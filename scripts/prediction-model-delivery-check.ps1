param(
  [switch]$RunTraining,
  [switch]$RunSmoke,
  [int]$BackendPort = 5070,
  [int]$ModelPort = 8090,
  [string]$FundCode = "510300"
)

$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$modelDir = Join-Path $repoRoot "model-training"
$backendDir = Join-Path $repoRoot "backend-go"
$frontendDir = Join-Path $repoRoot "frontend"
$modelSrc = Join-Path $modelDir "src"

function Invoke-Step([string]$Name, [scriptblock]$Action) {
  Write-Host ""
  Write-Host "==> $Name"
  & $Action
}

Invoke-Step "model-training tests" {
  Push-Location $modelDir
  try {
    python -m pytest
  } finally {
    Pop-Location
  }
}

Invoke-Step "backend-go tests" {
  Push-Location $backendDir
  try {
    go test ./...
  } finally {
    Pop-Location
  }
}

Invoke-Step "frontend production build" {
  Push-Location $frontendDir
  try {
    npm run build
  } finally {
    Pop-Location
  }
}

if ($RunTraining) {
  Invoke-Step "daily public MVP pipeline" {
    Push-Location $modelDir
    try {
      $env:PYTHONPATH = $modelSrc
      python -m fund_model_training.run_index_fund_pipeline --config configs/public_mvp_pipeline.example.yml --skip-existing
    } finally {
      Pop-Location
    }
  }

  Invoke-Step "daily tournament" {
    Push-Location $modelDir
    try {
      $env:PYTHONPATH = $modelSrc
      python -m fund_model_training.train_tournament --config configs/index_fund_tournament_train.example.yml
    } finally {
      Pop-Location
    }
  }

  Invoke-Step "weekly tournament" {
    Push-Location $modelDir
    try {
      $env:PYTHONPATH = $modelSrc
      python -m fund_model_training.train_tournament --config configs/index_fund_weekly_tournament.example.yml
    } finally {
      Pop-Location
    }
  }

  Invoke-Step "intraday 5m pipeline" {
    Push-Location $modelDir
    try {
      $env:PYTHONPATH = $modelSrc
      python -m fund_model_training.run_intraday_pipeline --config configs/public_mvp_intraday_pipeline.example.yml --skip-existing
    } finally {
      Pop-Location
    }
  }

  Invoke-Step "intraday 3m pipeline" {
    Push-Location $modelDir
    try {
      $env:PYTHONPATH = $modelSrc
      python -m fund_model_training.run_intraday_pipeline --config configs/public_mvp_intraday_3m_pipeline.example.yml --skip-existing
    } finally {
      Pop-Location
    }
  }
}

if ($RunSmoke) {
  Invoke-Step "model service + backend smoke" {
    & (Join-Path $PSScriptRoot "dev-model-backend-smoke.ps1") `
      -BackendPort $BackendPort `
      -ModelPort $ModelPort `
      -FundCode $FundCode
  }
}

Write-Host ""
Write-Host "Prediction model delivery check completed."
