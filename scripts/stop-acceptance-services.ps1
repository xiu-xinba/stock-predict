<#
.SYNOPSIS
    停止验收测试启动的全部服务

.DESCRIPTION
    读取 .run-logs/acceptance-pids.json 中保存的进程 PID，递归停止前端、后端、
    日内模型、周线模型和日线模型的进程树。如果 PID 文件不存在则安全退出。

.EXAMPLE
    .\scripts\stop-acceptance-services.ps1

.PREREQUISITES
    - 之前已通过 start-acceptance-services.ps1 启动服务
    - .run-logs/acceptance-pids.json 文件存在
    - PowerShell 5.1+ 或 PowerShell 7+

.NOTES
    此脚本会递归终止子进程，确保不留残留进程
    如果 PID 文件不存在，脚本会提示并安全退出
#>
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
