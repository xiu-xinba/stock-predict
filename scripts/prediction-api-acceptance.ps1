<#
.SYNOPSIS
    预测 API 验收测试脚本

.DESCRIPTION
    对后端 /api/v1/predict/{fundCode} 接口执行验收测试，验证信号状态、可操作性门控、
    收益分解和预测区间等字段是否正确返回，并将结果写入 JSON 报告文件。

.EXAMPLE
    .\scripts\prediction-api-acceptance.ps1
    .\scripts\prediction-api-acceptance.ps1 -BackendUrl "http://127.0.0.1:5070" -FundCodes @("510300","510050")
    .\scripts\prediction-api-acceptance.ps1 -SkipSync

.PREREQUISITES
    - 后端服务已启动（默认 http://127.0.0.1:5070）
    - 模型服务已启动并可通过后端访问
    - PowerShell 5.1+ 或 PowerShell 7+

.NOTES
    默认测试基金代码：510300、510050、510500、159915
    输出文件：docs/report/08-prediction-api-acceptance-results.json
    使用 -SkipSync 跳过基金数据同步（适用于已同步过的场景）
#>
param(
  [string]$BackendUrl = "http://127.0.0.1:5070",
  [string[]]$FundCodes = @("510300", "510050", "510500", "159915"),
  [string]$OutputPath = "docs/report/08-prediction-api-acceptance-results.json",
  [switch]$SkipSync
)

$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$output = if ([System.IO.Path]::IsPathRooted($OutputPath)) {
  $OutputPath
} else {
  Join-Path $repoRoot $OutputPath
}

function Invoke-Api($Method, $Path) {
  Invoke-RestMethod -Method $Method -Uri "$BackendUrl$Path" -TimeoutSec 20
}

$health = Invoke-Api Get "/api/v1/health"
$sync = $null
if (-not $SkipSync) {
  $sync = Invoke-Api Post "/api/v1/funds/sync"
}

$results = foreach ($code in $FundCodes) {
  $response = Invoke-Api Get "/api/v1/predict/$code"
  $data = $response.data
  $next = $data.next_day_prediction
  $decomposition = $next.return_decomposition
  $interval = $next.prediction_interval
  $gate = $next.actionability_gate
  [pscustomobject]@{
    fund_code = $data.fund_code
    fund_name = $data.fund_name
    horizon = $next.horizon
    direction = $next.direction
    confidence = $next.direction_confidence
    predicted_change_pct = $next.predicted_change_pct
    signal_status = $next.signal_status
    is_actionable = $next.is_actionable
    reliability = $next.reliability
    prediction_interval_method = if ($interval) { $interval.method } else { $null }
    prediction_interval_level = if ($interval) { $interval.level } else { $null }
    prediction_interval_coverage = if ($interval) { $interval.empirical_coverage } else { $null }
    actionability_gate_actionable = if ($gate) { $gate.actionable } else { $null }
    actionability_gate_reason = if ($gate) { $gate.reason } else { $null }
    return_decomposition_enabled = [bool]($decomposition -and $decomposition.enabled)
    index_return_pct = if ($decomposition) { $decomposition.index_return_pct } else { $null }
    tracking_error_pct = if ($decomposition) { $decomposition.tracking_error_pct } else { $null }
    direct_fund_return_pct = if ($decomposition) { $decomposition.direct_fund_return_pct } else { $null }
    checks = [pscustomobject]@{
      has_signal_status = [bool]$next.signal_status
      low_confidence_not_actionable = -not ($next.signal_status -eq "low_confidence" -and $next.is_actionable)
      no_signal_not_actionable = -not ($next.signal_status -eq "no_signal" -and $next.is_actionable)
      has_return_decomposition = [bool]($decomposition -and $decomposition.enabled)
      has_prediction_interval = [bool]($interval -and $interval.method -and $interval.empirical_coverage -ne $null)
      has_actionability_gate = [bool]($gate -and $gate.reason)
    }
  }
}

$allChecks = @()
foreach ($item in $results) {
  $allChecks += $item.checks.has_signal_status
  $allChecks += $item.checks.low_confidence_not_actionable
  $allChecks += $item.checks.no_signal_not_actionable
  $allChecks += $item.checks.has_return_decomposition
  $allChecks += $item.checks.has_prediction_interval
  $allChecks += $item.checks.has_actionability_gate
}

$report = [pscustomobject]@{
  ok = -not ($allChecks -contains $false)
  created_at = (Get-Date).ToString("o")
  backend_url = $BackendUrl
  health = $health
  sync = $sync
  fund_count = $results.Count
  results = $results
}

New-Item -ItemType Directory -Force -Path (Split-Path -Parent $output) | Out-Null
$report | ConvertTo-Json -Depth 12 | Set-Content -Encoding UTF8 $output
$report | ConvertTo-Json -Depth 12
