# AGENTS.md

Guidance for AI coding agents (and humans) working on this repository.

## What this is

`device-monitor` is a **self-hosted, LAN-only** system monitor. Each machine runs a
single static Go binary (the *agent*) that exposes a tiny JSON metrics API on
`:55555`. There is **no central server**: every host serves itself, and the Android
app polls several hosts at once. Design goals: zero external Go dependencies, no CGO,
fully offline builds, tiny footprint.

## Repository layout

```
agent/main.go            HTTP server: /metrics, /health, / (embedded self page)
agent/web/index.html     self page, embedded via //go:embed
collector/               portable metrics collection (see "Platform contract")
android/                 minimal Java app (no AndroidX/Gradle), built by scripts/build-apk.sh
scripts/                 build-apk.sh, deploy-pi.sh, dm-agent.service, install-windows.ps1
Makefile                 check / cross / windows / apk / all / clean
```

## Build & test

```bash
make check     # go vet ./...  +  go build ./...
make cross     # linux/amd64, linux/arm64, linux/arm7   -> dist/dm-agent-linux-*
make windows   # windows/amd64, windows/arm64           -> dist/dm-agent-windows-*.exe
make apk       # dist/devmon.apk  (needs ANDROID_HOME; compiles offline, no Gradle)
make all       # cross + windows + apk
```

Always run `make check` (and `gofmt -l .`, which must print nothing) before finishing.
Cross-compilation must keep working for **all** targets — verify with `make cross && make windows`.

Runtime smoke test (no VM needed on Linux):

```bash
go build -o /tmp/dm ./agent && /tmp/dm -addr :55999 &
curl -s http://127.0.0.1:55999/health     # -> ok
curl -s http://127.0.0.1:55999/metrics    # -> JSON
```

## Platform contract (important)

OS-specific code lives in `collector/*_linux.go` and `collector/*_windows.go`, selected
by `//go:build` tags. Shared logic (types, `usage`, `temps` filtering, `pct`/`round1`,
`normModel`) lives in untagged files and must stay OS-agnostic.

To **add a new OS**, create `collector/<name>_<goos>.go` implementing:

| Function | Returns | Notes |
|---|---|---|
| `readCPUStat() cpuStat` | cumulative idle/total ticks | any monotonic unit; `usage()` diffs two samples |
| `cpuCount() int` | logical CPU count | `runtime.NumCPU()` is fine |
| `cpuModel() string` | CPU marketing name | apply `normModel()` |
| `memInfo() (Mem, Swap)` | memory + swap bytes | |
| `loadavg() (f64,f64,f64)` | 1/5/15 load | return `0,0,0` if unsupported |
| `uptime() float64` | seconds since boot | |
| `rawTemps() []Temp` | raw sensors | may return `nil`; shared `temps()` filters/dedups |
| `diskMounts() []string` | mount points | honour `DISK_MOUNTS` env override |
| `disks(mounts []string) []Disk` | filesystem usage | dedup by device/volume |

`collector.go` and `Collect()` stay platform-neutral. Windows uses **only** `syscall`
via `syscall.NewLazyDLL("kernel32.dll"|"advapi32.dll")` (helpers in `cpu_windows.go`,
`mem_windows.go`, `disk_windows.go`, `registry_windows.go`) — do not add CGO or
third-party modules.

Known porting gaps (keep documented, don't silently fake):
- **Temperatures on Windows**: no portable API. `temp_windows.go:rawTemps()` returns
  `nil`; extend it via WMI/LibreHardwareMonitor if needed.
- **Load average on Windows**: does not exist → `0`.

## Conventions

- Spanish is used in user-facing docs/comments; code identifiers stay in English.
- Keep the agent dependency-free and static; the JSON schema in `Metrics` is the public
  contract used by the Android app and the self page — change it deliberately.
- The Android app is plain Java (`com.qvacloud.devmon`), no AndroidX. Its default
  endpoints and hints use **example** IPs (`192.168.1.x`); never commit real IPs,
  hostnames, passwords or keys. `dist/`, `android/build/`, `*.apk`, `*.exe` and
  keystores are gitignored on purpose.
- `DISK_MOUNTS` (e.g. `"/,/home"` on Linux, `"C:\\"` on Windows) limits monitored disks.
- Agent flags: `-addr`, `-token` (requires `X-Auth-Token` header), `-interval`.

## Releases

Compiled, clean artifacts are published as GitHub Release assets (not committed):

```bash
make all
gh release create vX.Y.Z dist/devmon.apk \
  dist/dm-agent-linux-amd64 dist/dm-agent-linux-arm64 dist/dm-agent-linux-arm7 \
  dist/dm-agent-windows-amd64.exe dist/dm-agent-windows-arm64.exe \
  --title "vX.Y.Z" --notes "..."
```
