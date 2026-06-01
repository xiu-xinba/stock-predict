# 部署指南

本文档说明 Stock Predict 在开发、验收和生产环境中的部署顺序、必要配置和回滚检查点。

## 1. 环境要求

| 组件 | 版本/要求 | 说明 |
| --- | --- | --- |
| Node.js | 18+ | 构建 Vue 前端 |
| Go | 1.26+ | 构建和运行 Go API |
| SQLite FTS5 | 由 `modernc.org/sqlite` 提供 | 后端搜索索引 |
| HTTPS 入口 | Nginx、Caddy 或云负载均衡 | 生产环境必须启用 |

## 2. 部署前门禁

在发布前必须运行以下命令并确认全部通过：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\verify-commercial-readiness.ps1
```

该脚本会串行执行 API 契约校验、Go 测试、Go vet、前端 lint、前端测试和前端构建。也可以手动分步执行：

```powershell
cd backend-go
go test ./...
go vet ./...

cd ../frontend
npm run lint
npm run test:run
npm run build
```

如果任一命令失败，不允许继续发布。先修复失败项，再重新执行完整门禁。

## Quality Gate

Run before deployment:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\verify-commercial-readiness.ps1
```

Deployment is blocked if API contract, Go tests, Go vet, frontend lint, frontend tests, or frontend build fail.

## 3. 后端配置

生产环境至少配置以下环境变量：

| 变量 | 要求 | 说明 |
| --- | --- | --- |
| `APP_ENV` | `production` | 启用生产安全语义 |
| `PORT` | 默认 `5070` | Go API 监听端口 |
| `ADMIN_TOKEN` | 32 字符以上强随机值 | 管理同步接口认证 |
| `CORS_ORIGINS` | 精确 HTTPS 域名 | 禁止生产使用 `*` |
| `FUND_STORE_PATH` | 可持久化路径 | 基金数据 JSON 存储 |
| `READ_TIMEOUT_SECONDS` | 默认 `8` | HTTP 读超时 |
| `WRITE_TIMEOUT_SECONDS` | 默认 `12` | HTTP 写超时 |
| `SHUTDOWN_TIMEOUT_SECONDS` | 默认 `8` | 优雅关闭超时 |

生产环境不得使用 `dev-admin-token`。

## 4. 运行指标

Go API 提供 `/api/v1/metrics` 作为最小运行指标接口，返回：

- `request_count`：已记录请求总数。
- `error_count`：HTTP 4xx/5xx 请求总数。
- `in_flight`：当前处理中请求数。
- `avg_duration_ms`：平均请求耗时。
- `status_counts`：按状态码统计的请求数。
- `uptime_seconds`：进程运行时间。

生产环境应在反向代理或内网层限制该接口访问范围。

## 5. 启动顺序

1. 启动 Go API。
2. 确认 `/api/v1/health` 返回 `status=ok`。
3. 构建前端静态资源并部署到静态站点目录。
4. 通过反向代理将 `/api/v1` 转发到 Go API。
5. 执行一次核心接口 smoke test：健康检查、搜索、行情、预测占位、自选报价。

## 6. 本地验收命令

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\verify-api-contract.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\verify-commercial-readiness.ps1
```

## 7. 回滚策略

| 回滚对象 | 回滚方式 | 验证 |
| --- | --- | --- |
| 前端静态资源 | 恢复上一版构建产物 | 打开首页、行情页、预测页 |
| Go API | 切回上一版可执行文件或镜像 | `/api/v1/health` 和核心接口通过 |
| 数据文件 | 恢复上一版 `funds.json` 或数据快照 | 搜索和自选报价可用 |

回滚后必须保留失败版本日志和配置，便于事后分析。

## 8. 发布后检查

- 访问前端首页、行情页、预测页、基金详情页和股票详情页。
- 调用 `/api/v1/health`，确认基金和股票状态符合预期。
- 调用 `/api/v1/metrics`，确认请求数、错误数、状态码分布和 uptime 正常返回。
- 检查同步接口必须需要管理员令牌。
- 检查生产响应头包含安全头。
- 检查预测页和预测接口展示独立项目占位语义。
