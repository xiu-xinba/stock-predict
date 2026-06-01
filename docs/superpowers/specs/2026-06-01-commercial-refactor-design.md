# Commercial Refactor Design

## Goal

将当前股票/基金行情系统整理为可维护、可审计、可部署的商业级 Go + Vue 项目。保留现有 Go 后端、Vue 3 前端、已通过质量门禁的业务能力和预测入口占位，同时重组后端领域边界、前端模块结构、API 契约、质量脚本和项目文档体系。

## Current Context

项目当前已经具备以下基础：

- 后端为 Go + Gin，包含基金、股票、市场、自选、统一搜索、详情页和预测占位接口。
- 前端为 Vue 3 + TypeScript + Pinia + Element Plus + ECharts。
- `docs/api/openapi.yaml` 已覆盖后端接口。
- `scripts/verify-commercial-readiness.ps1` 已统一执行 API 契约检查、Go 测试、Go vet、前端 lint、Vitest 和生产构建。
- 当前质量门禁已通过：API routes 19 个、OpenAPI routes 19 个、Frontend API routes 13 个；Go 测试通过；前端 21 个测试通过；生产构建通过。
- 工作区存在一批未提交的架构、安全、服务和测试改动，实施时允许替换其中不符合商业级目录与代码规范的部分。

## Scope

### In Scope

- 保留当前 Go + Vue 技术栈。
- 保留基金、股票、市场、自选、搜索、详情、预测占位等现有业务能力。
- 修复基金和股票数据在前端无法显示的问题，并将其作为 P0 验收项。
- 重组后端领域边界，使 HTTP、业务领域、基础设施和平台能力职责清晰。
- 重组前端模块结构，使业务 feature、共享组件、应用入口和 API 客户端边界清晰。
- 统一 API 契约，以 OpenAPI、后端路由和前端路由常量一致性检查作为质量门禁。
- 完善测试、质量脚本、编码规范、提交规范和项目文档。

### Out of Scope

- 不重新引入模型训练或在线推理。预测能力保持外部系统接入占位，当前接口继续返回功能禁用响应。
- 不更换技术栈，不引入微服务拆分。
- 不新增复杂权限系统、支付系统、用户系统或后台运营系统。
- 不以全量重写替代可验证的渐进重构。

## Recommended Approach

采用商业化收敛重组：

1. 保留可运行业务能力和已通过质量门禁的基础。
2. 按领域边界整理后端和前端目录。
3. 先锁定 API 契约和数据可见性，再做结构调整。
4. 每个阶段都通过自动化验证，避免大规模重构引入不可定位回归。

该方案比保守修补更符合商业项目规范，比全量重建风险更低。

## Backend Architecture

后端目标结构以职责边界组织：

```text
backend-go/
  cmd/api/                    # 进程入口
  internal/
    app/                      # 应用装配、依赖初始化、启动和关闭流程
    api/                      # HTTP 路由、handler、中间件、响应映射
    domain/
      fund/                   # 基金模型、服务、仓储接口、行情编排
      stock/                  # 股票模型、服务、仓储接口、行情、同步、排行
      market/                 # 市场指数、基金/股票榜单聚合
      search/                 # 统一搜索编排
      watchlist/              # 自选业务聚合
      prediction/             # 预测入口占位与 feature_disabled 响应
    infrastructure/
      persistence/            # JSON 持久化、内存仓储实现
      searchindex/            # SQLite FTS5 索引实现
      external/               # 东方财富、腾讯行情等外部客户端
      httpclient/             # HTTP 客户端工厂、URL 白名单和超时
      cache/                  # 缓存辅助
    platform/
      config/                 # 环境变量配置和生产校验
      errors/                 # 统一业务错误码和 HTTP 映射
      security/               # CORS、CSRF、管理员令牌、安全头
      telemetry/              # request id、日志、metrics
      response/               # 统一 JSON 响应
```

迁移时保持外部 API 路径不变。现有 `internal/service`、`internal/store`、`internal/dto` 可以分阶段迁移到新边界；若某些文件已经足够清晰，可保留并通过包名和文档明确职责。

### Backend Rules

- Handler 只负责 HTTP 边界：参数解析、认证、调用用例、响应映射。
- Domain service 不直接依赖 Gin，不直接读取环境变量。
- Infrastructure 实现接口，domain 只依赖接口。
- 外部数据源调用必须经过 URL 白名单、超时和错误分类。
- 业务错误必须使用统一错误码，禁止 handler 中散落硬编码错误字符串。
- 生产环境下管理员令牌和 CORS 配置必须有明确校验或警告。

## Frontend Architecture

前端目标结构以业务 feature 和共享能力组织：

```text
frontend/src/
  app/                         # main、router、pinia、应用级样式入口
  shared/
    api/                       # axios 客户端、路由常量、错误分类
    components/                # ErrorState、CollapsibleCard、通用详情布局
    composables/               # useECharts、useSearch、useTheme 等通用组合函数
    types/                     # API 通用类型
    utils/                     # 格式化、图表工具
  features/
    funds/
      api/                     # 基金 API
      components/              # 基金详情组件
      stores/                  # 基金详情和搜索状态
      types/                   # 基金类型
      views/                   # FundDetailView
    stocks/
      api/
      components/
      stores/
      types/
      views/
    market/
      api/
      components/
      stores/
      views/
    watchlist/
      components/
      stores/
      types/
      views/
    search/
      components/
      stores/
      types/
    prediction/
      components/
      views/
```

迁移时保持现有路由 URL 和用户可见页面不变。组件迁移后通过 barrel exports 或明确 import 路径避免循环依赖。

### Frontend Rules

- 页面组件负责组合 feature，不直接拼接底层 HTTP URL。
- Store 负责状态和动作，不负责格式化展示文案。
- API 层只返回已类型化数据，统一处理取消、重试、CSRF、认证和错误分类。
- 组件必须明确处理加载中、空数据、错误、正常数据四种状态。
- 基金和股票详情、市场榜单、搜索结果、自选刷新必须有回归测试或可重复验证步骤。

## Data Visibility P0

基金和股票数据当前在前端显示不出来，实施时作为最高优先级问题处理。

### Investigation Flow

按以下链路逐层复现并定位根因：

```text
后端接口响应
  -> 前端 API route 常量
  -> Axios baseURL / proxy / credentials / CSRF
  -> Pinia store action
  -> 组件 props / computed
  -> 页面渲染状态
```

### Acceptance Cases

以下场景必须能显示真实或后端返回的可用数据：

- 市场页显示市场指数、基金排行、股票排行。
- 基金详情页显示基金基础信息、行情、业绩、经理、持仓、风险和预测占位。
- 股票详情页显示股票基础信息、行情、K 线、资金流、财务、股东和预测占位。
- 搜索浮层能展示基金和股票搜索结果。
- 自选页能刷新基金和股票行情，并正确展示空状态和错误状态。

### Fix Discipline

- 先写或补充失败测试/复现脚本，证明当前不可见问题存在。
- 定位根因后只修根因，不用组件硬编码假数据掩盖问题。
- 修复完成后运行前后端测试、构建和商业门禁脚本。

## API Contract

OpenAPI 是接口真源：

- 后端路由、前端 API route 常量和 `docs/api/openapi.yaml` 必须一致。
- API 响应统一使用 `{ code, message, data }` 格式。
- 错误响应必须可被前端稳定分类。
- 同步接口继续使用管理员令牌保护。
- 预测占位接口继续返回 `501 feature_disabled`，并在 OpenAPI 中明确说明。

## Error Handling And Security

后端：

- 使用统一业务错误码和 HTTP 状态映射。
- 中间件顺序保持清晰：recover、request id、logger、安全头、CORS、CSRF、gzip、body limit、rate limit、handler。
- CSRF 使用 cookie + header 双令牌。
- 管理员令牌比较使用时间安全比较。
- 生产环境启用 HSTS、CSP、X-Frame-Options、Referrer-Policy 等安全头。

前端：

- API 错误统一转为业务错误、网络错误、取消请求、认证错误、服务器错误。
- 页面不直接展示低层异常栈。
- 用户界面必须在请求失败时展示可理解的错误状态。

## Testing Strategy

后端：

- 保留现有 Go 单元测试。
- 补充领域服务、仓储、搜索、错误映射、同步输入校验测试。
- 修复数据不可见问题时优先添加 handler 或 service 层回归测试。

前端：

- 保留 Vitest + Vue Test Utils。
- 补充 API route、store action、关键页面数据渲染测试。
- 对基金、股票、市场、自选、搜索的核心渲染路径建立最小回归覆盖。

质量门禁：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts\verify-commercial-readiness.ps1
```

该脚本必须继续执行：

- API contract check
- `go test ./...`
- `go vet ./...`
- `npm run lint`
- `npm run test:run`
- `npm run build`

## Documentation Deliverables

本轮完成后文档中心应包含：

- `README.md`：项目定位、快速启动、质量门禁、主要功能。
- `docs/architecture.md`：最新架构、模块边界、数据流、安全设计、部署架构。
- `docs/api/openapi.yaml`：完整接口契约。
- `docs/operations/deployment.md`：环境变量、启动方式、生产部署建议。
- `docs/operations/maintenance.md`：数据同步、监控、故障排查、常见问题。
- `docs/report/`：审计和重构报告。
- `docs/superpowers/specs/2026-06-01-commercial-refactor-design.md`：本设计规格。
- `docs/superpowers/plans/`：后续实施计划。

## Coding And Version Control Standards

- Go 使用 `gofmt`、`go test`、`go vet`。
- TypeScript/Vue 使用 ESLint、Prettier、Vitest 和生产构建校验。
- 命名以领域语言为核心：fund、stock、market、watchlist、search、prediction。
- 提交采用 Conventional Commits，例如 `refactor: reorganize backend domain boundaries`。
- 大重构分阶段提交：契约、后端、前端、文档、验证。
- 不提交本地运行产物、构建目录、日志、临时数据库或 IDE 私有状态。

## Implementation Order

1. 建立回归基线：记录当前门禁通过状态和数据不可见复现路径。
2. 修复数据不可见 P0：定位根因，补测试，修复并验证。
3. 收敛 API 契约：确保 OpenAPI、后端路由和前端 route 常量一致。
4. 重组后端边界：按低风险顺序迁移配置、错误、平台能力、领域服务和基础设施。
5. 重组前端边界：先迁移 shared，再迁移 features，保持路由和视觉行为不变。
6. 更新文档和脚本：README、架构、API、部署、维护、提交规范。
7. 运行完整质量门禁并整理最终审计报告。

## Success Criteria

- 当前业务功能保持可用。
- 基金和股票数据在前端关键页面正常显示。
- 后端和前端目录结构体现清晰领域边界。
- API 契约、后端路由和前端 route 常量一致。
- 质量门禁脚本完整通过。
- 文档足以支持新工程师启动、维护、部署和排障。
- 未引入预测模型训练或推理职责回流。
