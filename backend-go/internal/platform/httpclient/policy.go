// Package httpclient 提供了带重试、熔断和限流策略的 HTTP 客户端封装。
package httpclient

import (
	"net/url"
	"strings"
	"time"
)

const (
	SourceEastmoney = "eastmoney" // 东方财富数据源标识
	SourceTencent   = "tencent"   // 腾讯行情数据源标识
	SourceSina      = "sina"      // 新浪财经数据源标识
	SourceTHS       = "ths"       // 同花顺数据源标识
	SourceBiyingAPI = "biyingapi" // 必应 API 数据源标识
	SourceAKShare   = "akshare"   // AKShare 数据源标识
)

// CompliantUserAgent 是合规的市场数据客户端 User-Agent 标识。
const CompliantUserAgent = "StockPredict/1.0 (compliant market data client)"

// SourcePolicy 定义单个数据源的访问策略，包括请求头、限速和冷却时间。
type SourcePolicy struct {
	Source          string        // 数据源标识，对应 Source* 常量
	UserAgent       string        // 请求使用的 User-Agent 头
	Referer         string        // 请求使用的 Referer 头
	Accept          string        // 请求使用的 Accept 头
	MinInterval     time.Duration // 同一数据源两次请求的最小间隔
	CooldownOnLimit time.Duration // 触发限流（429/403/503）后的冷却时间
	CooldownOnError time.Duration // 请求出错后的冷却时间
}

// DefaultSourcePolicies 返回所有内置数据源的默认访问策略列表。
func DefaultSourcePolicies() []SourcePolicy {
	return []SourcePolicy{
		{
			Source:          SourceEastmoney,
			UserAgent:       CompliantUserAgent,
			Referer:         "https://quote.eastmoney.com/",
			Accept:          "application/json,text/plain,*/*",
			MinInterval:     time.Second,
			CooldownOnLimit: 30 * time.Second,
			CooldownOnError: 5 * time.Second,
		},
		{
			Source:          SourceTencent,
			UserAgent:       CompliantUserAgent,
			Referer:         "https://gu.qq.com/",
			Accept:          "application/json,text/plain,*/*",
			MinInterval:     500 * time.Millisecond,
			CooldownOnLimit: 20 * time.Second,
			CooldownOnError: 5 * time.Second,
		},
		{
			Source:          SourceSina,
			UserAgent:       CompliantUserAgent,
			Referer:         "https://finance.sina.com.cn/",
			Accept:          "application/json,text/plain,*/*",
			MinInterval:     time.Second,
			CooldownOnLimit: 30 * time.Second,
			CooldownOnError: 5 * time.Second,
		},
		{
			Source:          SourceTHS,
			UserAgent:       CompliantUserAgent,
			Referer:         "https://d.10jqka.com.cn/",
			Accept:          "*/*",
			MinInterval:     1500 * time.Millisecond,
			CooldownOnLimit: 30 * time.Second,
			CooldownOnError: 5 * time.Second,
		},
		{
			Source:          SourceBiyingAPI,
			UserAgent:       CompliantUserAgent,
			Referer:         "https://api.biyingapi.com/",
			Accept:          "application/json",
			MinInterval:     time.Second,
			CooldownOnLimit: 30 * time.Second,
			CooldownOnError: 5 * time.Second,
		},
		{
			Source:          SourceAKShare,
			UserAgent:       CompliantUserAgent,
			Referer:         "http://localhost:8900/",
			Accept:          "application/json",
			MinInterval:     500 * time.Millisecond,
			CooldownOnLimit: 10 * time.Second,
			CooldownOnError: 5 * time.Second,
		},
	}
}

// IsAllowedURL 检查给定 URL 是否属于允许访问的白名单域名，
// 仅允许 HTTPS 协议或回环地址的 HTTP 协议。
func IsAllowedURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if u.Scheme != "https" {
		isLoopback := host == "localhost" || host == "127.0.0.1" || host == "::1"
		if u.Scheme != "http" || !isLoopback {
			return false
		}
	}
	for _, suffix := range []string{".eastmoney.com", ".qq.com", ".biyingapi.com", ".10jqka.com.cn"} {
		if strings.HasSuffix(host, suffix) || host == strings.TrimPrefix(suffix, ".") {
			return true
		}
	}
	for _, exact := range []string{"push2.eastmoney.com", "push2his.eastmoney.com", "qt.gtimg.cn", "web.ifzq.gtimg.cn", "fundgz.1234567.com.cn", "money.finance.sina.com.cn", "vip.stock.finance.sina.com.cn", "stock.finance.sina.com.cn", "hq.sinajs.cn"} {
		if host == exact {
			return true
		}
	}
	if strings.HasSuffix(u.Host, ".biyingapi.com") {
		return true
	}
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}
	return false
}
