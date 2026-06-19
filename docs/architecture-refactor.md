# 商业架构强收敛设计

> 状态：已于 2026-06-12 完成实施并通过商业就绪门禁。

## 目标

将当前 Go + Vue 模块化单体重组为按业务领域垂直切分的商业项目结构，同时清理历史文档、构建产物、二进制、日志、缓存、IDE 配置和迁移完成后的旧源码目录。

除明确移除前端 `/predict/:fundCode` 流程外，本次重构不改变公开 API 路径、响应结构、数据库 schema 或用户功能。旧预测 API 路径保留为显式的 `410 Gone` 兼容契约。

## 核心原则

1. 保持行为不变，先移动和重命名，再拆分超大文件。
2. 不保留旧目录的兼容转发文件。
3. 每个阶段必须保持测试和构建通过。
4. 领域代码不依赖 HTTP、数据库实现或外部 provider。
5. 前端 feature 只能依赖自身模块、shared 和其他 feature 的公开入口。
6. 依赖目录不参与迁移和删除。

## 实施阶段

1. 清理生成物、历史文档并建立单一文档中心。
2. 迁移后端配置、HTTP、数据库和公共平台层。
3. 按 fund、stock、market、search 拆分后端领域服务。
4. 拆分超大服务文件到 provider 和市场子模块。
5. 迁移前端 shared 层。
6. 按 feature 迁移页面、组件、Store、API 和类型。
7. 删除旧目录，更新 Trae 规则、脚本、CI 和文档。
8. 执行完整商业门禁和浏览器验收。

## 后端目标结构

```text
backend-go/internal/
├── app/
├── transport/http/
│   ├── router/
│   ├── response/
│   ├── middleware/
│   └── handler/
├── domain/
│   ├── fund/
│   ├── stock/
│   ├── market/
│   └── search/
├── infrastructure/
│   ├── database/
│   └── providers/
│       ├── eastmoney/
│       ├── tencent/
│       ├── sina/
│       ├── tdx/
│       ├── ths/
│       ├── akshare/
│       └── biying/
└── platform/
    ├── config/
    ├── errors/
    ├── cache/
    ├── httpclient/
    └── observability/
```

### 后端边界

- `domain` 只依赖标准库、领域内部类型和仓储接口。
- `infrastructure` 实现领域接口，可依赖 GORM、HTTP 客户端和外部 SDK。
- `transport/http` 负责参数解析、认证、错误到响应的映射。
- `app` 是唯一同时引用 transport、domain 和 infrastructure 的装配层。
- provider 共享协议、能力路由和健康监控位于 `infrastructure/providers` 根部。
- HTTP 请求/响应类型归 transport；业务实体归对应 domain。
- 不保留全局 `dto` 或泛化 `util` 包。
- `index_quote.go` 按指数报价、分钟线、K 线和上游抓取适配拆分，单文件控制在约 400 行以内。
- 已迁移预测服务只保留 HTTP 410 兼容端点，不创建空 prediction domain。

## 前端目标结构

```text
frontend/src/
├── app/
│   ├── bootstrap.ts
│   ├── router.ts
│   ├── App.vue
│   ├── styles/
│   └── __tests__/
├── shared/
│   ├── api/
│   ├── components/
│   ├── composables/
│   ├── charts/
│   └── utils/
└── features/
    ├── funds/
    ├── stocks/
    ├── market/
    ├── watchlist/
    ├── search/
    ├── prediction/
    └── settings/
```

### 前端边界

- feature 内部保持 `api -> store -> components/view` 单向依赖。
- feature 可以依赖 shared，shared 禁止依赖 feature。
- feature 之间不直接读取对方 Store；跨 feature 协作通过公开入口或 app 组合。
- `App.vue`、路由、应用启动和全局样式归 app。
- 跨基金和股票复用的布局、错误、Skeleton、资产头部归 shared/components。
- ECharts 加载器、公共图表配置和主题辅助归 shared/charts。
- `RefreshFab` 归 app；搜索浮层归 search；市场 Dock 和健康状态归 market。
- 预测迁移提示归 prediction。
- 每个 feature 使用 `index.ts` 暴露受控公共入口。
- 测试优先与 feature 邻近，应用级集成测试位于 `app/__tests__`。

## 删除范围

- `.vscode/`
- `.superpowers/`
- `docs/superpowers/`
- 所有 `.run-logs/`
- `frontend/dist/`
- `backend-go/bin/`
- `backend-go/tmp/`
- `backend-go/*.exe`
- Python `__pycache__/`、`.pytest_cache/` 和 `.pyc`
- `backend-go/README.md`
- `frontend/README.md`
- 过期 `DESIGN.md`
- 迁移完成后的旧后端与前端源码目录
- 空目录和无引用代码

## 保留范围

- `.trae/`，并更新为 PostgreSQL 和新架构说明
- `frontend/node_modules/`
- `backend-go/akshare-service/.venv/`
- 根目录 `README.md`
- `docs/architecture.md`
- `docs/api/openapi.yaml`
- `docs/operations/`
- `.github/`
- `scripts/`
- Docker 配置

## 文档中心

根目录 `README.md` 是唯一项目入口；`architecture.md` 描述最终运行架构，本文件作为架构决策记录保留。API、部署、维护与数据源合规文档分别位于 `docs/api/` 和 `docs/operations/`。

## 验收标准

1. 不存在旧目录兼容文件或重复 README。
2. 搜索不到旧 import 路径。
3. OpenAPI 与后端路由完全一致。
4. Go 格式、测试、vet、build 和漏洞检查通过。
5. 前端 Prettier、lint、测试、build 和 audit 通过。
6. Python `pip check`、compileall 和单元测试通过。
7. 浏览器验证市场、自选、基金详情、股票详情、搜索、设置和预测迁移提示。
8. Git diff 不包含依赖目录改动，也不删除 `.trae`。
