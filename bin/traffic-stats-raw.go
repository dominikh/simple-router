package main

import (
	"fmt"
	"github.com/dominikh/simple-router/traffic"
	"github.com/dominikh/simple-router/monitor"
	"time"
)

func main() {
	tm := monitor.NewMonitor(&traffic.Monitor{}, 500*time.Millisecond)
	ch := make(chan interface{})
	tm.RegisterChannel(ch)
	go tm.Start()
	for {
		stats := (<-ch).([]traffic.ProgressiveStat)
		for _, stat := range stats {
			fmt.Println(stat)
		}
	}
}
