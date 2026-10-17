//go:build windows

package collector

// rawTemps returns no sensors on Windows.
//
// Unlike Linux, Windows exposes no portable temperature API: the classic ACPI
// thermal zone (MSAcpi_ThermalZoneTemperature) is often unavailable on desktops
// and vendor data needs a tool such as LibreHardwareMonitor/OpenHardwareMonitor.
// If you run one of those, read its data here (WMI) and return []Temp; the shared
// filtering/dedup logic in temp.go will take care of the rest. Returning nil is
// fine — the UI simply shows "sin sensores".
func rawTemps() []Temp { return nil }
