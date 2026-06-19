# Stock Predict - 基金/股票行情系统

Stock Predict 是一个基金与股票行情展示系统，提供市场指数、基金/股票搜索、自选管理、详情页和多数据源降级。预测服务已迁移至独立项目，本仓库保留的旧预测 API 仅用于兼容调用，并返回 HTTP 410 Gone。

## 功能

- A 股、港股、美股主要指数与分时/K 线展示
- 基金、股票搜索与筛选
- 基金/股票自选列表及批量报价
- 股票详情、资金流向、财务指标和股东信息
- PostgreSQL `pg_trgm` 拼音/代码检索
- 腾讯、东方财富、新浪、通达信、同花顺等数据源路由与健康监控
- 可选 AKShare 内部服务和 BiyingAPI 数据源

## 技术栈

| 模块 | 技术 |
| --- | --- |
| 前端 | Vue 3、TypeScript、Pinia、ECharts、Element Plus、Vite 8 |
| API | Go 1.26.4、Gin、GORM |
| 数据库 | PostgreSQL 16、`pg_trgm` |
| 可选数据服务 | Python 3.13、FastAPI、AKShare |
| 质量门禁 | Vitest、Go test/vet、govulncheck、GitHub Actions |

## 环境要求

- Node.js `>=20.19`（推荐 Node.js 24 LTS）
- Go `1.26.4`
- PostgreSQL `16+`
- Python `3.13`（仅 AKShare 服务需要）

## 快速开始

### 1. 启动 PostgreSQL

Compose 使用独立的 owner/migration 与 API runtime 身份，不提供默认账号或密码：

```powershell
$env:POSTGRES_DB="stock_predict"
$env:POSTGRES_USER="stock_owner"
$env:POSTGRES_PASSWORD="<strong-owner-password>"
$env:POSTGRES_RUNTIME_USER="stock_app"
$env:POSTGRES_RUNTIME_PASSWORD="<different-runtime-password>"
$env:MIGRATION_DATABASE_URL="postgres://stock_owner:<strong-owner-password>@localhost:5432/stock_predict?sslmode=disable"
$env:DATABASE_URL="postgres://stock_app:<different-runtime-password>@localhost:5432/stock_predict?sslmode=disable"
docker compose up -d postgres
```

数据库端口只绑定到 `127.0.0.1`。初始化脚本创建 runtime 角色，撤销 Schema
CREATE 与数据库 TEMPORARY 权限，并为迁移账号创建的表和序列设置默认 DML 权限。
仓库提供的本地 Compose PostgreSQL 未启用 TLS，因此容器内 DSN 使用
`sslmode=disable`；外部生产 PostgreSQL 必须启用证书校验，例如
`sslmode=verify-full` 配合受信任 CA。

### 2. 配置并迁移后端

```powershell
cd backend-go
Copy-Item .env.example .env
```

至少检查以下配置：

```dotenv
APP_ENV=development
HOST=127.0.0.1
PORT=5070
MIGRATION_DATABASE_URL=postgres://stock_owner:<owner-password>@localhost:5432/stock_predict?sslmode=disable
DATABASE_URL=postgres://stock_app:<runtime-password>@localhost:5432/stock_predict?sslmode=disable
ADMIN_TOKEN=<explicit-development-token>
CORS_ORIGINS=http://localhost:5173
RUN_DATABASE_MIGRATIONS=false
```

`APP_ENV` 和 `DATABASE_URL` 没有代码内默认值；缺失或非法时 API 会拒绝启动。
`cmd/migrate` 当前读取 `DATABASE_URL`，因此本地直接执行时应临时传入 owner DSN。

执行迁移并启动 API：

```powershell
$runtimeDatabaseUrl = $env:DATABASE_URL
$env:DATABASE_URL = $env:MIGRATION_DATABASE_URL
go run ./cmd/migrate
$env:DATABASE_URL = $runtimeDatabaseUrl
go run ./cmd/api
```

示例配置显式关闭 API 进程迁移。生产环境必须保持
`RUN_DATABASE_MIGRATIONS=false`，并在发布步骤中使用 owner DSN 先运行 `cmd/migrate`。

### 3. 启动前端

```powershell
cd ..\frontend
npm ci
npm run dev
```

前端默认使用 Vite 代理访问 `http://localhost:5070`。

## 开发命令

### 后端

```powershell
cd backend-go
go run ./cmd/migrate
go run ./cmd/api
go test ./...
go vet ./...
go build ./...
```

生产 API 不执行 DDL。使用当前 Compose 镜像时，`migrate` 服务会先运行
`stock-migrate`，再运行 `stock-grant-runtime` 为 runtime 角色刷新表和序列权限：

```powershell
docker compose --profile app build
docker compose --profile app run --rm migrate
docker compose --profile app up -d backend
```

### 前端

```powershell
cd frontend
npm run dev
npm run lint
npm run test:run
npm run build
npx prettier --check "src/**/*.{ts,tsx,vue,css}"
```

Axios 客户端默认携带 Cookie，并从 API 响应的 `X-CSRF-Token` 头获取写请求令牌。
生产静态站点可使用 `frontend/nginx.conf`，该配置包含 CSP 和其他浏览器安全响应头。

### AKShare 服务

```powershell
cd backend-go\akshare-service
.\.venv\Scripts\python.exe -m pip check
.\.venv\Scripts\python.exe -m unittest -v
```

## 健康检查

| 路径 | 用途 |
| --- | --- |
| `/api/v1/health/live` | 进程存活探测 |
| `/api/v1/health/ready` | 数据库就绪探测 |
| `/api/v1/health` | 兼容路径，语义等同 readiness |

公开的数据源健康接口只返回状态和失败次数，不返回上游原始错误或凭据相关 URL。

## 管理接口

`GET /api/v1/metrics` 要求 Bearer Token。`POST /api/v1/funds/sync`、
`POST /api/v1/stocks/sync` 和健康模拟接口同时要求 Bearer Token、CSRF Cookie
及匹配的 `X-CSRF-Token`：

```text
Authorization: Bearer <ADMIN_TOKEN>
```

项目不再内置 `dev-admin-token`。开发和生产环境都必须显式配置令牌。

浏览器写请求使用 HttpOnly CSRF Cookie 与 `X-CSRF-Token` 响应头。Axios 客户端会保存响应头令牌，并在后续 POST/PUT/PATCH/DELETE 请求中自动发送。命令行客户端必须先发送 GET，并在写请求中复用同一 Cookie 会话和响应头令牌。

## 数据与迁移安全

- 默认股票仅在空表时种入，重启不会清空已有股票。
- Schema 变更由 `schema_migrations` 记录，并在 PostgreSQL advisory lock 下执行。
- 历史 `updated_at` 字符串列使用类型转换迁移，不删除原有数据。
- 数据库日志只记录主机和数据库名，不记录用户名、密码或查询参数。

## AKShare 服务

AKShare 只应部署在回环地址或内部网络：

```powershell
cd backend-go\akshare-service
.\.venv\Scripts\python.exe -m pip install -r requirements.txt
$env:AKSHARE_SERVICE_TOKEN="<random-service-token>"
.\.venv\Scripts\python.exe main.py
```

Go API 同时配置：

```dotenv
AKSHARE_URL=http://localhost:8900
AKSHARE_SERVICE_TOKEN=<same-token>
```

未配置服务令牌时，Go API 不注册 AKShare provider。公网数据源必须使用 HTTPS。

## 数据同步

默认股票种子只会在股票表为空时写入，不会在重启时覆盖已有股票。远程同步使用 upsert/受控替换流程。

后端同步行为可通过 `backend-go/.env` 配置：

| 变量 | 作用 |
| --- | --- |
| `FUND_UNIVERSE_URL` | 基金全量代码列表来源，默认使用东方财富公开列表 |
| `FUND_METRICS_URL` | 基金净值与收益排行来源；留空时使用运行时默认地址 |
| `FUND_AUTO_SYNC_ON_START` | 启动时是否自动补全基金底库 |
| `FUND_AUTO_SYNC_MIN_COUNT` | 触发基金底库自动补全的最小记录数，默认 `1000` |
| `FUND_SYNC_CSV_PATH` | 可选本地基金 CSV，至少包含 `fund_code`、`fund_name`、`fund_type` |
| `FUND_REALTIME_QUOTES_ENABLED` | 自选刷新时是否启用腾讯场内基金行情和东方财富基金估值 |
| `STOCK_AUTO_SYNC_ON_START` | 启动时是否自动同步股票列表，默认关闭 |

离线开发时应关闭两类启动同步，并按需配置本地 CSV：

```dotenv
FUND_AUTO_SYNC_ON_START=false
STOCK_AUTO_SYNC_ON_START=false
FUND_SYNC_CSV_PATH=..\data\fund-universe.csv
```

```powershell
$session = New-Object Microsoft.PowerShell.Commands.WebRequestSession
$bootstrap = Invoke-WebRequest `
  -Uri "http://localhost:5070/api/v1/health/live" `
  -WebSession $session
$csrfToken = [string]$bootstrap.Headers["X-CSRF-Token"]
$adminHeaders = @{
  Authorization = "Bearer $env:ADMIN_TOKEN"
  "X-CSRF-Token" = $csrfToken
}

Invoke-RestMethod -Method Post `
  -Uri "http://localhost:5070/api/v1/funds/sync" `
  -WebSession $session `
  -Headers $adminHeaders

Invoke-RestMethod -Method Post `
  -Uri "http://localhost:5070/api/v1/stocks/sync" `
  -WebSession $session `
  -Headers $adminHeaders
```

## 质量门禁

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\verify-api-contract.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\verify-commercial-readiness.ps1
```

门禁覆盖：

- Go 格式、测试、vet、构建和可达调用链漏洞扫描
- 前端 Prettier、零警告 lint、测试、构建、全量及生产依赖审计
- Python 依赖一致性、字节码编译和单元测试；CI 额外执行 `pip-audit`
- Redocly OpenAPI 标准校验、后端路由方法/路径一致性及预测端点废弃/410 不变量

GitHub Actions 配置位于 `.github/workflows/ci.yml`。

## 项目结构

当前代码已收敛为以下商业架构边界：

```text
stock-predict/
├── backend-go/
│   ├── cmd/api/                 # API 进程
│   ├── cmd/migrate/             # 显式数据库迁移
│   ├── internal/app/            # 应用装配与用例编排
│   ├── internal/transport/http/ # Gin 路由、处理器、中间件与响应
│   ├── internal/domain/         # fund、stock、market、search 领域
│   ├── internal/infrastructure/ # PostgreSQL 与外部数据源实现
│   ├── internal/platform/       # 配置、错误、缓存、HTTP 与可观测性
│   └── akshare-service/         # 可选内部 FastAPI 服务
├── frontend/
│   └── src/
│       ├── app/                 # 启动、路由、全局样式与应用测试
│       ├── shared/              # 无业务依赖的 API、组件、组合式函数和工具
│       └── features/            # funds、stocks、market、watchlist 等业务模块
├── docs/api/openapi.yaml        # API 契约
├── docs/operations/             # 部署与维护说明
├── scripts/                     # 本地质量门禁
└── docker-compose.yml
```

## 文档

- [架构说明](docs/architecture.md)
- [架构重构决策记录](docs/architecture-refactor.md)
- [OpenAPI](docs/api/openapi.yaml)
- [部署指南](docs/operations/deployment.md)
- [维护手册](docs/operations/maintenance.md)
- [数据源合规](docs/operations/data-source-compliance.md)

## License

MIT
