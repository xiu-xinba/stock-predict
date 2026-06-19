// Package tencent 实现了腾讯行情数据源的 Provider。
package tencent

import core "stock-predict-go/internal/infrastructure/providers"

// Provider 是 core.TencentProvider 的类型别名，作为本包的公开入口。
type Provider = core.TencentProvider

// New 创建一个新的腾讯行情 Provider 实例。
func New(
	quoteClient *core.IndexQuoteClient,
	stockClient *core.StockQuoteClient,
	fundClient *core.FundQuoteClient,
) *Provider {
	return core.NewTencentProvider(quoteClient, stockClient, fundClient)
}
