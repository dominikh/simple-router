package traffic

import (
	"os/exec"
	"strconv"
	"strings"
)

type Stat struct {
	Host string
	In   uint64
	Out  uint64
}

type StatMap map[string]Stat

func Statistics() (StatMap, error) {
	var (
		total_in  uint64
		total_out uint64
		stats     StatMap = make(StatMap)
	)

	output_in, err := exec.Command("sudo", "/sbin/iptables", "-L", "TRAFFIC_IN", "-v", "-n", "-x").Output()
	if err != nil {
		return stats, err
	}

	output_out, err := exec.Command("sudo", "/sbin/iptables", "-L", "TRAFFIC_OUT", "-v", "-n", "-x").Output()
	if err != nil {
		return stats, err
	}

	entries_in := strings.Split(strings.TrimSpace(string(output_in)), "\n")[2:]
	entries_out := strings.Split(strings.TrimSpace(string(output_out)), "\n")[2:]

	for index, entry_in := range entries_in {
		entry_out := entries_out[index]

		parts_in := strings.Fields(entry_in)
		parts_out := strings.Fields(entry_out)

		bytes_in, _ := strconv.ParseUint(parts_in[1], 10, 64)
		bytes_out, _ := strconv.ParseUint(parts_out[1], 10, 64)
		destination := strings.TrimSpace(parts_in[8])

		stats[destination] = Stat{destination, bytes_in, bytes_out}

		total_in += bytes_in
		total_out += bytes_out
	}

	stats["total"] = Stat{"total", total_in, total_out}

	return stats, nil
}
