package dhcp

import (
	"io/ioutil"
	"net"
	"regexp"
	"strings"
)

var leaseRegexp = regexp.MustCompile("(?m:^lease (.+?) {\n((?s).+?)\\n}\n)")
var cleanupRegexp = regexp.MustCompile("^(binding state|next binding state)")

type Lease struct {
	// TODO Start, end, tstp, cltt
	IP              net.IP
	BindingState    string
	HardwareAddress net.HardwareAddr
	Hostname        string
	// TODO UID
}

func Leases() ([]Lease, error) {
	leases := make([]Lease, 0)
	mapLeases := make(map[string]Lease)

	buf, err := ioutil.ReadFile("/var/lib/dhcp/dhcpd.leases")
	if err != nil {
		return nil, err
	}

	for _, match := range leaseRegexp.FindAllStringSubmatch(string(buf), -1) {
		ip, data := match[1], match[2]
		values := make(map[string]string)

		for _, line := range strings.Split(data, "\n") {
			line = strings.TrimSpace(line)
			line = line[0 : len(line)-1]
			line = cleanupRegexp.ReplaceAllStringFunc(line, func(input string) string {
				return strings.Replace(input, " ", "-", -1)
			})
			fields := strings.SplitN(line, " ", 2)
			values[fields[0]] = fields[1]
		}

		ipAddr := net.ParseIP(ip)
		hwFields := strings.Fields(values["hardware"])
		mac, _ := net.ParseMAC(hwFields[len(hwFields)-1])

		hostname := values["client-hostname"]
		if hostname != "" {
			hostname = hostname[1 : len(hostname)-1]
		}

		lease := Lease{
			ipAddr,
			values["binding-state"],
			mac,
			hostname,
		}

		mapLeases[ip] = lease
	}

	for _, lease := range mapLeases {
		leases = append(leases, lease)
	}

	return leases, nil
}

func IPToHardwareAddress(ip net.IP) (net.HardwareAddr, error) {
	leases, err := Leases()
	if err != nil {
		return nil, err
	}

	for _, lease := range leases {
		if lease.IP.Equal(ip) {
			return lease.HardwareAddress, nil
		}
	}

	return nil, nil
}
