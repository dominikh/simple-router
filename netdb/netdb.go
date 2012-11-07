package netdb

import (
	"errors"
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
		if len(fields) < 3 {
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
		if len(fields) < 3 {
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

func GetProtoByNumber(num int) (Protoent, error) {
	for _, protoent := range Protocols {
		if protoent.Number == num {
			return protoent, nil
		}
	}
	return Protoent{}, errors.New("Unknown protocol number")
}

func GetProtoByName(name string) (Protoent, error) {
	for _, protoent := range Protocols {
		if protoent.Name == name {
			return protoent, nil
		}

		for _, alias := range protoent.Aliases {
			if alias == name {
				return protoent, nil
			}
		}
	}

	return Protoent{}, errors.New("Unknown protocol number")
}

func GetServByName(name, protocol string) (Servent, error) {
	for _, servent := range Services {
		if servent.Protocol != protocol && protocol != "" {
			continue
		}

		if servent.Name == name {
			return servent, nil
		}

		for _, alias := range servent.Aliases {
			if alias == name {
				return servent, nil
			}
		}
	}

	return Servent{}, errors.New("Unknown service name/protocol combination")
}

func GetServByPort(port int, protocol string) (Servent, error) {
	for _, servent := range Services {
		if servent.Port == port && (servent.Protocol == protocol || protocol == "") {
			return servent, nil
		}
	}

	return Servent{}, errors.New("Unknown port/protocol combination")
}
