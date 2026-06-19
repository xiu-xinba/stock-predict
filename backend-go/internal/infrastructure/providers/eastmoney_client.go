package providers

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"time"

	httpclient "stock-predict-go/internal/platform/httpclient"
)

// eastmoneyUserAgent 是访问东方财富 API 时使用的 User-Agent 标识。
const eastmoneyUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"

// EastmoneyFetchError 表示东方财富 HTTP 请求失败的错误。
type EastmoneyFetchError struct {
	URL    string
	Status int
	Err    error
}

// Error 返回东方财富请求失败的错误描述。
func (e EastmoneyFetchError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("eastmoney fetch %s failed: %v", e.URL, e.Err)
	}
	return fmt.Sprintf("eastmoney fetch %s failed with HTTP %d", e.URL, e.Status)
}

// EastmoneyClient 是东方财富 API 的 HTTP 客户端，内置请求频率限制和弹性重试机制。
type EastmoneyClient struct {
	client      *http.Client
	resilient   *ResilientHTTPClient
	minInterval time.Duration
	mu          sync.Mutex
	lastCall    time.Time
	now         func() time.Time
	sleep       func(time.Duration)
	jitter      func() time.Duration
}

// newEastmoneyClient 创建一个新的 EastmoneyClient 实例，配置默认的频率限制和弹性策略。
func newEastmoneyClient(client *http.Client) *EastmoneyClient {
	if client == nil {
		client = NewHTTPClient(HTTPClientConfig{})
	}
	return &EastmoneyClient{
		client: client,
		resilient: NewResilientHTTPClient(client, []SourcePolicy{{
			Source:          SourceEastmoney,
			UserAgent:       compliantUserAgent,
			Referer:         "https://quote.eastmoney.com/",
			Accept:          "application/json,text/plain,*/*",
			MinInterval:     0,
			CooldownOnLimit: 30 * time.Second,
			CooldownOnError: 5 * time.Second,
		}}),
		minInterval: time.Second,
		now:         time.Now,
		sleep:       time.Sleep,
		jitter: func() time.Duration {
			return time.Duration(100+rand.Intn(401)) * time.Millisecond
		},
	}
}

// Get 发送 GET 请求到指定 URL，使用默认的最大响应体大小限制。
func (c *EastmoneyClient) Get(ctx context.Context, rawURL string, referer string) ([]byte, error) {
	return c.GetWithLimit(ctx, rawURL, referer, MaxHTTPPayloadBytes)
}

// GetWithLimit 发送 GET 请求到指定 URL，并限制最大响应体字节数。
// 内置请求频率限制，确保两次请求之间满足最小间隔。
func (c *EastmoneyClient) GetWithLimit(ctx context.Context, rawURL string, referer string, maxBytes int) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.lastCall.IsZero() && c.minInterval > 0 {
		wait := c.minInterval - c.now().Sub(c.lastCall)
		if wait > 0 {
			if c.jitter != nil {
				wait += c.jitter()
			}
			c.sleep(wait)
		}
	}

	payload, err := c.get(ctx, rawURL, referer, maxBytes)
	c.lastCall = c.now()
	return payload, err
}

// get 执行实际的 HTTP GET 请求，失败时包装为 EastmoneyFetchError。
func (c *EastmoneyClient) get(ctx context.Context, rawURL string, referer string, maxBytes int) ([]byte, error) {
	payload, err := c.getViaGo(ctx, rawURL, referer, maxBytes)
	if err == nil {
		return payload, nil
	}
	return nil, EastmoneyFetchError{URL: rawURL, Err: err}
}

// getViaGo 使用 Go 原生 HTTP 客户端发送请求，通过弹性客户端处理重试和限流。
// 自动检测响应体编码，若非 UTF-8 则按 GBK 解码转换。
func (c *EastmoneyClient) getViaGo(ctx context.Context, rawURL string, referer string, maxBytes int) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	httpClient := c.resilient
	if httpClient == nil {
		httpClient = NewResilientHTTPClient(c.client, nil)
		c.resilient = httpClient
	}
	resp, err := httpClient.Do(ctx, SourceEastmoney, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	if maxBytes <= 0 {
		maxBytes = MaxHTTPPayloadBytes
	}
	payload, err := io.ReadAll(io.LimitReader(resp.Body, int64(maxBytes)))
	if err != nil {
		return nil, err
	}
	// 东方财富 push2 API 返回 UTF-8 编码的 JSON，
	// 但部分旧接口可能返回 GBK，使用 EnsureUTF8 自动检测编码：
	// 若数据已是有效 UTF-8 则保持不变，否则按 GBK 解码。
	decoded := httpclient.EnsureUTF8(payload)
	return decoded, nil
}
