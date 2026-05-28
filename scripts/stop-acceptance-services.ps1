$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$pidPath = Join-Path $repoRoot ".run-logs/acceptance-pids.json"

if (-not (Test-Path $pidPath)) {
  Write-Host "No acceptance pid file found: $pidPath"
  exit 0
}

function Stop-ProcessTree([int]$RootPid) {
  $children = Get-CimInstance Win32_Process | Where-Object { $_.ParentProcessId -eq $RootPid }
  foreach ($child in $children) {
    Stop-ProcessTree ([int]$child.ProcessId)
  }
  Stop-Process -Id $RootPid -Force -ErrorAction SilentlyContinue
}

$pids = Get-Content -Raw $pidPath | ConvertFrom-Json
foreach ($name in @("frontend", "backend", "intraday_model", "weekly_model", "model")) {
  $pidValue = $pids.$name
  if ($pidValue) {
    Stop-ProcessTree ([int]$pidValue)
    Write-Host "Stopped $name process tree: $pidValue"
  }
}
