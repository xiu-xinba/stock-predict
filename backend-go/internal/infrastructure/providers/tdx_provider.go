package providers

import (
	"context"
	"fmt"
	"log/slog"

	marketdomain "stock-predict-go/internal/domain/market"

	"gitee.com/quant1x/gotdx"
)

// TDXProvider 实现了通达信数据源的 Provider 接口。
type TDXProvider struct {
	quoteClient *IndexQuoteClient
	logger      *slog.Logger
}

// NewTDXProvider 创建一个新的 TDXProvider 实例。
func NewTDXProvider(quoteClient *IndexQuoteClient, logger *slog.Logger) *TDXProvider {
	return &TDXProvider{quoteClient: quoteClient, logger: logger}
}

// Name 返回数据源的唯一标识名称。
func (p *TDXProvider) Name() string {
	return "tdx"
}

// Capabilities 返回通达信数据源支持的能力及其适用的市场。
func (p *TDXProvider) Capabilities() map[Capability][]Market {
	return map[Capability][]Market{
		CapIndexQuote:  {MarketCN},
		CapIndexMinute: {MarketCN},
		CapIndexKline:  {MarketCN},
	}
}

// Priority 返回指定能力和市场组合下的优先级，数值越小优先级越高。
// 由于 gotdx 库存在日历数据溢出导致 panic 的不稳定性，通达信优先级较低。
func (p *TDXProvider) Priority(cap Capability, market Market) int {
	switch {
	case cap == CapIndexQuote && market == MarketCN:
		return 5
	case cap == CapIndexMinute && market == MarketCN:
		return 5
	case cap == CapIndexKline && market == MarketCN:
		return 5
	default:
		return 99
	}
}

// HealthCheck 通过测试 gotdx 初始化检测数据源健康状态。
func (p *TDXProvider) HealthCheck(_ context.Context) error {
	api, err := safeGetTdxApi()
	if err != nil {
		return newHealthCheckError("tdx", fmt.Sprintf("gotdx init failed: %v", err))
	}
	if api == nil {
		return newHealthCheckError("tdx", "gotdx returned nil api")
	}
	return nil
}

// safeGetTdxApi 包装 gotdx.GetTdxApi() 调用，内置 panic 恢复机制。
// gotdx 库在初始化时可能因日历数据溢出而触发 panic。
func safeGetTdxApi() (api interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("gotdx panic: %v", r)
		}
	}()
	a := gotdx.GetTdxApi()
	if a == nil {
		return nil, fmt.Errorf("gotdx returned nil")
	}
	return a, nil
}

// FetchIndexQuotes 从通达信获取 A 股指数行情数据。
func (p *TDXProvider) FetchIndexQuotes(ctx context.Context, market Market) ([]marketdomain.MarketIndex, error) {
	if market != MarketCN {
		return nil, newProviderError("tdx", "unsupported market")
	}
	quotes := p.quoteClient.fetchCNIndexQuotesTDX(ctx)
	if len(quotes) == 0 {
		return nil, newProviderError("tdx", "empty result")
	}
	return quotes, nil
}

// FetchIndexMinute 从通达信获取 A 股指数分时数据。
func (p *TDXProvider) FetchIndexMinute(ctx context.Context, code string, market Market) ([]marketdomain.IndexMinutePoint, error) {
	if market != MarketCN {
		return nil, newProviderError("tdx", "unsupported market")
	}
	points := p.quoteClient.fetchCNIndexMinuteTDX(ctx, code)
	if len(points) == 0 {
		return nil, newProviderError("tdx", "empty result")
	}
	return points, nil
}

// FetchIndexKline 从通达信获取 A 股指数 K 线数据。
func (p *TDXProvider) FetchIndexKline(ctx context.Context, code string, market Market, count int) ([]marketdomain.IndexKlinePoint, error) {
	if market != MarketCN {
		return nil, newProviderError("tdx", "unsupported market")
	}
	points := p.quoteClient.fetchCNIndexKlineTDX(ctx, code, count)
	if len(points) == 0 {
		return nil, newProviderError("tdx", "empty result")
	}
	return points, nil
}
