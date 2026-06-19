// Package biying 实现了币赢 API 数据源的 Provider。
package biying

import (
	"log/slog"

	core "stock-predict-go/internal/infrastructure/providers"
)

// Provider 是 core.BiyingApiProvider 的类型别名，作为本包的公开入口。
type Provider = core.BiyingApiProvider

// New 创建一个新的币赢 API Provider 实例。
func New(baseURL, token string, logger *slog.Logger) *Provider {
	return core.NewBiyingApiProvider(baseURL, token, logger)
}
