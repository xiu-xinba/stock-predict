$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot

function Invoke-QualityStep {
  param(
    [string]$Name,
    [string]$WorkingDirectory,
    [string[]]$Command
  )

  Write-Host ""
  Write-Host "==> $Name" -ForegroundColor Cyan
  Push-Location $WorkingDirectory
  try {
    & $Command[0] @($Command[1..($Command.Length - 1)])
    if ($LASTEXITCODE -ne 0) {
      throw "$Name failed with exit code $LASTEXITCODE"
    }
  } finally {
    Pop-Location
  }
}

Invoke-QualityStep "API contract check" $root @("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", "scripts\verify-api-contract.ps1")

$backend = Join-Path $root "backend-go"
Invoke-QualityStep "Go tests" $backend @("go", "test", "./...")
Invoke-QualityStep "Go vet" $backend @("go", "vet", "./...")

$frontend = Join-Path $root "frontend"
Invoke-QualityStep "Frontend lint" $frontend @("npm", "run", "lint")
Invoke-QualityStep "Frontend tests" $frontend @("npm", "run", "test:run")
Invoke-QualityStep "Frontend build" $frontend @("npm", "run", "build")

Write-Host ""
Write-Host "Commercial readiness verification passed." -ForegroundColor Green
