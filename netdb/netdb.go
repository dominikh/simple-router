package netdb

import (
	"io/ioutil"
	"strconv"
	"strings"
)

type Protoent struct {
	Name    string
	Aliases []string
	Number  int
}

type Servent struct {
	Name     string
	Aliases  []string
	Port     int
	Protocol string
}

var Protocols []Protoent
var Services []Servent

func init() {
	// Load protocols
	data, err := ioutil.ReadFile("/etc/protocols")
	if err != nil {
		panic(err)
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		split := strings.SplitN(line, "#", 2)
		fields := strings.Fields(split[0])
		if len(fields) < 2 {
			continue
		}

		num, err := strconv.ParseInt(fields[1], 10, 32)
		if err != nil {
			panic(err)
		}

		Protocols = append(Protocols, Protoent{
			Name:    fields[0],
			Aliases: fields[2:],
			Number:  int(num),
		})
	}

	// Load services
	data, err = ioutil.ReadFile("/etc/services")
	if err != nil {
		panic(err)
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		split := strings.SplitN(line, "#", 2)
		fields := strings.Fields(split[0])
		if len(fields) < 2 {
			continue
		}

		name := fields[0]
		portproto := strings.SplitN(fields[1], "/", 2)
		port, err := strconv.ParseInt(portproto[0], 10, 32)
		if err != nil {
			panic(err)
		}

		proto := portproto[1]
		aliases := fields[2:]

		Services = append(Services, Servent{
			Name:     name,
			Aliases:  aliases,
			Port:     int(port),
			Protocol: proto,
		})
	}
}

func (this Protoent) Equal(other Protoent) bool {
	return this.Number == other.Number
}

func GetProtoByNumber(num int) (Protoent, bool) {
	for _, protoent := range Protocols {
		if protoent.Number == num {
			return protoent, true
		}
	}
	return Protoent{}, false
}

func GetProtoByName(name string) (Protoent, bool) {
	for _, protoent := range Protocols {
		if protoent.Name == name {
			return protoent, true
		}

		for _, alias := range protoent.Aliases {
			if alias == name {
				return protoent, true
			}
		}
	}

	return Protoent{}, false
}

func GetServByName(name, protocol string) (Servent, bool) {
	for _, servent := range Services {
		if servent.Protocol != protocol && protocol != "" {
			continue
		}

		if servent.Name == name {
			return servent, true
		}

		for _, alias := range servent.Aliases {
			if alias == name {
				return servent, true
			}
		}
	}

	return Servent{}, false
}

func GetServByPort(port int, protocol string) (Servent, bool) {
	for _, servent := range Services {
		if servent.Port == port && (servent.Protocol == protocol || protocol == "") {
			return servent, true
		}
	}

	return Servent{}, false
}
