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
configured CSV. The CSV can come from `model-training` processed outputs or a
manually curated fund universe file, and should contain at least `fund_code`,
`fund_name`, and `fund_type`:

```powershell
$env:FUND_SYNC_CSV_PATH="..\model-training\data\processed\public_mvp_daily_weekly_index_fund_samples.csv"
curl -X POST http://localhost:5070/api/v1/funds/sync
```

Eastmoney's fund-code list includes pinyin abbreviations and full spellings, so
search supports code, Chinese fund name, pinyin abbreviation, and full pinyin.
Watchlist quote refreshes can additionally use Tencent listed-fund quotes and
Eastmoney fund valuation estimates when `FUND_REALTIME_QUOTES_ENABLED=true`.

To use the trained Python champion model, start `model-training`'s HTTP model
service first, then point the Go backend to it:

```powershell
cd ../model-training
python -m fund_model_training.serve_model `
  --model artifacts/public_mvp_index_fund_tournament_champion.joblib `
  --samples data/processed/public_mvp_daily_weekly_index_fund_samples.csv `
  --port 8090

cd ../backend-go
$env:MODEL_SERVICE_URL="http://127.0.0.1:8090"
go run ./cmd/api
```

`GET /api/v1/predict/{fundCode}` uses the model service for next-day prediction
when configured, and falls back to the Go baseline if the service is unavailable.
Model-service responses can include `return_decomposition`, `prediction_interval`,
and `signal_status`; the Go DTO preserves those fields so the frontend can show
the design-required interval coverage and block low-confidence actionable flags.
If a separate weekly model is running, set
`WEEKLY_MODEL_SERVICE_URL=http://127.0.0.1:8092`; only serve it after the
weekly retraining cycle has produced `model_registry/weekly_index_fund/current.json`,
and that service should return `next_week` in its prediction payload. If a
separate short-horizon model is running, set
`INTRADAY_MODEL_SERVICE_URL=http://127.0.0.1:8091`; that service should return
`intraday_3m` or `intraday_5m` in its prediction payload.

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
`model-training/`, and online model inference is currently plugged in through a
separate Python model service. A future ONNX runtime adapter can still be added
when the model contract stabilizes.

The current Go implementation uses a JSON-backed fund store with deterministic
seed data as fallback, plus CSV sync for bringing in collected fund metadata.
Baseline predictions remain available when Python model services are not
configured. The next production steps are:

1. Replace the JSON store with SQLite or Postgres once concurrent writes and
   historical NAV queries are needed.
2. Connect scheduled fund and market data sync workers.
3. Promote the model service behind rolling validation, model versioning, and
   rollback controls.
4. Add contract tests for the `/api/v1` response shapes used by the frontend.
