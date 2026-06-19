# 部署指南

## 环境要求

| 组件 | 版本 |
| --- | --- |
| Node.js | 20.19+，推荐 24 |
| Go | 1.26.4 |
| PostgreSQL | 16+，启用 `pg_trgm` |
| Python | 3.13（可选 AKShare 服务） |

## 发布门禁

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\verify-commercial-readiness.ps1
```

任何检查失败都应阻止发布。

## 生产配置

| 变量 | 要求 |
| --- | --- |
| `APP_ENV` | `production` |
| `HOST` | 容器内通常为 `0.0.0.0` |
| `POSTGRES_USER` / `POSTGRES_PASSWORD` | 数据库 owner/migration 身份；必须显式设置 |
| `POSTGRES_RUNTIME_USER` / `POSTGRES_RUNTIME_PASSWORD` | API runtime 身份；密码必须与 owner 不同 |
| `MIGRATION_DATABASE_URL` | owner/migration DSN，生产环境启用 TLS |
| `DATABASE_URL` | 最小权限 runtime DSN，生产环境启用 TLS |
| `ADMIN_TOKEN` | 至少 32 字符强随机值 |
| `CORS_ORIGINS` | 精确 HTTPS Origin，不允许 `*` |
| `TRUSTED_PROXIES` | 实际代理 IP/CIDR |
| `RUN_DATABASE_MIGRATIONS` | 生产 API 保持 `false` |

## 发布顺序

1. 备份 PostgreSQL。
2. 使用 `MIGRATION_DATABASE_URL` 对应的 owner 身份运行 `stock-migrate`。
3. 重新执行 `stock-grant-runtime`，授予新表/序列 DML 权限并撤销 runtime Schema CREATE 权限。
4. 使用 `DATABASE_URL` 对应的不具备 DDL 权限的 runtime 身份启动 API。
5. 检查 `/api/v1/health/live` 和 `/api/v1/health/ready`。
6. 部署前端，并使用 `frontend/nginx.conf` 或等价 CSP。
7. 执行搜索、行情、自选报价和管理接口认证 smoke test。

## Compose

```powershell
$env:POSTGRES_DB="stock_predict"
$env:POSTGRES_USER="stock_owner"
$env:POSTGRES_PASSWORD="<strong-owner-password>"
$env:POSTGRES_RUNTIME_USER="stock_app"
$env:POSTGRES_RUNTIME_PASSWORD="<different-runtime-password>"
$env:MIGRATION_DATABASE_URL="postgres://stock_owner:<strong-owner-password>@postgres:5432/stock_predict?sslmode=disable"
$env:DATABASE_URL="postgres://stock_app:<different-runtime-password>@postgres:5432/stock_predict?sslmode=disable"
$env:ADMIN_TOKEN="<32+-character-token>"
$env:CORS_ORIGINS="https://stock.example.com"
docker compose --profile app up -d --build
```

Compose 的 `migrate` 服务先执行 `stock-migrate`，随后运行权限脚本；`backend`
只在迁移和权限刷新成功后启动。Compose 只把 PostgreSQL/API 端口绑定到 `127.0.0.1`。
仓库提供的本地 Compose PostgreSQL 未配置 TLS，因此这两个容器内 DSN 使用
`sslmode=disable`。连接外部生产 PostgreSQL 时必须启用并校验证书，例如使用
`sslmode=verify-full` 和受信任的 CA；不要把本地 Compose DSN 直接用于生产外部数据库。

## 管理 POST 与 CSRF

Bearer Token 不能替代 CSRF。管理 POST 必须先通过 GET 获取 HttpOnly Cookie 和
`X-CSRF-Token`，然后在同一 Cookie 会话中发送二者：

```powershell
$session = New-Object Microsoft.PowerShell.Commands.WebRequestSession
$bootstrap = Invoke-WebRequest `
  -Uri "https://stock.example.com/api/v1/health/live" `
  -WebSession $session
$headers = @{
  Authorization = "Bearer $env:ADMIN_TOKEN"
  "X-CSRF-Token" = [string]$bootstrap.Headers["X-CSRF-Token"]
}
Invoke-RestMethod -Method Post `
  -Uri "https://stock.example.com/api/v1/funds/sync" `
  -WebSession $session `
  -Headers $headers
```

## 反向代理

- 同步配置 `TRUSTED_PROXIES`。
- 静态站点设置 CSP、HSTS 和 `frame-ancestors`。
- `/api/v1/metrics` 与健康模拟接口均要求 `ADMIN_TOKEN`，并应只允许运维网络访问。
- 不接受来自公网的伪造转发头。

## 回滚

| 对象 | 回滚 |
| --- | --- |
| 前端 | 恢复上一版不可变静态产物 |
| API | 恢复上一版镜像/二进制 |
| 数据库 | 使用迁移前快照或经过验证的向下迁移 |

PostgreSQL 是唯一数据源，不通过 JSON 文件回滚。
