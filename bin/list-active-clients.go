package main

import (
	"github.com/dominikh/simple-router/client"
	"fmt"
)

func main() {
    clients, err := client.Clients()
	if err != nil {
		panic(err)
	}

	for _, client := range clients {
		hostname := client.Hostname
		if hostname == "" {
			hostname = "-"
		}
		fmt.Printf("%s\t%s\t%s\t%d\n", client.IP, client.MAC, hostname, client.Connections())
	}
}
