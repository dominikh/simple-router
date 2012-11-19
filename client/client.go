package client

import (
	"github.com/dominikh/simple-router/arp"
	"github.com/dominikh/simple-router/dhcp"
	"github.com/dominikh/simple-router/lookup"
	"github.com/dominikh/simple-router/nat"

	"net"
)

type Client struct {
	IP       net.IP
	MAC      net.HardwareAddr
	Hostname string
}

func (client *Client) Connections() (count uint64) {
	entries, _ := nat.GetNAT(false)

	for _, entry := range entries {
		if entry.SourceAddress == client.IP.String() {
			count += 1
		}
	}

	return
}

// FIXME pass on errors
func Clients() ([]Client, error) {
	clients := make([]Client, 0)
	done := make(map[string]bool)

	leases, err := dhcp.Leases()
	if err != nil {
		return nil, err
	}
	for _, lease := range leases {
		if lease.BindingState != "active" {
			continue
		}

		if lease.Hostname == "" {
			lease.Hostname = lookup.Resolve(lease.IP, false)
		}
		clients = append(clients, Client{lease.IP, lease.HardwareAddress, lease.Hostname})
		done[lease.IP.String()] = true
	}

	natEntriesS, _ := nat.GetSNAT(false)
	natEntriesD, _ := nat.GetDNAT(false)
	natEntriesDSwapped := make([]nat.Entry, len(natEntriesD), len(natEntriesD))

	for index, natEntry := range natEntriesD {
		natEntry.SourceAddress, natEntry.SourcePort, natEntry.DestinationAddress, natEntry.DestinationPort = natEntry.DestinationAddress, natEntry.DestinationPort, natEntry.SourceAddress, natEntry.SourcePort
		natEntriesDSwapped[index] = natEntry
	}

	natEntriesCombined := append(natEntriesS, natEntriesDSwapped...)
	for _, natEntry := range natEntriesCombined {
		if done[natEntry.SourceAddress] {
			continue
		}

		ip := natEntry.SourceAddress
		hw, err := lookup.IPToHardwareAddress(net.ParseIP(ip))
		if err != nil {
			return nil, err
		}

		hostname, err := lookup.IPToHostname(net.ParseIP(ip))
		if err != nil {
			return nil, err
		}

		clients = append(clients, Client{net.ParseIP(ip), hw, hostname})

		done[natEntry.SourceAddress] = true
	}

	arpEntries, _ := arp.ARP()
	for _, arpEntry := range arpEntries {
		if done[arpEntry.IP.String()] {
			continue
		}

		hostname, err := lookup.IPToHostname(arpEntry.IP)
		if err != nil {
			return nil, err
		}

		clients = append(clients, Client{arpEntry.IP, arpEntry.HardwareAddress, hostname})

		done[arpEntry.IP.String()] = true
	}
	return clients, nil
}
