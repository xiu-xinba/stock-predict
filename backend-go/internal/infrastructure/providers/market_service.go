package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	funddomain "stock-predict-go/internal/domain/fund"
	marketdomain "stock-predict-go/internal/domain/market"
	httpclient "stock-predict-go/internal/platform/httpclient"
)

// MarketService 市场行情服务，提供指数行情、板块排行、北向资金等功能。
type MarketService struct {
	quote           *IndexQuoteClient
	logger          *slog.Logger
	client          *http.Client
	resilient       *ResilientHTTPClient
	eastmoney       *EastmoneyClient
	sectorCache     *DetailCache
	northboundCache *DetailCache
	router          *ProviderRouter
}

// NewMarketService 创建新的市场行情服务实例。
func NewMarketService(quote *IndexQuoteClient, logger *slog.Logger) *MarketService {
	if logger == nil {
		logger = slog.Default()
	}
	return &MarketService{
		quote:           quote,
		logger:          logger,
		client:          NewHTTPClient(HTTPClientConfig{}),
		resilient:       NewResilientHTTPClient(NewHTTPClient(HTTPClientConfig{}), nil),
		eastmoney:       newEastmoneyClient(NewHTTPClient(HTTPClientConfig{})),
		sectorCache:     NewDetailCache(CacheMaxEntries, SectorCacheTTL),
		northboundCache: NewDetailCache(CacheMaxEntries, NorthboundCacheTTL),
	}
}

// SetRouter 注入数据源路由器。
func (s *MarketService) SetRouter(router *ProviderRouter) {
	s.router = router
}

// Indices 获取主要市场指数行情数据。
func (s *MarketService) Indices(ctx context.Context) ([]marketdomain.MarketIndex, error) {
	if s.quote == nil {
		return nil, ErrMarketUnavailable
	}
	items := s.quote.FetchIndexQuotes(ctx)
	if len(items) == 0 {
		return nil, ErrMarketUnavailable
	}
	return items, nil
}

// IndexKline 获取指定指数的 K 线数据。
func (s *MarketService) IndexKline(ctx context.Context, code string, count int) ([]marketdomain.IndexKlinePoint, error) {
	if s.quote == nil {
		return nil, ErrMarketUnavailable
	}
	items := s.quote.FetchIndexKline(ctx, code, count)
	if len(items) == 0 {
		return nil, ErrMarketUnavailable
	}
	return items, nil
}

// IndexMinute 获取指定指数的分时数据。
func (s *MarketService) IndexMinute(ctx context.Context, code string) ([]marketdomain.IndexMinutePoint, error) {
	if s.quote == nil {
		return nil, ErrMarketUnavailable
	}
	items := s.quote.FetchIndexMinute(ctx, code)
	if len(items) == 0 {
		return nil, ErrMarketUnavailable
	}
	return items, nil
}

// StockMinute 获取指定股票的分时数据。
func (s *MarketService) StockMinute(ctx context.Context, code string) ([]marketdomain.IndexMinutePoint, error) {
	if s.router != nil {
		market := DetectMarket(code)
		var result []marketdomain.IndexMinutePoint
		err := s.router.Fetch(ctx, CapStockMinute, market, func(ctx context.Context, p Provider) error {
			provider, ok := p.(StockMinuteProvider)
			if !ok {
				return fmt.Errorf("provider %s does not implement StockMinuteProvider", p.Name())
			}
			items, err := provider.FetchStockMinute(ctx, code)
			if err != nil {
				return err
			}
			result = items
			return nil
		})
		if err == nil && len(result) > 0 {
			return result, nil
		}
	}

	if s.quote == nil {
		return nil, ErrMarketUnavailable
	}
	items := s.quote.FetchStockMinute(ctx, code)
	if len(items) == 0 {
		return nil, ErrMarketUnavailable
	}
	return items, nil
}

// SectorRanking 获取板块排行数据，优先使用路由器，回退到新浪源。
func (s *MarketService) SectorRanking(ctx context.Context) []marketdomain.MarketSectorItem {
	if s.router != nil {
		var result []marketdomain.MarketSectorItem
		err := s.router.Fetch(ctx, CapSectorRank, MarketCN, func(ctx context.Context, p Provider) error {
			provider, ok := p.(SectorRankingProvider)
			if !ok {
				return fmt.Errorf("provider %s does not implement SectorRankingProvider", p.Name())
			}
			items, err := provider.FetchSectorRanking(ctx)
			if err != nil {
				return err
			}
			result = items
			return nil
		})
		if err == nil && len(result) > 0 {
			return topGainersAndLosers(result, 10)
		}
	}

	const cacheKey = "sector_ranking"
	if cached, ok := s.sectorCache.Get(cacheKey); ok {
		if val, ok2 := cached.([]marketdomain.MarketSectorItem); ok2 {
			return val
		}
	}

	url := "https://push2his.eastmoney.com/api/qt/clist/get?pn=1&pz=50&fs=m:90+t:2&fields=f14,f3,f104,f105,f140"
	if !isAllowedURL(url) {
		s.logger.Warn("URL not in whitelist", "url", url)
		return nil
	}

	payload, err := s.fetchJSON(ctx, url)
	if err != nil {
		items := s.fetchSectorRankingSina(ctx)
		if items != nil {
			s.sectorCache.Set(cacheKey, items)
		}
		return items
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
		items := s.fetchSectorRankingSina(ctx)
		if items != nil {
			s.sectorCache.Set(cacheKey, items)
		}
		return items
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

	// Keep top 10 gainers + top 10 losers
	items = topGainersAndLosers(items, 10)

	s.sectorCache.Set(cacheKey, items)
	return items
}

// NorthboundFlow 获取北向资金流向数据，优先使用路由器，回退到东方财富源。
func (s *MarketService) NorthboundFlow(ctx context.Context) *marketdomain.NorthboundFlow {
	if s.router != nil {
		var result *marketdomain.NorthboundFlow
		err := s.router.Fetch(ctx, CapNorthbound, MarketCN, func(ctx context.Context, p Provider) error {
			provider, ok := p.(NorthboundProvider)
			if !ok {
				return fmt.Errorf("provider %s does not implement NorthboundProvider", p.Name())
			}
			flow, err := provider.FetchNorthboundFlow(ctx)
			if err != nil {
				return err
			}
			result = flow
			return nil
		})
		if err == nil && result != nil {
			return result
		}
	}

	const cacheKey = "northbound_flow"
	if cached, ok := s.northboundCache.Get(cacheKey); ok {
		if val, ok2 := cached.(*marketdomain.NorthboundFlow); ok2 {
			return val
		}
	}

	url := "https://push2his.eastmoney.com/api/qt/kamtbs.wss?fields1=f1,f2,f3&fields2=f51,f52,f53,f54,f55,f56"
	if !isAllowedURL(url) {
		s.logger.Warn("URL not in whitelist", "url", url)
		return marketdomain.NewNorthboundUnavailableFlow()
	}

	payload, err := s.fetchJSON(ctx, url)
	if err != nil {
		return marketdomain.NewNorthboundUnavailableFlow()
	}

	var result struct {
		Data struct {
			S2N []string `json:"s2n"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &result); err != nil {
		return marketdomain.NewNorthboundUnavailableFlow()
	}

	var flow marketdomain.NorthboundFlow
	timeline := make([]marketdomain.NorthboundPoint, 0, len(result.Data.S2N))
	now := time.Now().Format("15:04")
	for _, line := range result.Data.S2N {
		parts := strings.Split(line, ",")
		if len(parts) < 6 {
			continue
		}
		// Skip future time points beyond current time
		if len(parts[0]) >= 5 && parts[0][:5] > now {
			continue
		}
		shNet := httpclient.ParseQuoteFloat(parts[1])
		szNet := httpclient.ParseQuoteFloat(parts[2])
		shTotal := httpclient.ParseQuoteFloat(parts[3])
		szTotal := httpclient.ParseQuoteFloat(parts[4])
		total := httpclient.ParseQuoteFloat(parts[5])
		// Skip zero-value placeholder rows
		if shNet == 0 && szNet == 0 {
			continue
		}
		timeline = append(timeline, marketdomain.NorthboundPoint{
			Time:   parts[0],
			SHFlow: shNet,
			SZFlow: szNet,
		})
		flow.SHNetBuy = shTotal
		flow.SZNetBuy = szTotal
		flow.TotalBuy = total
	}
	flow.Timeline = timeline

	if !hasMeaningfulNorthboundFlow(&flow) {
		return marketdomain.NewNorthboundUnavailableFlow()
	}
	flow.Status = marketdomain.NorthboundStatusIntraday
	flow.DataSource = "eastmoney"
	s.northboundCache.Set(cacheKey, &flow)
	return &flow
}

// SortRanking 对基金排行项按指定类型排序（涨幅或跌幅）。
func SortRanking(items []funddomain.FundRankingItem, rankingType string) {
	sort.SliceStable(items, func(i, j int) bool {
		if rankingType == "losers" {
			return items[i].ChangePct < items[j].ChangePct
		}
		return items[i].ChangePct > items[j].ChangePct
	})
	for i := range items {
		items[i].Rank = i + 1
	}
}

// fetchSectorRankingSina 从新浪获取板块排行数据。
func (s *MarketService) fetchSectorRankingSina(ctx context.Context) []marketdomain.MarketSectorItem {
	url := "https://money.finance.sina.com.cn/q/view/newFLJK.php?param=class"
	if !isAllowedURL(url) {
		s.logger.Warn("URL not in whitelist", "url", url)
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
	resp, err := s.resilient.Do(ctx, SourceSina, req)
	if err != nil {
		s.logger.Warn("Sina sector fetch failed", "url", url, "error", err)
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.logger.Warn("Sina sector fetch non-2xx", "url", url, "status", resp.StatusCode)
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(MaxHTTPPayloadBytes)))
	if err != nil {
		return nil
	}
	body = httpclient.EnsureUTF8(body)

	bodyStr := string(body)
	start := strings.Index(bodyStr, "{")
	end := strings.LastIndex(bodyStr, "}")
	if start < 0 || end < 0 || end <= start {
		return nil
	}
	jsonStr := bodyStr[start : end+1]

	var data map[string]string
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil
	}

	items := make([]marketdomain.MarketSectorItem, 0, len(data))
	for _, v := range data {
		fields := strings.Split(v, ",")
		if len(fields) < 13 {
			continue
		}
		items = append(items, marketdomain.MarketSectorItem{
			Name:      fields[1],
			ChangePct: httpclient.ParseQuoteFloat(fields[5]),
			UpCount:   0,
			DownCount: 0,
			LeadStock: fields[12],
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].ChangePct > items[j].ChangePct
	})
	items = topGainersAndLosers(items, 10)
	return items
}

// fetchJSON 通过 HTTP GET 请求获取 JSON 数据。
func (s *MarketService) fetchJSON(ctx context.Context, url string) ([]byte, error) {
	if strings.Contains(url, "eastmoney.com") {
		return s.eastmoney.Get(ctx, url, "https://quote.eastmoney.com/")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
	req.Header.Set("Referer", "https://quote.eastmoney.com/")
	source := SourceSina
	if strings.Contains(url, "eastmoney.com") {
		source = SourceEastmoney
	}
	resp, err := s.resilient.Do(ctx, source, req)
	if err != nil {
		s.logger.Warn("HTTP fetch failed", "url", url, "error", err)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.logger.Warn("HTTP fetch non-2xx", "url", url, "status", resp.StatusCode)
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	payload, err := io.ReadAll(io.LimitReader(resp.Body, int64(MaxHTTPPayloadBytes)))
	if err != nil {
		return nil, err
	}
	return httpclient.EnsureUTF8(payload), nil
}

// topGainersAndLosers 从板块排行中提取涨幅前 n 和跌幅前 n 的板块。
func topGainersAndLosers(items []marketdomain.MarketSectorItem, n int) []marketdomain.MarketSectorItem {
	if len(items) <= n*2 {
		return items
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].ChangePct > items[j].ChangePct
	})

	topGainers := items[:n]
	topLosers := items[len(items)-n:]

	result := make([]marketdomain.MarketSectorItem, 0, n*2)
	result = append(result, topGainers...)
	result = append(result, topLosers...)
	return result
}
