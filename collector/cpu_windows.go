//go:build windows

package collector

import (
	"runtime"
	"syscall"
	"unsafe"
)

var (
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	procGetSystemTimes = kernel32.NewProc("GetSystemTimes")
	procGetTickCount64 = kernel32.NewProc("GetTickCount64")
)

// filetime is the Windows FILETIME layout (100ns intervals).
type filetime struct {
	lo uint32
	hi uint32
}

func (f filetime) u64() uint64 { return uint64(f.hi)<<32 | uint64(f.lo) }

// readCPUStat reads cumulative CPU times (idle / kernel / user).
// GetSystemTimes' kernel time already includes idle time, so the total is
// kernel + user while idle is reported separately — matching the Linux shape.
func readCPUStat() cpuStat {
	var idle, kernel, user filetime
	r, _, _ := procGetSystemTimes.Call(
		uintptr(unsafe.Pointer(&idle)),
		uintptr(unsafe.Pointer(&kernel)),
		uintptr(unsafe.Pointer(&user)),
	)
	if r == 0 {
		return cpuStat{}
	}
	return cpuStat{
		idle:  float64(idle.u64()),
		total: float64(kernel.u64() + user.u64()),
	}
}

func cpuCount() int { return runtime.NumCPU() }

// cpuModel reads the marketing name from the registry.
func cpuModel() string {
	return normModel(regReadString(`HARDWARE\DESCRIPTION\System\CentralProcessor\0`, "ProcessorNameString"))
}
