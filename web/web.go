package main

import (
	"github.com/dominikh/simple-router/lookup"
	"github.com/dominikh/simple-router/monitor"
	"github.com/dominikh/simple-router/system"
	"github.com/dominikh/simple-router/traffic"

	"github.com/dominikh/conntrack"

	eventsource "github.com/dominikh/eventsource/http"

	"encoding/gob"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type NATEntry struct {
	Protocol           string
	SourceAddress      string
	SourcePort         uint16
	DestinationAddress string
	DestinationPort    uint16
	State              string
}

var captures = NewCaptureManager()

func trafficServer(es eventsource.EventSource, tm *monitor.Monitor) {
	ch := make(chan interface{}, 1)
	tm.RegisterChannel(ch)

	for item := range ch {
		stats := item.([]traffic.ProgressiveStat)

		json, _ := json.Marshal(stats)
		es.SendMessage(string(json), "", "")
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	IndexTemplate, _ := template.New("index.html").ParseFiles("index.html", "header.html", "footer.html")

	IndexTemplate.Execute(w, nil)
}

func natJsonHandler(w http.ResponseWriter, r *http.Request) {
	natEntries := getConntrackFlows().FilterByType(conntrack.SNATFilter | conntrack.DNATFilter)
	natEntriesDumbedDown := make([]NATEntry, len(natEntries))

	for index, entry := range natEntries {
		natEntriesDumbedDown[index] = NATEntry{
			entry.Protocol.Name,
			lookup.Resolve(entry.Original.Source, false),
			entry.Original.SPort,
			lookup.Resolve(entry.Original.Destination, false),
			entry.Original.DPort,
			entry.State,
		}
	}

	b, err := json.Marshal(natEntriesDumbedDown)
	if err != nil {
		panic(err)
	}

	w.Write(b)
}

func memoryUsageJsonHandler(w http.ResponseWriter, r *http.Request) {
	memory := system.GetMemoryStats()
	b, err := json.Marshal(memory)
	if err != nil {
		panic(err)
	}

	w.Write(b)
}

func uuidHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(strconv.FormatInt(int64(time.Now().UnixNano()), 10)))
}

func trafficCaptureHandler(w http.ResponseWriter, r *http.Request) {
	uuid := r.FormValue("uuid")
	iface := r.FormValue("interface")

	w.Header().Add("Content-disposition", "attachment; filename=capture_"+iface+"_"+strconv.FormatInt(time.Now().Unix(), 10)+".cap")

	fmt.Println(uuid)
	tcpdump := exec.Command("sudo", "tcpdump", "-Z", "admin", "-i", iface, "-w", "-")
	err := captures.AddCapture(uuid, tcpdump)
	if err != nil {
		// Not a unique UUID
		return
	}

	pipe, err := tcpdump.StdoutPipe()
	if err != nil {
		panic(err)
	}
	tcpdump.Start()

	// This will return when we kill the tcpdump process
	io.Copy(w, pipe)
}

func resolveIPHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	splits := strings.Split(path, "/")
	ip := net.ParseIP(splits[len(splits)-1])
	// FIXME check error
	hostname := lookup.Resolve(ip, false)
	io.WriteString(w, hostname)
}

func trafficCaptureStopHandler(w http.ResponseWriter, r *http.Request) {
	uuid := r.FormValue("uuid")
	cmd, ok := captures.GetCapture(uuid)
	if !ok {
		// Invalid uuid
		return
	}

	captures.RemoveCapture(uuid)

	// We need to find the child of the sudo process because we lack
	// the rights to directly kill the sudo process
	b, _ := exec.Command("pgrep", "-P", strconv.FormatInt(int64(cmd.Process.Pid), 10)).Output()
	output := strings.TrimSpace(string(b))
	if output == "" {
		// this should not happen...
		return
	}

	pid, _ := strconv.ParseInt(output, 10, 32)
	process, _ := os.FindProcess(int(pid))
	process.Signal(os.Interrupt)

	// Make sure the sudo process can terminate and doesn't go defunct
	cmd.Wait()
}

type dataInterfaces struct {
	LAN dataInterface
	WAN dataInterface
}

type dataInterface struct {
	MAC string
	IPs []string
}

func systemDataServer(es eventsource.EventSource, sm *monitor.Monitor) {
	ch := make(chan interface{}, 1)
	sm.RegisterChannel(ch)
	sm.Force()
	for item := range ch {
		data := item.(*system.Data)

		var lanIPs []string
		var wanIPs []string

		lanAddrs, _ := data.Interfaces.LAN.Addrs()
		wanAddrs, _ := data.Interfaces.WAN.Addrs()

		for _, addr := range lanAddrs {
			lanIPs = append(lanIPs, addr.String())
		}

		for _, addr := range wanAddrs {
			wanIPs = append(wanIPs, addr.String())
		}

		dataToSerialize := struct {
			Memory       system.MemoryStats
			Temperatures map[string]float64
			Interfaces   dataInterfaces
		}{
			data.Memory,
			data.Temperatures,
			dataInterfaces{
				dataInterface{
					data.Interfaces.LAN.HardwareAddr.String(),
					lanIPs,
				},
				dataInterface{
					data.Interfaces.WAN.HardwareAddr.String(),
					wanIPs,
				},
			},
		}
		json, _ := json.Marshal(dataToSerialize)
		es.SendMessage(string(json), "", "")
	}
}

func getConntrackFlows() conntrack.FlowSlice {
	var flows conntrack.FlowSlice

	cmd := exec.Command("sudo", "/home/admin/bin/dump_conntrack")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	cmd.Start()
	dec := gob.NewDecoder(stdout)
	dec.Decode(&flows)
	cmd.Wait()

	return flows
}

func main() {
	esTraffic := eventsource.New()
	esSystemData := eventsource.New()

	tm := monitor.NewMonitor(&traffic.Monitor{}, 500*time.Millisecond)
	sm := monitor.NewMonitor(&system.Monitor{}, 10*time.Second)

	go tm.Start()
	go sm.Start()

	go trafficServer(esTraffic, tm)
	go systemDataServer(esSystemData, sm)

	srv := &http.Server{
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
		Addr:         "192.168.1.1:8000",
	}

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/nat.json", natJsonHandler)
	http.HandleFunc("/memory_usage.json", memoryUsageJsonHandler)
	http.HandleFunc("/uuid", uuidHandler)
	http.HandleFunc("/traffic_capture", trafficCaptureHandler)
	http.HandleFunc("/stop_capture", trafficCaptureStopHandler)

	http.Handle("/live/traffic_data/", esTraffic)
	http.Handle("/live/system_data/", esSystemData)

	http.HandleFunc("/resolve_ip/", resolveIPHandler)

	err := srv.ListenAndServe()
	if err != nil {
		panic("ListenANdServe: " + err.Error())
	}
}
