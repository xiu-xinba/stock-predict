<#
.SYNOPSIS
    启动验收测试所需的全部服务

.DESCRIPTION
    按顺序启动模型服务（日线/周线/日内三个实例）、Go 后端和前端开发服务器，
    等待各服务健康检查通过后输出进程 PID 和访问地址。PID 信息保存到
    .run-logs/acceptance-pids.json，供 stop-acceptance-services.ps1 使用。

.EXAMPLE
    .\scripts\start-acceptance-services.ps1
    .\scripts\start-acceptance-services.ps1 -BackendPort 5070 -DailyModelPort 8097
    .\scripts\start-acceptance-services.ps1 -FrontendPort 5173

.PREREQUISITES
    - Python 3.11+ 及 fund_model_training 包已安装（pip install -e ".[data,dev]"）
    - Go 1.21+ 已安装
    - Node.js 18+ 及前端依赖已安装（npm install）
    - 模型 artifact 文件已存在于 model-training/artifacts/
    - 训练样本 CSV 已存在于 model-training/data/processed/
    - PowerShell 5.1+ 或 PowerShell 7+

.NOTES
    日志文件保存在 .run-logs/ 目录下
    默认端口：后端 5070、前端 5173、日线模型 8097、周线模型 8098、日内模型 8099
    启动完成后使用 stop-acceptance-services.ps1 停止所有服务
#>
param(
  [int]$BackendPort = 5070,
  [int]$FrontendPort = 5173,
  [int]$DailyModelPort = 8097,
  [int]$WeeklyModelPort = 8098,
  [int]$IntradayModelPort = 8099
)

$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$modelDir = Join-Path $repoRoot "model-training"
$backendDir = Join-Path $repoRoot "backend-go"
$frontendDir = Join-Path $repoRoot "frontend"
$logDir = Join-Path $repoRoot ".run-logs"
$modelSrc = Join-Path $modelDir "src"

$dailySamples = Join-Path $modelDir "data/processed/public_mvp_daily_weekly_index_fund_samples.csv"
$dailyModel = Join-Path $modelDir "artifacts/public_mvp_index_fund_tournament_champion.joblib"
$weeklyModel = Join-Path $modelDir "artifacts/public_mvp_index_fund_weekly_tournament_champion.joblib"
$intradaySamples = Join-Path $modelDir "data/processed/public_mvp_intraday_index_fund_samples.csv"
$intradayModel = Join-Path $modelDir "artifacts/public_mvp_index_fund_intraday_tournament_champion.joblib"
$fundStorePath = Join-Path $backendDir "data/funds.json"

New-Item -ItemType Directory -Force -Path $logDir | Out-Null

function Wait-Json($Url, $TimeoutSeconds = 60) {
  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  do {
    try {
      return Invoke-RestMethod -Method Get -Uri $Url -TimeoutSec 3
    } catch {
      Start-Sleep -Milliseconds 700
    }
  } while ((Get-Date) -lt $deadline)
  throw "Timed out waiting for $Url"
}

function Wait-Port($Port, $TimeoutSeconds = 60) {
  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  do {
    $listener = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue
    if ($listener) {
      return
    }
    Start-Sleep -Milliseconds 700
  } while ((Get-Date) -lt $deadline)
  throw "Timed out waiting for local port $Port"
}

function Start-ModelService($Name, $Port, $ModelPath, $SamplesPath) {
  $logPath = Join-Path $logDir "$Name.log"
  $command = @"
`$env:PYTHONPATH = '$modelSrc'
python -m fund_model_training.serve_model --model '$ModelPath' --samples '$SamplesPath' --port $Port *> '$logPath'
"@
  $process = Start-Process powershell -WindowStyle Hidden -PassThru -WorkingDirectory $modelDir -ArgumentList @(
    "-NoProfile",
    "-ExecutionPolicy", "Bypass",
    "-Command", $command
  )
  Wait-Json "http://127.0.0.1:$Port/health" | Out-Null
  return $process
}

$dailyProcess = Start-ModelService "model-daily" $DailyModelPort $dailyModel $dailySamples
$weeklyProcess = Start-ModelService "model-weekly" $WeeklyModelPort $weeklyModel $dailySamples
$intradayProcess = Start-ModelService "model-intraday" $IntradayModelPort $intradayModel $intradaySamples

$backendLog = Join-Path $logDir "backend-go.log"
$backendCommand = @"
`$env:APP_ENV = 'development'
`$env:PORT = '$BackendPort'
`$env:FUND_STORE_PATH = '$fundStorePath'
`$env:FUND_SYNC_CSV_PATH = '$dailySamples'
`$env:MODEL_SERVICE_URL = 'http://127.0.0.1:$DailyModelPort'
`$env:WEEKLY_MODEL_SERVICE_URL = 'http://127.0.0.1:$WeeklyModelPort'
`$env:INTRADAY_MODEL_SERVICE_URL = 'http://127.0.0.1:$IntradayModelPort'
go run ./cmd/api *> '$backendLog'
"@
$backendProcess = Start-Process powershell -WindowStyle Hidden -PassThru -WorkingDirectory $backendDir -ArgumentList @(
  "-NoProfile",
  "-ExecutionPolicy", "Bypass",
  "-Command", $backendCommand
)
Wait-Json "http://127.0.0.1:$BackendPort/api/v1/health" | Out-Null

$frontendLog = Join-Path $logDir "frontend.log"
$frontendCommand = "npm run dev -- --host 127.0.0.1 --port $FrontendPort *> '$frontendLog'"
$frontendProcess = Start-Process powershell -WindowStyle Hidden -PassThru -WorkingDirectory $frontendDir -ArgumentList @(
  "-NoProfile",
  "-ExecutionPolicy", "Bypass",
  "-Command", $frontendCommand
)
Wait-Port $FrontendPort | Out-Null

$pids = [pscustomobject]@{
  model = $dailyProcess.Id
  weekly_model = $weeklyProcess.Id
  intraday_model = $intradayProcess.Id
  backend = $backendProcess.Id
  frontend = $frontendProcess.Id
  model_url = "http://127.0.0.1:$DailyModelPort"
  weekly_model_url = "http://127.0.0.1:$WeeklyModelPort"
  intraday_model_url = "http://127.0.0.1:$IntradayModelPort"
  backend_url = "http://127.0.0.1:$BackendPort"
  frontend_url = "http://127.0.0.1:$FrontendPort"
}

$pidPath = Join-Path $logDir "acceptance-pids.json"
$pids | ConvertTo-Json -Depth 4 | Set-Content -Encoding UTF8 $pidPath
$pids | ConvertTo-Json -Depth 4
