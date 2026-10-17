package collector

import (
	"sort"
	"strings"
)

// tempPrimaryKeys ranks the sensors we care about most (CPU/GPU/SoC/NVMe).
var tempPrimaryKeys = []string{
	"k10temp", "coretemp", "zenpower", "cpu", "soc", "package",
	"amdgpu", "gpu", "nvme", "edge", "junction", "composite",
}

// temps collects raw sensors from the platform, filters implausible readings,
// deduplicates by normalized label, prefers primary sensors, and caps the list.
func temps() []Temp {
	sensors := rawTemps()

	seen := map[string]bool{}
	var primary, other []Temp
	for _, t := range sensors {
		if t.Celsius <= 0 || t.Celsius > 150 {
			continue
		}
		key := normKey(t.Label)
		if seen[key] {
			continue
		}
		seen[key] = true
		if isPrimaryTemp(t.Label) {
			primary = append(primary, t)
		} else {
			other = append(other, t)
		}
	}
	if len(primary) > 0 {
		sensors = primary
	} else {
		sensors = other
	}
	if len(sensors) > 6 {
		sensors = sensors[:6]
	}
	sort.Slice(sensors, func(i, j int) bool { return sensors[i].Label < sensors[j].Label })
	return sensors
}

func isPrimaryTemp(label string) bool {
	l := strings.ToLower(label)
	for _, k := range tempPrimaryKeys {
		if strings.Contains(l, k) {
			return true
		}
	}
	return false
}

// normKey folds dashes/underscores/spaces so "gpu-thermal" == "gpu_thermal".
func normKey(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, " ", "")
	return s
}
