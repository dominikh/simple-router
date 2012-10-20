package main

import (
	"errors"
	"os/exec"
	"sync"
)

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
