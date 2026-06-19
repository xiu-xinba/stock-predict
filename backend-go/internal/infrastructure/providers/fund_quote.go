package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	funddomain "stock-predict-go/internal/domain/fund"
	httpclient "stock-predict-go/internal/platform/httpclient"
)

const (
	tencentQuoteURL    = "https://qt.gtimg.cn/q=%s"                     // 腾讯基金行情 API 地址
	eastmoneyFundGZURL = "https://fundgz.1234567.com.cn/js/%s.js?rt=%d" // 东方财富基金估值 API 地址
)

// fundQuoteProvider 基金报价刷新接口。
type fundQuoteProvider interface {
	RefreshQuotes(context.Context, []funddomain.FundItem) map[string]funddomain.FundItem
}

// FundQuoteClient 基金行情客户端，支持从腾讯和东方财富获取基金实时估值。
type FundQuoteClient struct {
	resilient *ResilientHTTPClient
	now       func() time.Time
	logger    *slog.Logger
	router    *ProviderRouter
}

// NewFundQuoteClient 创建新的基金行情客户端实例。
func NewFundQuoteClient(timeout time.Duration, logger *slog.Logger) *FundQuoteClient {
	if timeout <= 0 {
		timeout = HTTPClientTimeout
	}
	if logger == nil {
		logger = slog.Default()
	}
	client := NewHTTPClient(HTTPClientConfig{Timeout: timeout})
	return &FundQuoteClient{
		resilient: NewResilientHTTPClient(client, DefaultSourcePolicies()),
		now:       time.Now,
		logger:    logger,
	}
}

// SetRouter 注入数据源路由器。
func (c *FundQuoteClient) SetRouter(router *ProviderRouter) {
	c.router = router
}

// RefreshQuotes 批量刷新基金估值，货币基金直接使用收益率，其他基金从腾讯或东方财富获取实时估值。
func (c *FundQuoteClient) RefreshQuotes(ctx context.Context, funds []funddomain.FundItem) map[string]funddomain.FundItem {
	quotes := make(map[string]funddomain.FundItem, len(funds))
	listedSymbols := make([]string, 0, len(funds))
	for _, fund := range funds {
		if isMoneyFund(fund.FundType) {
			fund.QuoteSource = "money_fund_yield"
			quotes[fund.FundCode] = fund
			continue
		}
		if symbol, ok := listedFundSymbol(fund.FundCode); ok {
			listedSymbols = append(listedSymbols, symbol)
		}
	}
	if len(listedSymbols) > 0 {
		routerOK := false
		if c.router != nil {
			err := c.router.Fetch(ctx, CapFundQuote, MarketCN, func(ctx context.Context, p Provider) error {
				fp, ok := p.(FundQuoteProvider)
				if !ok {
					return fmt.Errorf("provider %s does not implement FundQuoteProvider", p.Name())
				}
				result, err := fp.FetchFundQuotes(ctx, listedSymbols)
				if err != nil {
					return err
				}
				for code, quote := range result {
					quotes[code] = quote
				}
				routerOK = true
				return nil
			})
			if err != nil {
				c.logger.Warn("router fetch for fund quotes failed, falling back to legacy", "error", err)
			}
		}
		if !routerOK {
			for code, quote := range c.fetchTencentQuotes(ctx, listedSymbols) {
				quotes[code] = quote
			}
		}
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, MaxFundGZConcurrency)
	for _, fund := range funds {
		fund := fund
		if _, ok := quotes[fund.FundCode]; ok {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}
			if quote, ok := c.fetchEastmoneyFundGZQuote(ctx, fund.FundCode); ok {
				mu.Lock()
				quotes[fund.FundCode] = quote
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return quotes
}

// fetchTencentQuotes 从腾讯行情 API 批量获取基金估值。
func (c *FundQuoteClient) fetchTencentQuotes(ctx context.Context, symbols []string) map[string]funddomain.FundItem {
	url := fmt.Sprintf(tencentQuoteURL, strings.Join(symbols, ","))
	if !isAllowedURL(url) {
		c.logger.Warn("URL not in whitelist, skipping request", "url", url)
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	resp, err := c.resilient.Do(ctx, SourceTencent, req)
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
	return parseTencentFundQuotes(payload)
}

// fetchEastmoneyFundGZQuote 从东方财富基金估值 API 获取单只基金估值。
func (c *FundQuoteClient) fetchEastmoneyFundGZQuote(ctx context.Context, code string) (funddomain.FundItem, bool) {
	url := fmt.Sprintf(eastmoneyFundGZURL, code, c.now().UnixMilli())
	if !isAllowedURL(url) {
		c.logger.Warn("URL not in whitelist, skipping request", "url", url)
		return funddomain.FundItem{}, false
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return funddomain.FundItem{}, false
	}
	req.Header.Set("Referer", "https://fund.eastmoney.com/"+code+".html")
	resp, err := c.resilient.Do(ctx, SourceEastmoney, req)
	if err != nil {
		return funddomain.FundItem{}, false
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return funddomain.FundItem{}, false
	}
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return funddomain.FundItem{}, false
	}
	return parseEastmoneyFundGZQuote(payload)
}

// parseEastmoneyFundGZQuote 解析东方财富基金估值 JSONP 响应。
func parseEastmoneyFundGZQuote(payload []byte) (funddomain.FundItem, bool) {
	text := strings.TrimSpace(strings.TrimPrefix(string(payload), "\ufeff"))
	start := strings.Index(text, "(")
	end := strings.LastIndex(text, ")")
	if start < 0 || end <= start {
		return funddomain.FundItem{}, false
	}
	var raw struct {
		FundCode string `json:"fundcode"`
		Name     string `json:"name"`
		NAVDate  string `json:"jzrq"`
		UnitNAV  string `json:"dwjz"`
		Estimate string `json:"gsz"`
		Change   string `json:"gszzl"`
		Time     string `json:"gztime"`
	}
	if err := json.Unmarshal([]byte(text[start+1:end]), &raw); err != nil {
		return funddomain.FundItem{}, false
	}
	code := strings.TrimSpace(raw.FundCode)
	if len(code) != 6 || !httpclient.IsAllDigits(code) {
		return funddomain.FundItem{}, false
	}
	latestNAV := httpclient.ParseQuoteFloat(raw.UnitNAV)
	estimatedNAV := httpclient.ParseQuoteFloat(raw.Estimate)
	if estimatedNAV == 0 {
		estimatedNAV = latestNAV
	}
	if estimatedNAV == 0 {
		return funddomain.FundItem{}, false
	}
	quoteTime := strings.TrimSpace(raw.Time)
	if quoteTime == "" {
		quoteTime = strings.TrimSpace(raw.NAVDate)
	}
	return funddomain.FundItem{
		FundCode:     code,
		FundName:     strings.TrimSpace(raw.Name),
		LatestNAV:    latestNAV,
		EstimatedNAV: estimatedNAV,
		ChangePct:    httpclient.ParseQuoteFloat(raw.Change),
		QuoteDate:    quoteTime,
		QuoteSource:  "eastmoney_fundgz",
	}, true
}

// parseTencentFundQuotes 解析腾讯行情 API 返回的基金估值数据。
func parseTencentFundQuotes(payload []byte) map[string]funddomain.FundItem {
	text := strings.TrimSpace(strings.TrimPrefix(string(payload), "\ufeff"))
	quotes := map[string]funddomain.FundItem{}
	for _, statement := range strings.Split(text, ";") {
		start := strings.Index(statement, "\"")
		end := strings.LastIndex(statement, "\"")
		if start < 0 || end <= start {
			continue
		}
		fields := strings.Split(statement[start+1:end], "~")
		if len(fields) <= 32 {
			continue
		}
		code := strings.TrimSpace(fields[2])
		if len(code) != 6 || !httpclient.IsAllDigits(code) {
			continue
		}
		price := httpclient.ParseQuoteFloat(fields[3])
		if price == 0 {
			continue
		}
		quotes[code] = funddomain.FundItem{
			FundCode:     code,
			FundName:     strings.TrimSpace(fields[1]),
			LatestNAV:    price,
			EstimatedNAV: price,
			ChangePct:    httpclient.ParseQuoteFloat(fields[32]),
			QuoteDate:    strings.TrimSpace(fields[30]),
			QuoteSource:  "tencent_quote",
		}
	}
	return quotes
}

// listedFundSymbol 将基金代码转换为腾讯行情 API 所需的上市代码格式。
func listedFundSymbol(code string) (string, bool) {
	switch {
	case len(code) == 6 && httpclient.IsAllDigits(code) && strings.HasPrefix(code, "5"):
		return "sh" + code, true
	case len(code) == 6 && httpclient.IsAllDigits(code) && strings.HasPrefix(code, "1"):
		return "sz" + code, true
	default:
		return "", false
	}
}

// isMoneyFund 判断基金类型是否为货币基金。
func isMoneyFund(fundType string) bool {
	t := strings.ToLower(fundType)
	return t == "货币型" || t == "货币市场型" || t == "money" || strings.Contains(t, "货币")
}
