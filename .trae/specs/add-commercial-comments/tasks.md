# Tasks

- [x] Task 1: Go 后端 domain 层注释 — 为 `internal/domain/` 下所有包添加包注释、导出类型/接口/字段注释
  - [x] SubTask 1.1: `domain/stock` 包 — 包注释 + StockItem/StockQuote/StockDetailData 等类型及字段注释
  - [x] SubTask 1.2: `domain/fund` 包 — 包注释 + FundItem/FundDetail 等类型及字段注释 + Repository 接口注释
  - [x] SubTask 1.3: `domain/market` 包 — 包注释 + MarketIndex/NorthboundFlow 等类型及字段注释
  - [x] SubTask 1.4: `domain/search` 包 — 包注释 + SearchResult 等类型注释

- [x] Task 2: Go 后端 platform 层注释 — 为 `internal/platform/` 下所有包添加包注释、导出符号注释
  - [x] SubTask 2.1: `platform/config` 包 — 包注释 + Config 结构体字段注释 + Load/Validate 等函数注释
  - [x] SubTask 2.2: `platform/errors` 包 — 包注释 + AppError 类型注释 + 错误码常量注释
  - [x] SubTask 2.3: `platform/cache` 包 — 包注释 + LRU 缓存类型和方法注释
  - [x] SubTask 2.4: `platform/httpclient` 包 — 包注释 + Client/ResilientClient 等类型和方法注释
  - [x] SubTask 2.5: `platform/observability` 包 — 包注释 + HTTP 指标相关类型注释

- [x] Task 3: Go 后端 infrastructure/database 层注释 — 为 `internal/infrastructure/database/` 下所有文件添加注释
  - [x] SubTask 3.1: 数据库核心 — db.go/migrations.go/models.go 包注释 + 导出函数/类型注释
  - [x] SubTask 3.2: Store 层 — fund_store.go/stock_store.go/market_store.go/search_store.go 导出方法注释
  - [x] SubTask 3.3: Seed 层 — seed/ 子包包注释 + 导出函数注释

- [x] Task 4: Go 后端 infrastructure/providers 层注释 — 为 `internal/infrastructure/providers/` 下所有文件添加注释
  - [x] SubTask 4.1: 核心接口 — provider.go/registry.go/provider_router.go/provider_types.go/constants.go 包注释 + 接口/类型注释
  - [x] SubTask 4.2: 服务层 — stock_service.go/fund_service.go/market_service.go/search_service.go/watchlist_service.go 等服务函数注释
  - [x] SubTask 4.3: 数据源实现 — eastmoney_provider.go/sina_provider.go/tdx_provider.go/tencent_provider.go/ths_provider.go/biyingapi_provider.go/akshare_provider.go 导出函数注释
  - [x] SubTask 4.4: 辅助模块 — cache_provider.go/data_source_health.go/data_source_recovery.go/fund_detail_service.go/stock_detail_service.go 等注释
  - [x] SubTask 4.5: 子目录数据源 — providers/akshare/provider.go, providers/biying/provider.go, providers/eastmoney/provider.go, providers/sina/provider.go, providers/tdx/provider.go, providers/tencent/provider.go, providers/ths/provider.go

- [x] Task 5: Go 后端 transport 层注释 — 为 `internal/transport/http/` 下所有文件添加注释
  - [x] SubTask 5.1: handler 层 — handler.go/stock_handler.go/fund_handler.go/market_handler.go/search_handler.go/prediction_handler.go 包注释 + 处理函数注释
  - [x] SubTask 5.2: middleware 层 — middleware.go 包注释 + 中间件函数注释
  - [x] SubTask 5.3: response 层 — response.go 包注释 + 响应工具函数注释
  - [x] SubTask 5.4: router 层 — router.go/metrics.go 包注释 + 路由注册函数注释

- [x] Task 6: Go 后端 app 层 + cmd 层注释
  - [x] SubTask 6.1: `app/app.go` — 包注释 + NewServer 函数注释 + 关键逻辑行内注释
  - [x] SubTask 6.2: `cmd/api/main.go` + `cmd/migrate/main.go` — 包注释 + main 函数注释

- [x] Task 7: 前端 shared 层注释 — 为 `frontend/src/shared/` 下所有文件添加 JSDoc 注释
  - [x] SubTask 7.1: API 层 — client.ts/routes.ts/types.ts 模块注释 + 导出函数/类型 JSDoc
  - [x] SubTask 7.2: Charts 层 — echarts.ts/useECharts.ts 模块注释 + composable JSDoc
  - [x] SubTask 7.3: Components 层 — 所有 .vue 组件说明注释 + AssetHeader/CollapsibleCard/ErrorState/SkeletonTable/DetailPageLayout
  - [x] SubTask 7.4: Composables 层 — useStaggerEntry.ts/useTheme.ts composable JSDoc
  - [x] SubTask 7.5: Utils 层 — format.ts 导出函数 JSDoc + types/errors.ts 类型 JSDoc

- [x] Task 8: 前端 features 层注释 — 为 `frontend/src/features/` 下所有模块添加注释
  - [x] SubTask 8.1: funds 模块 — types.ts/api/funds.ts/store/fundDetail.ts/composables + 所有组件
  - [x] SubTask 8.2: stocks 模块 — types.ts/api/stocks.ts/store/stockDetail.ts + 所有组件
  - [x] SubTask 8.3: market 模块 — types.ts/api/market.ts/store/market.ts/utils/marketTime.ts + 所有组件
  - [x] SubTask 8.4: search 模块 — types.ts/api/search.ts/store/search.ts/composables/useSearch.ts/utils/highlight.ts + 组件
  - [x] SubTask 8.5: watchlist 模块 — types.ts/api/watchlist.ts/store + 组件
  - [x] SubTask 8.6: prediction 模块 — index.ts/components/PredictionPlaceholder.vue
  - [x] SubTask 8.7: settings 模块 — store/settings.ts + SettingsView.vue

- [x] Task 9: 前端 app 层注释 — `frontend/src/app/` 下所有文件
  - [x] SubTask 9.1: App.vue/router.ts/bootstrap.ts 模块注释
  - [x] SubTask 9.2: NotFoundView.vue + RefreshFab.vue 组件注释

- [x] Task 10: Python AKShare 微服务注释
  - [x] SubTask 10.1: main.py — 模块 docstring + 所有路由函数 Google 风格 docstring

- [x] Task 11: 验证 — 确保注释添加后项目仍可正常构建和通过检查
  - [x] SubTask 11.1: 运行 `cd backend-go && go vet ./...` 确认无警告
  - [x] SubTask 11.2: 运行 `cd backend-go && go build ./...` 确认构建通过
  - [x] SubTask 11.3: 运行 `cd frontend && npx vue-tsc --noEmit` 确认类型检查通过
  - [x] SubTask 11.4: 运行 `cd frontend && npm run build` 确认构建通过

# Task Dependencies
- [Task 1-6] 可并行执行（Go 后端各层独立）
- [Task 7-9] 可并行执行（前端各层独立）
- [Task 10] 独立执行
- [Task 11] 依赖 Task 1-10 全部完成
