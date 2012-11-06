package nat

import (
	"github.com/dominikh/simple-router/conntrack"

	"net"
)

type Flag uint8

const (
	SNAT Flag = 1 << iota
	DNAT
	Routed
	Local
)

var localIPs = make([]*net.IPNet, 0)

func isLocalIP(ip net.IP) bool {
	for _, localIP := range localIPs {
		if localIP.IP.Equal(ip) {
			return true
		}
	}

	return false
}

func init() {
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		panic(err)
	}

	for _, address := range addresses {
		localIPs = append(localIPs, address.(*net.IPNet))
	}
}

func GetSNAT(flows []conntrack.Flow) []conntrack.Flow {
	return GetNAT(flows, SNAT)
}

func GetDNAT(flows []conntrack.Flow) []conntrack.Flow {
	return GetNAT(flows, DNAT)
}

func GetRouted(flows []conntrack.Flow) []conntrack.Flow {
	return GetNAT(flows, Routed)
}

func GetLocal(flows []conntrack.Flow) []conntrack.Flow {
	return GetNAT(flows, Local)
}

func GetNAT(flows []conntrack.Flow, which Flag) []conntrack.Flow {
	natFlows := make([]conntrack.Flow, 0, len(flows))

	snat := (which & SNAT) > 0
	dnat := (which & DNAT) > 0
	local := (which & Local) > 0
	routed := (which & Routed) > 0

	for _, flow := range flows {
		if (snat && isSNAT(flow)) ||
			(dnat && isDNAT(flow)) ||
			(local && isLocal(flow)) ||
			(routed && isRouted(flow)) {

			natFlows = append(natFlows, flow)
		}
	}

	return natFlows
}

func isSNAT(flow conntrack.Flow) bool {
	// SNATed flows should reply to our WAN IP, not a LAN IP.
	if flow.Original.Source.Equal(flow.Reply.Destination) {
		return false
	}

	if !flow.Original.Destination.Equal(flow.Reply.Source) {
		return false
	}

	return true
}

func isDNAT(flow conntrack.Flow) bool {
	// Reply must go back to the source; Reply mustn't come from the WAN IP
	if flow.Original.Source.Equal(flow.Reply.Destination) && !flow.Original.Destination.Equal(flow.Reply.Source) {
		return true
	}

	// Taken straight from original netstat-nat, labelled "DNAT (1 interface)"
	if !flow.Original.Source.Equal(flow.Reply.Source) && !flow.Original.Source.Equal(flow.Reply.Destination) && !flow.Original.Destination.Equal(flow.Reply.Source) && flow.Original.Destination.Equal(flow.Reply.Destination) {
		return true
	}

	return false
}

func isLocal(flow conntrack.Flow) bool {
	// no NAT
	if flow.Original.Source.Equal(flow.Reply.Destination) && flow.Original.Destination.Equal(flow.Reply.Source) {
		// At least one local address
		if isLocalIP(flow.Original.Source) || isLocalIP(flow.Original.Destination) || isLocalIP(flow.Reply.Source) || isLocalIP(flow.Reply.Destination) {
			return true
		}
	}

	return false
}

func isRouted(flow conntrack.Flow) bool {
	// no NAT
	if flow.Original.Source.Equal(flow.Reply.Destination) && flow.Original.Destination.Equal(flow.Reply.Source) {
		// No local addresses
		if !isLocalIP(flow.Original.Source) && !isLocalIP(flow.Original.Destination) && !isLocalIP(flow.Reply.Source) && !isLocalIP(flow.Reply.Destination) {
			return true
		}
	}

	return false
}
