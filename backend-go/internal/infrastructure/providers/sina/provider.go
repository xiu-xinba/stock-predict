// Package sina 实现了新浪财经数据源的 Provider。
package sina

import core "stock-predict-go/internal/infrastructure/providers"

// Provider 是 core.SinaProvider 的类型别名，作为本包的公开入口。
type Provider = core.SinaProvider

// New 创建一个新的新浪财经 Provider 实例。
func New(quoteClient *core.IndexQuoteClient) *Provider {
	return core.NewSinaProvider(quoteClient)
}
