package main

import (
	"github.com/dominikh/simple-router/lookup"
	"github.com/dominikh/simple-router/monitor"
	"github.com/dominikh/simple-router/traffic"

	"bytes"
	"fmt"
	"math"
	"net"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
)

const (
	maxIn  = 13107200
	maxOut = 655360
)

const header = "Host\t         Down\t           Up\tTotal down\t  Total up\n"

var colors = []int{237, 84, 83, 82, 46, 40, 226, 220, 214, 208, 202, 196}

func formatByteCount(bytes uint64) string {
	const format = "%.2f %siB"
	var units = []string{"K", "M", "G", "T", "P", "E"}

	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}

	exp := int(math.Floor(math.Log(float64(bytes)) / math.Log(1024)))

	return fmt.Sprintf(format, float64(bytes)/math.Pow(1024, float64(exp)), units[exp-1])
}

type statSlice []traffic.ProgressiveStat

func (s statSlice) Len() int {
	return len(s)
}

func (s statSlice) Less(i, j int) bool {
	return s[i].Host < s[j].Host
}

func (s statSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type lineColorizer struct {
	buf    bytes.Buffer
	colors []int
}

func (l *lineColorizer) Write(b []byte) (n int, err error) {
	n, err = l.buf.Write(b)
	return
}

func (l *lineColorizer) String() string {
	var buf bytes.Buffer
	for index, line := range strings.Split(l.buf.String(), "\n") {
		if line == "" {
			break
		}

		if index == 0 {
			buf.WriteString(fmt.Sprintf("\033[1m\033[38;05;%dm%s\033[0m\n", l.colors[index], line))
		} else {
			buf.WriteString(fmt.Sprintf("\033[38;05;%dm%s\033[0m\n", l.colors[index], line))
		}
	}

	return buf.String()
}

func main() {
	tm := monitor.NewMonitor(&traffic.Monitor{}, 500*time.Millisecond)
	ch := make(chan interface{})
	tm.RegisterChannel(ch)
	go tm.Start()

	hosts := make(map[string]string)
	for {
		lines := &lineColorizer{}
		stats := (<-ch).([]traffic.ProgressiveStat)

		os.Stdout.WriteString("\033[2J\033[;H")

		tabWriter := &tabwriter.Writer{}
		tabWriter.Init(lines, 0, 0, 2, ' ', 0)
		// The \b are a messy hack to remove the spaces that the
		// tabwriter inserts because it counts ansi colors as actual
		// characters
		tabWriter.Write([]byte("Host\t         Down\t           Up\t│ Total down\t\b\b\bTotal up\n"))
		lines.colors = append(lines.colors, 255)
		sort.Sort(statSlice(stats))

		for index, stat := range stats {
			var color_index int
			if stat.BPSIn == 0 && stat.BPSOut == 0 {
				color_index = 0
			} else if float64(stat.BPSIn)/maxIn > float64(stat.BPSOut)/maxOut {
				// IN is the culprit
				color_index = int(stat.BPSIn/(maxIn/uint64(len(colors))-1) + 1)
			} else {
				// OUT is the culprit
				color_index = int(stat.BPSOut/(maxOut/uint64(len(colors))-1) + 1)
			}

			if color_index >= len(colors) {
				color_index = len(colors) - 1
			}

			if index == len(stats)-1 {
				lines.colors = append(lines.colors, 255)
				tabWriter.Write([]byte("──────────────────────────────────────────────────────────────┼───────────────────────\n"))
			}

			lines.colors = append(lines.colors, colors[color_index])
			host, ok := hosts[stat.Host]
			if !ok {
				if stat.Host == "total" {
					host = "total"
				} else {
					host = lookup.Resolve(net.ParseIP(stat.Host), false)
					hosts[stat.Host] = host
				}
			}

			tabWriter.Write([]byte(fmt.Sprintf("%-30.30s\t%11s/s\t%11s/s\t\033[39m│ %10s\t%10s\n",
				host,
				formatByteCount(stat.BPSIn),
				formatByteCount(stat.BPSOut),
				formatByteCount(stat.In),
				formatByteCount(stat.Out))))
		}

		// fmt.Print(buf.String())
		tabWriter.Flush()
		os.Stdout.WriteString(lines.String())
	}
}
