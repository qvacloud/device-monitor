//go:build windows

package collector

import (
	"syscall"
	"unsafe"
)

var (
	advapi32             = syscall.NewLazyDLL("advapi32.dll")
	procRegOpenKeyExW    = advapi32.NewProc("RegOpenKeyExW")
	procRegQueryValueExW = advapi32.NewProc("RegQueryValueExW")
	procRegCloseKey      = advapi32.NewProc("RegCloseKey")
)

const (
	hkeyLocalMachine = 0x80000002
	keyRead          = 0x20019
)

// regReadString reads a REG_SZ value under HKLM\<subkey>. Returns "" on error.
func regReadString(subkey, value string) string {
	k, err := syscall.UTF16PtrFromString(subkey)
	if err != nil {
		return ""
	}
	v, err := syscall.UTF16PtrFromString(value)
	if err != nil {
		return ""
	}
	var hkey uintptr
	if r, _, _ := procRegOpenKeyExW.Call(hkeyLocalMachine, uintptr(unsafe.Pointer(k)), 0, keyRead, uintptr(unsafe.Pointer(&hkey))); r != 0 {
		return ""
	}
	defer procRegCloseKey.Call(hkey)

	var typ, size uint32
	if r, _, _ := procRegQueryValueExW.Call(hkey, uintptr(unsafe.Pointer(v)), 0, uintptr(unsafe.Pointer(&typ)), 0, uintptr(unsafe.Pointer(&size))); r != 0 || size == 0 {
		return ""
	}
	buf := make([]uint16, size/2+1)
	if r, _, _ := procRegQueryValueExW.Call(hkey, uintptr(unsafe.Pointer(v)), 0, uintptr(unsafe.Pointer(&typ)), uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size))); r != 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}
