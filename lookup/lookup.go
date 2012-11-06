package lookup

import (
	"github.com/dominikh/simple-router/arp"
	"github.com/dominikh/simple-router/dhcp"
	"net"
)

func Resolve(ip net.IP, noop bool) string {
	if noop {
		return ip.String()
	}

	lookup, err := net.LookupAddr(ip.String())
	if err == nil && len(lookup) > 0 {
		return lookup[0]
	}

	return ip.String()
}

func IPToHostname(ip net.IP) (string, error) {
	lookup, _ := net.LookupAddr(ip.String())

	if len(lookup) > 0 {
		return lookup[0], nil
	}

	hostname, err := dhcp.IPToHostname(ip)
	if err != nil {
		return "", err
	}

	if hostname == "" {
		hostname = ip.String()
	}

	return hostname, nil
}

func IPToHardwareAddress(ip net.IP) (net.HardwareAddr, error) {
	hw, err := dhcp.IPToHardwareAddress(ip)
	if err != nil {
		return nil, err
	}
	if hw != nil {
		return hw, nil
	}

	hw, err = arp.IPToHardwareAddress(ip)
	return hw, err
}
