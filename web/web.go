package main

import (
	"github.com/dominikh/simple-router/conntrack"
	"github.com/dominikh/simple-router/lookup"
	"github.com/dominikh/simple-router/system"
	"github.com/dominikh/simple-router/traffic"

	"code.google.com/p/go.net/websocket"

	"encoding/gob"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type InternetData struct {
	WAN *net.Interface
}

type Packet struct {
	Type string
	Data interface{}
}

type Rate struct {
	Time     string
	Host     string
	In       uint64
	Out      uint64
	TotalIn  uint64
	TotalOut uint64
}

type NATEntry struct {
	Protocol           string
	SourceAddress      string
	SourcePort         uint16
	DestinationAddress string
	DestinationPort    uint16
	State              string
}

var tm = traffic.NewMonitor(500 * time.Millisecond)
var sm = system.NewMonitor(10 * time.Second)
var captures = NewCaptureManager()

func trafficServer(ws *websocket.Conn) {
	ch := make(chan interface{}, 1)
	tm.RegisterChannel(ch)

	for item := range ch {
		stat := item.(*traffic.ProgressiveStat)

		msg := Packet{"rate", &Rate{stat.UnixMilliseconds(), stat.Host, stat.BPSIn, stat.BPSOut, stat.In, stat.Out}}
		err := websocket.JSON.Send(ws, msg)
		if err != nil {
			fmt.Println(err)
			tm.UnregisterChannel(ch)
			break
		}
	}
}

func formatByteCount(bytes uint64, base uint16, force int8) string {
	var exp uint8
	var units [6]string
	filler := ""
	format := "%.2f %s%sB"

	if bytes < uint64(base) {
		return fmt.Sprintf(format, bytes, "", "")
	}

	if force >= 0 {
		exp = uint8(force)
	} else {
		exp = uint8(math.Floor(math.Log10(float64(bytes)) / math.Log10(float64(base))))
	}

	if base == 1000 {
		units = [6]string{"k", "M", "G", "T", "P", "E"}
	} else {
		units = [6]string{"K", "M", "G", "T", "P", "E"}
		if exp > 0 {
			filler = "i"
		}
	}

	return fmt.Sprintf(format, float64(bytes)/math.Pow(float64(base), float64(exp)), units[exp-1], filler)
}

var funcMap = template.FuncMap{
	"list": func(addrs []net.Addr) string {
		ips := make([]string, 0, len(addrs))
		for _, addr := range addrs {
			ips = append(ips, addr.String())
		}

		return strings.Join(ips, ", ")
	},

	"downcase": func(s string) string {
		return strings.ToLower(s)
	},

	"temperatures": func(temperatures map[string]float64) string {
		temps := make([]string, 0, len(temperatures))
		for _, temp := range temperatures {
			temps = append(temps, fmt.Sprintf("%.2f°C", temp))
		}

		return strings.Join(temps, " / ")
	},

	"firstAddr": func(addrs []net.Addr) string {
		return addrs[0].String()
	},
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	IndexTemplate, _ := template.New("index.html").Funcs(funcMap).ParseFiles("index.html", "header.html", "footer.html")

	IndexTemplate.Execute(w, nil)
}

func natJsonHandler(w http.ResponseWriter, r *http.Request) {
	natEntries := conntrack.Filter(getConntrackFlows(), conntrack.SNATFilter|conntrack.DNATFilter)
	natEntriesDumbedDown := make([]NATEntry, len(natEntries))

	for index, entry := range natEntries {
		natEntriesDumbedDown[index] = NATEntry{
			entry.Protocol,
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
	hostname, _ := lookup.IPToHostname(ip)
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

func systemDataServer(ws *websocket.Conn) {
	ch := make(chan interface{}, 1)
	sm.RegisterChannel(ch)
	sm.Force()
	for item := range ch {
		data := item.(*system.Data)

		err := websocket.JSON.Send(ws, data)
		if err != nil {
			fmt.Println(err)
			sm.UnregisterChannel(ch)
			break
		}
	}
}

func getConntrackFlows() []conntrack.Flow {
	var flows []conntrack.Flow

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
	go tm.Start()
	go sm.Start()

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

	http.Handle("/websocket/traffic_data/", websocket.Handler(trafficServer))
	http.Handle("/websocket/system_data/", websocket.Handler(systemDataServer))

	http.HandleFunc("/resolve_ip/", resolveIPHandler)

	err := srv.ListenAndServe()
	if err != nil {
		panic("ListenANdServe: " + err.Error())
	}
}
