// Package ths 实现了同花顺数据源的 Provider。
package ths

import core "stock-predict-go/internal/infrastructure/providers"

// Provider 是 core.THSProvider 的类型别名，作为本包的公开入口。
type Provider = core.THSProvider

// New 创建一个新的同花顺 Provider 实例。
func New(quoteClient *core.IndexQuoteClient) *Provider {
	return core.NewTHSProvider(quoteClient)
}
