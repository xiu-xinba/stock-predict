# 维护手册

## 日常检查

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\verify-commercial-readiness.ps1
```

该脚本会执行 Go `govulncheck`、npm 全量/生产依赖审计、Python 依赖一致性检查和各语言测试。

- `/api/v1/health/live` 和 `/api/v1/health/ready` 应返回 200。
- `/api/v1/market/health` 不应包含上游错误或凭据。
- 使用 `Authorization: Bearer <ADMIN_TOKEN>` 访问 `/api/v1/metrics`，错误率和延迟应在预期范围。

## 数据同步

```powershell
Invoke-RestMethod -Method Post `
  -Uri "http://localhost:5070/api/v1/funds/sync" `
  -Headers @{ Authorization = "Bearer <ADMIN_TOKEN>" }

Invoke-RestMethod -Method Post `
  -Uri "http://localhost:5070/api/v1/stocks/sync" `
  -Headers @{ Authorization = "Bearer <ADMIN_TOKEN>" }
```

同步前后记录基金/股票数量。API 重启不应降低股票数量；默认种子只在空表时使用。

## 数据库变更

1. 创建数据库快照。
2. 在验收环境运行 `go run ./cmd/migrate`。
3. 验证历史 `updated_at` 和核心行数。
4. 在生产使用发布身份运行同一迁移。
5. API 保持 `RUN_DATABASE_MIGRATIONS=false`。

## 常见故障

### Readiness 失败

检查 PostgreSQL 网络、TLS、账号和连接数；确认已执行迁移。不要通过删除列或清空表
恢复启动。

### 前端写请求 403

确认 GET 响应包含 `X-CSRF-Token`，后续写请求同时携带 CSRF Cookie 和该响应头。
跨 Origin 部署还需检查 CORS exposed headers 与 credentials。

### 反向代理后大量 429

检查 `TRUSTED_PROXIES` 和代理追加的 `X-Forwarded-For`。不要配置宽泛公网 CIDR。

### 数据源失败

保留缓存并展示更新时间。不要恢复明文 HTTP 回退或宽松 TLS 重协商。

## AKShare

Go 和 Python 必须使用相同 `AKSHARE_SERVICE_TOKEN`。服务默认监听
`127.0.0.1:8900`，健康接口无需令牌，数据接口必须认证。

```powershell
cd backend-go\akshare-service
.\.venv\Scripts\python.exe -m unittest -v
```

## API 契约

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\verify-api-contract.ps1
```

路由变化时同步更新 OpenAPI 和前端 API routes。

## 日志

允许记录请求 ID、路径、状态码、耗时、数据源名和脱敏错误类别。禁止记录完整 DSN、
管理员令牌、AKShare 令牌、Biying licence、Cookie 或 Authorization。
