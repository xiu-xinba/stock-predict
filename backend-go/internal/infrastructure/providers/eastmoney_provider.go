package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	funddomain "stock-predict-go/internal/domain/fund"
	marketdomain "stock-predict-go/internal/domain/market"
	stockdomain "stock-predict-go/internal/domain/stock"
	httpclient "stock-predict-go/internal/platform/httpclient"
)

// EastmoneyProvider 实现了东方财富数据源的 Provider 接口。
type EastmoneyProvider struct {
	quoteClient  *IndexQuoteClient
	eastmoney    *EastmoneyClient
	stockService *StockService
}

// NewEastmoneyProvider 创建一个新的 EastmoneyProvider 实例。
func NewEastmoneyProvider(quoteClient *IndexQuoteClient, eastmoney *EastmoneyClient, stockService *StockService) *EastmoneyProvider {
	return &EastmoneyProvider{
		quoteClient:  quoteClient,
		eastmoney:    eastmoney,
		stockService: stockService,
	}
}

// SetStockService 注入股票服务，支持延迟初始化。
func (p *EastmoneyProvider) SetStockService(s *StockService) {
	p.stockService = s
}

// Name 返回数据源的唯一标识名称。
func (p *EastmoneyProvider) Name() string {
	return "eastmoney"
}

// Capabilities 返回东方财富数据源支持的能力及其适用的市场。
func (p *EastmoneyProvider) Capabilities() map[Capability][]Market {
	return map[Capability][]Market{
		CapIndexQuote:   {MarketCN},
		CapIndexMinute:  {MarketUS},
		CapIndexKline:   {MarketUS},
		CapStockSearch:  {MarketCN},
		CapStockSync:    {MarketCN},
		CapStockRanking: {MarketCN},
		CapSectorRank:   {MarketCN},
		CapNorthbound:   {MarketCN},
		CapFundQuote:    {MarketCN},
	}
}

// Priority 返回指定能力和市场组合下的优先级，数值越小优先级越高。
func (p *EastmoneyProvider) Priority(cap Capability, market Market) int {
	key := string(cap) + ":" + string(market)
	priorities := map[string]int{
		"index_quote:cn":   3,
		"index_minute:us":  3,
		"index_kline:us":   2,
		"stock_search:cn":  1,
		"stock_sync:cn":    1,
		"stock_ranking:cn": 2,
		"sector_rank:cn":   1,
		"northbound:cn":    1,
		"fund_quote:cn":    2,
	}
	if pr, ok := priorities[key]; ok {
		return pr
	}
	return 99
}

// HealthCheck 通过请求东方财富历史 K 线接口检测数据源健康状态。
func (p *EastmoneyProvider) HealthCheck(ctx context.Context) error {
	_, err := p.eastmoney.Get(ctx, "https://push2his.eastmoney.com/api/qt/stock/kline/get?secid=1.000001&fields1=f1&fields2=f51&klt=101&fqt=1&end=20500101&lmt=1", "https://quote.eastmoney.com/")
	if err != nil {
		return newHealthCheckError("eastmoney", fmt.Sprintf("health check failed: %v", err))
	}
	return nil
}

// FetchIndexQuotes 委托 IndexQuoteClient 获取 A 股指数行情数据。
func (p *EastmoneyProvider) FetchIndexQuotes(ctx context.Context, market Market) ([]marketdomain.MarketIndex, error) {
	if market != MarketCN {
		return nil, newProviderError("eastmoney", "unsupported market")
	}
	result := p.quoteClient.fetchCNIndexQuotesEastmoney(ctx)
	if len(result) == 0 {
		return nil, newProviderError("eastmoney", "empty result")
	}
	return result, nil
}

// FetchIndexMinute 委托 IndexQuoteClient 获取美股指数分时数据。
func (p *EastmoneyProvider) FetchIndexMinute(ctx context.Context, code string, market Market) ([]marketdomain.IndexMinutePoint, error) {
	if market != MarketUS {
		return nil, newProviderError("eastmoney", "unsupported market")
	}
	result := p.quoteClient.fetchUSIndexMinuteEastmoney(ctx, code)
	if len(result) == 0 {
		return nil, newProviderError("eastmoney", "empty result")
	}
	return result, nil
}

// FetchIndexKline 委托 IndexQuoteClient 获取美股指数 K 线数据。
func (p *EastmoneyProvider) FetchIndexKline(ctx context.Context, code string, market Market, count int) ([]marketdomain.IndexKlinePoint, error) {
	if market != MarketUS {
		return nil, newProviderError("eastmoney", "unsupported market")
	}
	result := p.quoteClient.fetchUSIndexKlineEastmoney(ctx, code, count)
	if len(result) == 0 {
		return nil, newProviderError("eastmoney", "empty result")
	}
	return result, nil
}

// SearchStocks 委托 StockService 搜索股票，并将 StockItem 转换为 StockSearchItem。
func (p *EastmoneyProvider) SearchStocks(ctx context.Context, keyword string) ([]stockdomain.StockSearchItem, error) {
	items := p.stockService.searchFromAPI(ctx, keyword)
	if len(items) == 0 {
		return nil, newProviderError("eastmoney", "empty result")
	}
	result := make([]stockdomain.StockSearchItem, len(items))
	for i, item := range items {
		result[i] = stockdomain.StockSearchItem{
			StockCode: item.StockCode,
			StockName: item.StockName,
			Market:    item.Market,
			Pinyin:    item.Pinyin,
		}
	}
	return result, nil
}

// SyncStocks 委托 StockService 从东方财富 API 全量同步股票列表。
func (p *EastmoneyProvider) SyncStocks(ctx context.Context) ([]stockdomain.StockItem, error) {
	result := p.stockService.fetchAllStocksFromAPI(ctx)
	if len(result) == 0 {
		return nil, newProviderError("eastmoney", "empty result")
	}
	return result, nil
}

// FetchStockRanking 委托 StockService 获取股票排行数据。
func (p *EastmoneyProvider) FetchStockRanking(ctx context.Context, rankingType string, size int) ([]stockdomain.StockRankingItem, error) {
	result := p.stockService.fetchRankingFromAPI(ctx, rankingType, size)
	if len(result) == 0 {
		return nil, newProviderError("eastmoney", "empty result")
	}
	return result, nil
}

// FetchSectorRanking 从东方财富 push2 API 获取板块排行数据。
func (p *EastmoneyProvider) FetchSectorRanking(ctx context.Context) ([]marketdomain.MarketSectorItem, error) {
	url := "https://push2his.eastmoney.com/api/qt/clist/get?pn=1&pz=50&fs=m:90+t:2&fields=f14,f3,f104,f105,f140"
	if !isAllowedURL(url) {
		return nil, newProviderError("eastmoney", "URL not in whitelist")
	}

	payload, err := p.eastmoney.Get(ctx, url, "https://quote.eastmoney.com/")
	if err != nil {
		return nil, newProviderError("eastmoney", fmt.Sprintf("fetch sector ranking: %v", err))
	}

	var result struct {
		Data struct {
			Diff []struct {
				Name      string  `json:"f14"`
				ChangePct float64 `json:"f3"`
				UpCount   int     `json:"f104"`
				DownCount int     `json:"f105"`
				LeadStock string  `json:"f140"`
			} `json:"diff"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, newProviderError("eastmoney", fmt.Sprintf("parse sector ranking: %v", err))
	}

	items := make([]marketdomain.MarketSectorItem, 0, len(result.Data.Diff))
	for _, d := range result.Data.Diff {
		items = append(items, marketdomain.MarketSectorItem{
			Name:      d.Name,
			ChangePct: d.ChangePct,
			UpCount:   d.UpCount,
			DownCount: d.DownCount,
			LeadStock: d.LeadStock,
		})
	}
	if len(items) == 0 {
		return nil, newProviderError("eastmoney", "empty result")
	}
	return items, nil
}

// FetchNorthboundFlow 从东方财富 API 获取北向资金流向数据。
func (p *EastmoneyProvider) FetchNorthboundFlow(ctx context.Context) (*marketdomain.NorthboundFlow, error) {
	url := "https://push2his.eastmoney.com/api/qt/kamtbs.wss?fields1=f1,f2,f3&fields2=f51,f52,f53,f54,f55,f56"
	if !isAllowedURL(url) {
		return nil, newProviderError("eastmoney", "URL not in whitelist")
	}

	payload, err := p.eastmoney.Get(ctx, url, "https://quote.eastmoney.com/")
	if err != nil {
		return nil, newProviderError("eastmoney", fmt.Sprintf("fetch northbound flow: %v", err))
	}

	var result struct {
		Data struct {
			S2N []string `json:"s2n"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, newProviderError("eastmoney", fmt.Sprintf("parse northbound flow: %v", err))
	}

	var flow marketdomain.NorthboundFlow
	timeline := make([]marketdomain.NorthboundPoint, 0, len(result.Data.S2N))
	now := time.Now().Format("15:04")

	for _, line := range result.Data.S2N {
		parts := strings.Split(line, ",")
		if len(parts) < 6 {
			continue
		}
		pointTime := strings.TrimSpace(parts[0])
		if len(pointTime) < 5 || pointTime[2] != ':' {
			continue
		}
		pointTime = pointTime[:5]
		if pointTime > now {
			continue
		}
		shNet := httpclient.ParseQuoteFloat(parts[1])
		szNet := httpclient.ParseQuoteFloat(parts[2])
		shTotal := httpclient.ParseQuoteFloat(parts[3])
		szTotal := httpclient.ParseQuoteFloat(parts[4])
		total := httpclient.ParseQuoteFloat(parts[5])
		if shNet == 0 && szNet == 0 {
			continue
		}
		timeline = append(timeline, marketdomain.NorthboundPoint{
			Time:   pointTime,
			SHFlow: shNet,
			SZFlow: szNet,
		})
		flow.SHNetBuy = shTotal
		flow.SZNetBuy = szTotal
		flow.TotalBuy = total
	}

	if flow.TotalBuy == 0 {
		flow.TotalBuy = flow.SHNetBuy + flow.SZNetBuy
	}
	flow.Timeline = timeline
	flow.Status = marketdomain.NorthboundStatusIntraday
	flow.DataSource = "eastmoney"

	if len(timeline) == 0 {
		return nil, newProviderError("eastmoney", "empty result")
	}
	return &flow, nil
}

// FetchFundQuotes 通过东方财富 fundgz API 批量获取未上市基金估值数据。
func (p *EastmoneyProvider) FetchFundQuotes(ctx context.Context, codes []string) (map[string]funddomain.FundItem, error) {
	quotes := make(map[string]funddomain.FundItem, len(codes))
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, MaxFundGZConcurrency)
	now := time.Now()
	for _, code := range codes {
		wg.Add(1)
		go func(code string) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}
			if quote, ok := p.fetchEastmoneyFundGZQuote(ctx, code, now); ok {
				mu.Lock()
				quotes[code] = quote
				mu.Unlock()
			}
		}(code)
	}
	wg.Wait()
	if len(quotes) == 0 {
		return nil, newProviderError("eastmoney", "empty result")
	}
	return quotes, nil
}

// fetchEastmoneyFundGZQuote 从东方财富 fundgz API 获取单只基金估值数据。
func (p *EastmoneyProvider) fetchEastmoneyFundGZQuote(ctx context.Context, code string, now time.Time) (funddomain.FundItem, bool) {
	url := fmt.Sprintf(eastmoneyFundGZURL, code, now.UnixMilli())
	if !isAllowedURL(url) {
		return funddomain.FundItem{}, false
	}
	payload, err := p.eastmoney.GetWithLimit(ctx, url, "https://fund.eastmoney.com/"+code+".html", 1<<20)
	if err != nil {
		return funddomain.FundItem{}, false
	}
	return parseEastmoneyFundGZQuote(payload)
}
