# 数据源合规与稳定性策略

本文档说明股票/指数数据获取的合规边界、请求治理策略、缓存回退和异常处置流程。

## 合规边界

允许的稳定性措施：

- 使用固定、透明的 `User-Agent` 和目标站公开页面对应的 `Referer`。
- 按数据源限频，遵守 `Retry-After`，在 403/429/503 后进入冷却窗口。
- 使用内存缓存、PostgreSQL 行情缓存和仅空库启用的本地种子降低重复请求。
- 使用合法备用数据源和本地缓存降级，并在响应中保留 `data_source`。
- 监控数据源健康状态，人工复核目标站 robots/TOS 和访问政策。

禁止的规避行为：

- 不使用代理池或 IP 轮换规避封禁。
- 不模拟真人浏览行为规避访问控制。
- 不绕过验证码、登录、JS challenge 或反自动化脚本。
- 不抓取未进入白名单的域名。

## 当前实现

后端在 `backend-go/internal/platform/httpclient` 与
`backend-go/internal/infrastructure/providers` 中提供统一请求治理层：

- `source_policy.go`：定义 Eastmoney、Tencent、Sina 的请求策略。
- `resilient_http_client.go`：统一应用请求头、同源限频、`Retry-After` 冷却、错误冷却和相同 GET 请求合并。
- `stock_quote.go`：股票批量报价支持 `balanced` 与 `realtime` 两种新鲜度策略；上游 429/403/5xx 或空结果时，使用 stale cache 回退。
- `cache_helper.go`：过期缓存不会立即删除，可用于短时故障回退，最终由 LRU 容量淘汰。

## 实时优先策略

`POST /api/v1/stocks/quotes` 支持可选请求字段：

```json
{
  "codes": ["600519"],
  "freshness": "realtime"
}
```

- `balanced`：默认策略，交易时段内 15 秒新鲜期，适合排行、后台批量刷新和非关键视图。
- `realtime`：交易时段内 3 秒新鲜期；3 至 15 秒的缓存会立即返回并触发后台刷新，超过 15 秒则同步请求上游。
- 非交易时段自动放宽到 60 秒新鲜期，降低无意义请求。
- 前端自选股与股票详情页 quote 刷新使用 `realtime`，刷新间隔为 3 秒。

该策略提升的是“可用数据的新鲜度”，不是绕过目标站限制。若上游返回 403/429/503，系统仍会尊重冷却窗口并返回可用缓存。

## 数据源策略

| 数据源 | 默认限频 | 冷却触发 | 默认冷却 | 主要用途 |
| --- | ---: | --- | ---: | --- |
| Eastmoney | 1 req/s | 403/429/503、网络错误 | 30s / 5s | 股票列表、排行、详情、K 线 |
| Tencent | 0.5s 间隔 | 403/429/503、网络错误 | 20s / 5s | 实时报价、指数、分时 |
| Sina | 1 req/s | 403/429/503、网络错误 | 30s / 5s | 板块备用数据 |

如果响应包含 `Retry-After`，实际冷却时间以该响应头为准。

## 缓存与降级

优先级：

1. 新鲜内存缓存。
2. `realtime` 模式下，短期 stale 内存缓存立即返回并后台刷新。
3. 受控请求访问目标数据源。
4. 上游失败时使用 stale memory cache。
5. 市场数据使用 PostgreSQL 缓存。
6. 排行和搜索回退到本地股票底库。

响应中的 `data_source` 可能为 `tencent`、`eastmoney`、`cache` 或 `local`。生产监控应区分这些来源，不应把缓存数据误认为实时行情。

## 403/429 处置

当 `/api/v1/market/health` 显示某个源 degraded 或 unhealthy：

1. 暂停手动同步和批量刷新任务。
2. 检查日志中的 `status`、`Retry-After`、`source` 和 `fallback_reason`。
3. 确认目标站 robots/TOS 是否变更。
4. 如业务需要更高稳定性，优先接入授权商业行情数据源。
5. 不通过代理池、验证码绕过或浏览器自动化规避限制。

## 验证命令

发布前运行：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\verify-commercial-readiness.ps1
```

关键单测：

```powershell
cd backend-go
go test ./internal/platform/httpclient ./internal/infrastructure/providers -run "Test(ResilientHTTPClient|StockQuoteClient|EastmoneyClient)" -count=1
```
