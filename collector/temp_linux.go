//go:build linux

package collector

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// rawTemps reads the hwmon and thermal-zone sysfs interfaces.
func rawTemps() []Temp {
	sensors := hwmonTemps()
	sensors = append(sensors, thermalZoneTemps()...)
	return sensors
}

func hwmonTemps() []Temp {
	dirs, _ := filepath.Glob("/sys/class/hwmon/hwmon*")
	var out []Temp
	for _, d := range dirs {
		name := readTrim(filepath.Join(d, "name"))
		if name == "" {
			continue
		}
		inputs, _ := filepath.Glob(filepath.Join(d, "temp*_input"))
		for _, in := range inputs {
			raw := readTrim(in)
			val, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				continue
			}
			label := normModel(readTrim(strings.Replace(in, "_input", "_label", 1)))
			full := name
			if len(inputs) > 1 {
				if label != "" {
					full = name + " " + label
				} else {
					base := strings.TrimSuffix(filepath.Base(in), "_input")
					full = name + " " + base
				}
			}
			out = append(out, Temp{Label: full, Celsius: round1(val / 1000)})
		}
	}
	return out
}

func thermalZoneTemps() []Temp {
	dirs, _ := filepath.Glob("/sys/class/thermal/thermal_zone*")
	var out []Temp
	for _, d := range dirs {
		typ := readTrim(filepath.Join(d, "type"))
		raw := readTrim(filepath.Join(d, "temp"))
		if raw == "" {
			continue
		}
		val, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			continue
		}
		if typ == "" {
			typ = filepath.Base(d)
		}
		out = append(out, Temp{Label: typ, Celsius: round1(val / 1000)})
	}
	return out
}

func readTrim(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}
