# 项目全面修复与架构收敛设计

## 状态

已于 2026-06-13 确认实施。

## 目标

逐项解决全面审查发现的发布阻断、并发安全、数据正确性、前端异步状态、API 契约和架构边界问题，使当前工作区能够在干净检出环境中重复构建、测试和部署。

## 范围与原则

1. 保持现有公开行情 API 路径和成功响应结构不变。
2. 后端旧预测 API 继续作为显式废弃契约返回 `410 Gone`。
3. 前端旧地址 `/predict/:fundCode` 重定向到统一迁移提示页 `/predict`。
4. 引入 `internal/application` 用例层，使 HTTP transport 不再直接编排数据库和 provider 实现。
5. 数据库迁移账号与 API 运行账号分离，API 使用最小权限账号。
6. 删除未引用的 `backend-go/data/funds.json`、构建二进制、缓存和生成物，并增加防回归检查。
7. 保留 `frontend/node_modules`、`backend-go/akshare-service/.venv` 和 `.trae`。
8. 所有行为修复先写失败测试，再实现最小修复。

## 阶段一：部署安全与契约

### Docker 构建边界

- 新增 `backend-go/.dockerignore`，排除 `.env`、`.venv`、`data/`、二进制、日志、缓存、测试产物和 Git 元数据。
- Dockerfile 使用明确复制范围，不把整个后端目录无条件复制进构建上下文。
- 运行镜像提供受信任 CA；移除对系统 `curl --insecure` 的依赖，所有公网 HTTPS 请求通过启用证书校验的 Go HTTP 客户端完成。
- CI 增加 Docker 镜像构建检查；在 Docker daemon 可用的环境执行容器启动 smoke test。

### 数据库最小权限

- Compose 使用独立的数据库 owner/migration 账号和 API runtime 账号。
- 初始化脚本创建 runtime 角色并授予所需 schema、表、序列权限，不授予建库、建角色或任意 DDL 权限。
- `stock-migrate` 使用迁移 DSN，API 使用 runtime DSN。
- README、`.env.example` 和部署文档同步新的变量契约与发布顺序。

### OpenAPI

- 修复所有无法解析的 `$ref` 和 OpenAPI 3.0 非法字段。
- 契约门禁增加标准 OpenAPI lint，不再只比较方法和路径。
- 继续验证 Gin 路由与 OpenAPI 路径集合，同时校验预测端点的 `deprecated: true` 和 `410` 响应。

## 阶段二：后端正确性

### Provider 竞速

- 每个竞速 goroutine 返回独立载荷和错误，不允许闭包并发写调用方共享变量。
- 第一个成功结果获胜；胜出后触发的内部取消不计入 Provider 失败统计。
- 增加多 Provider 同时成功、慢 Provider 被取消、全部失败和超时测试。

### 缓存与同步状态

- LRU 查询、读取和移动节点在同一个互斥锁临界区完成。
- 市场同步状态先在局部变量构造，再在一次加锁中原子更新。
- CI 增加 `go test -race ./...`，在支持 CGO 的 Linux runner 执行。

### 数据持久化

- 基金可选数值字段使用显式存在语义，合法 `0` 可以覆盖旧值。
- 股票拼音主字段和备用字段独立持久化，重复同步保持幂等。
- 数据库错误与“记录不存在”分开传播：只有 `gorm.ErrRecordNotFound` 映射为 404，其余错误映射为 500 或 503。
- Schema 启动检查验证最新迁移版本以及运行必需的表、列、索引和 `pg_trgm` 扩展。

## 阶段三：前端状态与兼容

### CSRF

- API 客户端在首次 mutation 前执行单例化 CSRF bootstrap GET。
- 并发 mutation 共享同一个 bootstrap Promise。
- 管理接口文档展示完整 Cookie 与 Header 流程；纯 Bearer Token 不被错误描述为足够条件。

### 搜索与行情状态

- 搜索 `reset()` 使当前请求序号失效，并取消仍在执行的请求。
- 搜索结果容器在查询完成且结果为空时展示可达的空状态。
- K 线与分钟线请求仅允许当前 AbortController 清理对应 loading 状态。
- 非法基金代码清空旧详情并展示明确错误，不保留上一个基金的内容。

### 预测兼容

- `/predict/:fundCode` 只做前端重定向，不恢复已移除的预测流程。
- `/predict` 和基金、股票详情占位继续使用一致的迁移说明。

## 阶段四：Application 用例层

### 目标依赖方向

```text
transport/http -> application -> domain
                           |
                           v
              domain ports/interfaces
                           ^
                           |
        infrastructure database/providers
```

### 迁移内容

- 新增 `internal/application/{fund,stock,market,search,watchlist}`。
- 将搜索、基金查询、行情编排、自选刷新、股票同步等业务用例从 `infrastructure/providers` 移入 application。
- Provider 包只保留外部数据源适配、路由、健康监控和底层客户端。
- Database 包只实现仓储和查询端口。
- HTTP handler 只依赖 application 暴露的接口，不直接依赖 GORM Store 或具体 Provider Service。
- `internal/app` 作为唯一装配层连接 application、transport 与 infrastructure。
- 删除迁移完成后的 alias facade 和无效 CacheProvider 装配。

### 架构门禁

- Go 增加包依赖测试，阻止 domain 引用 transport/infrastructure，阻止 transport 直接引用 database/providers。
- 前端架构检查解析别名和相对导入，阻止 shared 反向依赖、跨 feature 深层导入和跨 feature Store 访问。

## 阶段五：清理与发布验证

- 删除未引用的 `backend-go/data/funds.json`。
- 删除 `frontend/dist`、`backend-go/*.exe`、Python 缓存和运行日志。
- 清理脚本在 `finally` 中删除自身生成的构建产物，或把产物写入临时目录。
- 固定漏洞扫描工具版本，避免 `@latest` 造成不可重复结果。
- 增加清洁工作区检查，禁止提交 `.env`、构建产物、日志、缓存和历史数据 JSON。
- 添加 `.gitattributes` 固定文本文件使用 LF，避免 Windows 自动换行导致大面积 diff。

## 验收标准

1. Go 格式、单元与集成测试、race、vet、build 和漏洞扫描通过。
2. 前端 Prettier、lint、测试、类型检查、生产构建和依赖审计通过。
3. Python `pip check`、单元测试和依赖审计通过。
4. OpenAPI 标准 lint 为零错误，路由契约检查通过。
5. Docker 镜像可以构建，镜像中不包含项目 `.env`、本地虚拟环境或数据文件。
6. 使用迁移账号可执行迁移；使用 runtime 账号可正常运行 API 但无法执行 DDL。
7. Provider 竞速、LRU、市场同步状态在 race detector 下无数据竞争。
8. 合法零值更新、拼音重复同步和数据库故障映射均有回归测试。
9. 首次 mutation、搜索关闭、零结果、连续行情请求和非法基金路由均有前端回归测试。
10. `/predict/:fundCode` 重定向到 `/predict`，后端旧预测端点继续返回 `410 Gone`。
11. transport、application、domain 和 infrastructure 的依赖方向通过自动化架构测试。
12. 最终工作区不包含无引用历史数据、构建二进制、日志或缓存；保留指定依赖目录和 `.trae`。

## 实施顺序

阶段按“一至五”顺序执行。每个阶段必须先通过相关局部测试和独立代码复核，最后再执行全量发布验证。架构迁移不得改变公开 API；如果迁移暴露既有行为缺陷，先以回归测试固定正确契约，再修复实现。
