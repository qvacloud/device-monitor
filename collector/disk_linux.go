//go:build linux

package collector

import (
	"os"
	"sort"
	"strings"
	"syscall"
)

var defaultMounts = []string{"/", "/home", "/data", "/storage/emulated/0"}

func diskMounts() []string {
	if env := strings.TrimSpace(os.Getenv("DISK_MOUNTS")); env != "" {
		var m []string
		for _, p := range strings.Split(env, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				m = append(m, p)
			}
		}
		return m
	}
	var out []string
	for _, m := range defaultMounts {
		if fi, err := os.Stat(m); err == nil && fi.IsDir() {
			out = append(out, m)
		}
	}
	if len(out) == 0 {
		out = []string{"/"}
	}
	return out
}

func disks(mounts []string) []Disk {
	fstypes := fsTypes()
	var out []Disk
	seen := map[syscall.Fsid]bool{}
	for _, m := range mounts {
		var st syscall.Statfs_t
		if err := syscall.Statfs(m, &st); err != nil {
			continue
		}
		if seen[st.Fsid] {
			continue
		}
		seen[st.Fsid] = true

		bsize := uint64(st.Bsize)
		total := st.Blocks * bsize
		free := st.Bfree * bsize
		avail := st.Bavail * bsize
		used := uint64(0)
		if total > free {
			used = total - free
		}
		out = append(out, Disk{
			Mount:       m,
			Fs:          fstypes[m],
			Total:       total,
			Used:        used,
			Available:   avail,
			UsedPercent: pct(used, total),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Mount < out[j].Mount })
	return out
}

func fsTypes() map[string]string {
	b, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return nil
	}
	m := map[string]string{}
	for _, line := range strings.Split(string(b), "\n") {
		f := strings.Fields(line)
		if len(f) >= 3 {
			m[f[1]] = f[2]
		}
	}
	return m
}
