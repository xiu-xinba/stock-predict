package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	marketdomain "stock-predict-go/internal/domain/market"
)

// SinaProvider 实现了新浪财经数据源的 Provider 接口。
type SinaProvider struct {
	quoteClient *IndexQuoteClient
}

// NewSinaProvider 创建一个新的 SinaProvider 实例。
func NewSinaProvider(quoteClient *IndexQuoteClient) *SinaProvider {
	return &SinaProvider{quoteClient: quoteClient}
}

// Name 返回数据源的唯一标识名称。
func (p *SinaProvider) Name() string {
	return "sina"
}

// Capabilities 返回新浪财经数据源支持的能力及其适用的市场。
func (p *SinaProvider) Capabilities() map[Capability][]Market {
	return map[Capability][]Market{
		CapIndexQuote:  {MarketCN},
		CapIndexMinute: {MarketUS},
		CapNorthbound:  {MarketCN},
	}
}

// Priority 返回指定能力和市场组合下的优先级，数值越小优先级越高。
func (p *SinaProvider) Priority(cap Capability, market Market) int {
	switch {
	case cap == CapIndexQuote && market == MarketCN:
		return 4
	case cap == CapIndexMinute && market == MarketUS:
		return 1
	case cap == CapNorthbound && market == MarketCN:
		return 3
	case cap == CapSectorRank && market == MarketCN:
		return 2
	default:
		return 99
	}
}

// HealthCheck 通过获取 A 股指数行情检测数据源健康状态。
func (p *SinaProvider) HealthCheck(ctx context.Context) error {
	indices := p.quoteClient.fetchCNIndexQuotesSina(ctx)
	if len(indices) == 0 {
		return newHealthCheckError("sina", "CN index quotes returned empty")
	}
	return nil
}

// FetchIndexQuotes 从新浪财经获取 A 股指数行情数据。
func (p *SinaProvider) FetchIndexQuotes(ctx context.Context, market Market) ([]marketdomain.MarketIndex, error) {
	if market != MarketCN {
		return nil, newProviderError("sina", "unsupported market for index quotes")
	}
	indices := p.quoteClient.fetchCNIndexQuotesSina(ctx)
	if len(indices) == 0 {
		return nil, newProviderError("sina", "empty result")
	}
	return indices, nil
}

// FetchIndexMinute 从新浪财经获取指数分时数据。
func (p *SinaProvider) FetchIndexMinute(ctx context.Context, code string, market Market) ([]marketdomain.IndexMinutePoint, error) {
	points := p.quoteClient.fetchUSIndexMinuteSina(ctx, code)
	if len(points) == 0 {
		return nil, newProviderError("sina", "empty result")
	}
	return points, nil
}

// FetchNorthboundFlow 从新浪财经获取北向资金流向数据。
func (p *SinaProvider) FetchNorthboundFlow(ctx context.Context) (*marketdomain.NorthboundFlow, error) {
	url := "https://vip.stock.finance.sina.com.cn/q/getnewforex/getHK2SH.php?callback="
	if !isAllowedURL(url) {
		return nil, newProviderError("sina", "URL not in whitelist")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, newProviderError("sina", fmt.Sprintf("create request: %v", err))
	}
	req.Header.Set("Referer", "https://finance.sina.com.cn/")

	resp, err := p.quoteClient.resilient.Do(ctx, SourceSina, req)
	if err != nil {
		return nil, newProviderError("sina", fmt.Sprintf("fetch northbound flow: %v", err))
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, newProviderError("sina", fmt.Sprintf("northbound flow HTTP %d", resp.StatusCode))
	}

	payload, err := io.ReadAll(io.LimitReader(resp.Body, int64(MaxHTTPPayloadBytes)))
	if err != nil {
		return nil, newProviderError("sina", fmt.Sprintf("read response: %v", err))
	}

	// Parse response: var json={...} or plain JSON
	text := string(payload)
	idx := strings.Index(text, "{")
	if idx < 0 {
		return nil, newProviderError("sina", "no JSON in northbound response")
	}
	endIdx := strings.LastIndex(text, "}")
	if endIdx <= idx {
		return nil, newProviderError("sina", "malformed JSON in northbound response")
	}
	jsonStr := text[idx : endIdx+1]

	var raw struct {
		Data struct {
			HK2SH []struct {
				Date    string  `json:"d"`
				SHNet   float64 `json:"s2n"`
				SHTotal float64 `json:"n2s"`
			} `json:"hk2sh"`
			HK2SZ []struct {
				Date    string  `json:"d"`
				SZNet   float64 `json:"s2n"`
				SZTotal float64 `json:"n2s"`
			} `json:"hk2sz"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return nil, newProviderError("sina", fmt.Sprintf("parse northbound flow: %v", err))
	}

	var flow marketdomain.NorthboundFlow
	timeline := make([]marketdomain.NorthboundPoint, 0)

	for _, item := range raw.Data.HK2SH {
		timeline = append(timeline, marketdomain.NorthboundPoint{
			Time:   item.Date,
			SHFlow: item.SHNet,
		})
		flow.SHNetBuy = item.SHTotal
	}

	for i, item := range raw.Data.HK2SZ {
		if i < len(timeline) {
			timeline[i].SZFlow = item.SZNet
		}
		flow.SZNetBuy = item.SZTotal
	}

	flow.TotalBuy = flow.SHNetBuy + flow.SZNetBuy
	flow.Timeline = timeline
	flow.Status = marketdomain.NorthboundStatusIntraday
	flow.DataSource = "sina"

	if !hasMeaningfulNorthboundFlow(&flow) {
		return nil, newProviderError("sina", "empty result")
	}
	return &flow, nil
}
