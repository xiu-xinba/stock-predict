$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$routerPath = Join-Path $root "backend-go\internal\transport\http\router\router.go"
$openAPIPath = Join-Path $root "docs\api\openapi.yaml"
$frontendRoutesPath = Join-Path $root "frontend\src\shared\api\routes.ts"
$redoclyVersion = "2.20.3"
$redoclySkipRules = @(
    "--skip-rule", "operation-2xx-response",
    "--skip-rule", "operation-4xx-response",
    "--skip-rule", "info-license",
    "--skip-rule", "no-server-example.com"
)

& npx --yes "@redocly/cli@$redoclyVersion" lint $openAPIPath @redoclySkipRules
if ($LASTEXITCODE -ne 0) {
    throw "OpenAPI lint failed."
}

function Normalize-Path([string]$Path) {
    return [regex]::Replace($Path, ":([A-Za-z0-9_]+)", '{$1}')
}

function Normalize-FrontendPath([string]$Path) {
    $normalized = $Path `
        -replace '\$\{fundCode\}', '{fundCode}' `
        -replace '\$\{stockCode\}', '{stockCode}' `
        -replace '\$\{code\}', '{code}' `
        -replace '\$\{type\}', '{type}'
    $normalized = $normalized -replace '/stock/\{code\}', '/stock/{stockCode}'
    return "/api/v1$normalized"
}

function Get-OpenAPIPathBlock([string[]]$Lines, [string]$Path) {
    $start = -1
    for ($i = 0; $i -lt $Lines.Count; $i++) {
        if ($Lines[$i] -eq "  ${Path}:") {
            $start = $i
            break
        }
    }
    if ($start -lt 0) {
        throw "OpenAPI path is missing: $Path"
    }

    $end = $Lines.Count
    for ($i = $start + 1; $i -lt $Lines.Count; $i++) {
        if ($Lines[$i] -match '^  /api/v1' -or $Lines[$i] -eq "components:") {
            $end = $i
            break
        }
    }
    return ($Lines[$start..($end - 1)] -join "`n")
}

$backend = [System.Collections.Generic.HashSet[string]]::new()
$backendPaths = [System.Collections.Generic.HashSet[string]]::new()
$backendItems = New-Object System.Collections.Generic.List[string]
$extractorPath = Join-Path $root "scripts\verify-api-contract"
Push-Location $extractorPath
try {
    $routeJSON = & go run . --router $routerPath
    if ($LASTEXITCODE -ne 0) {
        throw "Go route extractor failed."
    }
}
finally {
    Pop-Location
}
$routes = New-Object System.Collections.Generic.List[object]
$convertedRoutes = $routeJSON | ConvertFrom-Json
foreach ($route in $convertedRoutes) {
    [void]$routes.Add($route)
}
if ($routes.Count -eq 0) {
    throw "Go route extractor returned no routes."
}
foreach ($route in $routes) {
    if (-not $route.method -or -not $route.path) {
        throw "Go route extractor returned an incomplete route."
    }
    $method = ([string]$route.method).ToLowerInvariant()
    $path = Normalize-Path ([string]$route.path)
    $key = "{0} {1}" -f $method, $path
    [void]$backend.Add($key)
    [void]$backendItems.Add($key)
    [void]$backendPaths.Add($path)
}

$documented = [System.Collections.Generic.HashSet[string]]::new()
$documentedPaths = [System.Collections.Generic.HashSet[string]]::new()
$documentedItems = New-Object System.Collections.Generic.List[string]
$currentPath = $null
$openAPILines = @(Get-Content -LiteralPath $openAPIPath)
foreach ($line in $openAPILines) {
    if ($line -match '^  (/api/v1[^:]*):\s*$') {
        $currentPath = $Matches[1]
        [void]$documentedPaths.Add($currentPath)
        continue
    }
    if ($currentPath -and $line -match '^    (get|post|put|delete|patch):\s*$') {
        $key = "{0} {1}" -f $Matches[1], $currentPath
        [void]$documented.Add($key)
        [void]$documentedItems.Add($key)
        continue
    }
    if ($line -match '^  [^ ]') {
        $currentPath = $null
    }
}

$missingDocsList = New-Object System.Collections.Generic.List[string]
foreach ($item in $backendItems) {
    if (-not $documented.Contains($item)) {
        [void]$missingDocsList.Add($item)
    }
}
$staleDocsList = New-Object System.Collections.Generic.List[string]
foreach ($item in $documentedItems) {
    if (-not $backend.Contains($item)) {
        [void]$staleDocsList.Add($item)
    }
}
$missingDocs = @($missingDocsList.ToArray() | Sort-Object)
$staleDocs = @($staleDocsList.ToArray() | Sort-Object)
if ($missingDocs.Count -gt 0 -or $staleDocs.Count -gt 0) {
    if ($missingDocs.Count -gt 0) {
        Write-Error ("OpenAPI is missing routes:`n" + ($missingDocs -join "`n"))
    }
    if ($staleDocs.Count -gt 0) {
        Write-Error ("OpenAPI contains stale routes:`n" + ($staleDocs -join "`n"))
    }
    exit 1
}

$frontendRoutes = [System.Collections.Generic.HashSet[string]]::new()
$frontendItems = New-Object System.Collections.Generic.List[string]
$frontendContent = Get-Content -Raw -LiteralPath $frontendRoutesPath
$frontendMatches = [regex]::Matches($frontendContent, '(?:''|`)(/[^''`]+)(?:''|`)')
foreach ($match in $frontendMatches) {
    $raw = [string]$match.Groups[1].Value
    if ($raw -notmatch '^/') {
        continue
    }
    $normalized = Normalize-FrontendPath $raw
    if ($frontendRoutes.Add($normalized)) {
        [void]$frontendItems.Add($normalized)
    }
}

$frontendMissingBackendList = New-Object System.Collections.Generic.List[string]
$frontendMissingDocsList = New-Object System.Collections.Generic.List[string]
foreach ($item in $frontendItems) {
    if (-not $backendPaths.Contains($item)) {
        [void]$frontendMissingBackendList.Add($item)
    }
    if (-not $documentedPaths.Contains($item)) {
        [void]$frontendMissingDocsList.Add($item)
    }
}
$frontendMissingBackend = @($frontendMissingBackendList.ToArray() | Sort-Object)
$frontendMissingDocs = @($frontendMissingDocsList.ToArray() | Sort-Object)
if ($frontendMissingBackend.Count -gt 0 -or $frontendMissingDocs.Count -gt 0) {
    if ($frontendMissingBackend.Count -gt 0) {
        Write-Error ("Frontend API routes missing from backend:`n" + ($frontendMissingBackend -join "`n"))
    }
    if ($frontendMissingDocs.Count -gt 0) {
        Write-Error ("Frontend API routes missing from OpenAPI:`n" + ($frontendMissingDocs -join "`n"))
    }
    exit 1
}

foreach ($predictionPath in @(
    "/api/v1/predict/{fundCode}",
    "/api/v1/stock/{stockCode}/predict"
)) {
    $block = Get-OpenAPIPathBlock $openAPILines $predictionPath
    if ($block -notmatch '(?m)^      deprecated:\s+true\s*$') {
        throw "Prediction compatibility endpoint must be deprecated: $predictionPath"
    }
    foreach ($status in @('"400"', '"410"')) {
        if ($block -notmatch "(?m)^        $([regex]::Escape($status)):\s*$") {
            throw "Prediction compatibility endpoint $predictionPath is missing response $status."
        }
    }
}

Write-Host "API contract verified: $($backend.Count) routes; frontend API routes: $($frontendRoutes.Count)."
