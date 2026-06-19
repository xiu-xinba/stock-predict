package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	marketdomain "stock-predict-go/internal/domain/market"
	stockdomain "stock-predict-go/internal/domain/stock"
)

// AKShareProvider 实现了 AKShare 数据源的 Provider 接口。
type AKShareProvider struct {
	resilient *ResilientHTTPClient
	baseURL   string
	token     string
	logger    *slog.Logger
}

// NewAKShareProvider 创建一个新的 AKShareProvider 实例（无 Token）。
func NewAKShareProvider(baseURL string, logger *slog.Logger) *AKShareProvider {
	return NewAKShareProviderWithToken(baseURL, "", logger)
}

// NewAKShareProviderWithToken 创建一个带 Token 认证的 AKShareProvider 实例。
func NewAKShareProviderWithToken(baseURL, token string, logger *slog.Logger) *AKShareProvider {
	client := NewHTTPClient(HTTPClientConfig{})
	return &AKShareProvider{
		resilient: NewResilientHTTPClient(client, DefaultSourcePolicies()),
		baseURL:   baseURL,
		token:     token,
		logger:    logger,
	}
}

// Name 返回数据源的唯一标识名称。
func (p *AKShareProvider) Name() string { return "akshare" }

// Capabilities 返回 AKShare 数据源支持的能力及其适用的市场。
func (p *AKShareProvider) Capabilities() map[Capability][]Market {
	return map[Capability][]Market{
		CapIndexQuote:  {MarketCN},
		CapIndexMinute: {MarketCN},
		CapIndexKline:  {MarketCN},
		CapStockSync:   {MarketCN},
		CapStockSearch: {MarketCN},
		CapNorthbound:  {MarketCN},
	}
}

// Priority 返回指定能力和市场组合下的优先级，数值越小优先级越高。
func (p *AKShareProvider) Priority(cap Capability, market Market) int {
	switch cap {
	case CapIndexQuote:
		return 5
	case CapIndexMinute:
		return 5
	case CapIndexKline:
		return 3
	case CapStockSync:
		return 2
	case CapStockSearch:
		return 2
	case CapNorthbound:
		return 3
	}
	return 99
}

// HealthCheck 通过请求 AKShare 健康检查接口检测数据源状态。
func (p *AKShareProvider) HealthCheck(ctx context.Context) error {
	url := p.baseURL + "/health"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := p.resilient.Do(ctx, SourceAKShare, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("akshare health check returned %d", resp.StatusCode)
	}
	return nil
}

// akshareResponse 表示 AKShare API 的标准响应结构。
type akshareResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// fetchJSON 通过 GET 请求获取 AKShare API 的 JSON 数据。
func (p *AKShareProvider) fetchJSON(ctx context.Context, path string, result any) error {
	return p.fetchJSONMethod(ctx, http.MethodGet, path, result)
}

// fetchJSONMethod 通过指定 HTTP 方法获取 AKShare API 的 JSON 数据。
func (p *AKShareProvider) fetchJSONMethod(ctx context.Context, method, path string, result any) error {
	url := p.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return err
	}
	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}
	resp, err := p.resilient.Do(ctx, SourceAKShare, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(MaxHTTPPayloadBytes)))
	if err != nil {
		return err
	}
	var akResp akshareResponse
	if err := json.Unmarshal(body, &akResp); err != nil {
		return err
	}
	if akResp.Code != 0 {
		return fmt.Errorf("akshare error: %s", akResp.Message)
	}
	return json.Unmarshal(akResp.Data, result)
}

// FetchIndexQuotes 从 AKShare 获取 A 股指数行情数据。
func (p *AKShareProvider) FetchIndexQuotes(ctx context.Context, market Market) ([]marketdomain.MarketIndex, error) {
	if p.baseURL == "" {
		return nil, newProviderError("akshare", "baseURL not configured")
	}
	if market != MarketCN {
		return nil, newProviderError("akshare", "market not supported")
	}

	type rawQuote struct {
		Code      string  `json:"code"`
		Name      string  `json:"name"`
		Price     float64 `json:"price"`
		ChangePct float64 `json:"change_pct"`
		Volume    float64 `json:"volume"`
	}

	var raw []rawQuote
	if err := p.fetchJSON(ctx, "/api/v1/index/quote?market=cn", &raw); err != nil {
		return nil, err
	}

	result := make([]marketdomain.MarketIndex, len(raw))
	for i, r := range raw {
		result[i] = marketdomain.MarketIndex{
			Code:       r.Code,
			Name:       r.Name,
			Market:     string(MarketCN),
			Value:      r.Price,
			ChangePct:  r.ChangePct,
			Volume:     r.Volume,
			DataSource: "akshare",
		}
	}
	return result, nil
}

// FetchIndexMinute 从 AKShare 获取 A 股指数分时数据。
func (p *AKShareProvider) FetchIndexMinute(ctx context.Context, code string, market Market) ([]marketdomain.IndexMinutePoint, error) {
	if p.baseURL == "" {
		return nil, newProviderError("akshare", "baseURL not configured")
	}
	if market != MarketCN {
		return nil, newProviderError("akshare", "market not supported")
	}

	type rawMinute struct {
		Time   string  `json:"time"`
		Price  float64 `json:"price"`
		Volume int64   `json:"volume"`
	}

	var raw []rawMinute
	path := fmt.Sprintf("/api/v1/index/minute?code=%s&market=cn", code)
	if err := p.fetchJSON(ctx, path, &raw); err != nil {
		return nil, err
	}

	result := make([]marketdomain.IndexMinutePoint, len(raw))
	for i, r := range raw {
		result[i] = marketdomain.IndexMinutePoint{
			Time:   r.Time,
			Price:  r.Price,
			Volume: r.Volume,
		}
	}
	return result, nil
}

// FetchIndexKline 从 AKShare 获取 A 股指数 K 线数据。
func (p *AKShareProvider) FetchIndexKline(ctx context.Context, code string, market Market, count int) ([]marketdomain.IndexKlinePoint, error) {
	if p.baseURL == "" {
		return nil, newProviderError("akshare", "baseURL not configured")
	}
	if market != MarketCN {
		return nil, newProviderError("akshare", "market not supported")
	}

	type rawKline struct {
		Date   string  `json:"date"`
		Open   float64 `json:"open"`
		Close  float64 `json:"close"`
		High   float64 `json:"high"`
		Low    float64 `json:"low"`
		Volume int64   `json:"volume"`
	}

	var raw []rawKline
	path := fmt.Sprintf("/api/v1/index/kline?code=%s&market=cn&count=%d", code, count)
	if err := p.fetchJSON(ctx, path, &raw); err != nil {
		return nil, err
	}

	result := make([]marketdomain.IndexKlinePoint, len(raw))
	for i, r := range raw {
		result[i] = marketdomain.IndexKlinePoint{
			Date:   r.Date,
			Open:   r.Open,
			Close:  r.Close,
			High:   r.High,
			Low:    r.Low,
			Volume: r.Volume,
		}
	}
	return result, nil
}

// SyncStocks 从 AKShare 全量同步 A 股股票列表。
func (p *AKShareProvider) SyncStocks(ctx context.Context) ([]stockdomain.StockItem, error) {
	if p.baseURL == "" {
		return nil, newProviderError("akshare", "baseURL not configured")
	}

	type rawStock struct {
		StockCode string `json:"stock_code"`
		StockName string `json:"stock_name"`
		Industry  string `json:"industry"`
	}

	var raw []rawStock
	if err := p.fetchJSONMethod(ctx, http.MethodPost, "/api/v1/stock/sync", &raw); err != nil {
		return nil, err
	}

	result := make([]stockdomain.StockItem, len(raw))
	for i, r := range raw {
		result[i] = stockdomain.StockItem{
			StockCode: r.StockCode,
			StockName: r.StockName,
			Market:    string(MarketCN),
			Industry:  r.Industry,
		}
	}
	return result, nil
}

// SearchStocks 尚未实现，返回 not yet implemented 错误。
func (p *AKShareProvider) SearchStocks(ctx context.Context, query string) ([]stockdomain.StockSearchItem, error) {
	return nil, newProviderError("akshare", "not yet implemented")
}

// FetchNorthboundFlow 从 AKShare 获取北向资金流向数据。
func (p *AKShareProvider) FetchNorthboundFlow(ctx context.Context) (*marketdomain.NorthboundFlow, error) {
	if p.baseURL == "" {
		return nil, newProviderError("akshare", "baseURL not configured")
	}

	type rawNorthboundPoint struct {
		Time   string  `json:"time"`
		SHFlow float64 `json:"sh_flow"`
		SZFlow float64 `json:"sz_flow"`
	}

	type rawNorthboundFlow struct {
		SHNetBuy float64              `json:"sh_net_buy"`
		SZNetBuy float64              `json:"sz_net_buy"`
		TotalBuy float64              `json:"total_net_buy"`
		Timeline []rawNorthboundPoint `json:"timeline"`
	}

	var raw rawNorthboundFlow
	if err := p.fetchJSON(ctx, "/api/v1/northbound/flow", &raw); err != nil {
		return nil, err
	}

	timeline := make([]marketdomain.NorthboundPoint, len(raw.Timeline))
	for i, p := range raw.Timeline {
		timeline[i] = marketdomain.NorthboundPoint{
			Time:   p.Time,
			SHFlow: p.SHFlow,
			SZFlow: p.SZFlow,
		}
	}

	flow := &marketdomain.NorthboundFlow{
		SHNetBuy:   raw.SHNetBuy,
		SZNetBuy:   raw.SZNetBuy,
		TotalBuy:   raw.TotalBuy,
		Timeline:   timeline,
		Status:     marketdomain.NorthboundStatusIntraday,
		DataSource: "akshare",
	}
	if !hasMeaningfulNorthboundFlow(flow) {
		return nil, newProviderError("akshare", "empty result")
	}
	return flow, nil
}
