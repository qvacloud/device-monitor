# Install device-monitor agent on Windows: firewall rule + start-on-boot task.
# Run in an ELEVATED (Administrator) PowerShell, after `make windows`.
param(
  [string]$Exe   = (Join-Path $PSScriptRoot "..\dist\dm-agent-windows-amd64.exe"),
  [string]$Dest  = (Join-Path $env:ProgramFiles "device-monitor"),
  [int]   $Port  = 55555,
  [string]$Token = ""
)
$ErrorActionPreference = "Stop"

if (-not (Test-Path $Exe)) { throw "Binary not found: $Exe  (build it with: make windows)" }
New-Item -ItemType Directory -Force -Path $Dest | Out-Null
Copy-Item $Exe (Join-Path $Dest "dm-agent.exe") -Force

$agentArgs = "-addr :$Port"
if ($Token) { $agentArgs = "$agentArgs -token $Token" }

# Inbound firewall rule for the metrics port.
if (-not (Get-NetFirewallRule -DisplayName "device-monitor" -ErrorAction SilentlyContinue)) {
  New-NetFirewallRule -DisplayName "device-monitor" -Direction Inbound -Action Allow `
    -Protocol TCP -LocalPort $Port | Out-Null
}

# Start on boot as SYSTEM (no user login required).
$action  = New-ScheduledTaskAction -Execute (Join-Path $Dest "dm-agent.exe") -Argument $agentArgs
$trigger = New-ScheduledTaskTrigger -AtStartup
$princ   = New-ScheduledTaskPrincipal -UserId "SYSTEM" -RunLevel Highest
Register-ScheduledTask -TaskName "device-monitor" -Action $action -Trigger $trigger `
  -Principal $princ -Force | Out-Null
Start-ScheduledTask -TaskName "device-monitor"

Write-Host "OK  ->  $Dest\dm-agent.exe  (port $Port, task 'device-monitor')"
Write-Host "Test: try { Invoke-RestMethod http://localhost:$Port/metrics }"
