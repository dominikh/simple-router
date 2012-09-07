package arp

import (
	"net"
	"os/exec"
	"regexp"
	"strings"
)

// TODO allow different interfaces

type Entry struct {
	IP              net.IP
	HardwareAddress net.HardwareAddr
}

var re = regexp.MustCompile("^\\S+ \\((.+?)\\) at (\\S+)")

func ARP() ([]Entry, error) {
	entries := make([]Entry, 0)

	output, err := exec.Command("/usr/sbin/arp", "-a", "-n", "-i", "eth1").Output()
	if err != nil {
		return entries, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		matches := re.FindStringSubmatch(line)
		mac, _ := net.ParseMAC(matches[2])
		entries = append(entries, Entry{net.ParseIP(matches[1]), mac})
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

	return nil, nil
}
