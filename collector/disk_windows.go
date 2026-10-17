//go:build windows

package collector

import (
	"os"
	"sort"
	"strings"
	"syscall"
	"unsafe"
)

var (
	procGetLogicalDrives      = kernel32.NewProc("GetLogicalDrives")
	procGetDriveTypeW         = kernel32.NewProc("GetDriveTypeW")
	procGetDiskFreeSpaceExW   = kernel32.NewProc("GetDiskFreeSpaceExW")
	procGetVolumeInformationW = kernel32.NewProc("GetVolumeInformationW")
)

const driveFixed = 3

// diskMounts enumerates fixed drives (or DISK_MOUNTS="" env override).
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
	mask, _, _ := procGetLogicalDrives.Call()
	var out []string
	for i := 0; i < 26; i++ {
		if mask&(1<<uint(i)) == 0 {
			continue
		}
		root := string(rune('A'+i)) + `:\`
		if driveType(root) == driveFixed {
			out = append(out, root)
		}
	}
	if len(out) == 0 {
		out = []string{`C:\`}
	}
	return out
}

func driveType(root string) uintptr {
	p, err := syscall.UTF16PtrFromString(root)
	if err != nil {
		return 0
	}
	t, _, _ := procGetDriveTypeW.Call(uintptr(unsafe.Pointer(p)))
	return t
}

func disks(mounts []string) []Disk {
	var out []Disk
	seen := map[string]bool{}
	for _, m := range mounts {
		root := strings.TrimRight(m, `\`) + `\`
		key := strings.ToUpper(root)
		if seen[key] {
			continue
		}
		seen[key] = true

		total, free, avail, ok := spaceInfo(root)
		if !ok {
			continue
		}
		used := uint64(0)
		if total > free {
			used = total - free
		}
		out = append(out, Disk{
			Mount:       strings.TrimRight(root, `\`), // "C:"
			Fs:          volumeFs(root),
			Total:       total,
			Used:        used,
			Available:   avail,
			UsedPercent: pct(used, total),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Mount < out[j].Mount })
	return out
}

func spaceInfo(root string) (total, free, avail uint64, ok bool) {
	p, err := syscall.UTF16PtrFromString(root)
	if err != nil {
		return 0, 0, 0, false
	}
	var freeAvail, totalBytes, totalFree uint64
	r, _, _ := procGetDiskFreeSpaceExW.Call(
		uintptr(unsafe.Pointer(p)),
		uintptr(unsafe.Pointer(&freeAvail)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFree)),
	)
	if r == 0 {
		return 0, 0, 0, false
	}
	return totalBytes, totalFree, freeAvail, true
}

func volumeFs(root string) string {
	p, err := syscall.UTF16PtrFromString(root)
	if err != nil {
		return ""
	}
	buf := make([]uint16, 256)
	r, _, _ := procGetVolumeInformationW.Call(
		uintptr(unsafe.Pointer(p)),
		0, 0, 0, 0, 0,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
	)
	if r == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}
