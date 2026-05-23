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
`model-training/`, and online model inference can later be plugged in as a
separate Python model service or an ONNX runtime adapter.

The current Go implementation uses deterministic in-memory seed data and
baseline predictions. The next production steps are:

1. Add a persistent store, preferably SQLite or Postgres.
2. Connect real fund and market data sync workers.
3. Add a `PredictionProvider` implementation that calls Python model serving or
   loads ONNX directly.
4. Add contract tests for the `/api/v1` response shapes used by the frontend.
