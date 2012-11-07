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

var Protocols []Protoent

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
		if len(fields) != 3 {
			continue
		}

		num, err := strconv.ParseInt(fields[1], 10, 32)
		if err != nil {
			panic(err)
		}

		Protocols = append(Protocols, Protoent{
			Name:    fields[0],
			Aliases: strings.Fields(fields[2]),
			Number:  int(num),
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
