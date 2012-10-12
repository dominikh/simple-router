package traffic

import (
	"container/list"
	"fmt"
	"log"
	"sync"
	"time"
)

type ProgressiveStat struct {
	Stat
	Time   time.Time
	BPSIn  uint64
	BPSOut uint64
}

func (s ProgressiveStat) UnixMilliseconds() string {
	return fmt.Sprintf("%f", float64(s.Time.UnixNano())/1000000000)
}

func (s ProgressiveStat) String() string {
	return fmt.Sprintf(
		"%f;%s;%d;%d;%d;%d",
		float64(s.Time.UnixNano())/1000000000,
		s.Host,
		s.BPSIn,
		s.BPSOut,
		s.In,
		s.Out,
	)
}

type Monitor struct {
	Channels   *list.List
	Delay      time.Duration
	channelMap map[chan ProgressiveStat]*list.Element
	sync.RWMutex
}

func NewMonitor(delay time.Duration) *Monitor {
	return &Monitor{list.New(), delay, make(map[chan ProgressiveStat]*list.Element), sync.RWMutex{}}
}

// TODO mutexes for it all!
func (tm *Monitor) RegisterChannel(channel chan ProgressiveStat) {
	tm.Lock()
	defer tm.Unlock()
	element := tm.Channels.PushBack(channel)
	tm.channelMap[channel] = element
}

func (tm *Monitor) UnregisterChannel(channel chan ProgressiveStat) {
	tm.Lock()
	defer tm.Unlock()
	item := tm.channelMap[channel]
	if item != nil {
		tm.Channels.Remove(item)
		delete(tm.channelMap, channel)
	}
}

func (tm *Monitor) sendStat(stat *ProgressiveStat) {
	tm.RLock()
	defer tm.RUnlock()

	for e := tm.Channels.Front(); e != nil; e = e.Next() {
		select {
		case e.Value.(chan ProgressiveStat) <- *stat:
		case <-time.After(1 * time.Second):
			log.Println("Monitor.sendStat: Channel send timed out, unregistered channel.")
			channel := e.Value.(chan ProgressiveStat)
			tm.RUnlock()
			tm.UnregisterChannel(channel)
			tm.RLock()
		}
	}
}

func (tm *Monitor) Start() {
	var (
		first           bool = true
		last_time       time.Time
		last_statistics StatMap
	)

	for {
		this_time := time.Now()

		statistics, err := Statistics()
		if err != nil {
			log.Fatal(err)
		}

		for _, stat := range statistics {
			if !first && (stat.In > 0 || stat.Out > 0) {
				last_stat := last_statistics[stat.Host]
				time_diff := float64(this_time.Sub(last_time).Nanoseconds()) / 1000000000

				bytes_per_second_in := float64(stat.In-last_stat.In) / time_diff
				bytes_per_second_out := float64(stat.Out-last_stat.Out) / time_diff

				stat := ProgressiveStat{
					stat,
					this_time,
					uint64(bytes_per_second_in),
					uint64(bytes_per_second_out),
				}
				tm.sendStat(&stat)
			}
		}

		first = false
		last_time = this_time
		last_statistics = statistics
		time.Sleep(tm.Delay)
	}
}
