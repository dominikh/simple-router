package traffic

import (
	"container/list"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Stat struct {
	Time     time.Time
	Host     string
	BPSIn    uint64
	BPSOut   uint64
	TotalIn  uint64
	TotalOut uint64
}

func (s Stat) UnixMilliseconds() string {
	return fmt.Sprintf("%f", float64(s.Time.UnixNano())/1000000000)
}

func (s Stat) String() string {
	return fmt.Sprintf(
		"%f;%s;%d;%d;%d;%d",
		float64(s.Time.UnixNano())/1000000000,
		s.Host,
		s.BPSIn,
		s.BPSOut,
		s.TotalIn,
		s.TotalOut,
	)
}

type Monitor struct {
	Channels   *list.List
	Delay      time.Duration
	channelMap map[chan Stat]*list.Element
	sync.RWMutex
}

func NewMonitor(delay time.Duration) *Monitor {
	return &Monitor{list.New(), delay, make(map[chan Stat]*list.Element), sync.RWMutex{}}
}

// TODO mutexes for it all!
func (tm *Monitor) RegisterChannel(channel chan Stat) {
	tm.Lock()
	defer tm.Unlock()
	element := tm.Channels.PushBack(channel)
	tm.channelMap[channel] = element
}

func (tm *Monitor) UnregisterChannel(channel chan Stat) {
	tm.Lock()
	defer tm.Unlock()
	item := tm.channelMap[channel]
	if item != nil {
		tm.Channels.Remove(item)
		delete(tm.channelMap, channel)
	}
}

func (tm *Monitor) sendStat(stat *Stat) {
	tm.RLock()
	defer tm.RUnlock()

	for e := tm.Channels.Front(); e != nil; e = e.Next() {
		select {
		case e.Value.(chan Stat) <- *stat:
		case <- time.After(1 * time.Second):
			log.Println("Monitor.sendStat: Channel send timed out, unregistered channel.")
			channel := e.Value.(chan Stat)
			tm.RUnlock()
			tm.UnregisterChannel(channel)
			tm.RLock()
		}
	}
}

func (tm *Monitor) Start() {
	last_bytes_in := make(map[string]uint64)
	last_bytes_out := make(map[string]uint64)
	first := true
	var last_time time.Time
	for {
		this_time := time.Now()

		output_in, err := exec.Command("sudo", "/sbin/iptables", "-L", "TRAFFIC_IN", "-v", "-n", "-x").Output()
		if err != nil {
			log.Fatal(err)
		}

		output_out, err := exec.Command("sudo", "/sbin/iptables", "-L", "TRAFFIC_OUT", "-v", "-n", "-x").Output()
		if err != nil {
			log.Fatal(err)
		}

		entries_in := strings.Split(strings.TrimSpace(string(output_in)), "\n")[2:]
		entries_out := strings.Split(strings.TrimSpace(string(output_out)), "\n")[2:]

		var total_in uint64
		var total_out uint64
		var total_per_second_in float64
		var total_per_second_out float64

		for index, entry_in := range entries_in {
			entry_out := entries_out[index]

			parts_in := strings.Fields(entry_in)
			parts_out := strings.Fields(entry_out)

			bytes_in, _ := strconv.ParseUint(parts_in[1], 10, 64)
			bytes_out, _ := strconv.ParseUint(parts_out[1], 10, 64)
			destination := strings.TrimSpace(parts_in[8])

			if !first && (bytes_in > 0 || bytes_out > 0) {
				time_diff := float64(this_time.Sub(last_time).Nanoseconds()) / 1000000000

				bytes_per_second_in := float64(bytes_in-last_bytes_in[destination]) / time_diff
				bytes_per_second_out := float64(bytes_out-last_bytes_out[destination]) / time_diff

				total_per_second_in += bytes_per_second_in
				total_per_second_out += bytes_per_second_out

				stat := Stat{
					this_time,
					destination,
					uint64(bytes_per_second_in),
					uint64(bytes_per_second_out),
					bytes_in, bytes_out,
				}

				tm.sendStat(&stat)
			}

			last_bytes_in[destination] = bytes_in
			last_bytes_out[destination] = bytes_out

			total_in += bytes_in
			total_out += bytes_out
		}

		stat := Stat{
			this_time,
			"total",
			uint64(total_per_second_in),
			uint64(total_per_second_out),
			total_in,
			total_out,
		}

		tm.sendStat(&stat)

		first = false
		last_time = this_time
		time.Sleep(tm.Delay)
	}
}
