package main

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dominikh/simple-router/hwmon"
	"github.com/dominikh/simple-router/lookup"
	"github.com/dominikh/simple-router/memory"
	"github.com/dominikh/simple-router/nat"
	"github.com/dominikh/simple-router/traffic"
	"html/template"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type InterfaceData struct {
	LAN *net.Interface
	WAN *net.Interface
}

type SystemData struct {
	Memory       memory.Stats
	Interfaces   InterfaceData
	Temperatures map[string]float64
}

type InternetData struct {
	WAN *net.Interface
}

type NATData struct {
	Connections []nat.Entry
}

type Packet struct {
	Type string
	Data interface{}
}

type Rate struct {
	Time     string
	Hostname string
	Host     string
	In       uint64
	Out      uint64
	TotalIn  uint64
	TotalOut uint64
}

var tm = traffic.NewMonitor(500 * time.Millisecond)
var captures = NewCaptureManager()

func trafficServer(ws *websocket.Conn) {
	ch := make(chan traffic.ProgressiveStat, 1)
	tm.RegisterChannel(ch)

	for {
		stat := <-ch
		var hostname string
		ip := net.ParseIP(stat.Host)
		if ip == nil {
			hostname = stat.Host
		} else {
			hostname, _ = lookup.IPToHostname(ip)
		}
		msg := Packet{"rate", &Rate{stat.UnixMilliseconds(), hostname, stat.Host, stat.BPSIn, stat.BPSOut, stat.In, stat.Out}}
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

func internetHandler(w http.ResponseWriter, r *http.Request) {
	InternetTemplate, _ := template.New("internet.html").Funcs(funcMap).ParseFiles("internet.html", "header.html", "footer.html")
	wanEth, _ := net.InterfaceByName("eth0")

	InternetTemplate.Execute(w, InternetData{wanEth})
}

func natJsonHandler(w http.ResponseWriter, r *http.Request) {
	natEntries, err := nat.GetNAT(true)
	if err != nil {
		panic(err)
	}

	b, err := json.Marshal(natEntries)
	if err != nil {
		panic(err)
	}

	w.Write(b)
}

func memoryUsageJsonHandler(w http.ResponseWriter, r *http.Request) {
	memory := memory.GetStats()
	b, err := json.Marshal(memory)
	if err != nil {
		panic(err)
	}

	w.Write(b)
}

func uuidHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(strconv.FormatInt(int64(time.Now().UnixNano()), 10)))
}

type CaptureManager struct {
	sync.RWMutex
	captures map[string]*exec.Cmd
}

func (cm *CaptureManager) AddCapture(uuid string, command *exec.Cmd) error {
	cm.Lock()
	defer cm.Unlock()

	_, ok := cm.captures[uuid]
	if ok {
		return errors.New("UUID already in use")
	}

	cm.captures[uuid] = command

	return nil
}

func (cm *CaptureManager) GetCapture(uuid string) (*exec.Cmd, bool) {
	cm.RLock()
	defer cm.RUnlock()

	capture, ok := cm.captures[uuid]
	return capture, ok
}

func (cm *CaptureManager) RemoveCapture(uuid string) {
	cm.Lock()
	defer cm.Unlock()

	delete(cm.captures, uuid)
}

func NewCaptureManager() *CaptureManager {
	return &CaptureManager{sync.RWMutex{}, make(map[string]*exec.Cmd)}
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

func trafficCaptureStopHandler(w http.ResponseWriter, r *http.Request) {
	uuid := r.FormValue("uuid")
	fmt.Println(uuid)
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
	for {
		memory := memory.GetStats()
		// usedPercentage := (float64(memory.Active) / float64(memory.Total)) * 100
		// buffersPercentage := (float64(memory.Buffers) / float64(memory.Total)) * 100
		// cachePercentage := (float64(memory.Cached) / float64(memory.Total)) * 100

		// usedText := formatByteCount(memory.Active*1000, 1024, -1)
		// buffersText := formatByteCount(memory.Buffers*1000, 1024, -1)
		// cacheText := formatByteCount(memory.Cached*1000, 1024, -1)

		// mData := MemoryData{usedText, usedPercentage, buffersText, buffersPercentage, cacheText, cachePercentage}

		lanEth, _ := net.InterfaceByName("eth1")
		wanEth, _ := net.InterfaceByName("eth0")

		iData := InterfaceData{lanEth, wanEth}

		mon1 := hwmon.HWMon{"hwmon0", []string{"2", "3"}}
		mon2 := hwmon.HWMon{"hwmon1", []string{"1"}}

		temps1, err := mon1.Temperatures()
		if err != nil {
			panic(err)
		}
		temps2, err := mon2.Temperatures()
		if err != nil {
			panic(err)
		}

		allTemps := make(map[string]float64)
		for key, value := range temps1 {
			allTemps[key] = value
		}

		for key, value := range temps2 {
			allTemps[key] = value
		}

		data := SystemData{memory, iData, allTemps}

		err = websocket.JSON.Send(ws, data)
		if err != nil {
			fmt.Println(err)
			break
		}

		time.Sleep(10 * time.Second)
	}
}

func main() {
	go tm.Start()

	srv := &http.Server{
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
		Addr:         "192.168.1.1:8000",
	}

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/internet/", internetHandler)
	http.HandleFunc("/nat.json", natJsonHandler)
	http.HandleFunc("/memory_usage.json", memoryUsageJsonHandler)
	http.HandleFunc("/uuid", uuidHandler)
	http.HandleFunc("/traffic_capture", trafficCaptureHandler)
	http.HandleFunc("/stop_capture", trafficCaptureStopHandler)

	http.Handle("/websocket/traffic_data/", websocket.Handler(trafficServer))
	http.Handle("/websocket/system_data/", websocket.Handler(systemDataServer))

	err := srv.ListenAndServe()
	if err != nil {
		panic("ListenANdServe: " + err.Error())
	}
}
