// Package tdx 实现了通达信数据源的 Provider。
package tdx

import (
	"log/slog"

	core "stock-predict-go/internal/infrastructure/providers"
)

// Provider 是 core.TDXProvider 的类型别名，作为本包的公开入口。
type Provider = core.TDXProvider

// New 创建一个新的通达信 Provider 实例。
func New(quoteClient *core.IndexQuoteClient, logger *slog.Logger) *Provider {
	return core.NewTDXProvider(quoteClient, logger)
}
