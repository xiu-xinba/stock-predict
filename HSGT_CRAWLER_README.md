# 北向南向资金爬虫系统

## 概述

这是一个自动爬取东方财富网北向资金（陆股通）和南向资金（港股通）数据的系统。功能包括：

- **每日定时爬取**：每天 20:00 自动爬取当日数据
- **历史数据收集**：支持批量爬取历史数据（近一年）
- **滚动存储**：自动删除超过一年的旧数据，保持数据库大小在合理范围
- **API 接口**：提供多个 API 查询数据

## 核心组件

### 1. 数据模型 (`HSGTFlowDaily`)
位置：`internal/infrastructure/database/hsgt_flow.go`

存储每日的北向/南向资金数据：
- **date** (YYYY-MM-DD)：交易日期
- **updateTime**：数据更新时间
- **NorthSHBuy / NorthSZBuy / NorthTotalBuy**：北向资金净买入（万元）
- **SouthHKBuy / SouthSHBuy / SouthSZBuy / SouthTotalBuy**：南向资金（万元）

### 2. 爬虫服务 (`HSGTScraper`)
位置：`internal/infrastructure/providers/hsgt_scraper.go`

主要方法：
- `ScrapeToday(ctx)` - 爬取今天的数据
- `ScrapeDate(ctx, date)` - 爬取指定日期的数据
- `FetchHistoricalData(ctx, startDate)` - 爬取历史数据
- `CleanupOldData(ctx)` - 清理超过一年的数据

### 3. 定时调度器 (`HSGTScheduler`)
位置：`internal/infrastructure/providers/hsgt_scheduler.go`

**执行时间**：每天 20:00-20:05（每30秒检查一次）

主要方法：
- `Start()` - 启动调度器
- `Stop()` - 停止调度器
- `SyncHistoricalData(ctx, days)` - 同步历史数据

## API 接口

### 获取最新数据
```
GET /api/v1/hsgt/latest
```
返回最新一天的北向南向资金数据。

### 获取最近 N 天的数据
```
GET /api/v1/hsgt/recent?days=30
```
查询参数：
- `days`: 1-365，默认 30

### 按日期范围查询
```
GET /api/v1/hsgt/range?start=2024-01-01&end=2024-01-31
```
查询参数：
- `start`: 开始日期 (YYYY-MM-DD)
- `end`: 结束日期 (YYYY-MM-DD)

### 按特定日期查询
```
GET /api/v1/hsgt/date/:date
```
路径参数：
- `date`: 日期 (YYYY-MM-DD)

### 获取统计信息
```
GET /api/v1/hsgt/stats
```
返回数据统计信息，包括：
- 总记录数
- 最新日期及数据
- 最早日期

## 初始化步骤

### 1. 数据库迁移
系统启动时会自动创建 `hsgt_flow_daily` 表。

### 2. 首次运行 - 导入历史数据
如果是首次使用，需要导入近一年的历史数据。在应用初始化代码中调用：

```go
// 在 app.go 或类似的初始化代码中
if err := services.HSGTScheduler.SyncHistoricalData(ctx, 365); err != nil {
    logger.Error("failed to sync historical HSGT data", "error", err)
}
```

### 3. 启动定时任务
应用启动时自动启动 HSGT 调度器（见 `internal/app/app.go`）。

## 数据流转图

```
┌─────────────────────┐
│  东方财富网         │  (https://data.eastmoney.com/hsgt/hsgtV2.html)
└──────────┬──────────┘
           │
           ↓
┌─────────────────────┐
│  HSGTScraper        │  (爬取和解析 HTML)
└──────────┬──────────┘
           │
           ↓
┌─────────────────────┐
│  HSGTScheduler      │  (定时调度：每天 20:00)
└──────────┬──────────┘
           │
           ↓
┌─────────────────────┐
│  HSGTFlowDaily      │  (数据库表)
│  (PostgreSQL)       │
└──────────┬──────────┘
           │
           ↓
┌─────────────────────┐
│  API Handlers       │  (GET /api/v1/hsgt/*)
└─────────────────────┘
```

## 数据流向示例

```json
{
  "id": 1,
  "date": "2024-06-16",
  "updateTime": "14:00:00",
  "northSHBuy": 27375.73,      // 沪股通净买入（万元）
  "northSZBuy": 25143.14,      // 深股通净买入（万元）
  "northTotalBuy": 52518.87,   // 北向合计
  "southHKBuy": 12345.67,      // 香港买入（万元）
  "southSHBuy": 6789.12,       // 香港买入沪股通
  "southSZBuy": 5556.55,       // 香港买入深股通
  "southTotalBuy": 12345.67,   // 南向合计
  "source": "eastmoney",
  "status": "completed",
  "createdAt": "2024-06-16T20:05:00Z",
  "updatedAt": "2024-06-16T20:05:00Z"
}
```

## 存储策略

### 滚动存储（Rolling Storage）
- **保留周期**：近 365 天的数据
- **自动清理**：每天爬取新数据后，自动删除超过一年的最早数据
- **清理时机**：`executeDaily()` 方法中，新数据保存后立即执行

### 示例：
假设今天是 2024-06-16，系统会删除 2023-06-15 及更早的数据。

## 性能考虑

1. **爬虫频率**：仅每天一次（20:00），不会对源网站造成压力
2. **数据量**：年度数据约 250 条记录（工作日），占用极少数据库空间
3. **API 查询速度**：
   - 单日查询：< 5ms
   - 年度数据查询：< 50ms

## 错误处理

### 爬虫失败
- 如果爬虫失败，会在日志中记录错误，但不会中断调度器
- 下次执行（次日 20:00）会重试

### 网络超时
- 爬虫请求超时设置为 5 分钟
- 如果源网站不可用，日志中会记录警告

## 监控和调试

### 获取爬虫统计信息
```bash
curl http://localhost:8080/api/v1/hsgt/stats
```

### 查看日志
爬虫活动会在应用日志中记录，查找关键字 "HSGT"。

## 常见问题

### Q: 爬虫何时开始运行？
A: 应用启动时立即启动调度器，然后在每天 20:00 执行爬虫。

### Q: 数据是否实时更新？
A: 不是实时的，仅在每天 20:00 更新一次。源网站在交易时段（09:30-11:30, 13:00-15:00）更新数据。

### Q: 历史数据需要多长时间导入？
A: 取决于网络速度，一年的数据（约 250 条）通常需要 5-10 分钟。

### Q: 如何手动触发爬虫？
A: 当前系统不支持手动触发。如需要，可以修改代码或通过管理接口实现。

## 扩展建议

1. **支持更多数据源**：可添加其他网站的爬虫实现
2. **数据验证**：添加数据有效性检查
3. **告警机制**：当数据异常时发送告警
4. **导出功能**：提供 CSV/Excel 导出接口
5. **统计分析**：添加资金流向趋势分析 API

## 依赖项

- Go 1.20+
- PostgreSQL 12+
- GORM ORM

## 测试

```bash
# 测试爬虫（手动调用）
go test ./internal/infrastructure/providers -run TestHSGTScraper

# 测试数据库存储
go test ./internal/infrastructure/database -run TestHSGTFlowDaily
```

## 许可和免责声明

本爬虫用于学习和研究目的。使用本工具请遵守数据来源网站的服务条款和机器人协议。
