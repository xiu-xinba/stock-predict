# Project Rules

## Tech Stack
- Backend: Go 1.26+ with Gin framework
- Frontend: Vue 3 + TypeScript + Vite + Element Plus + Pinia + ECharts
- Database: PostgreSQL 16 with GORM and pg_trgm search
- API Style: RESTful, JSON responses, `/api/v1` prefix

## Code Style
- Go: gofmt, go vet, tab indentation
- Frontend: ESLint + Prettier, 2-space indentation
- All responses wrapped in `{"code":0,"message":"success","data":...}`

## Commands
- API contract: `powershell -ExecutionPolicy Bypass -File scripts/verify-api-contract.ps1`
- Backend test: `cd backend-go && go test ./...`
- Backend vet: `cd backend-go && go vet ./...`
- Backend build: `cd backend-go && go build ./...`
- Frontend lint: `cd frontend && npm run lint`
- Frontend typecheck: `cd frontend && npx vue-tsc --noEmit`
- Frontend build: `cd frontend && npm run build`
- Frontend test: `cd frontend && npm run test:run`

## Commit Convention
- feat: new feature
- fix: bug fix
- refactor: code restructuring
- docs: documentation
- test: adding tests
- chore: maintenance tasks
- style: formatting
- perf: performance improvement

## Architecture
- Backend: `app -> transport/http -> domain interfaces`, with infrastructure implementations.
- Backend target: `backend-go/internal/{app,transport/http/{router,response,middleware,handler},domain/{fund,stock,market,search},infrastructure/{database,providers/{eastmoney,tencent,sina,tdx,ths,akshare,biying}},platform/{config,errors,cache,httpclient,observability}}`.
- Domain packages must not depend on Gin, GORM, HTTP handlers, or concrete providers.
- Infrastructure packages implement domain interfaces; `app` owns assembly and use-case orchestration.
- Frontend: `app + shared + feature-owned api/store/components/views`.
- Frontend target: `frontend/src/{app,shared,features/{funds,stocks,market,watchlist,search,prediction,settings}}`.
- Shared frontend code must not depend on feature packages; each feature exposes a deliberate public entry point.
- Prediction service has migrated to a separate project. The legacy prediction API remains an explicit `410 Gone` compatibility contract, and the UI must show a migrated-service placeholder rather than mock metrics or charts.

## Workspace Retention
- Keep dependency directories: `frontend/node_modules/` and `backend-go/akshare-service/.venv/`.
- Keep `.trae/`; do not add it to `.gitignore`.
- Generated output, logs, caches, `.vscode/`, `.superpowers/`, and `docs/superpowers/` must not be committed.
