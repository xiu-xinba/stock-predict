// Package httpclient 提供了带重试、熔断和限流策略的 HTTP 客户端封装。
package httpclient

import (
	"context"
	"net"
	"net/http"
	"time"
)

const (
	DefaultTimeout      = 8 * time.Second  // 默认 HTTP 请求超时时间
	DefaultDialTimeout  = 10 * time.Second // 默认 TCP 拨号超时时间
	DefaultKeepAlive    = 30 * time.Second // 默认 TCP Keep-Alive 间隔
	MaxPayloadBytes     = 2 << 20          // 普通请求最大响应体大小（2 MB）
	MaxSyncPayloadBytes = 5 << 20          // 同步请求最大响应体大小（5 MB）
	MaxProviderPayload  = 50 << 20         // 数据源请求最大响应体大小（50 MB）
)

// Config 定义 HTTP 客户端的配置参数。
type Config struct {
	Timeout      time.Duration // 请求总超时时间
	DialTimeout  time.Duration // TCP 拨号超时时间
	KeepAlive    time.Duration // TCP Keep-Alive 间隔
	MaxRedirects int           // 最大重定向次数，0 表示不限制
}

// New 根据配置创建一个新的 http.Client，使用 IPv4 拨号并启用 HTTP/2。
// 未指定的配置项将使用默认值。
func New(cfg Config) *http.Client {
	if cfg.Timeout <= 0 {
		cfg.Timeout = DefaultTimeout
	}
	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = DefaultDialTimeout
	}
	if cfg.KeepAlive <= 0 {
		cfg.KeepAlive = DefaultKeepAlive
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			dialer := net.Dialer{
				Timeout:   cfg.DialTimeout,
				KeepAlive: cfg.KeepAlive,
			}
			return dialer.DialContext(ctx, "tcp4", addr)
		},
		ForceAttemptHTTP2: true,
	}
	client := &http.Client{
		Timeout:   cfg.Timeout,
		Transport: transport,
	}
	if cfg.MaxRedirects > 0 {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if len(via) >= cfg.MaxRedirects {
				return http.ErrUseLastResponse
			}
			return nil
		}
	}
	return client
}
