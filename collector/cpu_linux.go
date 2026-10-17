//go:build linux

package collector

import (
	"os"
	"strconv"
	"strings"
)

// readCPUStat reads the aggregate CPU counters from /proc/stat.
func readCPUStat() cpuStat {
	b, err := os.ReadFile("/proc/stat")
	if err != nil {
		return cpuStat{}
	}
	for _, line := range strings.Split(string(b), "\n") {
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		fields := strings.Fields(line)[1:]
		var vals []float64
		for _, f := range fields {
			v, _ := strconv.ParseFloat(f, 64)
			vals = append(vals, v)
		}
		if len(vals) < 4 {
			return cpuStat{}
		}
		var total float64
		for _, v := range vals {
			total += v
		}
		idle := vals[3]
		if len(vals) > 4 {
			idle += vals[4] // iowait counts as idle-ish
		}
		return cpuStat{idle: idle, total: total}
	}
	return cpuStat{}
}

// cpuCount counts the "cpuN" lines in /proc/stat.
func cpuCount() int {
	b, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 1
	}
	n := 0
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "cpu") && !strings.HasPrefix(line, "cpu ") {
			n++
		}
	}
	if n == 0 {
		n = 1
	}
	return n
}

// cpuModel extracts a human model name from /proc/cpuinfo.
func cpuModel() string {
	b, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return ""
	}
	var hw, model, processor string
	for _, line := range strings.Split(string(b), "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = normModel(v)
		switch k {
		case "model name":
			if model == "" {
				model = v
			}
		case "Hardware":
			if hw == "" {
				hw = v
			}
		case "Model":
			if processor == "" {
				processor = v
			}
		}
	}
	if model != "" {
		return model
	}
	if hw != "" {
		return hw
	}
	return processor
}
