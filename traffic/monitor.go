package traffic

import (
	"fmt"
	"github.com/dominikh/simple-router/monitor"
	"log"
	"time"
)

type ProgressiveStat struct {
	Stat
	Time   time.Time
	BPSIn  uint64
	BPSOut uint64
}

func (s ProgressiveStat) UnixMilliseconds() string {
	return fmt.Sprintf("%f", float64(s.Time.UnixNano())/1000000000)
}

func (s ProgressiveStat) String() string {
	return fmt.Sprintf(
		"%f;%s;%d;%d;%d;%d",
		float64(s.Time.UnixNano())/1000000000,
		s.Host,
		s.BPSIn,
		s.BPSOut,
		s.In,
		s.Out,
	)
}

func NewMonitor(delay time.Duration) *monitor.Monitor {
	return monitor.NewMonitor(delay, func(m *monitor.Monitor) {
		var (
			first           bool = true
			last_time       time.Time
			last_statistics StatMap
		)

		for {
			this_time := time.Now()

			statistics, err := Statistics()
			if err != nil {
				log.Fatal(err)
			}

			for _, stat := range statistics {
				if !first && (stat.In > 0 || stat.Out > 0) {
					last_stat := last_statistics[stat.Host]
					time_diff := float64(this_time.Sub(last_time).Nanoseconds()) / 1000000000

					bytes_per_second_in := float64(stat.In-last_stat.In) / time_diff
					bytes_per_second_out := float64(stat.Out-last_stat.Out) / time_diff

					stat := ProgressiveStat{
						stat,
						this_time,
						uint64(bytes_per_second_in),
						uint64(bytes_per_second_out),
					}
					m.SendData(&stat)
				}
			}

			first = false
			last_time = this_time
			last_statistics = statistics
			time.Sleep(m.Delay)
		}
	})
}
