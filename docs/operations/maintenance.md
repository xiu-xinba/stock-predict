# 维护手册

本文档面向日常维护、数据同步、故障处理和质量门禁巡检。

## 1. 日常巡检

每个工作日建议执行：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\verify-commercial-readiness.ps1
```

需要分模块定位时可执行：

```powershell
cd backend-go
go test ./...

cd ../frontend
npm run lint
npm run test:run
```

生产环境巡检还应检查：

- `/api/v1/health` 是否返回 `status=ok`。
- `/api/v1/metrics` 是否返回请求计数、错误计数、状态码分布和 uptime。
- 搜索接口是否能返回基金和股票结果。
- 行情接口是否有合理更新时间。
- 预测接口是否返回明确的独立项目占位状态。
- 后端日志是否出现外部数据源持续失败。

## Quality Gate

Run before deployment:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\verify-commercial-readiness.ps1
```

Deployment is blocked if API contract, Go tests, Go vet, frontend lint, frontend tests, or frontend build fail.

## 2. 数据同步

### 基金同步

```powershell
Invoke-RestMethod -Method Post `
  -Uri "http://localhost:5070/api/v1/funds/sync" `
  -Headers @{ Authorization = "Bearer <ADMIN_TOKEN>" }
```

同步后检查：

- `/api/v1/funds/coverage` 覆盖率是否正常。
- `/api/v1/search?q=510300` 是否返回目标基金。
- 自选基金报价是否仍能刷新。

### 股票同步

```powershell
Invoke-RestMethod -Method Post `
  -Uri "http://localhost:5070/api/v1/stocks/sync" `
  -Headers @{ Authorization = "Bearer <ADMIN_TOKEN>" }
```

同步后检查：

- `/api/v1/stocks/search?keyword=600519` 是否返回目标股票。
- `/api/v1/market/stock-ranking/gainers` 是否返回排行数据。
- 股票详情页是否能加载行情和基础信息。

## 3. 故障分级

| 等级 | 示例 | 处理目标 |
| --- | --- | --- |
| P0 | 前端无法构建、API 无法启动、核心行情不可用 | 立即回滚或修复 |
| P1 | 搜索、行情、自选、预测占位入口部分失败 | 当天修复 |
| P2 | 单个数据源失败但有降级、部分页面展示不完整 | 规划修复 |
| P3 | 文档、样式、性能提示和开发体验问题 | 排入迭代 |

## 4. 常见问题处理

### 前端构建失败

1. 运行 `npm run build`，读取首个 TypeScript 错误。
2. 如果是无用导入或类型不匹配，先做最小修复。
3. 运行 `npm run lint` 和 `npm run test:run`，确认没有引入新问题。

### Go API 启动失败

1. 检查环境变量是否有效，尤其是端口、数据路径和 CORS。
2. 运行 `go test ./...` 和 `go vet ./...`。
3. 检查数据文件是否存在且可读写。
4. 检查外部数据源不可用时是否有兜底数据。

### 前端基金或股票数据不可见

1. 直接调用后端接口，确认接口返回纯 JSON：

   ```powershell
   curl.exe -s --compressed "http://localhost:5070/api/v1/stocks/search?size=1"
   ```

2. 输出中不得出现 gzip 尾部乱码。如果出现普通 JSON 后附加乱码，优先检查 `backend-go/internal/api/middleware.go` 的 gzip writer 生命周期。
3. 运行前端数据可见性测试：

   ```powershell
   cd frontend
   npm run test:run -- src/__tests__/data-visibility.test.ts
   ```

4. 如果接口正常但页面仍无数据，沿 `API route -> Axios -> Pinia store -> Vue view` 逐层定位。

### 预测入口异常

1. 调用 `/api/v1/predict/{fundCode}` 或 `/api/v1/stock/{stockCode}/predict`。
2. 有效 6 位代码应返回 HTTP 501 和“预测模型已拆分为独立项目”的提示。
3. 前端预测页和详情页预测卡片应显示同一占位语义。

## 5. 日志和审计

后端日志应至少保留：

- 请求 ID、方法、路径、状态码、耗时。
- 外部数据源请求失败原因。
- 管理接口调用时间和结果。

日志中不得输出管理员令牌、Cookie、完整用户认证信息或敏感请求头。

## 6. 契约校验

接口变更后必须运行：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\verify-api-contract.ps1
```

该脚本会对比：

- Go 后端 `backend-go/internal/api/router.go` 中注册的 API 路由。
- `docs/api/openapi.yaml` 中声明的 OpenAPI 路径。
- 前端 `frontend/src/shared/api/routes.ts` 中使用的 API 路由。

如果任一层缺失对应路径，脚本会失败并输出缺失路由。

## 7. 维护原则

- 修复故障前先复现和定位根因。
- 每次修复至少运行相关模块测试。
- 涉及接口字段时同步更新 OpenAPI、后端 DTO、前端类型和文档。
- 涉及预测入口时同步检查后端 501 响应、OpenAPI 和前端占位文案。
