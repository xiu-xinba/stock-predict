package providers

// Capability 表示数据源支持的能力
type Capability string

const (
	CapIndexQuote   Capability = "index_quote"   // 指数实时行情
	CapIndexMinute  Capability = "index_minute"  // 指数分时
	CapIndexKline   Capability = "index_kline"   // 指数K线
	CapStockQuote   Capability = "stock_quote"   // 股票实时行情
	CapStockMinute  Capability = "stock_minute"  // 股票分时
	CapStockSearch  Capability = "stock_search"  // 股票搜索
	CapStockSync    Capability = "stock_sync"    // 股票列表同步
	CapStockRanking Capability = "stock_ranking" // 股票排行
	CapSectorRank   Capability = "sector_rank"   // 板块排行
	CapNorthbound   Capability = "northbound"    // 北向资金
	CapFundQuote    Capability = "fund_quote"    // 基金估值
)

// FetchStrategy 请求策略
type FetchStrategy string

const (
	// StrategyFallback 串行回退：按优先级依次尝试，失败则降级到下一个
	StrategyFallback FetchStrategy = "fallback"
	// StrategyRace 竞速：同时请求 Top-N 源，取最快返回的成功结果
	StrategyRace FetchStrategy = "race"
	// StrategyRaceThenFallback 混合：竞速 Top-2，都失败则继续串行回退
	StrategyRaceThenFallback FetchStrategy = "race_then_fallback"
)

// providerError 表示来自特定数据源的错误，包含数据源名称和错误信息。
// 当 health 为 true 时，表示该错误来自健康检查。
type providerError struct {
	provider string
	msg      string
	health   bool
}

// Error 返回格式化的错误字符串。
func (e *providerError) Error() string {
	if e.health {
		return "health check " + e.provider + ": " + e.msg
	}
	return e.provider + ": " + e.msg
}

// newProviderError 创建一个普通的数据源错误。
func newProviderError(provider, msg string) *providerError {
	return &providerError{provider: provider, msg: msg}
}

// newHealthCheckError 创建一个健康检查类型的错误。
func newHealthCheckError(provider, msg string) *providerError {
	return &providerError{provider: provider, msg: msg, health: true}
}
