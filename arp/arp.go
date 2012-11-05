package arp

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
)

type Entry struct {
	IP              net.IP
	HardwareAddress net.HardwareAddr
}

func ARP() ([]Entry, error) {
	entries := make([]Entry, 0)

	data, err := ioutil.ReadFile("/proc/net/arp")
	if err != nil {
		return entries, err
	}

	lines := bytes.Split(bytes.TrimSpace(data), []byte("\n"))
	for _, line := range lines[1:] {
		fields := bytes.Fields(line)
		ip := fields[0]
		hw := fields[3]
		mac, _ := net.ParseMAC(string(hw))
		entries = append(entries, Entry{net.ParseIP(string(ip)), mac})
	}

	return entries, nil
}

func IPToHardwareAddress(ip net.IP) (net.HardwareAddr, error) {
	entries, err := ARP()
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IP.Equal(ip) {
			return entry.HardwareAddress, nil
		}
	}

	return nil, fmt.Errorf("Unknown IP %s", ip)
}
