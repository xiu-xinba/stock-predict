$ErrorActionPreference = "Stop"

$root = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$assertions = 0

function Read-ProjectFile([string]$RelativePath) {
    return Get-Content -Raw -LiteralPath (Join-Path $root $RelativePath)
}

function Assert-True([bool]$Condition, [string]$Message) {
    $script:assertions++
    if (-not $Condition) {
        throw "Assertion failed: $Message"
    }
}

function Assert-Match([string]$Content, [string]$Pattern, [string]$Message) {
    Assert-True ($Content -match $Pattern) $Message
}

function Assert-NotMatch([string]$Content, [string]$Pattern, [string]$Message) {
    Assert-True ($Content -notmatch $Pattern) $Message
}

$dockerIgnorePath = Join-Path $root "backend-go\.dockerignore"
Assert-True (Test-Path -LiteralPath $dockerIgnorePath) "backend-go/.dockerignore must exist"
$dockerIgnore = Get-Content -Raw -LiteralPath $dockerIgnorePath
Assert-Match $dockerIgnore "(?m)^\.env$" ".env must be excluded"
Assert-Match $dockerIgnore "(?m)^!\.env\.example$" ".env.example must remain available"
Assert-Match $dockerIgnore "(?m)^\.venv/$" ".venv must be excluded"
Assert-Match $dockerIgnore "(?m)^data/$" "data must be excluded"
Assert-Match $dockerIgnore "(?m)^bin/$" "bin must be excluded"
Assert-Match $dockerIgnore "(?m)^\*\.exe$" "executables must be excluded"
Assert-Match $dockerIgnore "(?m)^\*\.log$" "logs must be excluded"
Assert-Match $dockerIgnore "(?m)^\*\*/__pycache__/$" "Python caches must be excluded"

$dockerfile = Read-ProjectFile "backend-go\Dockerfile"
Assert-NotMatch $dockerfile "COPY\s+\.\s+\." "Dockerfile must not copy the whole backend context"
Assert-Match $dockerfile "COPY\s+cmd\s+\./cmd" "Dockerfile must copy cmd explicitly"
Assert-Match $dockerfile "COPY\s+internal\s+\./internal" "Dockerfile must copy internal explicitly"
Assert-Match $dockerfile "apk add --no-cache ca-certificates" "runtime image must install trusted CA certificates"
Assert-NotMatch $dockerfile "(?i)apk add[^\r\n]*\bcurl\b" "runtime image must not install curl"

$compose = Read-ProjectFile "docker-compose.yml"
$initScript = Read-ProjectFile "backend-go\docker\postgres\init\01-runtime-role.sh"
Assert-Match $compose "POSTGRES_USER:\s+\$\{POSTGRES_USER:\?" "owner identity must be mandatory"
Assert-Match $compose "POSTGRES_RUNTIME_USER:\s+\$\{POSTGRES_RUNTIME_USER:\?" "runtime user must be mandatory"
Assert-Match $compose "POSTGRES_RUNTIME_PASSWORD:\s+\$\{POSTGRES_RUNTIME_PASSWORD:\?" "runtime password must be mandatory"
Assert-Match $compose ':\s+"\$\$\{MIGRATION_DATABASE_URL:\?MIGRATION_DATABASE_URL must be set\}"' "migration service must require its DSN at startup"
Assert-Match $compose ':\s+"\$\$\{DATABASE_URL:\?DATABASE_URL must be set\}"' "backend service must require its DSN at startup"
Assert-Match $compose "01-runtime-role\.sh:/docker-entrypoint-initdb\.d/01-runtime-role\.sh:ro" "runtime role initializer must be mounted"
Assert-Match $initScript "ALTER DEFAULT PRIVILEGES" "default privileges must be configured"
Assert-Match $initScript "GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES" "runtime table DML must be granted"
Assert-Match $initScript "GRANT USAGE, SELECT, UPDATE ON ALL SEQUENCES" "runtime sequence access must be granted"
Assert-Match $initScript "REVOKE TEMPORARY ON DATABASE %I FROM PUBLIC" "temporary-table creation must be revoked from PUBLIC"
Assert-Match $initScript "REVOKE TEMPORARY ON DATABASE %I FROM %I" "temporary-table creation must be revoked from runtime"
Assert-Match $initScript "GRANT CONNECT ON DATABASE %I TO %I" "runtime database CONNECT must remain granted"
Assert-Match $initScript 'POSTGRES_RUNTIME_USER.*POSTGRES_USER' "shell guard must reject matching owner and runtime roles"
Assert-Match $initScript "current_user\s*=\s*.*runtime_user" "SQL guard must reject the migration identity as runtime role"
Assert-NotMatch $initScript "GRANT\s+(CREATE|ALL).+SCHEMA" "runtime must not receive schema DDL privileges"

$restrictedHTTPRoots = @(
    "backend-go\internal\infrastructure\providers",
    "backend-go\internal\infrastructure\database"
)
$restrictedHTTPGoFiles = @()
foreach ($restrictedRoot in $restrictedHTTPRoots) {
    $absoluteRoot = Join-Path $root $restrictedRoot
    $restrictedHTTPGoFiles += @(Get-ChildItem -LiteralPath $absoluteRoot -Recurse -File -Filter "*.go" | Where-Object { $_.Name -notlike "*_test.go" })
}
Assert-True ($restrictedHTTPGoFiles.Count -gt 0) "restricted infrastructure Go files must be discovered recursively"
foreach ($file in $restrictedHTTPGoFiles) {
    $content = Get-Content -Raw -LiteralPath $file.FullName
    $relativePath = $file.FullName.Substring($root.Length + 1)
    Assert-NotMatch $content '"os/exec"' "$relativePath must not import os/exec"
    Assert-NotMatch $content '\bexec\.(Command|CommandContext)\b' "$relativePath must not execute external commands"
    Assert-NotMatch $content "--insecure" "$relativePath must not disable TLS verification"
    Assert-NotMatch $content '\b(fetchViaSystemCurl|getViaCurl)\b' "$relativePath must not contain curl fallback identifiers"
    Assert-NotMatch $content '\bclient\.Do\(' "$relativePath must use the resilient HTTP client"
    Assert-NotMatch $content '&http\.Client\s*\{' "$relativePath must use the shared HTTP client factory"
}

$verifier = Read-ProjectFile "scripts\verify-api-contract.ps1"
$openAPI = Read-ProjectFile "docs\api\openapi.yaml"
Assert-Match $verifier '\$redoclyVersion\s*=\s*"[0-9]+\.[0-9]+\.[0-9]+"' "Redocly must be pinned"
Assert-Match $verifier '\$redoclyVersion\s*=\s*"2\.20\.3"' "Redocly must use the approved pinned version"
Assert-Match $verifier '@redocly/cli@\$redoclyVersion' "contract verifier must use the pinned Redocly version"
Assert-Match $verifier "lint" "contract verifier must lint OpenAPI"
foreach ($skipRule in @("operation-2xx-response", "operation-4xx-response", "info-license", "no-server-example.com")) {
    Assert-Match $verifier ('"--skip-rule",\s*"' + [regex]::Escape($skipRule) + '"') "contract verifier must skip expected Redocly warning rule $skipRule"
}
Assert-Match $verifier "frontend\\src\\shared\\api\\routes\.ts" "contract verifier must inspect frontend API routes"
Assert-Match $verifier "frontend" "contract verifier must compare frontend routes"
Assert-NotMatch $verifier '@\(\$routeJSON\s*\|\s*ConvertFrom-Json\)' "contract verifier must not wrap ConvertFrom-Json arrays in a single Windows PowerShell item"
Assert-Match $verifier "deprecated" "contract verifier must check prediction deprecation"
Assert-Match $verifier '"400"' "contract verifier must check prediction 400"
Assert-Match $verifier '"410"' "contract verifier must check prediction 410"
Assert-Match $openAPI "(?ms)^  /api/v1/predict/\{fundCode\}:.*?deprecated:\s+true.*?responses:.*?        ""400"":.*?        ""410"":" "fund prediction must remain deprecated with 400 and 410"
Assert-Match $openAPI "(?ms)^  /api/v1/stock/\{stockCode\}/predict:.*?deprecated:\s+true.*?responses:.*?        ""400"":.*?        ""410"":" "stock prediction must remain deprecated with 400 and 410"
Assert-Match $openAPI "(?m)^    ServiceUnavailable:$" "ServiceUnavailable response must exist"
Assert-NotMatch $openAPI "(?ms)^    ErrorCode:.*?^      examples:" "schema-level examples must not be used"

$workflow = Read-ProjectFile ".github\workflows\ci.yml"
Assert-Match $workflow "govulncheck@v?[0-9]+\.[0-9]+\.[0-9]+" "govulncheck must be pinned"
Assert-NotMatch $workflow "govulncheck@latest" "govulncheck must not use latest"
Assert-Match $workflow "docker build" "CI must build the backend container"
Assert-Match $workflow "actions/setup-go@v5" "contract CI must install Go for the route extractor"
Assert-Match $workflow "go-version:\s*1\.26\.4" "contract CI must pin Go"
Assert-Match $workflow "working-directory:\s*scripts/verify-api-contract" "contract CI must run route extractor tests in its module"
Assert-Match $workflow "go test \./\.\.\." "contract CI must test the route extractor"
Assert-Match $workflow "pwsh\s+\./scripts/tests/verify-api-contract\.Tests\.ps1" "CI must execute the assertion script directly"
Assert-Match $workflow "docker compose --profile app up -d --build" "CI must start a fresh Compose application stack"
Assert-Match $workflow "/api/v1/health/live" "CI must poll the live health endpoint"
Assert-Match $workflow "docker compose --profile app logs" "CI must dump Compose logs on failure"
Assert-Match $workflow "if:\s+always\(\)" "CI cleanup must always run"
Assert-Match $workflow "docker compose --profile app down -v" "CI must remove the smoke-test stack and volumes"
Assert-Match $workflow "CREATE TABLE public\.runtime_must_not_create" "CI must prove the runtime role cannot create permanent tables"
Assert-Match $workflow "FUND_AUTO_SYNC_ON_START:\s*false" "CI smoke must disable fund auto sync"
Assert-Match $workflow "STOCK_AUTO_SYNC_ON_START:\s*false" "CI smoke must disable stock auto sync"
Assert-Match $workflow "MARKET_SYNC_ENABLED:\s*false" "CI smoke must disable market sync"

Assert-Match $compose 'ADMIN_TOKEN:\s+\$\{ADMIN_TOKEN:-\}' "postgres-only Compose parsing must not require ADMIN_TOKEN"
Assert-Match $compose 'CORS_ORIGINS:\s+\$\{CORS_ORIGINS:-\}' "postgres-only Compose parsing must not require CORS_ORIGINS"
Assert-Match $compose 'MIGRATION_DATABASE_URL:\s+\$\{MIGRATION_DATABASE_URL:-\}' "postgres-only Compose parsing must not require migration DSN"
Assert-Match $compose 'DATABASE_URL:\s+\$\{DATABASE_URL:-\}' "postgres-only Compose parsing must not require runtime DSN"
Assert-Match $compose "FUND_AUTO_SYNC_ON_START:\s+\$\{FUND_AUTO_SYNC_ON_START:-false\}" "Compose must disable fund auto sync by default"
Assert-Match $compose "STOCK_AUTO_SYNC_ON_START:\s+\$\{STOCK_AUTO_SYNC_ON_START:-false\}" "Compose must disable stock auto sync by default"
Assert-Match $compose "MARKET_SYNC_ENABLED:\s+\$\{MARKET_SYNC_ENABLED:-false\}" "Compose must disable market sync by default"

$readme = Read-ProjectFile "README.md"
Assert-Match $readme "docker compose --profile app run --rm migrate" "README production flow must run migration and runtime grants"
Assert-NotMatch $readme 'MIGRATION_DATABASE_URL="postgres://[^"]+@postgres:5432' "README host-side migration DSN must use localhost"
Assert-NotMatch $readme 'DATABASE_URL="postgres://[^"]+@postgres:5432' "README host-side runtime DSN must use localhost"

$verifierPath = Join-Path $root "scripts\verify-api-contract.ps1"
$windowsPowerShell = Get-Command powershell -ErrorAction SilentlyContinue
if ($windowsPowerShell) {
    & $windowsPowerShell.Source -NoProfile -ExecutionPolicy Bypass -File $verifierPath
    if ($LASTEXITCODE -ne 0) {
        throw "API contract verifier failed under Windows PowerShell with exit code $LASTEXITCODE"
    }
}
& pwsh -NoProfile -File $verifierPath
if ($LASTEXITCODE -ne 0) {
    throw "API contract verifier failed with exit code $LASTEXITCODE"
}

Write-Host "Static deployment and API contract assertions passed: $assertions checks."
