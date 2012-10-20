package monitor

import (
	"container/list"
	"log"
	"sync"
	"time"
)

type Monitor struct {
	sync.RWMutex
	Channels     *list.List
	Delay        time.Duration
	MonitorFunc  MonitorFunc
	channelMap   map[chan interface{}]*list.Element
	sleepChannel chan bool
}

type MonitorFunc func(*Monitor)

func NewMonitor(delay time.Duration, function MonitorFunc) *Monitor {
	return &Monitor{
		RWMutex:      sync.RWMutex{},
		Channels:     list.New(),
		Delay:        delay,
		MonitorFunc:  function,
		channelMap:   make(map[chan interface{}]*list.Element),
		sleepChannel: make(chan bool),
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
	go m.MonitorFunc(m)
	for {
		time.Sleep(m.Delay)
		m.sleepChannel <- true
	}
}

func (m *Monitor) Force() {
	m.sleepChannel <- true
}

func (m *Monitor) Wait() {
	<-m.sleepChannel
}
