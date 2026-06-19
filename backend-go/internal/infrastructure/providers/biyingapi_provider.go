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

	marketdomain "stock-predict-go/internal/domain/market"
	stockdomain "stock-predict-go/internal/domain/stock"
)

// BiyingApiProvider 实现了币赢 API 数据源的 Provider 接口。
type BiyingApiProvider struct {
	client    *http.Client
	resilient *ResilientHTTPClient
	baseURL   string
	token     string
	logger    *slog.Logger
}

// NewBiyingApiProvider 创建一个新的 BiyingApiProvider 实例。
func NewBiyingApiProvider(baseURL, token string, logger *slog.Logger) *BiyingApiProvider {
	client := NewHTTPClient(HTTPClientConfig{})
	return &BiyingApiProvider{
		client:    client,
		resilient: NewResilientHTTPClient(client, DefaultSourcePolicies()),
		baseURL:   baseURL,
		token:     token,
		logger:    logger,
	}
}

// Name 返回数据源的唯一标识名称。
func (p *BiyingApiProvider) Name() string { return "biyingapi" }

// Capabilities 返回币赢 API 数据源支持的能力及其适用的市场。
func (p *BiyingApiProvider) Capabilities() map[Capability][]Market {
	return map[Capability][]Market{
		CapIndexQuote:  {MarketCN},
		CapIndexMinute: {MarketCN},
	}
}

// Priority 返回指定能力和市场组合下的优先级，固定为 4（深度后备）。
func (p *BiyingApiProvider) Priority(cap Capability, market Market) int {
	return 4 // deep fallback for all capabilities
}

// HealthCheck 通过请求单个轻量级指数行情接口检测数据源连通性和 Token 有效性。
func (p *BiyingApiProvider) HealthCheck(ctx context.Context) error {
	// Use a single lightweight index quote request to verify connectivity
	// and token validity, instead of FetchIndexQuotes which may be affected
	// by unrelated stock quote batch-size failures recorded by the router.
	reqCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()

	path := fmt.Sprintf("/hsindex/real/time/000001.SH/%s", p.token)
	url := p.baseURL + path
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("biyingapi health check failed: %w", err)
	}
	resp, err := p.resilient.Do(reqCtx, SourceBiyingAPI, req)
	if err != nil {
		return fmt.Errorf("biyingapi health check failed: %w", p.safeError(err))
	}
	defer resp.Body.Close()
	body, err := readBiyingBody(resp)
	if err != nil {
		return fmt.Errorf("biyingapi health check failed: %w", err)
	}
	// Verify the response contains valid data (has a price field)
	var result biyingIndexRealTimeResp
	if err := json.Unmarshal(body, &result); err != nil || result.P <= 0 {
		return fmt.Errorf("biyingapi health check failed: invalid response")
	}
	return nil
}

// fetchJSON 通过 URL 查询参数传递 Token 获取 JSON 数据（旧版接口辅助方法）。
func (p *BiyingApiProvider) fetchJSON(ctx context.Context, path string, result any) error {
	url := fmt.Sprintf("%s%s?token=%s", p.baseURL, path, p.token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := p.resilient.Do(ctx, SourceBiyingAPI, req)
	if err != nil {
		return p.safeError(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(MaxHTTPPayloadBytes)))
	if err != nil {
		return err
	}
	return json.Unmarshal(body, result)
}

// fetchBiyingJSON 通过路径段嵌入授权密钥的方式获取 JSON 数据。
// 例如请求路径格式为 /hsindex/real/time/000001/{licence}。
func (p *BiyingApiProvider) fetchBiyingJSON(ctx context.Context, path string, result any) error {
	url := fmt.Sprintf("%s%s/%s", p.baseURL, path, p.token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := p.resilient.Do(ctx, SourceBiyingAPI, req)
	if err != nil {
		return p.safeError(err)
	}
	defer resp.Body.Close()
	body, err := readBiyingBody(resp)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, result)
}

// biyingAPIErrorResp 表示币赢 API 的错误响应结构。
type biyingAPIErrorResp struct {
	Code    int    `json:"code"`
	Msg     string `json:"msg"`
	Message string `json:"message"`
	Error   string `json:"error"`
}

// readBiyingBody 读取币赢 API 响应体，检查 HTTP 状态码和 API 错误码。
func readBiyingBody(resp *http.Response) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(MaxHTTPPayloadBytes)))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, newProviderError("biyingapi", fmt.Sprintf("status %d: %s", resp.StatusCode, truncateHTTPBody(body)))
	}
	var apiErr biyingAPIErrorResp
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Code != 0 {
		msg := firstNonEmpty(apiErr.Message, apiErr.Msg, apiErr.Error, "request failed")
		return nil, newProviderError("biyingapi", fmt.Sprintf("code %d: %s", apiErr.Code, msg))
	}
	return body, nil
}

// truncateHTTPBody 截断 HTTP 响应体内容，限制最大长度为 200 字符。
func truncateHTTPBody(body []byte) string {
	const limit = 200
	text := strings.TrimSpace(string(body))
	if len(text) <= limit {
		return text
	}
	return text[:limit] + "..."
}

// firstNonEmpty 返回参数列表中第一个非空字符串。
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// safeError 将错误信息中的 Token 替换为 [REDACTED]，防止敏感信息泄露。
func (p *BiyingApiProvider) safeError(err error) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	if p.token != "" {
		message = strings.ReplaceAll(message, p.token, "[REDACTED]")
	}
	return fmt.Errorf("%s", message)
}

// biyingCNIndexCodes 列出了标准 A 股指数代码及其币赢 API 市场后缀。
var biyingCNIndexCodes = []struct {
	code   string
	suffix string // SH or SZ
}{
	{"000001", "SH"},
	{"399001", "SZ"},
	{"399006", "SZ"},
}

// biyingIndexCode 将指数代码和市场后缀组合为币赢 API 格式 "code.Market"。
func biyingIndexCode(code, suffix string) string {
	return code + "." + suffix
}

// biyingIndexRealTimeResp 映射币赢 API 指数实时行情响应字段。
type biyingIndexRealTimeResp struct {
	P   float64 `json:"p"`   // price
	O   float64 `json:"o"`   // open
	H   float64 `json:"h"`   // high
	L   float64 `json:"l"`   // low
	YC  float64 `json:"yc"`  // prev close
	CJE float64 `json:"cje"` // amount
	V   float64 `json:"v"`   // volume
	PV  float64 `json:"pv"`  // change
	UD  float64 `json:"ud"`  // change pct (as decimal, e.g. 0.0123 = 1.23%)
	PC  float64 `json:"pc"`  // amplitude
	ZF  float64 `json:"zf"`  // amplitude (alternate field)
	T   string  `json:"t"`   // time
}

// FetchIndexQuotes 从币赢 API 获取 A 股指数行情数据。
func (p *BiyingApiProvider) FetchIndexQuotes(ctx context.Context, market Market) ([]marketdomain.MarketIndex, error) {
	if p.baseURL == "" || p.token == "" {
		return nil, newProviderError("biyingapi", "baseURL or token not configured")
	}
	if market != MarketCN {
		return nil, newProviderError("biyingapi", "only CN market supported")
	}

	// Use an independent timeout so that the caller's short deadline
	// (e.g. RaceTimeout 3s) does not cancel in-flight requests.
	reqCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()

	var results []marketdomain.MarketIndex
	for _, idx := range biyingCNIndexCodes {
		path := fmt.Sprintf("/hsindex/real/time/%s", biyingIndexCode(idx.code, idx.suffix))
		var resp biyingIndexRealTimeResp
		if err := p.fetchBiyingJSON(reqCtx, path, &resp); err != nil {
			p.logger.Warn("biyingapi: failed to fetch index quote", "code", idx.code, "error", err)
			continue
		}

		changePct := resp.UD * 100 // BiyingAPI returns decimal, convert to percentage
		name := cnIndexNames[idx.code]
		results = append(results, marketdomain.MarketIndex{
			Code:       idx.code,
			Name:       name,
			Market:     string(market),
			Value:      resp.P,
			Change:     resp.PV,
			ChangePct:  changePct,
			High:       resp.H,
			Low:        resp.L,
			PrevClose:  resp.YC,
			Open:       resp.O,
			Volume:     resp.V,
			UpdateTime: resp.T,
			DataSource: "biyingapi",
		})
	}

	if len(results) == 0 {
		return nil, newProviderError("biyingapi", "all index quote requests failed")
	}
	return results, nil
}

// biyingIndexMinuteResp 映射币赢 API 指数历史分时响应中的单条数据。
type biyingIndexMinuteResp struct {
	T string  `json:"t"` // time
	O float64 `json:"o"` // open
	H float64 `json:"h"` // high
	L float64 `json:"l"` // low
	C float64 `json:"c"` // close
	V int64   `json:"v"` // volume
	A float64 `json:"a"` // amount
}

// FetchIndexMinute 从币赢 API 获取 A 股指数分时数据。
func (p *BiyingApiProvider) FetchIndexMinute(ctx context.Context, code string, market Market) ([]marketdomain.IndexMinutePoint, error) {
	if p.baseURL == "" || p.token == "" {
		return nil, newProviderError("biyingapi", "baseURL or token not configured")
	}
	if market != MarketCN {
		return nil, newProviderError("biyingapi", "only CN market supported")
	}

	// Use an independent timeout so that the caller's short deadline
	// (e.g. RaceTimeout 3s) does not cancel in-flight requests.
	reqCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()

	// Determine the market suffix for the code
	suffix := "SH"
	if code == "399001" || code == "399006" {
		suffix = "SZ"
	}
	codeMarket := biyingIndexCode(code, suffix)

	// period=5 for 5-minute bars (closest to 1-minute available)
	path := fmt.Sprintf("/hsindex/latest/%s/5/%s", codeMarket, p.token)
	// lt parameter is capped at 5 by the API; use 5 to get the most recent bars.
	url := fmt.Sprintf("%s%s?lt=5", p.baseURL, path)

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := p.resilient.Do(reqCtx, SourceBiyingAPI, req)
	if err != nil {
		return nil, p.safeError(err)
	}
	defer resp.Body.Close()
	body, err := readBiyingBody(resp)
	if err != nil {
		return nil, err
	}

	var bars []biyingIndexMinuteResp
	if err := json.Unmarshal(body, &bars); err != nil {
		return nil, fmt.Errorf("biyingapi: failed to parse index minute response: %w", err)
	}

	points := make([]marketdomain.IndexMinutePoint, 0, len(bars))
	for _, bar := range bars {
		// 提取 HH:MM 部分
		timeStr := bar.T
		if idx := strings.Index(bar.T, " "); idx >= 0 {
			timeStr = bar.T[idx+1:]
		}
		if len(timeStr) > 5 {
			timeStr = timeStr[:5]
		}
		points = append(points, marketdomain.IndexMinutePoint{
			Time:     timeStr,
			Price:    bar.C,
			AvgPrice: bar.C,
			Volume:   bar.V,
		})
	}
	return normalizeIndexMinutePoints(points, market), nil
}

// biyingStockRealTimeResp 映射币赢 API 股票实时行情响应字段。
type biyingStockRealTimeResp struct {
	P       float64 `json:"p"`        // price
	O       float64 `json:"o"`        // open
	H       float64 `json:"h"`        // high
	L       float64 `json:"l"`        // low
	YC      float64 `json:"yc"`       // prev close
	CJE     float64 `json:"cje"`      // amount
	V       float64 `json:"v"`        // volume
	PV      float64 `json:"pv"`       // change
	UD      float64 `json:"ud"`       // change pct (decimal)
	PC      float64 `json:"pc"`       // amplitude
	ZF      float64 `json:"zf"`       // amplitude (alternate)
	PE      float64 `json:"pe"`       // P/E ratio
	PBRatio float64 `json:"pb_ratio"` // P/B ratio
	TR      float64 `json:"tr"`       // turnover rate
	TV      float64 `json:"tv"`       // total volume
	T       string  `json:"t"`        // time
}

// FetchStockQuotes 从币赢 API 批量获取股票实时行情数据。
func (p *BiyingApiProvider) FetchStockQuotes(ctx context.Context, symbols []string) (map[string]stockdomain.StockQuote, error) {
	if p.baseURL == "" || p.token == "" {
		return nil, newProviderError("biyingapi", "baseURL or token not configured")
	}
	if len(symbols) == 0 {
		return map[string]stockdomain.StockQuote{}, nil
	}

	// BiyingAPI requires one HTTP call per stock, so fetching many symbols
	// is impractical. Return an error for large batches so the provider
	// router can fall back to a more suitable provider.
	const maxSymbols = 20
	if len(symbols) > maxSymbols {
		return nil, newProviderError("biyingapi", fmt.Sprintf("batch size %d exceeds limit of %d; use a bulk provider instead", len(symbols), maxSymbols))
	}

	// Use an independent timeout for each stock request so that the caller's
	// short deadline (e.g. RaceTimeout 3s) does not cancel in-flight HTTP
	// calls. The parent context is only used for cancellation signals (app
	// shutdown), not for deadline propagation.
	const perRequestTimeout = 10 * time.Second

	results := make(map[string]stockdomain.StockQuote, len(symbols))
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Limit concurrency to avoid overwhelming the API rate limits.
	sem := make(chan struct{}, 5)

	for _, symbol := range symbols {
		code := plainStockCode(symbol)
		if code == "" {
			continue
		}
		wg.Add(1)
		go func(code string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			reqCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), perRequestTimeout)
			defer cancel()

			path := fmt.Sprintf("/hsstock/real/time/%s", code)
			var resp biyingStockRealTimeResp
			if err := p.fetchBiyingJSON(reqCtx, path, &resp); err != nil {
				p.logger.Warn("biyingapi: failed to fetch stock quote", "code", code, "error", err)
				return
			}

			changePct := resp.UD * 100 // convert decimal to percentage
			quote := stockdomain.StockQuote{
				Price:        resp.P,
				Open:         resp.O,
				High:         resp.H,
				Low:          resp.L,
				PrevClose:    resp.YC,
				Volume:       resp.V,
				Amount:       resp.CJE,
				TurnoverRate: resp.TR,
				ChangePct:    changePct,
				ChangeAmt:    resp.PV,
				QuoteTime:    resp.T,
			}

			mu.Lock()
			results[code] = quote
			mu.Unlock()
		}(code)
	}
	wg.Wait()

	if len(results) == 0 {
		return nil, newProviderError("biyingapi", "all stock quote requests failed")
	}
	return results, nil
}

// plainStockCode 从带市场前缀的股票代码中提取纯数字代码。
// 例如 "sh600000" → "600000"，"sz000001" → "000001"。
func plainStockCode(symbol string) string {
	symbol = strings.TrimSpace(symbol)
	switch {
	case len(symbol) == 8 && (strings.HasPrefix(symbol, "sh") || strings.HasPrefix(symbol, "sz") || strings.HasPrefix(symbol, "bj")):
		return symbol[2:]
	case len(symbol) == 6:
		return symbol
	default:
		return ""
	}
}

// FetchStockRanking 为存根实现——需要多次 API 调用（股票列表 + 逐只行情），
// 对免费版速率限制而言成本过高。
func (p *BiyingApiProvider) FetchStockRanking(ctx context.Context, rankingType string, size int) ([]stockdomain.StockRankingItem, error) {
	return nil, newProviderError("biyingapi", "not yet implemented: requires multiple API calls")
}

// FetchSectorRanking 为存根实现——需要遍历层级概念树，
// 对免费版速率限制而言成本过高。
func (p *BiyingApiProvider) FetchSectorRanking(ctx context.Context) ([]marketdomain.MarketSectorItem, error) {
	return nil, newProviderError("biyingapi", "not yet implemented: requires multiple API calls")
}
