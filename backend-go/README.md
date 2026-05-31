# Stock Predict Go Backend

This is the Gin-based Go backend for Stock Predict.

It keeps the same public API prefix:

```text
/api/v1
```

The API layer uses Gin route groups and middleware while keeping the original
public response shapes used by the frontend.

## Run

Install Go 1.22+ first.

Install dependencies:

```powershell
go mod tidy
```

```powershell
cd backend-go
go run ./cmd/api
```

Default URL:

```text
http://localhost:5070/api/v1/health
```

The existing Vite proxy points to `http://localhost:5070`, so the frontend works
with the Go backend without code changes.

To use another port:

```powershell
$env:PORT="5071"
go run ./cmd/api
```

Fund metadata is file-backed by default:

```powershell
$env:FUND_STORE_PATH="data/funds.json"
```

The backend can fill that store from Eastmoney's public full fund-code list and
latest NAV ranking metrics, using the same direct-HTTP style as
`simonlin1212/a-stock-data`:

```powershell
$env:FUND_UNIVERSE_URL="https://fund.eastmoney.com/js/fundcode_search.js"
$env:FUND_METRICS_URL="" # blank uses the runtime default rankhandler URL
$env:FUND_AUTO_SYNC_ON_START="true"
```

On startup, development builds auto-sync when the local store has fewer than
`FUND_AUTO_SYNC_MIN_COUNT` funds, or when an older local store has no trusted
`quote_source` metrics yet. `POST /api/v1/funds/sync` imports the remote fund
universe, merges Eastmoney rankhandler metrics, and then merges an optional
configured CSV. The CSV should be a manually curated fund universe file with at
least `fund_code`, `fund_name`, and `fund_type`:

```powershell
$env:FUND_SYNC_CSV_PATH="..\data\fund-universe.csv"
curl -X POST http://localhost:5070/api/v1/funds/sync
```

Eastmoney's fund-code list includes pinyin abbreviations and full spellings, so
search supports code, Chinese fund name, pinyin abbreviation, and full pinyin.
Watchlist quote refreshes can additionally use Tencent listed-fund quotes and
Eastmoney fund valuation estimates when `FUND_REALTIME_QUOTES_ENABLED=true`.

Prediction modeling has been split into a standalone future project. The Go
backend keeps the public prediction entry points, but
`GET /api/v1/predict/{fundCode}` and `GET /api/v1/stock/{stockCode}/predict`
currently return HTTP 501 with `feature_disabled` semantics.

## Implemented API Compatibility

- `GET /api/v1/health`
- `GET /api/v1/funds/search`
- `GET /api/v1/funds/filters`
- `POST /api/v1/funds/sync`
- `GET /api/v1/market/indices`
- `GET /api/v1/market/ranking/{gainers|losers}`
- `GET /api/v1/predict/{fundCode}`
- `POST /api/v1/watchlist/quotes`

## Migration Notes

This service owns the web/API layer. Model training should remain in
an independent project. This repository only preserves prediction routes as
stable placeholders for a later integration.

The current Go implementation uses a JSON-backed fund store with deterministic
seed data as fallback, plus CSV sync for bringing in collected fund metadata.
The next production steps are:

1. Replace the JSON store with SQLite or Postgres once concurrent writes and
   historical NAV queries are needed.
2. Connect scheduled fund and market data sync workers.
3. Add contract tests for the `/api/v1` response shapes used by the frontend.
