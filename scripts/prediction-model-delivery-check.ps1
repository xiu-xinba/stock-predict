<#
.SYNOPSIS
    预测模型交付检查脚本

.DESCRIPTION
    执行完整的模型交付检查流程，包括：
    1. 运行 model-training 模块的单元测试
    2. 运行 backend-go 模块的单元测试
    3. 执行前端生产构建验证
    4. 可选：运行完整的模型训练流水线（日线/周线/日内）
    5. 可选：执行模型服务与后端的冒烟测试

.EXAMPLE
    .\scripts\prediction-model-delivery-check.ps1
    .\scripts\prediction-model-delivery-check.ps1 -RunTraining
    .\scripts\prediction-model-delivery-check.ps1 -RunTraining -RunSmoke
    .\scripts\prediction-model-delivery-check.ps1 -RunSmoke -FundCode "510050"

.PREREQUISITES
    - Python 3.11+ 及 fund_model_training 包已安装（pip install -e ".[data,dev]"）
    - Go 1.21+ 已安装
    - Node.js 18+ 及前端依赖已安装（npm install）
    - 使用 -RunTraining 时需要训练数据已就绪
    - 使用 -RunSmoke 时需要模型 artifact 文件已存在
    - PowerShell 5.1+ 或 PowerShell 7+

.NOTES
    不带参数运行时仅执行测试和构建检查
    -RunTraining：额外执行完整训练流水线（耗时较长）
    -RunSmoke：额外执行模型服务冒烟测试
    两个开关可组合使用
#>
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
