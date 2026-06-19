package providers

import "time"

const (
	DefaultSearchSize      = 20  // 默认搜索返回条数
	MaxSearchSize          = 50  // 搜索最大返回条数
	MaxStockQuoteBatch     = 50  // 股票行情单批最大请求数
	MaxWatchlistBatch      = 50  // 自选股单批最大请求数
	MaxSearchKeywordLen    = 100 // 搜索关键词最大长度
	MaxSearchMatches       = 5000 // 搜索最大匹配数
	FTSSearchLimit         = 200  // 全文搜索返回上限
	DefaultRankingSize     = 10   // 默认排行返回条数
	MaxRankingSize         = 50   // 排行最大返回条数
	DefaultFundRankingSize = 5    // 默认基金排行返回条数

	CacheMaxEntries = 1000            // 详情缓存最大条目数
	CacheTTL        = 5 * time.Minute // 详情缓存默认过期时间
	RankingCacheTTL = 30 * time.Second // 排行缓存过期时间

	IndexQuoteCacheTTL         = 15 * time.Second  // 指数行情缓存过期时间
	IndexKlineCacheTTL         = 5 * time.Minute   // 指数K线缓存过期时间
	IndexMinuteCacheTTL        = 30 * time.Second  // 指数分时缓存过期时间
	StockQuoteCacheTTL         = 15 * time.Second  // 股票行情缓存过期时间
	StockQuoteRealtimeFreshTTL = 3 * time.Second   // 股票行情实时新鲜度TTL
	StockQuoteRealtimeStaleTTL = 15 * time.Second  // 股票行情实时过期TTL
	StockQuoteIdleFreshTTL     = 60 * time.Second  // 股票行情盘后新鲜度TTL
	SectorCacheTTL             = 60 * time.Second  // 板块排行缓存过期时间
	NorthboundCacheTTL         = 60 * time.Second  // 北向资金缓存过期时间

	StockSyncTimeout      = 120  // 股票同步总超时（秒）
	StockSyncPageDelay    = 200  // 股票同步分页延迟（毫秒）
	DataCenterPageDelay   = 150  // 数据中心分页延迟（毫秒）
	MaxFundGZConcurrency  = 8    // 基金估值最大并发数
	StockQuoteConcurrency = 5    // 股票行情最大并发数
	StockQuoteBatchSize   = 30   // 股票行情每批请求数

	RiskFreeRate          = 0.015 // 无风险利率
	NAVHistoryPages       = 3     // 净值历史获取页数
	NAVHistoryDaysPerPage = 365   // 每页净值历史天数
	MinNAVHistoryForRisk  = 15    // 风险指标计算最少净值记录数
	MinReturnsForRisk     = 20    // 风险指标计算最少日收益率数
	TradingDaysPerYear    = 252   // 每年交易日数

	MarketSyncTradingInterval     = 5 * time.Minute  // 交易时段同步间隔
	MarketSyncIdleInterval        = 30 * time.Minute // 非交易时段同步间隔
	MarketSyncCleanupInterval     = 24 * time.Hour   // 数据清理间隔
	MarketDataRetentionDays       = 1095             // 市场数据保留天数（3年）
	MarketSyncValidationThreshold = 0.5              // 数据校验偏差阈值（百分比）
)

// 数据源健康监控相关常量
const (
	HealthCheckFailThreshold = 3 // 连续失败次数标记为 unhealthy
	HealthRecoveryInterval   = 5 // 健康恢复探测间隔（分钟）
)

// TDX 预加载相关常量
const (
	TDXPreloadBatchSize = 100 // TDX预加载每批获取条数
	TDXPreloadTimeout   = 120 // TDX预加载总超时（秒）
)

// 缓存策略相关常量
const (
	MarketMinuteRetentionDays = 7  // 分时保留7天
	KlineIncrementalDays      = 5  // K线增量更新天数
	CacheWarmupDays           = 30 // LRU预热天数
)
