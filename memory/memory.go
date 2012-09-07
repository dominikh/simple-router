package memory

import (
	"io/ioutil"
	"strconv"
	"strings"
)

type Stats struct {
	Total   uint64
	Free    uint64
	Buffers uint64
	Cached  uint64
	Active  uint64
}

func GetStats() Stats {
	values := make(map[string]uint64)
	b, err := ioutil.ReadFile("/proc/meminfo")
	if err != nil {
		panic(err)
	}

	for _, line := range strings.Split(string(b), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		key := fields[0][0 : len(fields[0])-1]
		value, _ := strconv.ParseUint(fields[1], 10, 64)
		values[key] = value
	}

	return Stats{
		values["MemTotal"],
		values["MemFree"],
		values["Buffers"],
		values["Cached"],
		values["MemTotal"] - values["MemFree"] - values["Buffers"] - values["Cached"],
	}
}
