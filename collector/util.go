package collector

import "math"

// pct returns used/total as a percentage rounded to one decimal.
func pct(used, total uint64) float64 {
	if total == 0 {
		return 0
	}
	return round1(float64(used) / float64(total) * 100)
}

// round1 rounds to one decimal, mapping NaN/Inf to zero.
func round1(f float64) float64 {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0
	}
	return math.Round(f*10) / 10
}
