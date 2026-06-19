$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot

function Invoke-Checked([string]$WorkingDirectory, [string]$Command) {
    Push-Location $WorkingDirectory
    try {
        Write-Host ">>> $Command"
        Invoke-Expression $Command
        if ($LASTEXITCODE -ne 0) {
            throw "Command failed with exit code ${LASTEXITCODE}: $Command"
        }
    }
    finally {
        Pop-Location
    }
}

& (Join-Path $PSScriptRoot "verify-api-contract.ps1")
if (-not $?) {
    throw "API contract verification failed."
}

$backend = Join-Path $root "backend-go"
$frontend = Join-Path $root "frontend"
$akshare = Join-Path $backend "akshare-service"

Invoke-Checked $backend '$files = @(gofmt -l .); if ($files.Count -gt 0) { $files; exit 1 }'
Invoke-Checked $backend 'go test -count=2 ./...'
Invoke-Checked $backend 'go vet ./...'
Invoke-Checked $backend 'go build ./...'
Invoke-Checked $backend 'go run golang.org/x/vuln/cmd/govulncheck@latest ./...'

Invoke-Checked $frontend 'npx prettier --check "src/**/*.{ts,tsx,vue,css}"'
Invoke-Checked $frontend 'npm run lint -- --max-warnings 0'
Invoke-Checked $frontend 'npm run test:run'
Invoke-Checked $frontend 'npm run build'
Invoke-Checked $frontend 'npm audit --audit-level=high'
Invoke-Checked $frontend 'npm audit --omit=dev --audit-level=high'

$python = Join-Path $akshare ".venv\Scripts\python.exe"
if (-not (Test-Path -LiteralPath $python)) {
    throw "AKShare virtual environment is missing: $python"
}
Invoke-Checked $akshare "& '$python' -m pip check"
Invoke-Checked $akshare "& '$python' -m compileall -q ."
Invoke-Checked $akshare "& '$python' -m unittest -v"

Write-Host "Commercial readiness checks passed."
