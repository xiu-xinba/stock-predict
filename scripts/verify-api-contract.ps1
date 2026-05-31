param(
  [string]$BackendRouter = "backend-go/internal/api/router.go",
  [string]$OpenApiPath = "docs/api/openapi.yaml",
  [string]$FrontendRoutes = "frontend/src/api/routes.ts"
)

$ErrorActionPreference = "Stop"

function Convert-RoutePath {
  param([string]$Path)

  return ($Path -replace ':([A-Za-z][A-Za-z0-9_]*)', '{$1}' `
    -replace '\$\{fundCode\}', '{fundCode}' `
    -replace '\$\{stockCode\}', '{stockCode}' `
    -replace '\$\{code\}', '{stockCode}' `
    -replace '\$\{type\}', '{type}')
}

function Get-BackendRoutes {
  $content = Get-Content -Raw $BackendRouter
  $matches = [regex]::Matches($content, 'v1\.(GET|POST|PUT|DELETE|PATCH)\("([^"]+)"')
  $routes = foreach ($match in $matches) {
    Convert-RoutePath "/api/v1$($match.Groups[2].Value)"
  }
  return $routes | Sort-Object -Unique
}

function Get-OpenApiRoutes {
  $routes = Select-String -Path $OpenApiPath -Pattern '^\s{2}(/api/v1/[^:]+):\s*$' | ForEach-Object {
    $_.Matches[0].Groups[1].Value.Trim()
  }
  return $routes | Sort-Object -Unique
}

function Get-FrontendRoutes {
  $content = Get-Content -Raw $FrontendRoutes
  $matches = [regex]::Matches($content, '(?s)(?:''|`)(/[^''`]+)(?:''|`)')
  $routes = foreach ($match in $matches) {
    $raw = $match.Groups[1].Value
    if ($raw -notmatch '^/') { continue }
    Convert-RoutePath "/api/v1$raw"
  }
  return $routes | Sort-Object -Unique
}

function Compare-RouteSet {
  param(
    [string]$Name,
    [string[]]$Expected,
    [string[]]$Actual
  )

  $missing = $Expected | Where-Object { $_ -notin $Actual }
  if ($missing.Count -gt 0) {
    Write-Error "$Name missing routes:`n$($missing -join "`n")"
  }
}

$backend = @(Get-BackendRoutes)
$openapi = @(Get-OpenApiRoutes)
$frontend = @(Get-FrontendRoutes)

Compare-RouteSet "OpenAPI" $backend $openapi
Compare-RouteSet "Backend" $frontend $backend
Compare-RouteSet "OpenAPI for frontend routes" $frontend $openapi

Write-Host "API contract check passed."
Write-Host "Backend routes: $($backend.Count)"
Write-Host "OpenAPI routes: $($openapi.Count)"
Write-Host "Frontend API routes: $($frontend.Count)"
