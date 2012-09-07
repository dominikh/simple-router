package hwmon

import (
	"io/ioutil"
	"strconv"
	"strings"
)

type HWMon struct {
	Name    string
	Sensors []string
}

func (mon HWMon) Temperatures() (map[string]float64, error) {
	temps := make(map[string]float64)

	for _, sensor := range mon.Sensors {
		name, err := ioutil.ReadFile("/sys/class/hwmon/" + mon.Name + "/device/temp" + sensor + "_label")
		if err != nil {
			return nil, err
		}

		temp, err := ioutil.ReadFile("/sys/class/hwmon/" + mon.Name + "/device/temp" + sensor + "_input")
		if err != nil {
			return nil, err
		}

		float, err := strconv.ParseFloat(strings.TrimSpace(string(temp)), 64)
		if err != nil {
			return nil, err
		}
		temps[string(name)] = float / 1000

	}

	return temps, nil
}
