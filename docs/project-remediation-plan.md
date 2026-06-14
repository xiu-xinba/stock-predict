# Project Remediation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Every behavior change follows superpowers:test-driven-development.

**Goal:** Resolve the audited deployment, concurrency, data integrity, frontend state, API contract, architecture, and workspace hygiene issues without changing the public market-data API.

**Architecture:** Work proceeds in seven independently verifiable tasks. Deployment and contracts are fixed first, correctness bugs second, frontend behavior third, and the application-layer migration only after behavior is protected by regression tests. The final task verifies the repository from the same commands used by CI and removes generated artifacts in a `finally` path.

**Tech Stack:** Go 1.26.4, Gin, GORM, PostgreSQL 16, Vue 3, TypeScript 6, Pinia, Vitest, Docker Compose, OpenAPI 3.0.3, PowerShell.

---

## Task 1: Secure Docker, Database Roles, and OpenAPI Contract

**Files:**
- Create: `backend-go/.dockerignore`
- Create: `backend-go/docker/postgres/init/01-runtime-role.sh`
- Modify: `backend-go/Dockerfile`
- Modify: `docker-compose.yml`
- Modify: `backend-go/.env.example`
- Modify: `docs/api/openapi.yaml`
- Modify: `scripts/verify-api-contract.ps1`
- Modify: `.github/workflows/ci.yml`
- Modify: `README.md`
- Modify: `docs/operations/deployment.md`
- Test: `scripts/tests/verify-api-contract.Tests.ps1`

- [ ] **Step 1: Write failing contract and deployment tests**

Create Pester-free PowerShell assertions so they run on a stock Windows runner:

```powershell
$ErrorActionPreference = "Stop"
$root = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)

& (Join-Path $root "scripts\verify-api-contract.ps1")
if ($LASTEXITCODE -ne 0) { throw "contract verification failed" }

$dockerIgnore = Get-Content -Raw (Join-Path $root "backend-go\.dockerignore")
foreach ($required in @(".env", ".venv", "data/", "*.exe", "__pycache__")) {
    if ($dockerIgnore -notmatch [regex]::Escape($required)) {
        throw "missing dockerignore rule: $required"
    }
}

$compose = Get-Content -Raw (Join-Path $root "docker-compose.yml")
foreach ($required in @("MIGRATION_DATABASE_URL", "DATABASE_URL", "POSTGRES_RUNTIME_USER")) {
    if ($compose -notmatch $required) { throw "compose missing $required" }
}
```

- [ ] **Step 2: Run the test and verify RED**

Run:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/tests/verify-api-contract.Tests.ps1
```

Expected: FAIL because `.dockerignore`, standard OpenAPI lint, and split database variables do not exist.

- [ ] **Step 3: Restrict the Docker build context**

Create `backend-go/.dockerignore` with:

```dockerignore
.git
.gitignore
.env
.env.*
!.env.example
.venv
akshare-service/.venv
data/
*.exe
*.test
bin/
tmp/
coverage/
__pycache__/
*.py[cod]
*.log
```

Change the build stage to copy only required Go sources:

```dockerfile
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
    && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate
```

Install trusted roots in the runtime image:

```dockerfile
RUN apk add --no-cache ca-certificates \
    && addgroup -S app \
    && adduser -S -G app app
```

- [ ] **Step 4: Split migration and runtime database identities**

Add a PostgreSQL init script which creates a login-only runtime role using environment variables:

```sh
#!/bin/sh
set -eu

psql --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" \
  --set=runtime_user="$POSTGRES_RUNTIME_USER" \
  --set=runtime_password="$POSTGRES_RUNTIME_PASSWORD" <<'SQL'
SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', :'runtime_user', :'runtime_password')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = :'runtime_user')\gexec
GRANT CONNECT ON DATABASE :"POSTGRES_DB" TO :"runtime_user";
GRANT USAGE ON SCHEMA public TO :"runtime_user";
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO :"runtime_user";
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO :"runtime_user";
SQL
```

Mount the init directory into PostgreSQL. Configure `stock-migrate` with `MIGRATION_DATABASE_URL` using the owner account and the API with `DATABASE_URL` using `POSTGRES_RUNTIME_USER` and `POSTGRES_RUNTIME_PASSWORD`. After migrations, grant runtime table and sequence privileges from the migration command before starting the API.

- [ ] **Step 5: Remove insecure curl fallbacks**

Delete `getViaCurl` and `fetchViaSystemCurl`, remove `os/exec` imports and `--insecure`, and make all call sites use the existing resilient Go HTTP clients with normal certificate verification. Update tests to assert no source file under `internal/infrastructure/providers` contains `--insecure` or `exec.CommandContext(..., "curl", ...)`.

- [ ] **Step 6: Make OpenAPI standards-valid**

Add `components.responses.ServiceUnavailable`:

```yaml
ServiceUnavailable:
  description: 上游服务或数据库暂不可用
  content:
    application/json:
      schema:
        $ref: "#/components/schemas/APIResponse"
      example:
        code: -1
        message: 服务暂不可用
        data: null
```

Replace the schema-level `examples` under `ErrorCode` with one valid `example`. Preserve both prediction operations with `deprecated: true`, a six-digit path pattern, `400`, and `410`.

Update `verify-api-contract.ps1` to:

1. Run `npx --yes @redocly/cli@2.20.3 lint docs/api/openapi.yaml --skip-rule operation-2xx-response --skip-rule operation-4xx-response --skip-rule info-license --skip-rule no-server-example.com`.
2. Compare Gin methods and paths as before.
3. Parse the prediction blocks and fail unless both contain `deprecated: true` and a `410` response.

- [ ] **Step 7: Add CI coverage**

Add a Docker job:

```yaml
docker:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - run: docker build -t stock-predict-api:test backend-go
```

Pin Redocly and `govulncheck` versions. Add the PowerShell contract test to the contract job.

- [ ] **Step 8: Verify GREEN**

Run:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/tests/verify-api-contract.Tests.ps1
npx --yes @redocly/cli@2.20.3 lint docs/api/openapi.yaml --skip-rule operation-2xx-response --skip-rule operation-4xx-response --skip-rule info-license --skip-rule no-server-example.com
docker compose config --quiet
```

Expected: all commands exit 0. If Docker Desktop is unavailable, record the build as environment-blocked but keep CI coverage.

## Task 2: Fix Provider Races, LRU Locking, and Sync State

**Files:**
- Modify: `backend-go/internal/infrastructure/providers/provider_router.go`
- Modify: `backend-go/internal/infrastructure/providers/index_quote.go`
- Modify: `backend-go/internal/infrastructure/providers/index_minute.go`
- Modify: `backend-go/internal/infrastructure/providers/stock_quote.go`
- Modify: `backend-go/internal/infrastructure/providers/stock_service.go`
- Modify: `backend-go/internal/infrastructure/providers/fund_quote.go`
- Modify: `backend-go/internal/platform/cache/lru.go`
- Modify: `backend-go/internal/infrastructure/providers/market_sync.go`
- Test: `backend-go/internal/infrastructure/providers/provider_router_test.go`
- Test: `backend-go/internal/platform/cache/lru_test.go`
- Create: `backend-go/internal/infrastructure/providers/market_sync_test.go`
- Modify: `.github/workflows/ci.yml`

- [ ] **Step 1: Write failing Provider race tests**

Add a typed result API test:

```go
func TestRouterRaceReturnsFirstSuccessfulValueWithoutPenalizingCanceledProvider(t *testing.T) {
    fast := newDelayedProvider("fast", 5*time.Millisecond, nil)
    slow := newDelayedProvider("slow", 200*time.Millisecond, nil)
    health := NewHealthMonitor(slog.Default(), fast.Name(), slow.Name())
    router := NewProviderRouter([]Provider{fast, slow}, health, RouterConfig{
        DefaultStrategy: StrategyRace,
        RaceTimeout: time.Second,
    }, slog.Default())

    got, err := FetchValue(router, context.Background(), CapIndexQuote, MarketCN, 2,
        func(ctx context.Context, provider Provider) (string, error) {
            return provider.Name(), provider.wait(ctx)
        })

    if err != nil || got != "fast" {
        t.Fatalf("got value=%q err=%v", got, err)
    }
    if status := health.GetStatus("slow"); status.FailCount != 0 {
        t.Fatalf("internal cancellation counted as failure: %+v", status)
    }
}
```

- [ ] **Step 2: Verify Provider RED**

Run:

```powershell
go test ./internal/infrastructure/providers -run "TestRouterRaceReturnsFirstSuccessfulValue" -count=1
```

Expected: FAIL because a value-returning race API and cancellation distinction do not exist.

- [ ] **Step 3: Implement value-returning Provider routing**

Introduce a generic package function because Go methods cannot have independent type parameters:

```go
type providerResult[T any] struct {
    provider Provider
    value    T
    err      error
}

func FetchValue[T any](
    router *ProviderRouter,
    ctx context.Context,
    capability Capability,
    market Market,
    raceCount int,
    fetch func(context.Context, Provider) (T, error),
) (T, error)
```

Each goroutine sends its own value. The receiver cancels after the first success. A goroutine must not call `RecordFailure` when `errors.Is(err, context.Canceled)` and the parent context is still active. Convert result-bearing call sites to this API; keep side-effect-only fallback routing only where no value is shared.

- [ ] **Step 4: Write failing LRU concurrency test**

Add a test that repeatedly races `Get`, `Set`, and eviction, then validates every map node appears exactly once in the linked list and `head.prev == nil`, `tail.next == nil`.

Run:

```powershell
go test ./internal/platform/cache -run TestDetailCacheConcurrentGetAndEvict -count=20
```

Expected: FAIL under the existing read-lock upgrade implementation or fail the explicit invariant.

- [ ] **Step 5: Fix LRU locking**

Use one write lock for the complete lookup:

```go
func (c *DetailCache) get(key string, maxAge time.Duration, enforceAge bool) (any, bool) {
    c.mu.Lock()
    defer c.mu.Unlock()
    entry, ok := c.items[key]
    if !ok || enforceAge && time.Since(entry.cachedAt) > maxAge {
        return nil, false
    }
    c.moveToFront(entry)
    return entry.value, true
}
```

- [ ] **Step 6: Write failing market-sync atomicity test**

Create a test that calls `syncOnce` and `Status` concurrently with stub dependencies and asserts a successful status never has an empty source. It must run under the race detector.

- [ ] **Step 7: Atomically update sync status**

Build `source` and `result` locally. At completion update `lastSyncTime`, `lastSyncSource`, `lastSyncResult`, and `syncCount` under one lock. Copy fields under lock before logging.

- [ ] **Step 8: Verify GREEN**

Run:

```powershell
go test ./internal/infrastructure/providers ./internal/platform/cache -count=2
go test -race ./internal/infrastructure/providers ./internal/platform/cache
```

Expected: zero failures and no race reports. Add `go test -race ./...` to the Linux backend CI job.

## Task 3: Fix Persistence Semantics, Errors, and Schema Verification

**Files:**
- Modify: `backend-go/internal/domain/fund/types.go`
- Modify: `backend-go/internal/domain/fund/repository.go`
- Modify: `backend-go/internal/domain/stock/repository.go`
- Modify: `backend-go/internal/infrastructure/database/fund_store.go`
- Modify: `backend-go/internal/infrastructure/database/stock_store.go`
- Modify: `backend-go/internal/infrastructure/database/search_store.go`
- Modify: `backend-go/internal/infrastructure/database/migrations.go`
- Modify: handlers and services consuming repository lookups
- Test: `backend-go/internal/infrastructure/database/persistence_test.go`
- Test: `backend-go/internal/infrastructure/database/db_test.go`
- Test: relevant handler tests

- [ ] **Step 1: Write failing zero-value and pinyin idempotency tests**

Add PostgreSQL integration tests:

```go
func TestSaveFundListAllowsRealZeroQuoteToReplaceOldValue(t *testing.T) {
    db := InitTestDB(t)
    store := NewFundStore(db)
    requireNoError(t, store.SaveFundList([]funddomain.FundItem{
        {FundCode: "000001", FundName: "测试基金", ChangePct: 1.25, QuoteDate: "2026-06-13"},
    }))
    requireNoError(t, store.SaveFundList([]funddomain.FundItem{
        {FundCode: "000001", FundName: "测试基金", ChangePct: 0, QuoteDate: "2026-06-14"},
    }))
    got, err := store.GetFundByCode("000001")
    requireNoError(t, err)
    if got.ChangePct != 0 { t.Fatalf("got %v", got.ChangePct) }
}

func TestSyncStocksIsIdempotentForPinyinAlt(t *testing.T) {
    db := InitTestDB(t)
    store := NewSearchStore(db)
    input := []stockdomain.StockItem{{StockCode: "600519", StockName: "贵州茅台", Pinyin: "gzmt", PinyinAlt: "guizhoumaotai"}}
    requireNoError(t, store.SyncStocks(input))
    requireNoError(t, store.SyncStocks(input))
    var model Stock
    requireNoError(t, db.First(&model, "stock_code = ?", "600519").Error)
    if model.Pinyin != "gzmt" { t.Fatalf("pinyin duplicated: %q", model.Pinyin) }
}
```

- [ ] **Step 2: Verify data RED**

Run:

```powershell
go test ./internal/infrastructure/database -run "TestSaveFundListAllowsRealZero|TestSyncStocksIsIdempotent" -count=1
```

Expected: the zero update and/or pinyin test fails with current upsert conversion.

- [ ] **Step 3: Implement explicit patch presence**

Do not use numeric zero as a missing sentinel. Add presence flags to ingestion-only patch types:

```go
type FundPatch struct {
    FundItem
    HasLatestNAV     bool
    HasEstimatedNAV  bool
    HasChangePct     bool
    HasReturn1M      bool
    HasReturn3M      bool
    HasReturn6M      bool
    HasReturn1Y      bool
    HasReturn3Y      bool
}
```

Provider parsers set flags when the source field is present. Repository upsert assignments update flagged fields even when their value is zero and preserve existing fields only when the flag is false. Keep API JSON models unchanged.

Persist `Pinyin` and `PinyinAlt` separately. Search SQL may concatenate them, but `stockDTOToModel` must not mutate `Pinyin`.

- [ ] **Step 4: Write failing database-error mapping tests**

Use a closed SQL connection or injected failing repository and assert:

```go
if _, err := store.GetFundByCode("000001"); err == nil {
    t.Fatal("expected database error")
}
```

At handler level, assert repository failure produces 503/500 and missing record produces 404.

- [ ] **Step 5: Propagate repository errors**

Change repository lookups to `(T, error)` or `(*T, error)`. Return `domain.ErrNotFound` only for `gorm.ErrRecordNotFound`; wrap all other errors. Update application/handler mapping accordingly.

- [ ] **Step 6: Write failing schema verification tests**

Create a database with only `funds`, `stocks`, and `schema_migrations`, then assert `VerifyDatabaseSchema` fails. Create a fully migrated schema and assert it passes.

- [ ] **Step 7: Verify complete schema**

Define:

```go
const latestMigrationVersion = 2
```

Record a migration version after all current schema and indexes exist. Verify:

- latest migration version is present;
- every model table exists;
- `pg_trgm` exists;
- required GIN indexes exist.

- [ ] **Step 8: Verify GREEN**

Run with PostgreSQL:

```powershell
go test ./internal/infrastructure/database ./internal/transport/http/router -count=2
go vet ./...
```

Expected: zero failures.

## Task 4: Fix Frontend CSRF, Async State, and Prediction Compatibility

**Files:**
- Modify: `frontend/src/shared/api/client.ts`
- Modify: `frontend/src/shared/api/__tests__/client.test.ts`
- Modify: `frontend/src/features/search/store/search.ts`
- Create: `frontend/src/features/search/__tests__/search-store.test.ts`
- Modify: `frontend/src/features/search/components/SearchOverlay.vue`
- Modify: `frontend/src/features/market/store/market.ts`
- Modify: `frontend/src/features/market/__tests__/market-store.test.ts`
- Modify: `frontend/src/features/funds/FundDetailView.vue`
- Modify: `frontend/src/features/funds/store/fundDetail.ts`
- Create: `frontend/src/features/funds/__tests__/fund-detail-route.test.ts`
- Modify: `frontend/src/app/router.ts`
- Modify: `frontend/src/features/prediction/__tests__/prediction-surface.test.ts`

- [ ] **Step 1: Write failing CSRF bootstrap tests**

Add tests proving a cold POST first performs one GET and concurrent POSTs share that GET:

```ts
it('bootstraps one CSRF token before concurrent cold mutations', async () => {
  const calls: string[] = []
  api.defaults.adapter = async (config) => {
    calls.push(`${config.method}:${config.url}`)
    const headers =
      config.method === 'get' ? new AxiosHeaders({ 'x-csrf-token': 'bootstrap-token' }) : new AxiosHeaders()
    if (config.method === 'post') {
      expect(config.headers.get('X-CSRF-Token')).toBe('bootstrap-token')
    }
    return { config, data: {}, headers, status: 200, statusText: 'OK' }
  }
  await Promise.all([api.post('/a'), api.post('/b')])
  expect(calls.filter((call) => call.startsWith('get:'))).toHaveLength(1)
})
```

- [ ] **Step 2: Verify CSRF RED**

Run:

```powershell
npm run test:run -- src/shared/api/__tests__/client.test.ts
```

Expected: FAIL because cold mutations do not bootstrap a token.

- [ ] **Step 3: Implement single-flight CSRF bootstrap**

Add an internal axios instance without mutation interceptors, plus:

```ts
let csrfBootstrap: Promise<string> | null = null

async function ensureCSRFToken(): Promise<string> {
  if (csrfToken) return csrfToken
  csrfBootstrap ??= csrfClient
    .get(API_ROUTES.health, { timeout: 5000 })
    .then((response) => {
      const token = response.headers['x-csrf-token']
      if (!token) throw new Error('CSRF bootstrap did not return a token')
      csrfToken = token
      return token
    })
    .finally(() => {
      csrfBootstrap = null
    })
  return csrfBootstrap
}
```

Make the request interceptor async and call `ensureCSRFToken` for mutation methods before setting the header.

- [ ] **Step 4: Write failing search and market-state tests**

Test that:

- `reset()` invalidates a deferred search response;
- a completed zero-result query renders `.search-empty`;
- canceling an older K-line/minute request does not clear loading owned by the newer request.

- [ ] **Step 5: Fix search and chart request ownership**

Increment `searchSeq` in `reset()` and pass an AbortSignal through `unifiedSearch`. Store the active controller and abort it on reset.

Render the search body whenever a non-empty query has completed:

```vue
v-if="store.query.trim() && (store.loading || store.searched || store.error)"
```

Track `searched` explicitly.

In every chart request `finally`, clear loading and delete the controller only when:

```ts
if (klineAbortControllers.get(code) === controller) {
  klineAbortControllers.delete(code)
  indexKlineLoading.set(code, false)
}
```

- [ ] **Step 6: Write failing route and invalid-code tests**

Assert `/predict/000001` resolves to `/predict`, and navigation from `/fund/000001` to `/fund/invalid` clears prior detail and displays a validation error.

- [ ] **Step 7: Implement compatibility and invalid-route handling**

Add:

```ts
{
  path: '/predict/:fundCode',
  redirect: '/predict',
}
```

Expose validity from `useFundCodeRoute`, and have `FundDetailView` call a store method that invalidates pending requests, clears `detail`, and stores `基金代码必须为6位数字`.

- [ ] **Step 8: Verify GREEN**

Run:

```powershell
npm run test:run
npm run lint -- --max-warnings 0
npx prettier --check "src/**/*.{ts,tsx,vue,css}"
npm run build
```

Expected: all commands exit 0.

## Task 5: Introduce the Application Use-Case Layer

**Files:**
- Create: `backend-go/internal/application/contracts.go`
- Create: `backend-go/internal/application/registry.go`
- Create/move: `backend-go/internal/application/fund/service.go`
- Create/move: `backend-go/internal/application/stock/service.go`
- Create/move: `backend-go/internal/application/market/service.go`
- Create/move: `backend-go/internal/application/search/service.go`
- Create/move: `backend-go/internal/application/watchlist/service.go`
- Modify: `backend-go/internal/app/app.go`
- Modify: `backend-go/internal/transport/http/handler/*.go`
- Modify: `backend-go/internal/transport/http/router/router.go`
- Modify: `backend-go/internal/infrastructure/providers/registry.go`
- Delete after migration: service files under `backend-go/internal/infrastructure/providers`
- Delete after migration: `backend-go/internal/infrastructure/providers/platform_aliases.go`
- Delete after migration: `backend-go/internal/infrastructure/providers/httpclient_aliases.go`
- Delete after migration if unused: `backend-go/internal/infrastructure/providers/cache_provider.go`
- Create: `backend-go/internal/architecture/architecture_test.go`
- Move/update relevant tests alongside application packages

- [ ] **Step 1: Write failing backend architecture tests**

Use `go list -deps -json` or `golang.org/x/tools/go/packages` from a test to assert:

```go
func TestTransportDoesNotImportInfrastructureImplementations(t *testing.T) {
    forbidden := []string{
        "stock-predict-go/internal/infrastructure/database",
        "stock-predict-go/internal/infrastructure/providers",
    }
    assertPackageDoesNotImport(t, "stock-predict-go/internal/transport/http/...", forbidden)
}

func TestDomainDoesNotImportOuterLayers(t *testing.T) {
    assertPackageDoesNotImport(t, "stock-predict-go/internal/domain/...", []string{
        "stock-predict-go/internal/application",
        "stock-predict-go/internal/infrastructure",
        "stock-predict-go/internal/transport",
    })
}
```

- [ ] **Step 2: Verify architecture RED**

Run:

```powershell
go test ./internal/architecture -count=1
```

Expected: FAIL because handlers import concrete database/provider packages.

- [ ] **Step 3: Define application contracts**

Create interfaces used by HTTP:

```go
type Services struct {
    Funds       FundUseCases
    Market      MarketUseCases
    Watchlist   WatchlistUseCases
    FundDetail  FundDetailUseCases
    Stocks      StockUseCases
    StockDetail StockDetailUseCases
    Search      SearchUseCases
    Operations  OperationsUseCases
}
```

Each interface contains only methods actually called by handlers. Transport owns no concrete GORM or provider types.

- [ ] **Step 4: Move business orchestration into application**

Move and repackage:

- fund search, filters, coverage, sync orchestration;
- stock search, ranking, synchronization orchestration;
- unified search and pagination;
- watchlist quote aggregation;
- market endpoint orchestration and sync status;
- fund/stock detail use cases.

External HTTP clients, Provider interfaces, ProviderRouter, health monitoring, TDX preloading, parsing, and upstream-specific code remain in infrastructure. Database stores implement domain/application ports.

- [ ] **Step 5: Rewire assembly and handlers**

`internal/app` constructs infrastructure adapters, then application services, then passes `application.Services` to `handler.New`. Handler fields must no longer include `*database.SearchStore` or `*providers.Registry`.

- [ ] **Step 6: Remove compatibility aliases and dead cache wiring**

Replace alias usages with imports from `platform/cache`, `platform/errors`, and `platform/httpclient`. Remove the unregistered CacheProvider unless it is deliberately inserted into the provider list and covered by tests.

- [ ] **Step 7: Verify GREEN**

Run:

```powershell
go test ./internal/architecture ./internal/application/... ./internal/transport/http/... ./internal/infrastructure/... -count=1
go vet ./...
go build ./...
```

Expected: zero failures and architecture tests pass.

## Task 6: Strengthen Frontend Architecture and Workspace Hygiene

**Files:**
- Modify: `frontend/src/app/__tests__/architecture.test.ts`
- Create: `.gitattributes`
- Modify: `.gitignore`
- Modify: `scripts/verify-commercial-readiness.ps1`
- Create: `scripts/verify-workspace-clean.ps1`
- Create: `scripts/tests/verify-workspace-clean.Tests.ps1`
- Modify: `.github/workflows/ci.yml`
- Delete: `backend-go/data/funds.json`
- Delete generated caches/logs/binaries found by the verification script

- [ ] **Step 1: Write failing frontend architecture fixtures**

Create temporary fixture files during the test and assert the scanner catches:

- `shared/x.ts` importing `../../features/funds`;
- one feature importing `../other/store/privateStore`;
- one feature importing another feature’s internal alias path.

The scanner must resolve both `@/` and relative paths before applying boundaries.

- [ ] **Step 2: Verify frontend architecture RED**

Run:

```powershell
npm run test:run -- src/app/__tests__/architecture.test.ts
```

Expected: FAIL because the current regex only checks alias imports.

- [ ] **Step 3: Implement resolved import boundary checks**

Extract imports with a parser-compatible regex for static imports and resolve:

```ts
function resolveSourceImport(importer: string, specifier: string): string | null {
  if (specifier.startsWith('@/')) return resolve(srcRoot, specifier.slice(2))
  if (specifier.startsWith('.')) return resolve(dirname(importer), specifier)
  return null
}
```

Normalize extension and `index` resolution before checking package ownership.

- [ ] **Step 4: Write failing workspace-clean test**

Assert the checker fails when a temporary `backend-go/test.exe`, `frontend/dist`, `__pycache__`, `.run-logs`, or `backend-go/data/funds.json` exists, while allowing `frontend/node_modules`, `backend-go/akshare-service/.venv`, and `.trae`.

- [ ] **Step 5: Implement workspace hygiene**

Add `.gitattributes`:

```gitattributes
* text=auto eol=lf
*.bat text eol=crlf
*.ps1 text eol=crlf
```

Create `verify-workspace-clean.ps1` using `Get-ChildItem` with explicit exclusions. Delete the legacy funds JSON.

Wrap commercial readiness in `try/finally`; remove `frontend/dist`, root Go binaries, Python `__pycache__`, `.pyc`, and logs created by the run. Set `PYTHONDONTWRITEBYTECODE=1` for Python tests. Pin `govulncheck`.

- [ ] **Step 6: Verify GREEN**

Run:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/tests/verify-workspace-clean.Tests.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify-workspace-clean.ps1
npm run test:run -- src/app/__tests__/architecture.test.ts
```

Expected: all commands exit 0 and retained dependency directories remain present.

## Task 7: Full Verification and Final Review

**Files:**
- Modify only files required by failures found during verification
- Review: all changed files from `git diff --name-status`

- [ ] **Step 1: Start PostgreSQL with split roles**

Set strong test-only environment values and run:

```powershell
$env:POSTGRES_PASSWORD="owner-test-password"
$env:POSTGRES_RUNTIME_USER="stock_app"
$env:POSTGRES_RUNTIME_PASSWORD="runtime-test-password"
$env:ADMIN_TOKEN="0123456789abcdef0123456789abcdef"
$env:CORS_ORIGINS="http://localhost:5173"
docker compose up -d postgres
docker compose run --rm --entrypoint stock-migrate backend
docker compose --profile app up -d --build backend
```

- [ ] **Step 2: Verify runtime role is not a DDL owner**

Connect with the runtime DSN and prove ordinary SELECT succeeds while `CREATE TABLE runtime_must_fail(id int)` fails with permission denied.

- [ ] **Step 3: Run the complete backend gate**

```powershell
cd backend-go
$files = @(gofmt -l .); if ($files.Count) { $files; exit 1 }
go test -count=2 ./...
go test -race ./...
go vet ./...
go build ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...
```

- [ ] **Step 4: Run the complete frontend and Python gates**

```powershell
cd frontend
npx prettier --check "src/**/*.{ts,tsx,vue,css}"
npm run lint -- --max-warnings 0
npm run test:run
npx vue-tsc --noEmit
npm run build
npm audit --audit-level=high
npm audit --omit=dev --audit-level=high

cd ..\backend-go\akshare-service
$env:PYTHONDONTWRITEBYTECODE="1"
.\.venv\Scripts\python.exe -m pip check
.\.venv\Scripts\python.exe -m unittest -v
```

- [ ] **Step 5: Run contract, Docker, and cleanliness gates**

```powershell
cd ..\..
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify-api-contract.ps1
docker build -t stock-predict-api:verification backend-go
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify-workspace-clean.ps1
git diff --check
```

- [ ] **Step 6: Request final independent reviews**

Dispatch one specification reviewer against `docs/project-remediation-design.md` and this plan. After approval, dispatch a code-quality reviewer over the complete diff. Fix every Critical and Important issue and rerun the affected gates.

- [ ] **Step 7: Record actual verification status**

Report exact passing test counts and any environment-blocked checks. Do not claim completion unless every non-environment-blocked command above exits 0 and generated artifacts are removed.
