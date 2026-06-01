# 系统架构设计文档

## 1. 系统概述

### 项目定位

基金/股票行情系统是一个面向中国 A 股市场的金融数据平台，提供基金与股票的实时行情、搜索、自选管理和预测入口占位。系统采用前后端分离架构，后端使用 Go 语言构建高性能 API 服务，前端使用 Vue 3 构建响应式单页应用。

### 核心功能

| 功能 | 描述 |
|------|------|
| 统一搜索 | 支持基金代码、名称、拼音缩写/全拼、股票代码等多维度混合搜索 |
| 自选管理 | 基金/股票自选列表，支持实时行情刷新、排序、涨跌统计 |
| 行情展示 | 大盘指数、基金/股票排行榜（涨幅/跌幅/成交量） |
| 预测入口 | 当前主项目保留页面和 API 占位，模型训练与推理由独立项目后续接入 |
| 详情页面 | 基金详情（净值/经理/持仓/风险）、股票详情（K线/资金流/财务/股东） |
| 数据同步 | 从东方财富等数据源自动同步基金/股票基础数据 |

### 技术栈

| 层次 | 技术 |
|------|------|
| 前端框架 | Vue 3 + TypeScript + Vite |
| 状态管理 | Pinia |
| UI/图表 | Element Plus + ECharts |
| HTTP 客户端 | Axios |
| 后端框架 | Go + Gin |
| 数据存储 | 内存 + JSON 文件持久化 + SQLite FTS5 |
| 外部数据源 | 东方财富 API、腾讯行情 API |

---

## 2. 系统架构图

```
┌─────────────────────────────────────────────────────────────────────┐
│                          前端 (Vue 3 SPA)                          │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌─────────┐ │
│  │ Watchlist │ │  Market  │ │ Predict  │ │FundDetail│ │StockDtl │ │
│  │   View   │ │   View   │ │   View   │ │   View   │ │  View   │ │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬────┘ │
│       └─────────────┴─────────────┴─────────────┴────────────┘     │
│                         Pinia Stores                                │
│       ┌─────────────┬──────────────┬──────────────┬──────────┐     │
│       │  Watchlist  │    Search    │  Prediction  │  Detail  │     │
│       │   Store     │    Store    │    Store     │  Stores  │     │
│       └──────┬──────┴──────┬──────┴──────┬───────┴────┬─────┘     │
│              └─────────────┴─────────────┴────────────┘            │
│                        API 集成层 (Axios)                           │
│              请求去重 · CSRF 令牌 · 自动重试 · 错误分类              │
└────────────────────────────┬────────────────────────────────────────┘
                             │ HTTP /api/v1
┌────────────────────────────┴────────────────────────────────────────┐
│                     后端 API (Go + Gin)                              │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                    中间件链                                    │   │
│  │  Recover → RequestID → Logger → SecurityHeaders → CORS →     │   │
│  │  CSRF → Gzip → MaxBody → RateLimiter                         │   │
│  └──────────────────────────┬───────────────────────────────────┘   │
│                             │                                       │
│  ┌──────────────────────────┴───────────────────────────────────┐   │
│  │                    Handler 层 (Router)                         │   │
│  │  fund_handler · market_handler · search_handler ·             │   │
│  │  stock_handler                                                │   │
│  └──────────────────────────┬───────────────────────────────────┘   │
│                             │                                       │
│  ┌──────────────────────────┴───────────────────────────────────┐   │
│  │                 Service 层 (Registry)                          │   │
│  │  FundService · MarketService · WatchlistService ·             │   │
│  │  FundDetailService · StockService · StockDetailService ·      │   │
│  │  SearchService · FundQuoteClient · StockQuoteClient           │   │
│  └──────────────────────────┬───────────────────────────────────┘   │
│                             │                                       │
│  ┌──────────────────────────┴───────────────────────────────────┐   │
│  │                   Store 层                                     │   │
│  │  MemoryStore (基金) · SearchIndex (SQLite FTS5) ·             │   │
│  │  JSON 持久化                                                   │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                 外部数据源                                      │   │
│  │  东方财富基金JS · 东方财富排行API · 东方财富股票列表API ·       │   │
│  │  腾讯行情API                                                    │   │
│  └──────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 3. 后端架构

### 3.1 分层设计

后端采用经典的四层架构，职责清晰，依赖方向自上而下：

```
cmd/api/main.go          → 应用入口，组装依赖，启动服务
    │
    ▼
internal/api/            → HTTP 层（路由 + Handler + 中间件）
    │                       职责：参数绑定、输入校验、响应序列化
    ▼
internal/service/        → 业务逻辑层（Registry 模式组装）
    │                       职责：搜索、行情、自选、同步等核心业务
    ▼
internal/store/          → 数据存储层
                            职责：内存存储、JSON 持久化、SQLite FTS5 索引
```

**各层文件职责：**

| 层次 | 文件 | 职责 |
|------|------|------|
| cmd/api | main.go | 加载配置、初始化 Store/Service/Router、优雅关闭 |
| api | router.go | 路由注册、中间件链组装 |
| api | fund_handler.go | 基金搜索/同步/详情 Handler |
| api | market_handler.go | 大盘指数/排行榜 Handler |
| api | search_handler.go | 统一搜索 Handler |
| api | stock_handler.go | 股票搜索/详情/排行 Handler |
| api | middleware.go | CORS/CSRF/限流/安全头/Gzip/日志/恢复 |
| api | response.go | 统一 JSON 响应格式 |
| service | registry.go | 服务注册中心，组装所有 Service |
| service | fund_service.go | 基金搜索/排行/过滤/同步 |
| service | watchlist_service.go | 自选基金报价聚合 |
| service | search_service.go | 统一搜索（线性扫描 + FTS5 混合） |
| service | stock_service.go | 股票搜索/排行/同步 |
| service | stock_sync.go | 东方财富 API 同步 + 拼音生成 |
| service | fund_detail_service.go | 基金详情（净值/经理/持仓/风险） |
| service | stock_detail_service.go | 股票详情（K线/资金流/财务/股东） |
| service | fund_quote.go | 基金实时行情（东方财富/腾讯） |
| service | stock_quote.go | 股票实时行情 |
| service | url_validator.go | URL 白名单验证 |
| service | errors.go | 统一业务错误码体系 |
| service | constants.go | 共享常量和 HTTP 客户端工厂 |
| store | interfaces.go | FundRepository、StockRepository 接口定义 |
| store | memory.go | 内存存储实现（MemoryStore，同时实现 FundRepository 和 StockRepository） |
| store | persistence.go | JSON 文件持久化 |
| store | persistence_sync.go | 东方财富数据同步解析 |
| store | search_index.go | SQLite FTS5 搜索索引 |

### 3.2 依赖注入（Registry 模式）

系统采用 Registry 模式实现依赖注入，所有 Service 在 `NewRegistry` 中一次性组装：

```go
type Registry struct {
    Funds       *FundService
    Market      *MarketService
    Watchlist   *WatchlistService
    Detail      *FundDetailService
    Stocks      *StockService
    StockDetail *StockDetailService
    StockQuote  *StockQuoteClient
    Search      *SearchService
}
```

**依赖关系：**

```
Registry
 ├── FundService          ← FundRepository
 ├── MarketService        ← (独立，无外部依赖)
 ├── FundQuoteClient      ← (HTTP 客户端)
 ├── StockQuoteClient     ← (HTTP 客户端)
 ├── StockService         ← StockRepository, Logger
 ├── WatchlistService     ← FundRepository, Config, Logger
 ├── FundDetailService    ← FundRepository, FundQuoteClient, Logger
 ├── StockDetailService   ← StockRepository, StockQuoteClient, Logger
 └── SearchService        ← FundRepository, StockRepository, SearchIndex
```

**统一业务错误码体系（service/errors.go）：**

| 错误变量 | 业务码 | HTTP状态 | 说明 |
|---------|--------|---------|------|
| ErrInvalidFundCode | 10001 | 400 | 无效基金代码 |
| ErrFundNotFound | 10002 | 404 | 基金不存在 |
| ErrInvalidStockCode | 10003 | 400 | 无效股票代码 |
| ErrStockNotFound | 10004 | 404 | 股票不存在 |
| ErrInvalidRankingType | 10005 | 400 | 无效排行类型 |
| ErrSyncSourceRequired | 10006 | 400 | 同步源未指定 |
| ErrSyncUnsupported | 10007 | 500 | 同步源不支持 |

**初始化流程（main.go）：**

1. 加载配置 `config.Load()`
2. 初始化基金存储 `store.NewPersistentStore()`
3. 可选：自动同步基金数据
4. 初始化搜索索引 `store.NewSearchIndex()`
5. 创建服务注册中心 `service.NewRegistry()`
6. 可选：自动同步股票数据
7. 同步基金/股票到搜索索引
8. 创建路由 `api.NewRouter()`
9. 启动 HTTP 服务器
10. 监听信号，优雅关闭

### 3.3 中间件链

中间件按以下顺序注册，每个请求依次经过：

```
请求 → Recoverer → RequestID → RequestLogger → SecurityHeaders
     → CORS → CSRF → Gzip → MaxBody → RateLimiter → Handler
```

| 中间件 | 功能 | 关键参数 |
|--------|------|----------|
| Recoverer | Panic 恢复，返回 500 | 记录堆栈到日志 |
| RequestID | 生成唯一请求 ID | 8 字节随机 hex，写入 `X-Request-ID` 头 |
| RequestLogger | 请求日志 | 记录方法/路径/状态码/耗时/请求ID |
| SecurityHeaders | 安全响应头 | `X-Content-Type-Options: nosniff`、`X-Frame-Options: DENY`、`Referrer-Policy: no-referrer`、`Cache-Control: no-store`、`Permissions-Policy`、`X-Permitted-Cross-Domain-Policies: none`、`Strict-Transport-Security`（非开发环境）、`Content-Security-Policy`（非开发环境）、`X-XSS-Protection: 0` |
| CORS | 跨域控制 | 基于 `CORS_ORIGINS` 配置，开发模式默认允许 `localhost:5173` |
| CSRF | 跨站请求伪造防护 | Cookie + Header 双令牌验证，24h 过期，5 分钟清理 |
| Gzip | 响应压缩 | 检查 `Accept-Encoding`，仅对 JSON/文本内容压缩 |
| MaxBody | 请求体大小限制 | 最大 1MB |
| RateLimiter | IP 限流 | 每分钟 60 次，1 分钟清理过期条目 |

### 3.4 数据流

典型请求处理流程：

```
HTTP 请求
    │
    ▼
中间件链（安全/日志/限流）
    │
    ▼
Handler（参数绑定 + 输入校验）
    │
    ▼
Service（业务逻辑处理）
    │
    ├──→ Store（内存读取/写入）
    ├──→ SearchIndex（FTS5 查询）
    ├──→ 外部 API（行情数据源）
    │
    ▼
Handler（响应序列化）
    │
    ▼
HTTP 响应（JSON + Gzip）
```

**API 路由表：**

| 方法 | 路径 | Handler | 说明 |
|------|------|---------|------|
| GET | /api/v1/health | health | 健康检查 |
| GET | /api/v1/search | unifiedSearch | 统一搜索（基金+股票） |
| GET | /api/v1/funds/search | searchFunds | 基金搜索 |
| GET | /api/v1/funds/filters | fundFilters | 基金筛选项 |
| GET | /api/v1/funds/coverage | fundCoverage | 基金覆盖率报告 |
| POST | /api/v1/funds/sync | syncFunds | 基金数据同步（需管理员令牌） |
| GET | /api/v1/market/indices | marketIndices | 大盘指数 |
| GET | /api/v1/market/ranking/:type | marketRanking | 基金排行榜 |
| GET | /api/v1/predict/:fundCode | predict | 基金预测入口占位 |
| POST | /api/v1/watchlist/quotes | watchlistQuotes | 自选基金行情 |
| GET | /api/v1/fund/:fundCode/detail | fundDetail | 基金详情 |
| GET | /api/v1/stocks/search | searchStocks | 股票搜索 |
| GET | /api/v1/stocks/filters | stockFilters | 股票筛选项 |
| GET | /api/v1/stock/:stockCode/detail | stockDetail | 股票详情 |
| GET | /api/v1/stock/:stockCode/predict | predictStock | 股票预测入口占位 |
| POST | /api/v1/stocks/quotes | stockQuotes | 股票批量行情 |
| GET | /api/v1/market/stock-ranking/:type | stockRanking | 股票排行榜 |
| POST | /api/v1/stocks/sync | syncStocks | 股票数据同步（需管理员令牌） |

---

## 4. 前端架构

### 4.1 组件层级图

```
App.vue
 ├── RouterView
 │    ├── WatchlistView          ← 自选列表页
 │    │    ├── SearchOverlay     ← 搜索浮层
 │    │    ├── WatchlistEmpty    ← 空状态
 │    │    └── RefreshFab        ← 刷新浮动按钮
 │    │
 │    ├── MarketView             ← 行情页
 │    │    ├── MarketDock        ← 大盘指数
 │    │    ├── FundRanking       ← 基金排行
 │    │    └── StockRanking      ← 股票排行
 │    │
 │    ├── PredictView            ← 预测入口占位页
 │    │
 │    ├── FundDetailView         ← 基金详情页
 │    │    ├── DetailPageLayout  ← 通用详情布局
 │    │    ├── AssetHeader       ← 资产头部
 │    │    ├── FundHeader        ← 基金头部信息
 │    │    ├── FundPerformance   ← 业绩表现
 │    │    ├── FundManager       ← 基金经理
 │    │    ├── FundPortfolio     ← 持仓明细
 │    │    ├── FundRisk          ← 风险指标
 │    │    └── FundPrediction    ← 基金预测入口占位
 │    │
 │    ├── StockDetailView        ← 股票详情页
 │    │    ├── DetailPageLayout
 │    │    ├── AssetHeader
 │    │    ├── StockHeader       ← 股票头部信息
 │    │    ├── StockQuote        ← 实时行情
 │    │    ├── StockKline        ← K线图
 │    │    ├── StockCapitalFlow  ← 资金流向
 │    │    ├── StockFinancials   ← 财务数据
 │    │    ├── StockShareholders ← 股东结构
 │    │    └── StockPrediction   ← 股票预测入口占位
 │    │
 │    └── NotFoundView           ← 404 页面
 │
 └── 通用组件
      ├── CollapsibleCard        ← 可折叠卡片
      └── ErrorState             ← 错误状态
```

### 4.2 状态管理（Pinia Stores）

```
Pinia
 ├── watchlist/                  ← 自选模块
 │    ├── index.ts               ← 聚合 Store（统一导出）
 │    │    └── useWatchlistStore   聚合基金+股票自选
 │    ├── fundWatchlist.ts       ← 基金自选 Store
 │    │    ├── items              自选基金列表
 │    │    ├── sortedItems        排序后列表
 │    │    ├── directionCounts    涨跌统计
 │    │    ├── addItem/removeItem 增删操作
 │    │    └── refreshQuotes      批量行情刷新
 │    └── stockWatchlist.ts      ← 股票自选 Store
 │         ├── stockItems         自选股票列表
 │         ├── addStockItem/removeStockItem
 │         └── refreshStockQuotes 批量行情刷新
 │
 ├── search.ts                   ← 搜索 Store
 │    ├── query/activeTab         搜索状态
 │    ├── fundResults/stockResults 搜索结果
 │    ├── fundFilters/stockFilters 筛选项
 │    ├── history                 搜索历史（localStorage）
 │    └── search()               执行搜索
 │
 ├── market.ts                   ← 行情 Store
 │
 ├── fundDetail.ts               ← 基金详情 Store
 │
 └── stockDetail.ts              ← 股票详情 Store
```

**自选列表持久化策略：**

- 基金自选：`localStorage` key `fund-watchlist`，最多 50 条
- 股票自选：`localStorage` key `stock-watchlist`，最多 50 条
- 使用 `watch` + 防抖（300ms）自动保存
- 行情数据不持久化，每次加载时从 API 刷新

### 4.3 API 集成层（Axios 拦截器/去重/重试）

前端 API 层基于 Axios 封装，提供以下核心能力：

**请求拦截器：**

```
请求发出
    │
    ├── 附加 Authorization 头（Bearer Token，从 localStorage 读取）
    ├── 附加 X-CSRF-Token 头（从 Cookie 读取，仅 POST/PUT/DELETE/PATCH）
    └── 请求去重（GET/POST 相同请求自动取消前一个）
         ├── 生成请求唯一 Key（method + url + params + body）
         ├── 已有相同请求 → 取消前一个
         └── 关联外部 AbortSignal
```

**响应拦截器：**

```
响应返回
    │
    ├── 成功 → 清理 pending 请求
    │
    └── 失败 → 错误分类处理
         ├── 请求取消 → CancelError
         ├── 401 → 清除 Token，派发 auth:expired 事件
         ├── 5xx + GET → 自动重试（最多 2 次，递增延迟 500ms×n）
         └── 统一错误映射
              ├── 无响应 → "网络连接失败"（retryable）
              ├── 401 → "登录已过期"（business）
              ├── 403 → "没有权限"（business）
              ├── 404 → "请求的资源不存在"（business）
              ├── 429 → "请求过于频繁"（retryable）
              ├── 5xx → "服务器繁忙"（retryable）
              └── 其他 → 原始错误信息
```

**API 模块划分：**

| 文件 | 职责 |
|------|------|
| api/index.ts | Axios 实例、拦截器、去重、重试、认证 |
| api/routes.ts | API 路径常量 |
| api/search.ts | 统一搜索、筛选 |
| api/watchlist.ts | 自选行情 |
| api/market.ts | 大盘指数、排行 |
| api/fundDetail.ts | 基金详情 |
| api/stock.ts | 股票搜索、详情、行情 |

### 4.4 路由设计

| 路径 | 名称 | 组件 | 说明 |
|------|------|------|------|
| `/` | — | → redirect `/watchlist` | 默认跳转 |
| `/watchlist` | Watchlist | WatchlistView | 自选列表 |
| `/market` | Market | MarketView | 行情中心 |
| `/predict` | Predict | PredictView | 预测入口 |
| `/predict/:fundCode` | PredictDetail | PredictView | 指定基金预测入口占位 |
| `/fund/:fundCode` | FundDetail | FundDetailView | 基金详情 |
| `/stock/:stockCode` | StockDetail | StockDetailView | 股票详情 |
| `/:pathMatch(.*)*` | NotFound | NotFoundView | 404 |

**路由守卫：**

- `beforeEach`：设置页面标题
- 预留认证检查点（当前未启用）
- `scrollBehavior`：路由切换时滚动到顶部

---

## 5. 数据模型

### 5.1 基金数据流

```
┌──────────────────────────────────────────────────────────────┐
│                     基金数据来源                               │
│                                                              │
│  ① 东方财富 fundcode_search.js                               │
│     URL: fund.eastmoney.com/js/fundcode_search.js            │
│     格式: JS 数组 → 正则提取 ["代码","拼音","类型","名称","全拼"] │
│     数据: 基金代码/名称/类型/拼音缩写/拼音全拼                    │
│                                                              │
│  ② 东方财富 rankhandler.aspx                                 │
│     URL: fund.eastmoney.com/data/rankhandler.aspx?...         │
│     格式: JS 对象 → CSV 解析 datas 字段                        │
│     数据: 净值/涨跌幅/收益率/成立日期等行情指标                    │
│                                                              │
│  ③ CSV 文件导入                                               │
│     格式: 标准 CSV，支持多种列名映射                             │
│     数据: 完整基金信息                                         │
└──────────────────────┬───────────────────────────────────────┘
                       │
                       ▼
┌──────────────────────────────────────────────────────────────┐
│              数据解析与合并 (persistence_sync.go)               │
│                                                              │
│  ReadEastmoneyFundCodeSearchJS()  → JS 正则解析               │
│  ReadEastmoneyFundRankHandlerJS() → CSV 子解析                │
│  ReadFundsCSV()                    → CSV 标准解析              │
│                                                              │
│  mergeFund() → 字段级合并策略：                                 │
│    - 非空值覆盖空值                                            │
│    - "未知"类型不覆盖已知类型                                    │
│    - 行情数据仅在 QuoteSource 为空时继承                         │
│    - mergeSeedFunds() → 种子数据与持久化数据合并                 │
└──────────────────────┬───────────────────────────────────────┘
                       │
                       ▼
┌──────────────────────────────────────────────────────────────┐
│                  内存存储 (MemoryStore)                        │
│                                                              │
│  结构: map[string]FundItem  (key = FundCode)                  │
│  并发: sync.RWMutex 读写锁                                    │
│  操作: ListFunds / FindFund / AddFund / RemoveFund / MergeFunds│
└──────────────────────┬───────────────────────────────────────┘
                       │
                       ▼
┌──────────────────────────────────────────────────────────────┐
│              JSON 持久化 (persistence.go)                      │
│                                                              │
│  格式: JSON Array of fundRecord                               │
│  路径: data/funds.json (可配置)                                │
│  写入: 原子写入（先写 .tmp 再 rename）                          │
│  读取: 启动时加载，与种子数据合并                                │
│  触发: 每次内存变更后自动持久化                                  │
└──────────────────────────────────────────────────────────────┘
```

### 5.2 股票数据流

```
┌──────────────────────────────────────────────────────────────┐
│                     股票数据来源                               │
│                                                              │
│  ① 东方财富股票列表 API (主)                                   │
│     URL: push2.eastmoney.com/api/qt/clist/get                 │
│     分页: 每页 5000 条，按市场类型分批                           │
│     字段: 代码/名称/价格/涨跌幅/成交量/PE/市值/行业等             │
│                                                              │
│  ② 东方财富数据中心 API (备)                                   │
│     URL: datacenter-web.eastmoney.com/api/data/v1/get         │
│     分页: 每页 500 条                                         │
│     字段: 代码/名称/市场/板块                                   │
│                                                              │
│  ③ 默认股票数据 (data/default_stocks.json)                     │
│     内嵌 JSON，作为兜底数据源                                   │
└──────────────────────┬───────────────────────────────────────┘
                       │
                       ▼
┌──────────────────────────────────────────────────────────────┐
│              数据处理 (stock_sync.go)                          │
│                                                              │
│  ① 主 API 失败 → 自动降级到数据中心 API                         │
│  ② 数据清洗：                                                 │
│     - 验证 6 位纯数字代码                                      │
│     - 去重（seen map）                                        │
│     - 过滤空名称                                               │
│  ③ 拼音生成：                                                 │
│     - pinyinAbbr() → 首字母缩写                               │
│     - pinyinAbbrAll() → 多音字全排列                           │
│  ④ 与默认数据合并                                              │
└──────────────────────┬───────────────────────────────────────┘
                       │
                       ▼
┌──────────────────────────────────────────────────────────────┐
│              内存存储 (MemoryStore - StockRepository)         │
│                                                              │
│  结构: map[string]StockItem  (key = StockCode)                │
│  并发: sync.RWMutex                                          │
│  缓存: rankingCache (30 秒过期)                               │
│  操作: 通过 StockRepository 接口访问                           │
│        Search / FindStock / ListStocks / SyncStocks / Ranking │
└──────────────────────────────────────────────────────────────┘
```

### 5.3 搜索索引（SQLite FTS5）

```
┌──────────────────────────────────────────────────────────────┐
│                  SearchIndex 架构                              │
│                                                              │
│  存储引擎: SQLite (modernc.org/sqlite, 纯 Go 实现)            │
│  运行模式: 内存模式 (file:search_index?mode=memory)             │
│  并发控制: MaxOpenConns=1 + sync.RWMutex                     │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐  │
│  │  基金表 (funds)                                         │  │
│  │  fund_code | fund_name | fund_type | pinyin_abbr |     │  │
│  │  pinyin_full | company | manager | risk_level          │  │
│  └──────────────────────┬─────────────────────────────────┘  │
│                         │                                    │
│  ┌──────────────────────┴─────────────────────────────────┐  │
│  │  基金 FTS 虚拟表 (funds_fts)                            │  │
│  │  索引字段: fund_code, pinyin_abbr, pinyin_full          │  │
│  │  触发器: INSERT/UPDATE/DELETE 自动同步                    │  │
│  └────────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐  │
│  │  股票表 (stocks)                                        │  │
│  │  stock_code | stock_name | market | industry | pinyin   │  │
│  └──────────────────────┬─────────────────────────────────┘  │
│                         │                                    │
│  ┌──────────────────────┴─────────────────────────────────┐  │
│  │  股票 FTS 虚拟表 (stocks_fts)                           │  │
│  │  索引字段: stock_code, pinyin                            │  │
│  │  触发器: INSERT/UPDATE/DELETE 自动同步                    │  │
│  └────────────────────────────────────────────────────────┘  │
│                                                              │
│  同步策略:                                                    │
│  - 启动时全量同步（SyncFunds / SyncStocks）                    │
│  - 事务操作：DELETE ALL → INSERT OR REPLACE                   │
│  - 运行时数据变更后需手动触发同步                                │
└──────────────────────────────────────────────────────────────┘
```

---

## 6. 搜索系统

### 6.1 FTS5 全文搜索 + 线性扫描混合策略

搜索系统采用**双引擎混合**策略，兼顾准确性和性能：

```
搜索请求 (keyword)
    │
    ├── 线性扫描引擎（主）
    │    ├── 遍历内存中所有基金/股票
    │    ├── 多字段模糊匹配（Contains）
    │    ├── 相关性评分（0-9 级）
    │    └── 上限 5000 条匹配结果
    │
    ├── FTS5 索引引擎（辅）
    │    ├── MATCH 查询（精确 + 前缀通配）
    │    ├── 按 rank 排序
    │    └── 上限 200 条匹配结果
    │
    └── 合并策略
         ├── 线性扫描结果为准（保留其评分）
         ├── FTS5 结果补充（仅添加线性扫描未匹配的项，评分=10）
         └── 按评分排序 → 分页返回
```

**基金搜索匹配字段：** FundCode、FundName、PinyinAbbr、PinyinFull、Company、Manager

**股票搜索匹配字段：** StockCode、StockName、Pinyin、PinyinAlt、Industry、Market

### 6.2 拼音系统（GBK 编码 + 23 项二分查找）

拼音首字母提取基于 GBK 编码区间映射：

```
汉字 → GBK 编码 → 区间查找 → 拼音首字母

示例：
  "华" → GBK: 0xBBAA → code=48042 → 区间 [47614, 48119) → "H"
  "夏" → GBK: 0xCFC4 → code=53188 → 区间 [52980, 53689) → "X"
```

**23 项拼音首字母映射表：**

| GBK 起始码 | 首字母 |
|-----------|--------|
| 45217 | A |
| 45253 | B |
| 45761 | C |
| 46318 | D |
| 46826 | E |
| 47010 | F |
| 47297 | G |
| 47614 | H |
| 48119 | J |
| 49062 | K |
| 49324 | L |
| 49896 | M |
| 50371 | N |
| 50614 | O |
| 50622 | P |
| 50906 | Q |
| 51387 | R |
| 51446 | S |
| 52218 | T |
| 52698 | W |
| 52980 | X |
| 53689 | Y |
| 54481 | Z |

> 注意：I/U/V 不存在对应拼音首字母。

### 6.3 多音字支持

系统内置 5 个常见金融领域多音字覆盖：

| Unicode | 汉字 | 默认读音 | 备选读音 | 示例 |
|---------|------|---------|---------|------|
| 0x884C | 行 | H | X | 行情(H)/银行(X) |
| 0x91CD | 重 | Z | C | 重仓(Z)/重合(C) |
| 0x957F | 长 | C | Z | 成长(C)/增长(Z) |
| 0x4E50 | 乐 | L | Y | 乐普(L)/乐视(Y) |
| 0x53C2 | 参 | C | S | 参考(C)/参差(S) |
| 0x5355 | 单 | D | S | 单位(D)/单县(S) |

**多音字拼音生成算法：**

1. `pinyinAbbr()` — 取默认读音首字母
2. `pinyinAbbrAll()` — 生成所有多音字排列组合
   - 基础缩写（默认读音）
   - 每个多音字的备选读音替换
   - 结果存入 `Pinyin`（默认）和 `PinyinAlt`（备选）

### 6.4 相关性评分算法

评分越低排名越靠前（0 = 完全匹配，9 = 最弱匹配）：

**基金搜索评分：**

| 评分 | 匹配条件 | 示例 |
|------|---------|------|
| 0 | 代码或名称完全匹配 | keyword="000001" 或 "华夏成长混合" |
| 1 | 代码前缀匹配 | keyword="0000" |
| 2 | 名称前缀匹配 | keyword="华夏" |
| 3 | 拼音缩写前缀匹配 | keyword="hx" |
| 4 | 拼音全拼前缀匹配 | keyword="huaxia" |
| 5 | 代码包含匹配 | keyword="001" |
| 6 | 名称包含匹配 | keyword="成长" |
| 7 | 拼音缩写包含匹配 | keyword="ch" |
| 8 | 拼音全拼包含匹配 | keyword="cheng" |
| 9 | 默认（FTS5 补充结果） | — |

**股票搜索评分：** 类似结构，额外支持 PinyinAlt 字段匹配。

**排序规则：** 评分升序 → 评分相同时按代码字典序升序 → 插入排序保证稳定性。

---

## 7. 安全设计

### 7.1 CORS 策略

```go
// 配置项：CORS_ORIGINS（逗号分隔）
// 开发模式默认：http://localhost:5173

// 策略：
// 1. 严格 Origin 匹配（非通配符）
// 2. Vary: Origin 防止缓存污染
// 3. 允许凭据：Access-Control-Allow-Credentials: true
// 4. 限制方法：GET, POST, OPTIONS
// 5. 限制请求头：Content-Type, Authorization, X-CSRFToken, X-CSRF-Token
// 6. 通配符 "*" 在非开发模式下发出安全警告
```

### 7.2 CSRF 防护

采用 **Double Submit Cookie** 模式：

```
GET/HEAD/OPTIONS 请求：
    → 检查 Cookie 中是否存在 csrf_token
    → 不存在 → 生成 32 字节随机令牌
    → 设置 Cookie（HttpOnly=true, Secure=非开发模式, SameSite=Lax）
    → 写入响应头 X-CSRF-Token
    → 服务端维护令牌映射（24h 过期）

POST/PUT/DELETE/PATCH 请求：
    → 读取 Cookie: csrf_token
    → 读取 Header: X-CSRF-Token
    → 验证：两者均非空且相等（使用 crypto/subtle.ConstantTimeCompare 进行时间安全比较，防止时序攻击）
    → 验证：令牌在服务端映射中存在且未过期
    → 失败 → 403 "CSRF token 验证失败"

清理机制：
    → 每 5 分钟清理过期令牌
    → 优雅关闭时停止清理协程
```

### 7.3 管理员令牌验证

```
需要管理员权限的接口：
    POST /api/v1/funds/sync
    POST /api/v1/stocks/sync

验证流程：
    1. 读取 Authorization 头，去除 "Bearer " 前缀
    2. 空令牌 → 401 "未提供管理员令牌"（生产环境直接拒绝，开发模式允许 dev-admin-token）
    3. 开发模式 + "dev-admin-token" → 放行
    4. 与 ADMIN_TOKEN 环境变量匹配（使用 crypto/subtle.ConstantTimeCompare 时间安全比较） → 放行
    5. 不匹配 → 401 "管理员令牌无效"
```

### 7.4 输入验证

| 验证项 | 规则 | 位置 |
|--------|------|------|
| 基金代码 | 6 位纯数字 | Handler + Service |
| 股票代码 | 6 位纯数字 | Handler + Service |
| 请求体大小 | 最大 1MB | MaxBody 中间件 |
| 自选列表 | 最多 50 个代码，每个 6 位数字 | Handler binding tag |
| 股票行情请求 | StockQuoteRequest binding tag 验证（Codes 必填，每项 6 位数字，最多 50 个） | Handler binding tag |
| 搜索分页 | Page ≥ 1, 1 ≤ Size ≤ 50 | Service 层 |
| 排行数量 | 1 ≤ Size ≤ 50 | Service 层 |
| FTS 查询 | 转义特殊字符 `("*,()^+-:)` | SearchIndex 层 |

### 7.5 URL 白名单

后端仅允许请求以下外部域名：

| 域名 | 用途 |
|------|------|
| `*.eastmoney.com` | 基金/股票数据源 |
| `push2.eastmoney.com` | 股票实时行情 |
| `push2his.eastmoney.com` | 股票历史数据 |
| `*.qq.com` | 腾讯数据源 |
| `qt.gtimg.cn` | 腾讯行情接口 |

### 7.6 Config 验证

生产环境启动时对关键配置进行校验：

```
校验项：
    ├── ADMIN_TOKEN 为空 → 启动警告（生产环境强烈建议配置）
    ├── CORS_ORIGINS 包含 "*" → 启动警告（非开发模式不推荐通配符）
    └── 配置冲突检测（如端口占用等）
```

### 7.7 安全头增强

非开发环境下自动启用以下安全响应头：

| 响应头 | 值 | 说明 |
|--------|------|------|
| Strict-Transport-Security | max-age=31536000; includeSubDomains | HSTS，强制 HTTPS（仅生产环境） |
| Content-Security-Policy | default-src 'self'; ... | CSP 策略，限制资源加载来源（仅生产环境） |
| X-XSS-Protection | 0 | 禁用浏览器 XSS 过滤器（现代浏览器已弃用，设为 0 避免误判） |

---

## 8. 部署架构

### 8.1 开发环境

```
┌─────────────────────────────────────────────────────────┐
│                    开发环境                               │
│                                                         │
│  前端 (Vite Dev Server)                                  │
│  ├── 端口: 5173                                          │
│  ├── 热更新: HMR                                         │
│  ├── API 代理: VITE_API_BASE_URL → http://localhost:5070 │
│  └── 环境: VITE_API_WITH_CREDENTIALS=true                │
│                                                         │
│  后端 (Go)                                               │
│  ├── 端口: 5070                                          │
│  ├── 环境: APP_ENV=development                           │
│  ├── 日志级别: Debug                                     │
│  ├── CORS: 默认允许 localhost:5173                        │
│  ├── 管理员令牌: "dev-admin-token"                        │
│  ├── 数据目录: data/funds.json                           │
│  ├── 自动同步: FUND_AUTO_SYNC_ON_START=true               │
│  └── 自动同步: STOCK_AUTO_SYNC_ON_START=true              │
│                                                         │
│  预测：当前仅保留入口占位，模型训练与推理将由独立项目接入    │
└─────────────────────────────────────────────────────────┘
```

### 8.2 生产环境建议

```
┌─────────────────────────────────────────────────────────┐
│                    生产环境                               │
│                                                         │
│  ┌─────────────┐    ┌──────────────────────────────┐    │
│  │   Nginx     │    │        Go API 服务             │    │
│  │  反向代理   │───▶│  端口: 5070                    │    │
│  │  SSL 终止   │    │  APP_ENV=production            │    │
│  │  静态资源   │    │  ADMIN_TOKEN=<强随机令牌>       │    │
│  └─────────────┘    │  CORS_ORIGINS=<实际域名>        │    │
│                     │  FUND_AUTO_SYNC_ON_START=true    │    │
│                     │  STOCK_AUTO_SYNC_ON_START=true   │    │
│                     │  CACHE_TTL_MINUTES=5             │    │
│                     └──────────────┬───────────────────┘    │
│                                    │                       │
│                     ┌──────────────┴───────────────────┐    │
│                     │         外部数据源                  │    │
│                     │  东方财富 API · 腾讯行情 API        │    │
│                     └──────────────────────────────────┘    │
│                                                         │
│  关键配置：                                               │
│  ├── ADMIN_TOKEN: 至少 32 字符随机字符串                   │
│  ├── CORS_ORIGINS: 精确域名，禁止 "*"                      │
│  ├── READ_TIMEOUT_SECONDS: 8                              │
│  ├── WRITE_TIMEOUT_SECONDS: 12                            │
│  ├── SHUTDOWN_TIMEOUT_SECONDS: 8                          │
│  └── FUND_STORE_PATH: 持久化存储路径                       │
│                                                         │
│  安全加固：                                               │
│  ├── HTTPS（Nginx SSL 终止）                              │
│  ├── CSRF Secure Cookie（非开发模式自动启用）               │
│  ├── 限流 60 次/分钟/IP                                   │
│  ├── 请求体 1MB 上限                                      │
│  └── 安全响应头（X-Frame-Options, CSP 等）                 │
└─────────────────────────────────────────────────────────┘
```

**环境变量完整清单：**

| 变量 | 默认值 | 说明 |
|------|--------|------|
| PORT | 5070 | 服务端口 |
| APP_ENV | development | 运行环境 |
| CORS_ORIGINS | — | 允许的来源（逗号分隔） |
| ADMIN_TOKEN | — | 管理员令牌 |
| FUND_STORE_PATH | data/funds.json | 基金数据持久化路径 |
| FUND_UNIVERSE_URL | eastmoney fundcode_search.js | 基金宇宙数据源 |
| FUND_METRICS_URL | eastmoney rankhandler.aspx | 基金指标数据源 |
| FUND_SYNC_CSV_PATH | — | CSV 同步路径 |
| FUND_AUTO_SYNC_ON_START | true | 启动时自动同步基金 |
| FUND_AUTO_SYNC_MIN_COUNT | 1000 | 触发自动同步的最小基金数 |
| FUND_REALTIME_QUOTES_ENABLED | true | 启用基金实时行情 |
| STOCK_AUTO_SYNC_ON_START | true | 启动时自动同步股票 |
| CACHE_TTL_MINUTES | 5 | 缓存过期时间 |
| EASTMONEY_BASE_URL | push2his.eastmoney.com | 东方财富历史数据基址 |
| TENCENT_QUOTE_BASE_URL | qt.gtimg.cn | 腾讯行情基址 |
| READ_TIMEOUT_SECONDS | 8 | 读超时 |
| WRITE_TIMEOUT_SECONDS | 12 | 写超时 |
| SHUTDOWN_TIMEOUT_SECONDS | 8 | 优雅关闭超时 |

---

## 9. 商业级边界收敛

### 9.1 后端边界

后端新增 `internal/app` 作为应用装配层，`cmd/api` 只负责配置加载、日志初始化、进程启动和优雅关闭。HTTP 处理仍位于 `internal/api`，业务能力继续由现有 `internal/service` 与 `internal/store` 承载；后续领域迁移以 `fund`、`stock`、`market`、`watchlist`、`search` 和 `prediction` 为边界逐步收敛。

`internal/platform` 作为平台能力入口，当前包含统一响应 envelope 与错误码基础类型，后续可逐步承接通用响应、错误映射、安全策略和 telemetry 边界。

### 9.2 前端边界

前端新增 `src/app` 作为应用启动入口，`src/shared` 作为共享 API 与通用能力边界，`src/features` 作为业务模块入口。现有 `src/api`、`src/stores`、`src/components`、`src/views` 路径保持兼容，避免一次性迁移造成大范围回归。

### 9.3 P0 数据可见性修复

本轮排查发现前端基金/股票数据不可见的关键风险来自后端 gzip 中间件响应污染：当客户端发送 `Accept-Encoding: gzip` 时，旧实现会输出普通 JSON 后附加 gzip trailer 字节，导致浏览器或 Axios 无法稳定解析 JSON。修复后 gzip writer 只在确认 JSON/text 响应需要压缩时创建，并只在实际创建后关闭。

回归测试覆盖：

- `backend-go/internal/api/router_test.go`：验证 gzip 响应可被正常解压并解析为 API JSON。
- `frontend/src/__tests__/data-visibility.test.ts`：验证市场基金排行、热门股票、基金详情和股票详情能消费 API payload 并渲染关键数据。
- `scripts/verify-api-contract.ps1`：以 `frontend/src/shared/api/routes.ts` 作为前端 API route 真源，与后端路由和 OpenAPI 对齐。
