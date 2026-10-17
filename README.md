# device-monitor

Monitoriza CPU, RAM, temperatura y almacenamiento de tus equipos. Cada máquina
**se sirve a sí misma** una API JSON mínima en el puerto **55555** (sin servidor
central) y una **app Android** propia muestra todos los equipos a la vez, con
estética *terminal verde* sobre negro AMOLED, a pantalla completa y en horizontal.

Funciona en **Linux** (PC, Raspberry/Orange Pi, servidores) y **Windows**.

```
 equipo A ─┐
 equipo B ─┼─ :55555/metrics ──►  APK Android (móvil)  ◄─ peticiones
 equipo C ─┘        (cada uno se sirve a sí mismo)
```

## Componentes

| Pieza | Qué es | Salida |
|---|---|---|
| `agent/` | Agente Go, binario estático único | API en `:55555` |
| `collector/` | Colector multiplataforma (`linux` + `windows`, con build tags) | CPU / RAM / swap / temps / discos |
| `android/` | App Android en Java puro (sin AndroidX) | APK ~24 KB |
| `scripts/` | Despliegue y build (Linux y Windows) | — |

Sin dependencias externas de Go ni de terceros en el APK: **todo compila offline**
(la parte Windows usa `syscall` puro, sin CGO).

## Compilar

```bash
make check     # go vet + build
make cross     # Linux: amd64 (PC), arm64 y arm7 (Orange Pi / Raspberry)
make windows   # Windows: amd64 y arm64 (.exe)
make apk       # dist/devmon.apk  (requiere el SDK de Android)
make all       # cross + windows + apk
```

El APK se compila sin Gradle (aapt2 + javac + d8 + zipalign + apksigner) con el
SDK en `$ANDROID_HOME`. Se firma con `android/debug.keystore` (se genera si no
existe; está en `.gitignore`).

## Desplegar el agente

### Windows

Eleva PowerShell y ejecuta el instalador (copia el binario a `Program Files`,
abre el puerto en el firewall y lo deja arrancando al inicio):

```powershell
make windows
powershell -ExecutionPolicy Bypass -File .\scripts\install-windows.ps1
```

Manual:

```powershell
.\dist\dm-agent-windows-amd64.exe -addr :55555
netsh advfirewall firewall add rule name="device-monitor" ^
  dir=in action=allow protocol=TCP localport=55555
```

### Linux (systemd, acceso root)

```bash
PI_PASS='tu_password' ARCH=arm64 ./scripts/deploy-pi.sh 192.168.1.49 192.168.1.50
```

Copia el binario a `/usr/local/bin/dm-agent`, instala el unit
`dm-agent.service`, lo habilita y lo reinicia.

### Prueba rápida (cualquier SO)

```bash
./dist/dm-agent-linux-arm64 -addr :55555          # o -token SECRETO
DISK_MOUNTS="/,/home" ./dist/dm-agent-linux-amd64 # limitar discos monitorizados
curl http://IP:55555/health
```

## API (súper ligera)

| Ruta | Descripción |
|---|---|
| `GET /metrics` | JSON con todas las métricas |
| `GET /health` | `ok` (comprobación de vida) |
| `GET /` | Página propia del equipo (verde terminal) |

`/metrics` responde en < 1 ms y pesa ~1 KB. Si se arranca con `-token X`, exige
la cabecera `X-Auth-Token: X`.

```json
{
  "hostname": "my-pc", "os": "windows", "arch": "amd64",
  "cpu": {"model":"AMD Ryzen 5 8600G","cores":12,"usage_percent":12.4,"load1":0,"load5":0,"load15":0},
  "mem": {"total":32000000000,"used":18000000000,"available":14000000000,"used_percent":56.2},
  "swap": {"total":8000000000,"used":1500000000,"used_percent":18.7},
  "temps": [{"label":"k10temp","celsius":55.5}],
  "disks": [{"mount":"C:","fs":"NTFS","total":1000000000000,"used":620000000000,"used_percent":62.0}]
}
```

## App Android

- Pantalla completa real (inmersiva, sin barras), horizontal, fondo `#000000`.
- Una **columna por dispositivo** con líneas finas: CPU, RAM, SWAP, temperaturas
  y discos. Verde terminal `#39FF14`; amarillo ≥75 %, rojo ≥90 % y offline.
- **Bajo consumo**: consulta cada 5 s, timeouts de 1,8 s, sin `keep-screen-on`, y
  **deja de consultar** en `onStop` (pantalla apagada / app en segundo plano).
- **Añadir endpoints**: botón `+ ADD` (nombre + `http://IP:55555`). **Mantén
  pulsado** un dispositivo para eliminarlo. Se guardan en preferencias.

Instalar en el móvil conectado por ADB:

```bash
adb install -r dist/devmon.apk
adb shell am start -n com.qvacloud.devmon/.MainActivity
```

## Notas por plataforma

| Métrica | Linux | Windows |
|---|---|---|
| CPU % | `/proc/stat` | `GetSystemTimes` |
| CPU modelo | `/proc/cpuinfo` | registro (`ProcessorNameString`) |
| RAM / swap | `/proc/meminfo` | `GlobalMemoryStatusEx` |
| Uptime | `/proc/uptime` | `GetTickCount64` |
| Discos | `statfs` (puntos de montaje) | `GetLogicalDrives` + `GetDiskFreeSpaceEx` (letras `C:`, `D:`…) |
| Load average | `/proc/loadavg` | *(no existe en Windows → 0)* |
| Temperaturas | hwmon + thermal_zone | *(no hay API portable → vacío; ver más abajo)* |

- **Temperaturas en Windows**: el SO no expone una API universal de sensores. El
  `collector` deja el punto de extensión listo en `collector/temp_windows.go`
  (`rawTemps()`): si ejecutas LibreHardwareMonitor/OpenHardwareMonitor, puedes
  leer sus datos por WMI y devolver `[]Temp`; el filtrado/dedup ya es compartido.
  Sin eso, la app muestra `sin sensores`.
- **Discos**: deduplica por sistema de ficheros / letra; ajustable con la variable
  `DISK_MOUNTS` (p. ej. `DISK_MOUNTS="C:\"` en Windows o `"/,/home"` en Linux).
- Tráfico HTTP en claro (LAN): `usesCleartextTraffic="true"` en el manifiesto Android.

## Licencia

MIT — ver [`LICENSE`](LICENSE).
