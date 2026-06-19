package providers

import (
	"net/http"

	httpclient "stock-predict-go/internal/platform/httpclient"
)

// HTTPClientConfig HTTP 客户端配置的类型别名，引用 httpclient.Config。
type HTTPClientConfig = httpclient.Config

// ResilientHTTPClient 弹性 HTTP 客户端的类型别名，引用 httpclient.ResilientHTTPClient。
type ResilientHTTPClient = httpclient.ResilientHTTPClient

// SourcePolicy 数据源策略的类型别名，引用 httpclient.SourcePolicy。
type SourcePolicy = httpclient.SourcePolicy

const (
	// HTTPClientTimeout HTTP 客户端默认超时时间
	HTTPClientTimeout = httpclient.DefaultTimeout
	// HTTPDialTimeout 默认拨号超时时间
	HTTPDialTimeout = httpclient.DefaultDialTimeout
	// HTTPKeepAlive 默认 Keep-Alive 间隔
	HTTPKeepAlive = httpclient.DefaultKeepAlive
	// MaxHTTPPayloadBytes HTTP 响应体的最大字节数
	MaxHTTPPayloadBytes = httpclient.MaxPayloadBytes
	// MaxSyncPayloadBytes 同步响应体的最大字节数
	MaxSyncPayloadBytes = httpclient.MaxSyncPayloadBytes
	// MaxEastmoneyPayload 东方财富接口响应体的最大字节数
	MaxEastmoneyPayload = httpclient.MaxProviderPayload

	// SourceEastmoney 东方财富数据源标识
	SourceEastmoney = httpclient.SourceEastmoney
	// SourceTencent 腾讯数据源标识
	SourceTencent = httpclient.SourceTencent
	// SourceSina 新浪数据源标识
	SourceSina = httpclient.SourceSina
	// SourceTHS 同花顺数据源标识
	SourceTHS = httpclient.SourceTHS
	// SourceBiyingAPI 币赢数据源标识
	SourceBiyingAPI = httpclient.SourceBiyingAPI
	// SourceAKShare AKShare 数据源标识
	SourceAKShare = httpclient.SourceAKShare

	// compliantUserAgent 合规 User-Agent 字符串
	compliantUserAgent = httpclient.CompliantUserAgent
)

// NewHTTPClient 使用给定配置创建标准 HTTP 客户端。
func NewHTTPClient(cfg HTTPClientConfig) *http.Client {
	return httpclient.New(cfg)
}

// NewResilientHTTPClient 创建弹性 HTTP 客户端，包装标准客户端并应用数据源策略。
func NewResilientHTTPClient(client *http.Client, policies []SourcePolicy) *ResilientHTTPClient {
	return httpclient.NewResilientHTTPClient(client, policies)
}

// DefaultSourcePolicies 返回默认的数据源策略列表。
func DefaultSourcePolicies() []SourcePolicy {
	return httpclient.DefaultSourcePolicies()
}

// isAllowedURL 检查给定 URL 是否在允许列表中。
func isAllowedURL(rawURL string) bool {
	return httpclient.IsAllowedURL(rawURL)
}
