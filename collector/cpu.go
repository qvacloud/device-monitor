package collector

import "math"

// cpuStat holds cumulative CPU tick counters (Linux "jiffies", Windows 100ns).
// idle is the cumulative idle time, total is the cumulative total time.
type cpuStat struct {
	idle  float64
	total float64
}

// usage returns the busy percentage between two cumulative samples.
func usage(prev, cur cpuStat) float64 {
	dt := cur.total - prev.total
	di := cur.idle - prev.idle
	if dt <= 0 {
		return 0
	}
	p := (1 - di/dt) * 100
	if p < 0 {
		p = 0
	}
	if p > 100 {
		p = 100
	}
	return math.Round(p*10) / 10
}
