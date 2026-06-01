# Commercial Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the frontend fund/stock data visibility failure, then reorganize the project toward commercial-grade backend, frontend, API, scripts, and documentation boundaries without changing the public product behavior.

**Architecture:** Keep the existing Go + Gin API and Vue 3 + TypeScript SPA. First fix the response-corruption bug in the backend gzip middleware, then add focused frontend rendering coverage, then introduce domain-oriented backend/frontend structure with compatibility imports so behavior stays stable throughout migration.

**Tech Stack:** Go 1.26, Gin, SQLite FTS5, Vue 3, TypeScript, Pinia, Element Plus, ECharts, Vitest, PowerShell quality scripts.

---

## Root-Cause Evidence For P0

Current local evidence shows the backend returns valid data without gzip, but corrupts responses when clients advertise gzip support:

```powershell
curl.exe -s "http://localhost:5070/api/v1/stocks/search?size=1"
```

Expected and observed: valid JSON with `code:0` and stock items.

```powershell
curl.exe -s --compressed -H "Accept-Encoding: gzip" "http://localhost:5070/api/v1/stocks/search?size=1"
```

Observed before fix: response body contains normal JSON followed by gzip footer bytes such as `‹...`, and `Content-Encoding` is missing. Root cause is `backend-go/internal/api/middleware.go`: `gzipMiddleware` sets `Content-Encoding: gzip` before writing, then `gzipResponseWriter.Write` sees that header, deletes it, writes uncompressed JSON, and the deferred `gzip.Writer.Close()` still writes gzip trailer bytes.

## File Structure Map

P0 files:

- Modify: `backend-go/internal/api/middleware.go` — fix gzip writer lifecycle.
- Modify: `backend-go/internal/api/router_test.go` — add gzip regression test that fails before the fix.
- Create: `frontend/src/__tests__/data-visibility.test.ts` — verify market, stock list, fund detail, stock detail, and watchlist-visible data paths render or store data when API data is available.

Commercial structure files:

- Create: `backend-go/internal/app/app.go` — application composition entry point.
- Modify: `backend-go/cmd/api/main.go` — delegate dependency assembly to `internal/app`.
- Create: `backend-go/internal/platform/response/response.go` — future response boundary documentation and types.
- Create: `backend-go/internal/platform/errors/errors.go` — future business error boundary documentation and aliases.
- Create: `frontend/src/app/router.ts` — router re-export entry point.
- Create: `frontend/src/app/bootstrap.ts` — Vue app bootstrap entry point.
- Modify: `frontend/src/main.ts` — delegate to app bootstrap.
- Create: `frontend/src/shared/api/routes.ts` — shared route constants.
- Modify: `frontend/src/api/routes.ts` — compatibility re-export.
- Create: `frontend/src/features/market/index.ts`, `frontend/src/features/funds/index.ts`, `frontend/src/features/stocks/index.ts`, `frontend/src/features/watchlist/index.ts`, `frontend/src/features/search/index.ts` — feature boundary entry points.
- Modify: `README.md`, `docs/architecture.md`, `docs/operations/deployment.md`, `docs/operations/maintenance.md` — document final structure and P0 troubleshooting.
- Modify: `scripts/verify-commercial-readiness.ps1` — keep quality gate stable and ensure no new checks skip existing gates.

## Task 1: Fix Gzip Response Corruption P0

**Files:**
- Modify: `backend-go/internal/api/router_test.go`
- Modify: `backend-go/internal/api/middleware.go`

- [ ] **Step 1: Add failing gzip regression test**

Append this test to `backend-go/internal/api/router_test.go` near the other router tests:

```go
func TestGzipMiddlewareReturnsValidCompressedJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stocks/search?size=1", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	newTestHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("expected gzip content encoding, got %q", got)
	}

	reader, err := gzip.NewReader(bytes.NewReader(rec.Body.Bytes()))
	if err != nil {
		t.Fatalf("expected gzip body, got invalid gzip stream: %v; raw=%q", err, rec.Body.String())
	}
	defer reader.Close()

	var response dto.APIResponse
	if err := json.NewDecoder(reader).Decode(&response); err != nil {
		t.Fatalf("decode gzipped API response: %v", err)
	}
	if response.Code != 0 {
		t.Fatalf("expected API code 0, got %d: %s", response.Code, response.Message)
	}
	data, err := remarshal[dto.StockSearchData](response.Data)
	if err != nil {
		t.Fatalf("decode stock search data: %v", err)
	}
	if len(data.Items) == 0 {
		t.Fatalf("expected stock items in gzipped response")
	}
}
```

Also add `compress/gzip` to the import list in `router_test.go`.

- [ ] **Step 2: Run test to verify it fails**

Run:

```powershell
cd backend-go
go test ./internal/api -run TestGzipMiddlewareReturnsValidCompressedJSON -v
```

Expected before fix: FAIL because `Content-Encoding` is not `gzip` or gzip reader reports an invalid gzip header.

- [ ] **Step 3: Fix gzip writer lifecycle**

Replace the `gzipResponseWriter` and `gzipMiddleware` implementation in `backend-go/internal/api/middleware.go` with this implementation:

```go
type gzipResponseWriter struct {
	gin.ResponseWriter
	gw      *gzip.Writer
	gzipped bool
}

func (w *gzipResponseWriter) Write(data []byte) (int, error) {
	if !w.gzipped {
		ct := w.Header().Get("Content-Type")
		ce := w.Header().Get("Content-Encoding")
		if ce != "" || !(strings.Contains(ct, "json") || strings.Contains(ct, "text")) {
			return w.ResponseWriter.Write(data)
		}
		w.Header().Del("Content-Length")
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")
		w.gw = gzip.NewWriter(w.ResponseWriter)
		w.gzipped = true
	}
	return w.gw.Write(data)
}

func (w *gzipResponseWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func (w *gzipResponseWriter) Close() error {
	if w.gw == nil {
		return nil
	}
	return w.gw.Close()
}

func gzipMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}
		writer := &gzipResponseWriter{ResponseWriter: c.Writer}
		c.Writer = writer
		c.Next()
		if err := writer.Close(); err != nil && !c.Writer.Written() {
			writeError(c, http.StatusInternalServerError, -1, "服务器内部错误")
		}
	}
}
```

- [ ] **Step 4: Run focused backend test**

Run:

```powershell
cd backend-go
go test ./internal/api -run TestGzipMiddlewareReturnsValidCompressedJSON -v
```

Expected after fix: PASS.

- [ ] **Step 5: Run backend API package tests**

Run:

```powershell
cd backend-go
go test ./internal/api -v
```

Expected: PASS.

- [ ] **Step 6: Commit P0 gzip fix**

Run:

```powershell
git add backend-go/internal/api/middleware.go backend-go/internal/api/router_test.go
git commit -m "fix: correct gzip response encoding"
```

## Task 2: Add Frontend Data Visibility Regression Coverage

**Files:**
- Create: `frontend/src/__tests__/data-visibility.test.ts`

- [ ] **Step 1: Write failing or protective frontend visibility tests**

Create `frontend/src/__tests__/data-visibility.test.ts`:

```ts
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import MarketView from '@/views/MarketView.vue'
import WatchlistView from '@/views/WatchlistView.vue'
import FundDetailView from '@/views/FundDetailView.vue'
import StockDetailView from '@/views/StockDetailView.vue'

vi.mock('@/composables/useStaggerEntry', () => ({
  useStaggerEntry: vi.fn(),
}))

vi.mock('@/api/market', () => ({
  fetchMarketIndices: vi.fn(async () => ({
    code: 0,
    message: 'success',
    data: [
      { code: '000001', name: '上证指数', market: 'cn', value: 3100, change: 12, change_pct: 0.39, high: 3120, low: 3080, prev_close: 3088, volume: 100000, mini_chart_data: [1, 2, 3], update_time: '09:30', data_source: 'test' },
    ],
  })),
  fetchFundRanking: vi.fn(async (type: 'gainers' | 'losers') => ({
    code: 0,
    message: 'success',
    data: [
      { rank: 1, fund_code: type === 'gainers' ? '000001' : '000002', fund_name: type === 'gainers' ? '华夏成长混合' : '易方达蓝筹精选', fund_type: '混合型', change_pct: type === 'gainers' ? 2.3 : -1.2, estimated_nav: 1.2345, quote_date: '2026-06-01', quote_source: 'test' },
    ],
  })),
}))

vi.mock('@/api/stock', () => ({
  fetchStockList: vi.fn(async () => ({
    code: 0,
    message: 'success',
    data: {
      items: [
        { stock_code: '600519', stock_name: '贵州茅台', market: 'SH', industry: '白酒', list_date: '', total_shares: 0, float_shares: 0, current_price: 1307.02, change_pct: -1.43, volume: 32923, amount: 431438, turnover_rate: 0.26, pe_ratio: 0, pb_ratio: 0, total_mv: 0, pinyin: 'gzmt' },
      ],
      total: 1,
      page: 1,
      size: 20,
    },
  })),
  fetchStockRanking: vi.fn(async (type: string) => ({
    code: 0,
    message: 'success',
    data: [
      { rank: 1, stock_code: type === 'gainers' ? '600519' : '000858', stock_name: type === 'gainers' ? '贵州茅台' : '五粮液', change_pct: type === 'gainers' ? 1.2 : -1.1, current_price: 1307.02, volume: 32923, amount: 431438 },
    ],
  })),
  fetchStockDetail: vi.fn(async () => ({
    code: 0,
    message: 'success',
    data: {
      basic: { stock_code: '600519', stock_name: '贵州茅台', market: 'SH', industry: '白酒', list_date: '', total_shares: 0, float_shares: 0 },
      quote: { price: 1307.02, open: 1327, high: 1327, low: 1301.31, prev_close: 1326, volume: 32923, amount: 431438, turnover_rate: 0.26, change_pct: -1.43, change_amt: -18.98, bid_price: 1306.99, ask_price: 1307.01, quote_time: '2026-06-01 13:22:20' },
      kline: { period: 'daily', klines: [{ date: '2026-06-01', open: 1327, close: 1307.02, high: 1327, low: 1301.31, volume: 32923, amount: 431438 }] },
      capital_flow: { main_net_inflow: 0, retail_net_inflow: 0, flow_history: [] },
      financials: { pe_ratio: 0, pb_ratio: 0, roe: 0, revenue: 0, net_profit: 0, eps: 0, gross_margin: 0, net_margin: 0, quarterly: [] },
      shareholders: { top10: [], institution_count: 0, institution_ratio: 0 },
    },
  })),
}))

vi.mock('@/api/fundDetail', () => ({
  fetchFundDetail: vi.fn(async () => ({
    code: 0,
    message: 'success',
    data: {
      basic: { fund_code: '000001', fund_name: '华夏成长混合', fund_type: '混合型', company: '华夏基金', manager: '阳琨', latest_nav: 1.333, cumulative_nav: 3.906, risk_level: '中高', inception_date: '2001-12-18' },
      quote: { fund_code: '000001', fund_name: '华夏成长混合', fund_type: '混合型', latest_nav: 1.333, estimated_nav: 1.356, change_pct: 1.73, quote_date: '2026-06-01', quote_source: 'test' },
      performance: { nav_history: [{ date: '2026-06-01', nav: 1.333, cumulative_nav: 3.906, change_pct: 1.73 }], return_1m: 1, return_3m: 2, return_6m: 3, return_1y: 4, return_3y: 5 },
      manager: { name: '阳琨', tenure_days: 8931, managed_size: '', fund_count: 1, bio: '' },
      portfolio: { top_holdings: [], sector_allocation: [] },
      risk: { volatility_1y: 1, max_drawdown_1y: -1, sharpe_1y: 1, beta_1y: 1 },
    },
  })),
}))

function createTestRouter(path: string) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/market', component: MarketView },
      { path: '/watchlist', component: WatchlistView },
      { path: '/fund/:fundCode', component: FundDetailView },
      { path: '/stock/:stockCode', component: StockDetailView },
    ],
  })
  router.push(path)
  return router
}

async function mountWithRouter(component: object, path: string) {
  const pinia = createPinia()
  setActivePinia(pinia)
  const router = createTestRouter(path)
  await router.isReady()
  const wrapper = mount(component, {
    global: {
      plugins: [pinia, router],
      stubs: {
        teleport: true,
        transition: false,
        'transition-group': false,
        ElIcon: true,
      },
    },
  })
  await flushPromises()
  return wrapper
}

describe('data visibility', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders fund rankings on market page', async () => {
    const wrapper = await mountWithRouter(MarketView, '/market')
    expect(wrapper.text()).toContain('华夏成长混合')
    expect(wrapper.text()).toContain('易方达蓝筹精选')
  })

  it('renders stock list on watchlist stock tab', async () => {
    const wrapper = await mountWithRouter(WatchlistView, '/watchlist?tab=stock')
    await flushPromises()
    expect(wrapper.text()).toContain('贵州茅台')
  })

  it('renders fund detail payload', async () => {
    const wrapper = await mountWithRouter(FundDetailView, '/fund/000001')
    expect(wrapper.text()).toContain('华夏成长混合')
    expect(wrapper.text()).toContain('华夏基金')
  })

  it('renders stock detail payload', async () => {
    const wrapper = await mountWithRouter(StockDetailView, '/stock/600519')
    expect(wrapper.text()).toContain('贵州茅台')
    expect(wrapper.text()).toContain('1307.02')
  })
})
```

- [ ] **Step 2: Run frontend visibility test**

Run:

```powershell
cd frontend
npm run test:run -- src/__tests__/data-visibility.test.ts
```

Expected after P0 backend fix and test stabilization: PASS. If the first run fails due to component assumptions, fix the component/store path that blocks rendering real API payloads; do not replace payloads with fake production data.

- [ ] **Step 3: Commit frontend visibility coverage**

Run:

```powershell
git add frontend/src/__tests__/data-visibility.test.ts
git commit -m "test: cover frontend data visibility paths"
```

## Task 3: Extract Backend Application Composition Boundary

**Files:**
- Create: `backend-go/internal/app/app.go`
- Modify: `backend-go/cmd/api/main.go`

- [ ] **Step 1: Write backend startup test through existing packages**

Add this test to a new file `backend-go/internal/app/app_test.go`:

```go
package app

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"stock-predict-go/internal/config"
)

func TestNewServerRespondsToHealth(t *testing.T) {
	cfg := config.Config{
		Port:            "0",
		Env:             "test",
		CORSOrigins:     []string{"http://localhost:5173"},
		FundStorePath:   "",
		ReadTimeout:     time.Second,
		WriteTimeout:    time.Second,
		ShutdownTimeout: time.Second,
	}
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

	server, cleanup, err := NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	defer cleanup()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	server.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected health 200, got %d: %s", rec.Code, rec.Body.String())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```powershell
cd backend-go
go test ./internal/app -run TestNewServerRespondsToHealth -v
```

Expected before implementation: FAIL because `internal/app` does not exist.

- [ ] **Step 3: Implement app composition**

Create `backend-go/internal/app/app.go`:

```go
package app

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"stock-predict-go/internal/api"
	"stock-predict-go/internal/config"
	"stock-predict-go/internal/data"
	"stock-predict-go/internal/service"
	"stock-predict-go/internal/store"
)

type CleanupFunc func()

func NewServer(cfg config.Config, logger *slog.Logger) (*http.Server, CleanupFunc, error) {
	mem, err := store.NewPersistentStore(cfg.FundStorePath)
	if err != nil {
		return nil, nil, err
	}
	if cfg.FundAutoSyncOnStart && (mem.CountFunds() < cfg.FundAutoSyncMinCount || (cfg.FundMetricsURL != "" && mem.CountQuotedFunds() == 0)) {
		result, err := service.NewFundService(mem).SyncFromSources(cfg.FundUniverseURL, cfg.FundMetricsURL, cfg.FundSyncCSVPath)
		if err != nil {
			logger.Warn("fund auto sync skipped", "error", err)
		} else {
			logger.Info("fund auto sync completed", "imported", result.Imported, "total", result.Total, "source", result.Source)
		}
	}

	searchIdx, err := store.NewSearchIndex("file:search_index?mode=memory", logger)
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() {
		_ = searchIdx.Close()
	}

	if err := mem.ReplaceStocks(data.LoadDefaultStocks()); err != nil {
		logger.Warn("failed to load default stocks", "error", err)
	}

	services := service.NewRegistry(mem, mem, cfg, logger, searchIdx)

	if cfg.StockAutoSyncOnStart {
		logger.Info("starting stock auto sync from eastmoney API...")
		syncCtx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		result, err := services.Stocks.SyncStocks(syncCtx)
		cancel()
		if err != nil {
			logger.Warn("stock auto sync failed, using default stocks", "error", err)
		} else {
			logger.Info("stock auto sync completed", "imported", result.Imported, "total", result.Total, "errors", result.Errors)
		}
	}

	if err := searchIdx.SyncFunds(mem.ListFunds()); err != nil {
		logger.Warn("failed to sync funds to search index", "error", err)
	}
	if err := searchIdx.SyncStocks(services.Stocks.ListStocks()); err != nil {
		logger.Warn("failed to sync stocks to search index", "error", err)
	}

	router := api.NewRouter(cfg, services, mem, logger, searchIdx)
	previousCleanup := cleanup
	cleanup = func() {
		router.Close()
		previousCleanup()
	}

	return &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: cfg.ReadTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}, cleanup, nil
}
```

- [ ] **Step 4: Modify main to delegate to app**

Update `backend-go/cmd/api/main.go` so it loads config, creates logger, calls `app.NewServer`, starts the returned server, and keeps the existing graceful shutdown behavior.

- [ ] **Step 5: Run backend app tests**

Run:

```powershell
cd backend-go
go test ./cmd/api ./internal/app ./internal/api -v
```

Expected: PASS.

- [ ] **Step 6: Commit backend app boundary**

Run:

```powershell
git add backend-go/cmd/api/main.go backend-go/internal/app/app.go backend-go/internal/app/app_test.go
git commit -m "refactor: extract backend app composition"
```

## Task 4: Add Backend Platform Boundary Packages

**Files:**
- Create: `backend-go/internal/platform/response/response.go`
- Create: `backend-go/internal/platform/errors/errors.go`

- [ ] **Step 1: Create response package**

Create `backend-go/internal/platform/response/response.go`:

```go
package response

type Envelope[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

func Success[T any](data T) Envelope[T] {
	return Envelope[T]{Code: 0, Message: "success", Data: data}
}
```

- [ ] **Step 2: Create errors package**

Create `backend-go/internal/platform/errors/errors.go`:

```go
package errors

type Code int

const (
	CodeSuccess         Code = 0
	CodeBadRequest     Code = -1
	CodeFeatureDisabled Code = -2
)
```

- [ ] **Step 3: Run platform package tests**

Run:

```powershell
cd backend-go
go test ./internal/platform/...
```

Expected: PASS or `[no test files]`.

- [ ] **Step 4: Commit platform boundary**

Run:

```powershell
git add backend-go/internal/platform
git commit -m "refactor: add backend platform boundaries"
```

## Task 5: Introduce Frontend App And Shared API Boundaries

**Files:**
- Create: `frontend/src/app/bootstrap.ts`
- Create: `frontend/src/app/router.ts`
- Modify: `frontend/src/main.ts`
- Create: `frontend/src/shared/api/routes.ts`
- Modify: `frontend/src/api/routes.ts`

- [ ] **Step 1: Create shared API routes**

Move the exact `API_ROUTES` object from `frontend/src/api/routes.ts` into `frontend/src/shared/api/routes.ts`.

- [ ] **Step 2: Preserve compatibility export**

Replace `frontend/src/api/routes.ts` with:

```ts
export { API_ROUTES } from '@/shared/api/routes'
```

- [ ] **Step 3: Create app router re-export**

Create `frontend/src/app/router.ts`:

```ts
export { default } from '@/router'
```

- [ ] **Step 4: Create app bootstrap**

Create `frontend/src/app/bootstrap.ts`:

```ts
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from '@/App.vue'
import router from '@/app/router'
import '@/style.css'

export function bootstrap() {
  const debounceResizeObserverErr = (() => {
    let timer: ReturnType<typeof setTimeout> | null = null
    return (e: ErrorEvent) => {
      if (e.message === 'ResizeObserver loop completed with undelivered notifications.') {
        e.stopImmediatePropagation()
        if (timer) clearTimeout(timer)
        timer = setTimeout(() => { timer = null }, 100)
      }
    }
  })()
  window.addEventListener('error', debounceResizeObserverErr)

  const app = createApp(App)
  app.use(createPinia())
  app.use(router)
  app.mount('#app')
}
```

- [ ] **Step 5: Update frontend main**

Replace `frontend/src/main.ts` with:

```ts
import { bootstrap } from '@/app/bootstrap'

bootstrap()
```

- [ ] **Step 6: Run frontend build**

Run:

```powershell
cd frontend
npm run build
```

Expected: PASS.

- [ ] **Step 7: Commit frontend app/shared boundary**

Run:

```powershell
git add frontend/src/app frontend/src/shared/api/routes.ts frontend/src/api/routes.ts frontend/src/main.ts
git commit -m "refactor: introduce frontend app and shared api boundaries"
```

## Task 6: Add Feature Boundary Entry Points

**Files:**
- Create: `frontend/src/features/market/index.ts`
- Create: `frontend/src/features/funds/index.ts`
- Create: `frontend/src/features/stocks/index.ts`
- Create: `frontend/src/features/watchlist/index.ts`
- Create: `frontend/src/features/search/index.ts`

- [ ] **Step 1: Create feature index files**

Create `frontend/src/features/market/index.ts`:

```ts
export { default as MarketView } from '@/views/MarketView.vue'
export { useMarketStore } from '@/stores/market'
export * from '@/api/market'
```

Create `frontend/src/features/funds/index.ts`:

```ts
export { default as FundDetailView } from '@/views/FundDetailView.vue'
export { useFundDetailStore } from '@/stores/fundDetail'
export * from '@/api/fundDetail'
export type * from '@/types/fund'
export type * from '@/types/fundDetail'
```

Create `frontend/src/features/stocks/index.ts`:

```ts
export { default as StockDetailView } from '@/views/StockDetailView.vue'
export { useStockDetailStore } from '@/stores/stockDetail'
export * from '@/api/stock'
export type * from '@/types/stock'
```

Create `frontend/src/features/watchlist/index.ts`:

```ts
export { useWatchlistStore } from '@/stores/watchlist'
export type * from '@/types/watchlist'
```

Create `frontend/src/features/search/index.ts`:

```ts
export { default as SearchOverlay } from '@/components/SearchOverlay.vue'
export { useSearchStore } from '@/stores/search'
export * from '@/api/search'
```

- [ ] **Step 2: Run TypeScript build**

Run:

```powershell
cd frontend
npm run build
```

Expected: PASS.

- [ ] **Step 3: Commit feature boundaries**

Run:

```powershell
git add frontend/src/features
git commit -m "refactor: add frontend feature boundaries"
```

## Task 7: Update Documentation And Quality Scripts

**Files:**
- Modify: `README.md`
- Modify: `docs/architecture.md`
- Modify: `docs/operations/deployment.md`
- Modify: `docs/operations/maintenance.md`
- Modify: `scripts/verify-commercial-readiness.ps1`

- [ ] **Step 1: Update README project structure**

Edit `README.md` so the project structure section describes:

```text
backend-go/internal/app
backend-go/internal/api
backend-go/internal/domain
backend-go/internal/infrastructure
backend-go/internal/platform
frontend/src/app
frontend/src/shared
frontend/src/features
docs/api
docs/operations
scripts
```

Also add a troubleshooting note:

```markdown
### 前端数据不可见排查

如果基金或股票数据在前端不可见，先确认后端 gzip 响应没有被污染：

```powershell
curl.exe -s --compressed "http://localhost:5070/api/v1/stocks/search?size=1"
```

输出必须是纯 JSON，不能出现 gzip 尾部乱码。随后运行：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\verify-commercial-readiness.ps1
```
```

- [ ] **Step 2: Update architecture document**

Edit `docs/architecture.md` to match the backend/frontend boundary names from the design spec and record that gzip response corruption was fixed as a P0 data visibility issue.

- [ ] **Step 3: Update operations docs**

Edit `docs/operations/deployment.md` and `docs/operations/maintenance.md` with:

```markdown
## Quality Gate

Run before deployment:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\verify-commercial-readiness.ps1
```

Deployment is blocked if API contract, Go tests, Go vet, frontend lint, frontend tests, or frontend build fail.
```

- [ ] **Step 4: Keep commercial readiness script stable**

Open `scripts/verify-commercial-readiness.ps1` and confirm it still runs these steps in order:

```powershell
API contract check
Go tests
Go vet
Frontend lint
Frontend tests
Frontend build
```

If any step is missing, restore it.

- [ ] **Step 5: Commit docs and script updates**

Run:

```powershell
git add README.md docs/architecture.md docs/operations/deployment.md docs/operations/maintenance.md scripts/verify-commercial-readiness.ps1
git commit -m "docs: document commercial project structure"
```

## Task 8: Final Verification

**Files:**
- No source edits unless verification exposes a regression.

- [ ] **Step 1: Verify gzip endpoint manually**

Run:

```powershell
curl.exe -s --compressed "http://localhost:5070/api/v1/stocks/search?size=1"
```

Expected: pure JSON response ending with `}}`, no raw gzip footer characters.

- [ ] **Step 2: Run backend tests**

Run:

```powershell
cd backend-go
go test ./...
go vet ./...
```

Expected: PASS.

- [ ] **Step 3: Run frontend tests and build**

Run:

```powershell
cd frontend
npm run lint
npm run test:run
npm run build
```

Expected: PASS.

- [ ] **Step 4: Run commercial readiness gate**

Run:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\verify-commercial-readiness.ps1
```

Expected: `Commercial readiness verification passed.`

- [ ] **Step 5: Record final status**

Run:

```powershell
git status --short --branch
git log --oneline -5
```

Expected: only intentional uncommitted files remain, and recent commits show the P0 fix plus structure/doc commits.
