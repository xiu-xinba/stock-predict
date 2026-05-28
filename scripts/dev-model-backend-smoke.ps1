param(
  [int]$BackendPort = 5070,
  [int]$ModelPort = 8090,
  [string]$FundCode = "510300"
)

$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$modelDir = Join-Path $repoRoot "model-training"
$backendDir = Join-Path $repoRoot "backend-go"
$modelSrc = Join-Path $modelDir "src"
$modelPath = Join-Path $modelDir "artifacts/public_mvp_index_fund_tournament_champion.joblib"
$samplesPath = Join-Path $modelDir "data/processed/public_mvp_daily_weekly_index_fund_samples.csv"
$fundStorePath = Join-Path $backendDir "data/funds.json"
$modelLog = Join-Path $repoRoot ".run-logs/model-service.log"
$backendLog = Join-Path $repoRoot ".run-logs/backend-go.log"

New-Item -ItemType Directory -Force -Path (Split-Path -Parent $modelLog) | Out-Null

function Wait-Json($Url, $TimeoutSeconds = 45) {
  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  do {
    try {
      return Invoke-RestMethod -Method Get -Uri $Url -TimeoutSec 3
    } catch {
      Start-Sleep -Milliseconds 600
    }
  } while ((Get-Date) -lt $deadline)
  throw "Timed out waiting for $Url"
}

function Stop-ProcessTree([int]$RootPid) {
  $children = Get-CimInstance Win32_Process | Where-Object { $_.ParentProcessId -eq $RootPid }
  foreach ($child in $children) {
    Stop-ProcessTree ([int]$child.ProcessId)
  }
  Stop-Process -Id $RootPid -Force -ErrorAction SilentlyContinue
}

$modelProcess = $null
$backendProcess = $null
try {
  $modelCommand = @"
`$env:PYTHONPATH = '$modelSrc'
python -m fund_model_training.serve_model --model '$modelPath' --samples '$samplesPath' --port $ModelPort *> '$modelLog'
"@
  $modelProcess = Start-Process powershell -WindowStyle Hidden -PassThru -WorkingDirectory $modelDir -ArgumentList @(
    "-NoProfile",
    "-ExecutionPolicy", "Bypass",
    "-Command", $modelCommand
  )
  Wait-Json "http://127.0.0.1:$ModelPort/health" | Out-Null

  $backendCommand = @"
`$env:APP_ENV = 'development'
`$env:PORT = '$BackendPort'
`$env:FUND_STORE_PATH = '$fundStorePath'
`$env:FUND_SYNC_CSV_PATH = '$samplesPath'
`$env:MODEL_SERVICE_URL = 'http://127.0.0.1:$ModelPort'
go run ./cmd/api *> '$backendLog'
"@
  $backendProcess = Start-Process powershell -WindowStyle Hidden -PassThru -WorkingDirectory $backendDir -ArgumentList @(
    "-NoProfile",
    "-ExecutionPolicy", "Bypass",
    "-Command", $backendCommand
  )
  Wait-Json "http://127.0.0.1:$BackendPort/api/v1/health" | Out-Null

  $sync = Invoke-RestMethod -Method Post -Uri "http://127.0.0.1:$BackendPort/api/v1/funds/sync" -TimeoutSec 20
  $prediction = Invoke-RestMethod -Method Get -Uri "http://127.0.0.1:$BackendPort/api/v1/predict/$FundCode" -TimeoutSec 20

  [pscustomobject]@{
    ok = $true
    sync = $sync.data
    fund_code = $prediction.data.fund_code
    fund_name = $prediction.data.fund_name
    direction = $prediction.data.next_day_prediction.direction
    confidence = $prediction.data.next_day_prediction.direction_confidence
    signal_status = $prediction.data.next_day_prediction.signal_status
    is_actionable = $prediction.data.next_day_prediction.is_actionable
    prediction_interval = $prediction.data.next_day_prediction.prediction_interval
    return_decomposition = $prediction.data.next_day_prediction.return_decomposition
    reliability = $prediction.data.next_day_prediction.reliability
    model_service = "http://127.0.0.1:$ModelPort"
    backend = "http://127.0.0.1:$BackendPort"
  } | ConvertTo-Json -Depth 6
} finally {
  if ($backendProcess) {
    Stop-ProcessTree $backendProcess.Id
  }
  if ($modelProcess) {
    Stop-ProcessTree $modelProcess.Id
  }
}
