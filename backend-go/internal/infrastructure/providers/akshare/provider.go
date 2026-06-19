// Package akshare 实现了 AKShare 数据源的 Provider。
package akshare

import (
	"log/slog"

	core "stock-predict-go/internal/infrastructure/providers"
)

// Provider 是 core.AKShareProvider 的类型别名，作为本包的公开入口。
type Provider = core.AKShareProvider

// New 创建一个新的 AKShare Provider 实例。
func New(baseURL, token string, logger *slog.Logger) *Provider {
	return core.NewAKShareProviderWithToken(baseURL, token, logger)
}
