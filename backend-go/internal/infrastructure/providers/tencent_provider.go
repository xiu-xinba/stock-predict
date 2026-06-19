package providers

import (
	"context"

	funddomain "stock-predict-go/internal/domain/fund"
	marketdomain "stock-predict-go/internal/domain/market"
	stockdomain "stock-predict-go/internal/domain/stock"
)

// TencentProvider 实现了腾讯行情数据源的 Provider 接口。
type TencentProvider struct {
	quoteClient *IndexQuoteClient
	stockClient *StockQuoteClient
	fundClient  *FundQuoteClient
}

// NewTencentProvider 创建一个新的 TencentProvider 实例。
func NewTencentProvider(quoteClient *IndexQuoteClient, stockClient *StockQuoteClient, fundClient *FundQuoteClient) *TencentProvider {
	return &TencentProvider{
		quoteClient: quoteClient,
		stockClient: stockClient,
		fundClient:  fundClient,
	}
}

// Name 返回数据源的唯一标识名称。
func (p *TencentProvider) Name() string {
	return "tencent"
}

// Capabilities 返回腾讯行情数据源支持的能力及其适用的市场。
func (p *TencentProvider) Capabilities() map[Capability][]Market {
	return map[Capability][]Market{
		CapIndexQuote:  {MarketCN, MarketHK, MarketUS},
		CapIndexMinute: {MarketCN, MarketHK, MarketUS},
		CapIndexKline:  {MarketCN, MarketHK, MarketUS},
		CapStockQuote:  {MarketCN, MarketHK, MarketUS},
		CapStockMinute: {MarketCN, MarketHK, MarketUS},
		CapFundQuote:   {MarketCN},
	}
}

// Priority 返回指定能力和市场组合下的优先级，数值越小优先级越高。
func (p *TencentProvider) Priority(cap Capability, market Market) int {
	switch cap {
	case CapIndexQuote:
		return 1
	case CapIndexMinute:
		if market == MarketCN {
			return 2
		}
		return 1
	case CapIndexKline:
		if market == MarketCN {
			return 2
		}
		return 1
	case CapStockQuote:
		return 1
	case CapStockMinute:
		return 1
	case CapFundQuote:
		return 1
	default:
		return 99
	}
}

// HealthCheck 通过获取 A 股指数行情检测数据源健康状态。
func (p *TencentProvider) HealthCheck(ctx context.Context) error {
	result := p.quoteClient.fetchCNIndexQuotesTencent(ctx)
	if len(result) == 0 {
		return newHealthCheckError("tencent", "empty result")
	}
	return nil
}

// FetchIndexQuotes 根据市场类型获取对应的指数行情数据。
func (p *TencentProvider) FetchIndexQuotes(ctx context.Context, market Market) ([]marketdomain.MarketIndex, error) {
	var result []marketdomain.MarketIndex
	switch market {
	case MarketCN:
		result = p.quoteClient.fetchCNIndexQuotesTencent(ctx)
	case MarketHK:
		result = p.quoteClient.fetchHKIndexQuotesTencent(ctx)
	case MarketUS:
		result = p.quoteClient.fetchUSIndexQuotesTencent(ctx)
	default:
		return nil, newProviderError("tencent", "unsupported market")
	}
	if len(result) == 0 {
		return nil, newProviderError("tencent", "empty result")
	}
	return result, nil
}

// FetchIndexMinute 根据市场和代码获取指数分时数据。
func (p *TencentProvider) FetchIndexMinute(ctx context.Context, code string, market Market) ([]marketdomain.IndexMinutePoint, error) {
	var result []marketdomain.IndexMinutePoint
	switch market {
	case MarketCN:
		result = p.quoteClient.fetchCNIndexMinuteTencent(ctx, code)
	case MarketHK:
		result = p.quoteClient.fetchHKIndexMinuteTencent(ctx, code)
	case MarketUS:
		result = p.quoteClient.fetchUSIndexMinuteTencent(ctx, code)
	default:
		return nil, newProviderError("tencent", "unsupported market")
	}
	if len(result) == 0 {
		return nil, newProviderError("tencent", "empty result")
	}
	return result, nil
}

// FetchIndexKline 根据市场和代码获取指数 K 线数据。
func (p *TencentProvider) FetchIndexKline(ctx context.Context, code string, market Market, count int) ([]marketdomain.IndexKlinePoint, error) {
	var result []marketdomain.IndexKlinePoint
	switch market {
	case MarketCN:
		result = p.quoteClient.fetchCNIndexKlineTencent(ctx, code, count)
	default:
		// HK and US both use the same method
		result = p.quoteClient.fetchHKUSIndexKlineTencent(ctx, code, count)
	}
	if len(result) == 0 {
		return nil, newProviderError("tencent", "empty result")
	}
	return result, nil
}

// FetchStockQuotes 根据股票代码列表获取实时行情数据。
func (p *TencentProvider) FetchStockQuotes(ctx context.Context, symbols []string) (map[string]stockdomain.StockQuote, error) {
	result := p.stockClient.fetchTencentStockQuotes(ctx, symbols)
	if len(result) == 0 {
		return nil, newProviderError("tencent", "empty result")
	}
	return result, nil
}

// FetchStockMinute 根据股票代码获取分时数据。
func (p *TencentProvider) FetchStockMinute(ctx context.Context, code string) ([]marketdomain.IndexMinutePoint, error) {
	market := DetectMarket(code)
	var symbol string
	switch market {
	case MarketCN:
		prefix := stockMarketPrefix(code)
		if prefix == "" {
			return nil, newProviderError("tencent", "invalid stock code")
		}
		symbol = prefix + code
	case MarketHK:
		symbol = "hk" + code
	case MarketUS:
		symbol = "us" + code
	default:
		return nil, newProviderError("tencent", "unsupported market")
	}
	result := p.quoteClient.fetchTencentMinuteData(ctx, symbol, market)
	if len(result) == 0 {
		return nil, newProviderError("tencent", "empty result")
	}
	return result, nil
}

// FetchFundQuotes 根据基金代码列表获取实时估值数据。
func (p *TencentProvider) FetchFundQuotes(ctx context.Context, codes []string) (map[string]funddomain.FundItem, error) {
	symbols := make([]string, 0, len(codes))
	for _, code := range codes {
		if symbol, ok := listedFundSymbol(code); ok {
			symbols = append(symbols, symbol)
		}
	}
	if len(symbols) == 0 {
		return nil, newProviderError("tencent", "no valid fund codes")
	}
	result := p.fundClient.fetchTencentQuotes(ctx, symbols)
	if len(result) == 0 {
		return nil, newProviderError("tencent", "empty result")
	}
	return result, nil
}
