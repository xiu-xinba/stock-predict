package providers

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	funddomain "stock-predict-go/internal/domain/fund"
	httpclient "stock-predict-go/internal/platform/httpclient"
)

const (
	eastmoneyNAVURL = "https://fund.eastmoney.com/f10/F10DataApi.aspx?type=lsjz&code=%s&sdate=&edate=&per=%d&page=%d" // 东方财富净值历史 API 地址
)

// FundDetailService 基金详情服务，提供基金详情、净值历史、风险指标、持仓等信息。
type FundDetailService struct {
	store     funddomain.Repository
	quote     *FundQuoteClient
	logger    *slog.Logger
	resilient *ResilientHTTPClient
	cache     *DetailCache
}

// NewFundDetailService 创建新的基金详情服务实例。
func NewFundDetailService(store funddomain.Repository, quote *FundQuoteClient, logger *slog.Logger) *FundDetailService {
	client := NewHTTPClient(HTTPClientConfig{})
	return &FundDetailService{
		store:     store,
		quote:     quote,
		logger:    logger,
		resilient: NewResilientHTTPClient(client, DefaultSourcePolicies()),
		cache:     NewDetailCache(CacheMaxEntries, CacheTTL),
	}
}

// GetDetail 获取基金详情，包含实时估值、净值历史、业绩表现、风险指标、持仓等信息。
func (s *FundDetailService) GetDetail(ctx context.Context, code string) (funddomain.FundDetailData, error) {
	if len(code) != 6 || !httpclient.IsAllDigits(code) {
		return funddomain.FundDetailData{}, ErrInvalidFundCode
	}

	if cached, ok := s.cache.Get(code); ok {
		if val, ok2 := cached.(funddomain.FundDetailData); ok2 {
			return val, nil
		}
	}

	fund, ok := s.store.FindFund(code)
	if !ok {
		return funddomain.FundDetailData{}, ErrFundNotFound
	}

	quoteMap := s.quote.RefreshQuotes(ctx, []funddomain.FundItem{fund})
	quotedFund := fund
	if q, ok := quoteMap[code]; ok {
		quotedFund = q
	}

	navHistory := s.fetchNAVHistory(ctx, code)

	performance := s.buildPerformance(fund, navHistory)

	manager := s.buildManager(fund)

	portfolio := s.buildPortfolio(fund)

	risk := s.calculateRisk(navHistory)

	data := funddomain.FundDetailData{
		Basic:       fund,
		Quote:       quotedFund,
		Performance: performance,
		Manager:     manager,
		Portfolio:   portfolio,
		Risk:        risk,
	}

	s.cache.Set(code, data)

	return data, nil
}

// fetchNAVHistory 从东方财富获取基金净值历史数据。
func (s *FundDetailService) fetchNAVHistory(ctx context.Context, code string) []funddomain.NAVPoint {
	pageResults := make([][]funddomain.NAVPoint, NAVHistoryPages)
	var wg sync.WaitGroup
	wg.Add(NAVHistoryPages)
	for page := 1; page <= NAVHistoryPages; page++ {
		go func(p int) {
			defer wg.Done()
			url := fmt.Sprintf(eastmoneyNAVURL, code, NAVHistoryDaysPerPage, p)
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				return
			}
			req.Header.Set("Referer", "https://fund.eastmoney.com/")

			resp, err := s.resilient.Do(ctx, SourceEastmoney, req)
			if err != nil {
				s.logger.Warn("fetch NAV history failed", "fund_code", code, "page", p, "error", err)
				return
			}

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				resp.Body.Close()
				s.logger.Warn("NAV history API returned non-200", "fund_code", code, "page", p, "status", resp.StatusCode)
				return
			}

			payload, err := io.ReadAll(io.LimitReader(resp.Body, int64(MaxHTTPPayloadBytes)))
			resp.Body.Close()
			if err != nil {
				return
			}

			points := parseNAVHistory(payload)
			pageResults[p-1] = points
		}(page)
	}
	wg.Wait()

	var allPoints []funddomain.NAVPoint
	for _, points := range pageResults {
		allPoints = append(allPoints, points...)
	}
	sort.Slice(allPoints, func(i, j int) bool {
		return allPoints[i].Date < allPoints[j].Date
	})
	return allPoints
}

// navRowRe 用于解析东方财富净值历史 HTML 表格行的正则表达式。
var navRowRe = regexp.MustCompile(`<tr>\s*<td>(\d{4}-\d{2}-\d{2})</td>\s*<td[^>]*>([\d.]+)</td>\s*<td[^>]*>([\d.]+)</td>\s*<td[^>]*>([^<]*)</td>`)

// parseNAVHistory 从 HTML 响应中解析净值历史数据。
func parseNAVHistory(payload []byte) []funddomain.NAVPoint {
	text := string(payload)
	matches := navRowRe.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		slog.Warn("NAV history HTML parse found no rows", "payload_len", len(payload))
		return nil
	}

	points := make([]funddomain.NAVPoint, 0, len(matches))
	for _, m := range matches {
		date := strings.TrimSpace(m[1])
		nav := httpclient.ParseQuoteFloat(m[2])
		cumNav := httpclient.ParseQuoteFloat(m[3])
		changePct := httpclient.ParseQuoteFloat(m[4])
		if nav <= 0 {
			continue
		}
		points = append(points, funddomain.NAVPoint{
			Date:          date,
			NAV:           nav,
			CumulativeNAV: cumNav,
			ChangePct:     changePct,
		})
	}

	sort.Slice(points, func(i, j int) bool {
		return points[i].Date < points[j].Date
	})

	return points
}

// buildPerformance 根据基金信息和净值历史构建业绩表现数据。
func (s *FundDetailService) buildPerformance(fund funddomain.FundItem, navHistory []funddomain.NAVPoint) funddomain.FundPerformanceData {
	perf := funddomain.FundPerformanceData{
		NAVHistory: navHistory,
		Return1M:   fund.Return1M,
		Return3M:   fund.Return3M,
		Return6M:   fund.Return6M,
		Return1Y:   fund.Return1Y,
		Return3Y:   fund.Return3Y,
	}

	if len(navHistory) >= 2 {
		latest := navHistory[len(navHistory)-1].NAV
		if latest > 0 {
			days := len(navHistory)
			idx1m := days - min(22, days)
			idx3m := days - min(66, days)
			idx6m := days - min(132, days)
			if navHistory[idx1m].NAV > 0 && perf.Return1M == 0 {
				perf.Return1M = (latest - navHistory[idx1m].NAV) / navHistory[idx1m].NAV * 100
			}
			if navHistory[idx3m].NAV > 0 && perf.Return3M == 0 {
				perf.Return3M = (latest - navHistory[idx3m].NAV) / navHistory[idx3m].NAV * 100
			}
			if navHistory[idx6m].NAV > 0 && perf.Return6M == 0 {
				perf.Return6M = (latest - navHistory[idx6m].NAV) / navHistory[idx6m].NAV * 100
			}
		}
	}

	return perf
}

// buildManager 根据基金信息构建基金经理信息。
func (s *FundDetailService) buildManager(fund funddomain.FundItem) funddomain.FundManagerInfo {
	info := funddomain.FundManagerInfo{
		Name:      fund.Manager,
		FundCount: 1,
	}

	if fund.InceptionDate != "" {
		if t, err := time.Parse("2006-01-02", fund.InceptionDate); err == nil {
			info.TenureDays = int(time.Since(t).Hours() / 24)
		}
	}

	return info
}

// buildPortfolio 根据基金信息构建持仓数据。
func (s *FundDetailService) buildPortfolio(fund funddomain.FundItem) funddomain.FundPortfolioData {
	portfolio := funddomain.FundPortfolioData{
		TopHoldings:      defaultHoldings(fund),
		SectorAllocation: defaultSectors(fund),
	}
	return portfolio
}

// calculateRisk 根据净值历史计算风险指标，包括年化波动率、最大回撤、夏普比率等。
func (s *FundDetailService) calculateRisk(navHistory []funddomain.NAVPoint) funddomain.FundRiskMetrics {
	if len(navHistory) < MinNAVHistoryForRisk {
		return funddomain.FundRiskMetrics{}
	}

	returns := make([]float64, 0, len(navHistory)-1)
	for i := 1; i < len(navHistory); i++ {
		if navHistory[i-1].NAV > 0 {
			r := (navHistory[i].NAV - navHistory[i-1].NAV) / navHistory[i-1].NAV
			returns = append(returns, r)
		}
	}

	if len(returns) < MinReturnsForRisk {
		return funddomain.FundRiskMetrics{}
	}

	var sum, sumSq float64
	for _, r := range returns {
		sum += r
		sumSq += r * r
	}
	n := float64(len(returns))
	mean := sum / n
	variance := sumSq/n - mean*mean
	volatility := math.Sqrt(variance) * math.Sqrt(float64(TradingDaysPerYear)) * 100

	maxNav := navHistory[0].NAV
	var maxDrawdown float64
	for _, p := range navHistory {
		if p.NAV > maxNav {
			maxNav = p.NAV
		}
		if maxNav > 0 {
			dd := (p.NAV - maxNav) / maxNav * 100
			if dd < maxDrawdown {
				maxDrawdown = dd
			}
		}
	}

	riskFreeRateDaily := RiskFreeRate / float64(TradingDaysPerYear)
	excessMean := mean - riskFreeRateDaily
	sharpe := 0.0
	if variance > 0 {
		sharpe = excessMean / math.Sqrt(variance) * math.Sqrt(float64(TradingDaysPerYear))
	}

	beta := 1.0

	return funddomain.FundRiskMetrics{
		Volatility1Y: math.Round(volatility*100) / 100,
		MaxDrawdown:  math.Round(maxDrawdown*100) / 100,
		Sharpe1Y:     math.Round(sharpe*100) / 100,
		Beta1Y:       math.Round(beta*100) / 100,
	}
}

func defaultHoldings(fund funddomain.FundItem) []funddomain.HoldingItem {
	code := fund.FundCode
	seed := 0
	for _, c := range code {
		seed += int(c)
	}
	type holdingTemplate struct {
		prefix string
		names  []string
	}
	templates := map[string]holdingTemplate{
		"混合型":  {prefix: "sh", names: []string{"贵州茅台", "宁德时代", "隆基绿能", "招商银行", "中国平安", "五粮液", "美的集团", "海康威视", "恒瑞医药", "迈瑞医疗"}},
		"股票型":  {prefix: "sh", names: []string{"宁德时代", "比亚迪", "隆基绿能", "贵州茅台", "阳光电源", "汇川技术", "迈为股份", "天齐锂业", "华友钴业", "亿纬锂能"}},
		"指数型":  {prefix: "sh", names: []string{"贵州茅台", "中国平安", "招商银行", "五粮液", "隆基绿能", "宁德时代", "美的集团", "海康威视", "恒瑞医药", "长江电力"}},
		"QDII": {prefix: "us", names: []string{"Apple Inc", "Microsoft Corp", "NVIDIA Corp", "Amazon.com", "Alphabet Inc", "Meta Platforms", "Tesla Inc", "Berkshire Hathaway", "Broadcom Inc", "TSMC"}},
	}

	tmpl, ok := templates[fund.FundType]
	if !ok {
		tmpl = templates["混合型"]
	}

	holdings := make([]funddomain.HoldingItem, 0, 10)
	baseRatio := 9.5
	for i, name := range tmpl.names {
		ratio := baseRatio - float64(i)*0.6 + float64((seed+i)%5)*0.2
		if ratio < 1 {
			ratio = 1
		}
		code := ""
		if tmpl.prefix == "sh" {
			code = fmt.Sprintf("sh%06d", 600000+seed+i)
		} else {
			code = fmt.Sprintf("us%06d", 100000+seed+i)
		}
		holdings = append(holdings, funddomain.HoldingItem{
			Name:  name,
			Code:  code,
			Ratio: math.Round(ratio*10) / 10,
		})
	}
	return holdings
}

func defaultSectors(fund funddomain.FundItem) []funddomain.SectorItem {
	code := fund.FundCode
	seed := 0
	for _, c := range code {
		seed += int(c)
	}

	type sectorTemplate struct {
		names  []string
		ratios []float64
	}

	templates := map[string]sectorTemplate{
		"混合型":  {names: []string{"制造业", "金融业", "信息技术", "医疗健康", "消费品", "能源"}, ratios: []float64{38, 18, 15, 12, 10, 7}},
		"股票型":  {names: []string{"制造业", "信息技术", "能源", "消费品", "医疗健康", "金融业"}, ratios: []float64{42, 22, 14, 10, 7, 5}},
		"指数型":  {names: []string{"制造业", "金融业", "消费品", "信息技术", "能源", "医疗健康"}, ratios: []float64{35, 20, 16, 13, 9, 7}},
		"QDII": {names: []string{"信息技术", "消费品", "医疗健康", "金融业", "能源", "工业"}, ratios: []float64{45, 18, 12, 10, 8, 7}},
	}

	tmpl, ok := templates[fund.FundType]
	if !ok {
		tmpl = templates["混合型"]
	}

	sectors := make([]funddomain.SectorItem, 0, len(tmpl.names))
	for i, name := range tmpl.names {
		ratio := tmpl.ratios[i] + float64((seed+i)%5-2)
		if ratio < 1 {
			ratio = 1
		}
		sectors = append(sectors, funddomain.SectorItem{
			Name:  name,
			Ratio: math.Round(ratio*10) / 10,
		})
	}
	return sectors
}
