package monitor

import (
	"container/list"
	"log"
	"sync"
	"time"
)

type Monitor struct {
	sync.RWMutex
	Channels   *list.List
	Delay      time.Duration
	Runnable   Runnable
	channelMap map[chan interface{}]*list.Element
}

type Runnable interface {
	Run(*Monitor)
}

func NewMonitor(runnable Runnable, delay time.Duration) *Monitor {
	return &Monitor{
		RWMutex:    sync.RWMutex{},
		Channels:   list.New(),
		Delay:      delay,
		Runnable:   runnable,
		channelMap: make(map[chan interface{}]*list.Element),
	}
}

func (m *Monitor) RegisterChannel(channel chan interface{}) {
	m.Lock()
	defer m.Unlock()
	element := m.Channels.PushBack(channel)
	m.channelMap[channel] = element
}

func (m *Monitor) UnregisterChannel(channel chan interface{}) {
	m.Lock()
	defer m.Unlock()
	item := m.channelMap[channel]
	if item != nil {
		m.Channels.Remove(item)
		delete(m.channelMap, channel)
	}
}

func (m *Monitor) SendData(data interface{}) {
	m.RLock()
	defer m.RUnlock()

	for e := m.Channels.Front(); e != nil; e = e.Next() {
		select {
		case e.Value.(chan interface{}) <- data:
		case <-time.After(1 * time.Second):
			log.Println("Monitor.sendStat: Channel send timed out, unregistered channel.")
			channel := e.Value.(chan interface{})
			m.RUnlock()
			m.UnregisterChannel(channel)
			m.RLock()
		}
	}
}

func (m *Monitor) Start() {
	for {
		go m.Runnable.Run(m)
		time.Sleep(m.Delay)
	}
}

func (m *Monitor) Force() {
	go m.Runnable.Run(m)
}
