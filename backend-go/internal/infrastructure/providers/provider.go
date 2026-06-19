// Package providers 实现了多数据源行情服务的基础设施层，包含数据源接口定义、
// 路由与回退策略、健康监控、缓存装饰器以及各数据源（腾讯、东方财富、新浪、通达信、
// 同花顺、币赢、AKShare）的具体实现。本包是 domain 层与外部数据 API 之间的桥梁。
package providers

import (
	"context"

	funddomain "stock-predict-go/internal/domain/fund"
	marketdomain "stock-predict-go/internal/domain/market"
	stockdomain "stock-predict-go/internal/domain/stock"
)

// Provider 表示一个市场数据源，是所有数据源必须实现的核心接口。
type Provider interface {
	// Name 返回数据源的唯一标识名称，例如 "tencent"、"eastmoney"。
	Name() string
	// Capabilities 返回该数据源支持的能力及其覆盖的市场。
	Capabilities() map[Capability][]Market
	// Priority 返回指定能力和市场下的优先级，数值越小优先级越高。
	Priority(cap Capability, market Market) int
	// HealthCheck 对数据源执行健康检查，正常时返回 nil。
	HealthCheck(ctx context.Context) error
}

// StockRankingProvider 由支持获取股票排行的数据源实现。
type StockRankingProvider interface {
	Provider
	FetchStockRanking(ctx context.Context, rankingType string, size int) ([]stockdomain.StockRankingItem, error)
}

// SectorRankingProvider 由支持获取板块排行的数据源实现。
type SectorRankingProvider interface {
	Provider
	FetchSectorRanking(ctx context.Context) ([]marketdomain.MarketSectorItem, error)
}

// NorthboundProvider 由支持获取北向资金流向的数据源实现。
type NorthboundProvider interface {
	Provider
	FetchNorthboundFlow(ctx context.Context) (*marketdomain.NorthboundFlow, error)
}

// IndexQuoteProvider 由支持获取指数实时行情的数据源实现。
type IndexQuoteProvider interface {
	Provider
	FetchIndexQuotes(ctx context.Context, market Market) ([]marketdomain.MarketIndex, error)
}

// IndexMinuteProvider 由支持获取指数分时数据的数据源实现。
type IndexMinuteProvider interface {
	Provider
	FetchIndexMinute(ctx context.Context, code string, market Market) ([]marketdomain.IndexMinutePoint, error)
}

// IndexKlineProvider 由支持获取指数 K 线数据的数据源实现。
type IndexKlineProvider interface {
	Provider
	FetchIndexKline(ctx context.Context, code string, market Market, count int) ([]marketdomain.IndexKlinePoint, error)
}

// StockMinuteProvider 由支持获取股票分时数据的数据源实现。
type StockMinuteProvider interface {
	Provider
	FetchStockMinute(ctx context.Context, code string) ([]marketdomain.IndexMinutePoint, error)
}

// StockSearchProvider 由支持股票搜索的数据源实现。
type StockSearchProvider interface {
	Provider
	SearchStocks(ctx context.Context, keyword string) ([]stockdomain.StockSearchItem, error)
}

// StockSyncProvider 由支持同步股票列表的数据源实现。
type StockSyncProvider interface {
	Provider
	SyncStocks(ctx context.Context) ([]stockdomain.StockItem, error)
}

// FundQuoteProvider 由支持获取基金估值的数据源实现。
type FundQuoteProvider interface {
	Provider
	FetchFundQuotes(ctx context.Context, codes []string) (map[string]funddomain.FundItem, error)
}

// StockQuoteProvider 由支持获取股票实时行情的数据源实现。
type StockQuoteProvider interface {
	Provider
	FetchStockQuotes(ctx context.Context, symbols []string) (map[string]stockdomain.StockQuote, error)
}
