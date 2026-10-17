//go:build linux

package collector

import (
	"os"
	"strconv"
	"strings"
)

func memInfo() (Mem, Swap) {
	b, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return Mem{}, Swap{}
	}
	vals := map[string]uint64{}
	for _, line := range strings.Split(string(b), "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		f := strings.Fields(v)
		if len(f) == 0 {
			continue
		}
		n, _ := strconv.ParseUint(f[0], 10, 64)
		vals[strings.TrimSpace(k)] = n * 1024
	}
	total := vals["MemTotal"]
	avail := vals["MemAvailable"]
	if avail == 0 {
		avail = vals["MemFree"] + vals["Buffers"] + vals["Cached"]
	}
	used := uint64(0)
	if total > avail {
		used = total - avail
	}
	mem := Mem{Total: total, Used: used, Available: avail, UsedPercent: pct(used, total)}

	swTotal := vals["SwapTotal"]
	swFree := vals["SwapFree"]
	swUsed := uint64(0)
	if swTotal > swFree {
		swUsed = swTotal - swFree
	}
	swap := Swap{Total: swTotal, Used: swUsed, UsedPercent: pct(swUsed, swTotal)}
	return mem, swap
}

func loadavg() (float64, float64, float64) {
	b, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, 0, 0
	}
	f := strings.Fields(string(b))
	if len(f) < 3 {
		return 0, 0, 0
	}
	a, _ := strconv.ParseFloat(f[0], 64)
	c, _ := strconv.ParseFloat(f[1], 64)
	d, _ := strconv.ParseFloat(f[2], 64)
	return a, c, d
}

func uptime() float64 {
	b, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	f := strings.Fields(string(b))
	if len(f) == 0 {
		return 0
	}
	v, _ := strconv.ParseFloat(f[0], 64)
	return v
}
