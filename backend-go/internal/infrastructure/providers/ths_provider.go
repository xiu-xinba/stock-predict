package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	marketdomain "stock-predict-go/internal/domain/market"
	stockdomain "stock-predict-go/internal/domain/stock"
	httpclient "stock-predict-go/internal/platform/httpclient"
)

const (
	// thsKlineBaseURL 是同花顺 K 线和分时数据的 API 基础地址（JSONP 格式）
	thsKlineBaseURL = "https://d.10jqka.com.cn/v6/line/"
	// thsPeriodDaily 表示日 K 线周期
	thsPeriodDaily = "01"
	// thsPeriodWeekly 表示周 K 线周期
	thsPeriodWeekly = "11"
	// thsPeriodMonthly 表示月 K 线周期
	thsPeriodMonthly = "21"
)

// thsCodePrefix 根据代码类型返回同花顺 API 的代码前缀。
// hs_ = 沪深个股，zs_ = 指数
func thsCodePrefix(code string) string {
	if isCNIndex(code) {
		return "zs_"
	}
	return "hs_"
}

// thsIndexCodeMap 将标准 A 股指数代码映射为同花顺 API 使用的代码格式。
// 例如上证指数 000001 → 1A0001。
var thsIndexCodeMap = map[string]string{
	"000001": "1A0001",
	"399001": "399001",
	"399006": "399006",
}

// thsIndexCode 将标准指数代码转换为同花顺 API 格式，若无映射则原样返回。
func thsIndexCode(code string) string {
	if mapped, ok := thsIndexCodeMap[code]; ok {
		return mapped
	}
	return code
}

// THSProvider 实现了同花顺数据源的 Provider 接口。
type THSProvider struct {
	quoteClient *IndexQuoteClient
	client      *http.Client
	resilient   *ResilientHTTPClient
	logger      *slog.Logger
}

// NewTHSProvider 创建一个新的 THSProvider 实例。
func NewTHSProvider(quoteClient *IndexQuoteClient) *THSProvider {
	policies := DefaultSourcePolicies()
	return &THSProvider{
		quoteClient: quoteClient,
		client:      NewHTTPClient(HTTPClientConfig{}),
		resilient:   NewResilientHTTPClient(NewHTTPClient(HTTPClientConfig{}), policies),
		logger:      slog.Default(),
	}
}

// Name 返回数据源的唯一标识名称。
func (p *THSProvider) Name() string {
	return "ths"
}

// Capabilities 返回同花顺数据源支持的能力及其适用的市场。
func (p *THSProvider) Capabilities() map[Capability][]Market {
	return map[Capability][]Market{
		CapIndexKline: {MarketCN},
		CapIndexQuote: {MarketCN},
	}
}

// Priority 返回指定能力和市场组合下的优先级，数值越小优先级越高。
func (p *THSProvider) Priority(cap Capability, market Market) int {
	key := string(cap) + ":" + string(market)
	priorities := map[string]int{
		"index_kline:cn": 2,
		"index_quote:cn": 5,
	}
	if pr, ok := priorities[key]; ok {
		return pr
	}
	return 99
}

// HealthCheck 通过请求同花顺 K 线接口检测数据源健康状态。
func (p *THSProvider) HealthCheck(ctx context.Context) error {
	url := thsKlineBaseURL + "zs_" + thsIndexCode("399001") + "/" + thsPeriodDaily + "/last.js"
	if !isAllowedURL(url) {
		return newHealthCheckError("ths", "URL not in whitelist")
	}
	_, err := p.fetchTHSJSONP(ctx, url)
	if err != nil {
		return newHealthCheckError("ths", fmt.Sprintf("health check failed: %v", err))
	}
	return nil
}

// FetchIndexKline 从同花顺获取 A 股指数 K 线数据。
func (p *THSProvider) FetchIndexKline(ctx context.Context, code string, market Market, count int) ([]marketdomain.IndexKlinePoint, error) {
	if market != MarketCN {
		return nil, newProviderError("ths", "unsupported market")
	}
	if !isCNIndex(code) {
		return nil, newProviderError("ths", "unsupported index code")
	}

	prefix := thsCodePrefix(code)
	thsCode := thsIndexCode(code)
	url := thsKlineBaseURL + prefix + thsCode + "/" + thsPeriodDaily + "/last.js"
	if !isAllowedURL(url) {
		return nil, newProviderError("ths", "URL not in whitelist")
	}

	data, err := p.fetchTHSJSONP(ctx, url)
	if err != nil {
		return nil, newProviderError("ths", fmt.Sprintf("fetch index kline: %v", err))
	}

	points := parseTHSKlineData(data)
	if len(points) == 0 {
		return nil, newProviderError("ths", "empty result")
	}

	// Return the last `count` points
	if count > 0 && len(points) > count {
		points = points[len(points)-count:]
	}

	return normalizeIndexKlinePoints(points), nil
}

// FetchStockQuotes 从同花顺获取股票实时行情数据。
func (p *THSProvider) FetchStockQuotes(ctx context.Context, symbols []string) (map[string]stockdomain.StockQuote, error) {
	if len(symbols) == 0 {
		return nil, newProviderError("ths", "no symbols")
	}

	quotes := make(map[string]stockdomain.StockQuote, len(symbols))
	for _, symbol := range symbols {
		code := plainStockCode(symbol)
		if code == "" {
			continue
		}
		// THS does not support BJ (北交所) stocks
		if len(symbol) == 8 && strings.HasPrefix(symbol, "bj") {
			continue
		}
		prefix := thsCodePrefix(code)
		thsCode := thsIndexCode(code)
		url := thsKlineBaseURL + prefix + thsCode + "/" + thsPeriodDaily + "/last.js"
		if !isAllowedURL(url) {
			continue
		}

		data, err := p.fetchTHSJSONP(ctx, url)
		if err != nil {
			continue
		}

		points := parseTHSKlineData(data)
		if len(points) < 2 {
			continue
		}

		latest := points[len(points)-1]
		prev := points[len(points)-2]
		change := latest.Close - prev.Close
		var changePct float64
		if prev.Close > 0 {
			changePct = change / prev.Close * 100
		}

		quotes[code] = stockdomain.StockQuote{
			Price:     latest.Close,
			Open:      latest.Open,
			High:      latest.High,
			Low:       latest.Low,
			PrevClose: prev.Close,
			Volume:    float64(latest.Volume),
			Amount:    latest.Amount,
			ChangePct: changePct,
			ChangeAmt: change,
			QuoteTime: latest.Date,
		}
	}

	if len(quotes) == 0 {
		return nil, newProviderError("ths", "empty result")
	}
	return quotes, nil
}

// FetchIndexQuotes 从同花顺获取 A 股指数行情数据。
func (p *THSProvider) FetchIndexQuotes(ctx context.Context, market Market) ([]marketdomain.MarketIndex, error) {
	if market != MarketCN {
		return nil, newProviderError("ths", "unsupported market")
	}

	var result []marketdomain.MarketIndex
	for _, code := range cnIndexCodes {
		prefix := thsCodePrefix(code)
		thsCode := thsIndexCode(code)
		url := thsKlineBaseURL + prefix + thsCode + "/" + thsPeriodDaily + "/last.js"
		if !isAllowedURL(url) {
			p.logger.Warn("ths: URL not allowed", "code", code, "url", url)
			continue
		}

		data, err := p.fetchTHSJSONP(ctx, url)
		if err != nil {
			p.logger.Warn("ths: fetchTHSJSONP failed", "code", code, "error", err)
			continue
		}

		points := parseTHSKlineData(data)
		if len(points) < 2 {
			p.logger.Warn("ths: not enough kline points", "code", code, "points", len(points))
			continue
		}

		latest := points[len(points)-1]
		prev := points[len(points)-2]
		change := latest.Close - prev.Close
		var changePct float64
		if prev.Close > 0 {
			changePct = change / prev.Close * 100
		}

		result = append(result, marketdomain.MarketIndex{
			Code:      code,
			Name:      cnIndexNames[code],
			Market:    cnIndexMarkets[code],
			Value:     latest.Close,
			Change:    change,
			ChangePct: changePct,
			High:      latest.High,
			Low:       latest.Low,
			PrevClose: prev.Close,
			Open:      latest.Open,
			Volume:    float64(latest.Volume),
		})
	}

	if len(result) == 0 {
		return nil, newProviderError("ths", "empty result")
	}
	return normalizeMarketIndices(result), nil
}

// fetchTHSJSONP 请求同花顺 JSONP 接口并提取其中的 JSON 数据。
// 同花顺 API 返回 JSONP 格式：quotebridge_v6_line_{market}_{code}_{period}_{page}({...})
func (p *THSProvider) fetchTHSJSONP(ctx context.Context, url string) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Referer", "https://d.10jqka.com.cn/")
	req.Header.Set("Accept", "*/*")

	resp, err := p.resilient.Do(ctx, SourceTHS, req)
	if err != nil {
		p.logger.Warn("ths: HTTP request failed", "url", url, "error", err)
		return nil, fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		p.logger.Warn("ths: HTTP status error", "url", url, "status", resp.StatusCode)
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	payload, err := io.ReadAll(io.LimitReader(resp.Body, int64(MaxHTTPPayloadBytes)))
	if err != nil {
		p.logger.Warn("ths: read response failed", "url", url, "error", err)
		return nil, fmt.Errorf("read response: %w", err)
	}

	// Extract JSON from JSONP callback: callbackName({...})
	text := string(payload)
	idx := strings.Index(text, "(")
	if idx < 0 {
		p.logger.Warn("ths: no JSONP callback", "url", url, "response_len", len(text))
		return nil, fmt.Errorf("no JSONP callback in response")
	}
	endIdx := strings.LastIndex(text, ")")
	if endIdx <= idx {
		p.logger.Warn("ths: malformed JSONP", "url", url)
		return nil, fmt.Errorf("malformed JSONP response")
	}
	jsonStr := text[idx+1 : endIdx]

	var raw json.RawMessage
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		p.logger.Warn("ths: JSON parse failed", "url", url, "error", err)
		return nil, fmt.Errorf("parse JSON: %w", err)
	}
	return raw, nil
}

// thsKlineResponse 表示同花顺 K 线 JSONP 响应结构。
type thsKlineResponse struct {
	Total int    `json:"total"`
	Start int64  `json:"start"`
	Name  string `json:"name"`
	Data  string `json:"data"`
}

// parseTHSKlineData 解析同花顺响应中以分号分隔的 K 线数据。
// 数据格式："20260605,3368.07,3396.27,3401.68,3350.52,4216728,46032700000,0.78,,,0"
func parseTHSKlineData(raw json.RawMessage) []marketdomain.IndexKlinePoint {
	var resp thsKlineResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil
	}
	if resp.Data == "" {
		return nil
	}

	entries := strings.Split(resp.Data, ";")
	points := make([]marketdomain.IndexKlinePoint, 0, len(entries))
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		fields := strings.Split(entry, ",")
		if len(fields) < 7 {
			continue
		}

		date := formatTHSDate(fields[0])
		if date == "" {
			continue
		}

		open := httpclient.ParseQuoteFloat(fields[1])
		close_ := httpclient.ParseQuoteFloat(fields[2])
		high := httpclient.ParseQuoteFloat(fields[3])
		low := httpclient.ParseQuoteFloat(fields[4])
		volume := int64(httpclient.ParseQuoteFloat(fields[5]))
		amount := httpclient.ParseQuoteFloat(fields[6])

		points = append(points, marketdomain.IndexKlinePoint{
			Date:   date,
			Open:   open,
			Close:  close_,
			High:   high,
			Low:    low,
			Volume: volume,
			Amount: amount,
		})
	}
	return points
}

// formatTHSDate 将同花顺日期格式 "20260605" 转换为 ISO 格式 "2026-06-05"。
func formatTHSDate(dateStr string) string {
	if len(dateStr) != 8 {
		return ""
	}
	return dateStr[:4] + "-" + dateStr[4:6] + "-" + dateStr[6:8]
}
