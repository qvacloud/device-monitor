//go:build windows

package collector

import "unsafe"

var procGlobalMemoryStatusEx = kernel32.NewProc("GlobalMemoryStatusEx")

type memoryStatusEx struct {
	length               uint32
	memoryLoad           uint32
	totalPhys            uint64
	availPhys            uint64
	totalPageFile        uint64
	availPageFile        uint64
	totalVirtual         uint64
	availVirtual         uint64
	availExtendedVirtual uint64
}

func memInfo() (Mem, Swap) {
	ms := memoryStatusEx{length: uint32(unsafe.Sizeof(memoryStatusEx{}))}
	if r, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&ms))); r == 0 {
		return Mem{}, Swap{}
	}
	used := uint64(0)
	if ms.totalPhys > ms.availPhys {
		used = ms.totalPhys - ms.availPhys
	}
	mem := Mem{Total: ms.totalPhys, Used: used, Available: ms.availPhys, UsedPercent: pct(used, ms.totalPhys)}

	swTotal := ms.totalPageFile
	swFree := ms.availPageFile
	swUsed := uint64(0)
	if swTotal > swFree {
		swUsed = swTotal - swFree
	}
	swap := Swap{Total: swTotal, Used: swUsed, UsedPercent: pct(swUsed, swTotal)}
	return mem, swap
}

// Windows has no load average concept; report zeros.
func loadavg() (float64, float64, float64) { return 0, 0, 0 }

// uptime returns milliseconds-since-boot from GetTickCount64, in seconds.
func uptime() float64 {
	r, _, _ := procGetTickCount64.Call()
	return float64(r) / 1000.0
}
