# Market Data Source Repair Design

## Context

The prediction interface currently renders market indices, fund rankings, stock rankings, sector heat, and northbound flow from the Go backend. The backend already has real providers for several endpoints, but the behavior is uneven: some upstream failures collapse to empty successful responses, ranking fallbacks can reuse stale local repository values, and the frontend does not consistently show source freshness, loading, validation, or retrieval errors.

This design follows the data-source policy from `simonlin1212/a-stock-data`: use Tongdaxin/mootdx-style data first for quotes and curves, Tencent as the quote/index fallback, and Eastmoney only where it is the practical source for market-wide ranking or sector data. Because this project is Go-based and already uses `gitee.com/quant1x/gotdx`, the implementation will map the mootdx principle to the existing `gotdx` client instead of adding a Python sidecar.

## Goals

- Show authentic market data in the prediction interface: real-time stock top gainers, real-time stock top losers, Shanghai Composite current quote, Shanghai Composite historical K-line data, and complete intraday fluctuation curves with precise time points.
- Prefer data sources in this order: Tongdaxin via `gotdx`, Tencent Finance, then Eastmoney with throttling for source-specific data.
- Remove mock generators and hardcoded placeholder values from the affected market-data path.
- Surface loading states, error states, and last-updated timestamps in the prediction interface.
- Refresh market data without a full page reload.
- Validate numerical values, percentage changes, and curve time ordering before showing them as valid market data.

## Non-Goals

- Do not add a Python `mootdx` service or another runtime.
- Do not implement predictive model output; this work is limited to real market-data display in the prediction interface.
- Do not replace unrelated fund/stock detail pages unless shared contracts require a narrow update.
- Do not attempt to guarantee exchange-grade real-time delivery beyond public provider latency and market trading-session availability.

## Architecture

The Go backend remains the only integration boundary for third-party market data. The frontend continues calling `/api/v1/...`, and the backend owns provider selection, fallback, validation, caching, and source metadata.

Market data should be organized as a provider chain:

1. `gotdx` provider for CN index quotes, index K-line, and index minute curves.
2. Tencent provider for index quote and index minute fallback, plus stock quote/ranking support where appropriate.
3. Eastmoney provider for market-wide stock rankings and sector rankings, wrapped by a shared throttled HTTP helper.

The provider chain returns structured success only when validated data exists. If every provider fails or returns invalid data, the API should return an error response rather than an empty success payload. Partial success is allowed only when the response shape can identify what succeeded and what failed, or when stale cached data is explicitly marked as stale.

## Backend Design

### Data Providers

- Keep `IndexQuoteClient` as the index provider facade, but make fallback order explicit and testable.
- Use `gotdx` first for:
  - `FetchIndexQuotes`
  - `FetchIndexMinute`
  - `FetchIndexKline`
- Use Tencent as fallback for:
  - CN index current quote
  - CN index minute curve
  - CN index K-line where Tencent supports the required query
- Use Eastmoney only where this project needs market-wide ranking or sector data:
  - `/market/stock-ranking/:type`
  - `/market/sectors`

### Eastmoney Throttling

Add a shared Eastmoney request helper in the backend service layer. It should:

- Reuse a single HTTP client.
- Serialize Eastmoney calls through a mutex.
- Enforce a minimum interval of at least one second between calls, with small jitter.
- Apply a standard browser user agent and proper referer.
- Return typed errors for network, status, and parse failures.

This applies to `push2`, `push2his`, and any future `datacenter-web.eastmoney.com` endpoint.

### Validation

Validate provider payloads before caching or returning them:

- Index code must be one of the supported CN index codes for CN index endpoints.
- Prices, highs, lows, previous close, change, and change percentage must be finite numbers.
- Index price must be positive for quote and curve points.
- `high >= low` when both values are present.
- `change_pct` must be consistent with `(value - prev_close) / prev_close * 100` within a small tolerance when previous close is available.
- Minute curve points must have valid `HH:mm` times, be sorted ascending, and stay inside the CN market session windows.
- Duplicate minute points should be collapsed deterministically, keeping the latest provider value for that minute.
- Ranking items must have valid code, name, rank, finite `change_pct`, and nonzero current price when a price is available.

Invalid data is discarded. If all rows are discarded, the provider attempt is treated as failed.

### API Behavior

Existing endpoints remain stable:

- `GET /api/v1/market/indices`
- `GET /api/v1/market/index/:code/kline`
- `GET /api/v1/market/index/:code/minute`
- `GET /api/v1/market/stock-ranking/:type`
- `GET /api/v1/market/sectors`

Responses should include source metadata already present in the DTOs where possible:

- `data_source`
- `update_time`
- `is_closed`

Index minute and K-line endpoints keep their existing array payload shape. The prediction page displays source and update metadata from the associated index quote for the same code, and treats minute/K-line data as valid only after backend validation passes.

Handlers should return non-2xx API errors when required market data cannot be retrieved and no valid cached/stale data is available.

## Frontend Design

`PredictView.vue` remains the page-level coordinator. It should show:

- A page-level market refresh indicator while the first real load is running.
- A visible last-updated timestamp after successful data retrieval.
- A Shanghai Composite historical trend area backed by `GET /api/v1/market/index/000001/kline`.
- A compact error state with retry when required data cannot be retrieved.
- Partial section errors when stock rankings or index minute curves fail independently.
- Manual refresh without a full page reload.
- Automatic refresh using the existing store timer, with stale request cancellation retained.

Ranking components should no longer treat an empty item array as an indefinite skeleton state. They should receive explicit `loading` and `error` props, show skeleton rows only while loading, and show an error or empty state when loading has finished.

Index cards should distinguish:

- Quote loaded but minute curve still loading.
- Quote loaded and valid curve available.
- Quote loaded but curve failed or unavailable.

The Shanghai Composite historical trend area should distinguish:

- Quote loaded but historical K-line still loading.
- Historical K-line loaded and valid.
- Historical K-line failed or unavailable.

The frontend should not compute or invent market values beyond formatting provider data.

## Data Refresh

Refresh intervals should remain conservative:

- Index quotes and rankings: about 30 seconds.
- Index minute curves: about 30 seconds while the prediction view is mounted.
- Eastmoney-backed sector/ranking calls should respect backend throttling and frontend request de-duplication.

The store should track per-resource loading and error states for:

- market indices
- fund rankings
- stock rankings
- index minute curves by code
- Shanghai Composite historical K-line
- sector ranking

## Testing

Backend tests should cover provider parsing and validation with deterministic fixtures:

- Tencent index quote parsing preserves price, previous close, change amount, change percentage, high, low, volume, and update time.
- Tencent minute parsing produces ordered `HH:mm` points and correct per-minute volume deltas.
- Invalid minute times, duplicate points, nonfinite values, and impossible price values are rejected.
- Eastmoney throttled helper serializes calls and enforces a minimum interval using an injectable clock/sleeper.
- Stock ranking API does not silently fall back to stale local placeholder values when real ranking data is unavailable for the prediction interface path.
- Market handlers return errors when all providers fail and no valid cache exists.

Frontend tests should cover:

- Prediction page shows loading state before market data resolves.
- Prediction page renders last-updated/source information after success.
- Ranking panels show explicit error state after failed ranking fetch.
- Ranking panels do not show skeleton rows forever after an empty successful response.
- Shanghai Composite historical trend renders K-line data after success and an explicit error state after failure.
- Manual refresh calls the store refresh path without reloading the page.

## Rollout

Implement in small TDD slices:

1. Backend validation helpers and provider parse tests.
2. Eastmoney throttled request helper.
3. Provider fallback behavior and handler error behavior.
4. Frontend market store state expansion.
5. Prediction page loading/error/freshness UI.
6. End-to-end verification with backend tests, frontend tests, build, and local browser smoke test.

## Risks

- Public market APIs can change fields or temporarily block traffic. The provider chain and explicit error states reduce silent data corruption.
- `gotdx` connectivity may depend on network location. Tencent fallback covers quote and curve availability for common CN indices.
- Eastmoney ranking endpoints have anti-abuse controls. The shared throttled helper reduces request bursts and aligns with the referenced data-source guidance.
