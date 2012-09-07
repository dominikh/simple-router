package nat

import (
	"net"
	"os/exec"
	"strings"
)

type Entry struct {
	Protocol           string
	SourceAddress      string
	SourcePort         string
	DestinationAddress string
	DestinationPort    string
	State              string
}

func GetDNAT(resolve bool) ([]Entry, error) {
	return getNAT("-D", resolve)
}

func GetSNAT(resolve bool) ([]Entry, error) {
	return getNAT("-S", resolve)
}

func GetNAT(resolve bool) ([]Entry, error) {
	return getNAT("", resolve)
}

func getNAT(flag string, resolve bool) ([]Entry, error){
	output, err := exec.Command("sudo", "/usr/bin/netstat-nat", "-o", "-n", flag).Output()
	if err != nil {
		return nil, err
	}

	return parseNetstat(string(output), resolve), nil
}

func parseNetstat(input string, resolve bool) []Entry {
	entries := make([]Entry, 0)

	lines := strings.Split(strings.TrimSpace(input), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		protocol := fields[0]
		source := fields[1]
		destination := fields[2]
		state := fields[3]

		sourceParts := strings.Split(source, ":")
		destinationParts := strings.Split(destination, ":")

		sourceAddress := sourceParts[0]
		sourcePort := sourceParts[1]

		destinationAddress := destinationParts[0]
		destinationPort := destinationParts[1]

		if resolve {
			lookup, err := net.LookupAddr(destinationAddress)
			if err == nil && len(lookup) > 0 {
				destinationAddress = lookup[0]
			}

			lookup, err = net.LookupAddr(sourceAddress)
			if err == nil && len(lookup) > 0 {
				sourceAddress = lookup[0]
			}
		}

		entries = append(entries, Entry{protocol, sourceAddress, sourcePort, destinationAddress, destinationPort, state})
	}

	return entries
}
