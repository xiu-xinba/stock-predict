package providers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	stockdomain "stock-predict-go/internal/domain/stock"
	httpclient "stock-predict-go/internal/platform/httpclient"
)

// StockQuoteClient 股票行情客户端，支持从腾讯获取实时行情并缓存。
type StockQuoteClient struct {
	client     *http.Client
	resilient  *ResilientHTTPClient
	quoteCache *DetailCache
	marketOpen func([]string) bool
	router     *ProviderRouter
}

// StockQuoteFreshness 行情数据新鲜度策略。
type StockQuoteFreshness string

const (
	StockQuoteFreshnessBalanced StockQuoteFreshness = "balanced" // 平衡模式，交易时段短缓存，盘后长缓存
	StockQuoteFreshnessRealtime StockQuoteFreshness = "realtime" // 实时模式，始终使用短缓存
)

// StockQuoteOptions 行情请求选项。
type StockQuoteOptions struct {
	Freshness StockQuoteFreshness // 新鲜度策略
}

// NewStockQuoteClient 创建新的股票行情客户端实例。
func NewStockQuoteClient(timeout time.Duration) *StockQuoteClient {
	if timeout <= 0 {
		timeout = HTTPClientTimeout
	}
	client := NewHTTPClient(HTTPClientConfig{Timeout: timeout})
	return &StockQuoteClient{
		client:     client,
		resilient:  NewResilientHTTPClient(client, nil),
		quoteCache: NewDetailCache(CacheMaxEntries, StockQuoteCacheTTL),
		marketOpen: anyStockMarketOpen,
	}
}

// SetRouter 注入数据源路由器。
func (c *StockQuoteClient) SetRouter(router *ProviderRouter) {
	c.router = router
}

// FetchQuotes 批量获取股票行情，使用平衡模式。
func (c *StockQuoteClient) FetchQuotes(ctx context.Context, codes []string) map[string]stockdomain.StockQuote {
	return c.FetchQuotesWithOptions(ctx, codes, StockQuoteOptions{Freshness: StockQuoteFreshnessBalanced})
}

// FetchQuotesWithOptions 批量获取股票行情，支持指定新鲜度策略。
func (c *StockQuoteClient) FetchQuotesWithOptions(ctx context.Context, codes []string, opts StockQuoteOptions) map[string]stockdomain.StockQuote {
	quotes := make(map[string]stockdomain.StockQuote, len(codes))
	if len(codes) == 0 {
		return quotes
	}

	freshness := normalizeStockQuoteFreshness(opts.Freshness)
	freshTTL := c.quoteFreshTTL(codes, freshness)
	missingCodes := make([]string, 0, len(codes))
	refreshCodes := make([]string, 0, len(codes))
	for _, code := range codes {
		if quote, ok := c.cachedQuote(code, freshTTL); ok {
			quotes[code] = quote
			continue
		}
		if freshness == StockQuoteFreshnessRealtime {
			if quote, ok := c.cachedQuote(code, StockQuoteRealtimeStaleTTL); ok {
				quotes[code] = quote
				refreshCodes = append(refreshCodes, code)
				continue
			}
		}
		missingCodes = append(missingCodes, code)
	}
	if len(refreshCodes) > 0 {
		c.refreshQuotesAsync(ctx, refreshCodes)
	}
	if len(missingCodes) == 0 {
		return quotes
	}

	result := c.fetchAndCacheQuotes(ctx, missingCodes)
	for k, v := range result {
		quotes[k] = v
	}
	for _, code := range missingCodes {
		if _, ok := quotes[code]; ok {
			continue
		}
		if c.quoteCache != nil {
			if cached, ok := c.quoteCache.GetStale(code); ok {
				if quote, ok := cached.(stockdomain.StockQuote); ok {
					quotes[code] = quote
				}
			}
		}
	}
	return quotes
}

func (c *StockQuoteClient) cachedQuote(code string, maxAge time.Duration) (stockdomain.StockQuote, bool) {
	if c.quoteCache == nil {
		return stockdomain.StockQuote{}, false
	}
	cached, ok := c.quoteCache.GetWithMaxAge(code, maxAge)
	if !ok {
		return stockdomain.StockQuote{}, false
	}
	quote, ok := cached.(stockdomain.StockQuote)
	return quote, ok
}

// fetchAndCacheQuotes 从远程获取行情数据并写入缓存。
func (c *StockQuoteClient) fetchAndCacheQuotes(ctx context.Context, codes []string) map[string]stockdomain.StockQuote {
	quotes := c.fetchQuotesFromProvider(ctx, codes)
	if c.quoteCache != nil {
		for k, v := range quotes {
			c.quoteCache.Set(k, v)
		}
	}
	return quotes
}

func (c *StockQuoteClient) refreshQuotesAsync(ctx context.Context, codes []string) {
	codes = uniqueStockCodes(codes)
	if len(codes) == 0 {
		return
	}
	refreshCtx := context.WithoutCancel(ctx)
	go c.fetchAndCacheQuotes(refreshCtx, codes)
}

// fetchQuotesFromProvider 从数据源获取股票行情，优先使用路由器，回退到腾讯源。
func (c *StockQuoteClient) fetchQuotesFromProvider(ctx context.Context, codes []string) map[string]stockdomain.StockQuote {
	quotes := make(map[string]stockdomain.StockQuote, len(codes))

	if c.router != nil {
		// Group codes by market for router-based fetching
		marketCodes := make(map[Market][]string)
		for _, code := range codes {
			market := DetectMarket(code)
			marketCodes[market] = append(marketCodes[market], code)
		}
		routerOK := true
		for market, marketCodeList := range marketCodes {
			symbols := make([]string, 0, len(marketCodeList))
			for _, code := range marketCodeList {
				prefix := stockMarketPrefix(code)
				if prefix != "" {
					symbols = append(symbols, prefix+code)
				}
			}
			if len(symbols) == 0 {
				continue
			}
			err := c.router.Fetch(ctx, CapStockQuote, market, func(ctx context.Context, p Provider) error {
				sp, ok := p.(StockQuoteProvider)
				if !ok {
					return fmt.Errorf("provider %s does not implement StockQuoteProvider", p.Name())
				}
				result, err := sp.FetchStockQuotes(ctx, symbols)
				if err != nil {
					return err
				}
				for k, v := range result {
					quotes[k] = v
				}
				return nil
			})
			if err != nil {
				routerOK = false
				break
			}
		}
		if routerOK && len(quotes) > 0 {
			return quotes
		}
		// Router failed, fall back to legacy
		quotes = make(map[string]stockdomain.StockQuote, len(codes))
	}

	symbols := make([]string, 0, len(codes))
	for _, code := range codes {
		market := stockMarketPrefix(code)
		if market != "" {
			symbol := market + code
			symbols = append(symbols, symbol)
		}
	}
	if len(symbols) == 0 {
		return quotes
	}

	sem := make(chan struct{}, StockQuoteConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i := 0; i < len(symbols); i += StockQuoteBatchSize {
		end := i + StockQuoteBatchSize
		if end > len(symbols) {
			end = len(symbols)
		}
		batch := symbols[i:end]
		wg.Add(1)
		go func(batch []string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			result := c.fetchTencentStockQuotes(ctx, batch)
			if result != nil {
				mu.Lock()
				for k, v := range result {
					quotes[k] = v
				}
				mu.Unlock()
			}
		}(batch)
	}
	wg.Wait()
	return quotes
}

func normalizeStockQuoteFreshness(freshness StockQuoteFreshness) StockQuoteFreshness {
	if freshness == StockQuoteFreshnessRealtime {
		return StockQuoteFreshnessRealtime
	}
	return StockQuoteFreshnessBalanced
}

func (c *StockQuoteClient) quoteFreshTTL(codes []string, freshness StockQuoteFreshness) time.Duration {
	if c.quoteCache != nil && c.quoteCache.TTL() <= 0 {
		return 0
	}
	open := true
	if c.marketOpen != nil {
		open = c.marketOpen(codes)
	}
	if !open {
		return StockQuoteIdleFreshTTL
	}
	if freshness == StockQuoteFreshnessRealtime {
		return StockQuoteRealtimeFreshTTL
	}
	return StockQuoteCacheTTL
}

func anyStockMarketOpen(codes []string) bool {
	for _, code := range codes {
		if IsMarketOpen(DetectMarket(code)) {
			return true
		}
	}
	return false
}

func uniqueStockCodes(codes []string) []string {
	seen := make(map[string]struct{}, len(codes))
	result := make([]string, 0, len(codes))
	for _, code := range codes {
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		result = append(result, code)
	}
	return result
}

// fetchTencentStockQuotes 从腾讯行情 API 批量获取股票行情。
func (c *StockQuoteClient) fetchTencentStockQuotes(ctx context.Context, symbols []string) map[string]stockdomain.StockQuote {
	url := fmt.Sprintf("https://qt.gtimg.cn/q=%s", strings.Join(symbols, ","))

	if !isAllowedURL(url) {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://gu.qq.com/")

	httpClient := c.resilient
	if httpClient == nil {
		httpClient = NewResilientHTTPClient(c.client, nil)
		c.resilient = httpClient
	}
	resp, err := httpClient.Do(ctx, SourceTencent, req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil
	}
	payload, err := io.ReadAll(io.LimitReader(resp.Body, int64(MaxHTTPPayloadBytes)))
	if err != nil {
		return nil
	}
	return parseTencentStockQuotes(payload)
}

// parseTencentStockQuotes 解析腾讯行情 API 返回的股票行情数据。
func parseTencentStockQuotes(payload []byte) map[string]stockdomain.StockQuote {
	text := strings.TrimSpace(strings.TrimPrefix(string(payload), "\ufeff"))
	quotes := map[string]stockdomain.StockQuote{}
	const minFields = 39
	for _, statement := range strings.Split(text, ";") {
		start := strings.Index(statement, "\"")
		end := strings.LastIndex(statement, "\"")
		if start < 0 || end <= start {
			continue
		}
		fields := strings.Split(statement[start+1:end], "~")
		if len(fields) < minFields {
			continue
		}
		code := strings.TrimSpace(fields[2])
		if len(code) != 6 || !httpclient.IsAllDigits(code) {
			continue
		}
		name := strings.TrimSpace(fields[1])
		if name == "" {
			continue
		}
		priceStr := strings.TrimSpace(fields[3])
		if priceStr == "" {
			continue
		}
		price := httpclient.ParseQuoteFloat(priceStr)
		if price == 0 {
			continue
		}
		changePctStr := strings.TrimSpace(fields[32])
		if changePctStr == "" {
			continue
		}
		quotes[code] = stockdomain.StockQuote{
			Price:        price,
			Open:         httpclient.ParseQuoteFloat(fields[5]),
			High:         httpclient.ParseQuoteFloat(fields[33]),
			Low:          httpclient.ParseQuoteFloat(fields[34]),
			PrevClose:    httpclient.ParseQuoteFloat(fields[4]),
			Volume:       httpclient.ParseQuoteFloat(fields[6]),
			Amount:       httpclient.ParseQuoteFloat(fields[37]),
			TurnoverRate: httpclient.ParseQuoteFloat(fields[38]),
			ChangePct:    httpclient.ParseQuoteFloat(changePctStr),
			ChangeAmt:    httpclient.ParseQuoteFloat(fields[31]),
			BidPrice:     httpclient.ParseQuoteFloat(fields[9]),
			AskPrice:     httpclient.ParseQuoteFloat(fields[18]),
			QuoteTime:    strings.TrimSpace(fields[30]),
		}
	}
	return quotes
}

// stockMarketPrefix 根据股票代码返回市场前缀（sh/sz）。
func stockMarketPrefix(code string) string {
	if len(code) == 0 {
		return ""
	}
	// US stocks: alphabetic codes like AAPL, MSFT
	if code[0] >= 'A' && code[0] <= 'Z' {
		return "us"
	}
	// HK stocks: 5-digit numeric codes like 00700, 09988
	if len(code) == 5 && httpclient.IsAllDigits(code) {
		return "hk"
	}
	// A stocks: 6-digit numeric codes
	if len(code) != 6 || !httpclient.IsAllDigits(code) {
		return ""
	}
	switch {
	case strings.HasPrefix(code, "6"), strings.HasPrefix(code, "9"):
		return "sh"
	case strings.HasPrefix(code, "0"), strings.HasPrefix(code, "3"):
		return "sz"
	case strings.HasPrefix(code, "8"), strings.HasPrefix(code, "4"):
		return "bj"
	default:
		return ""
	}
}
