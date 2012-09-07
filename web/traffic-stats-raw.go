package main

import (
	"fmt"
	"github.com/dominikh/simple-router/traffic"
	"time"
)

func main() {
	tm := traffic.NewMonitor(500 * time.Millisecond)
	ch := make(chan traffic.Stat)
	tm.RegisterChannel(ch)
	go tm.Start()
	for {
		stat := <-ch
		fmt.Println(stat)
	}
}
