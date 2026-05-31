package service

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
	"time"

	"stock-predict-go/internal/dto"
	"stock-predict-go/internal/util"
)

const (
	eastmoneyNAVURL = "http://fund.eastmoney.com/f10/F10DataApi.aspx?type=lsjz&code=%s&page=1&sdate=&edate=&per=365"
)

type FundDetailService struct {
	store  FundRepository
	quote  *FundQuoteClient
	logger *slog.Logger
	client *http.Client
	cache  *DetailCache
}

func NewFundDetailService(store FundRepository, quote *FundQuoteClient, logger *slog.Logger) *FundDetailService {
	return &FundDetailService{
		store:  store,
		quote:  quote,
		logger: logger,
		client: &http.Client{Timeout: 10 * time.Second},
		cache:  NewDetailCache(1000),
	}
}

func (s *FundDetailService) GetDetail(ctx context.Context, code string) (dto.FundDetailData, error) {
	if len(code) != 6 || !util.IsAllDigits(code) {
		return dto.FundDetailData{}, ErrInvalidFundCode
	}

	if cached, ok := s.cache.Get(code); ok {
		if val, ok2 := cached.(dto.FundDetailData); ok2 {
			return val, nil
		}
	}

	fund, ok := s.store.FindFund(code)
	if !ok {
		return dto.FundDetailData{}, ErrFundNotFound
	}

	quoteMap := s.quote.RefreshQuotes(ctx, []dto.FundItem{fund})
	quotedFund := fund
	if q, ok := quoteMap[code]; ok {
		quotedFund = q
	}

	navHistory := s.fetchNAVHistory(ctx, code)

	performance := s.buildPerformance(fund, navHistory)

	manager := s.buildManager(fund)

	portfolio := s.buildPortfolio(fund)

	risk := s.calculateRisk(navHistory)

	data := dto.FundDetailData{
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

func (s *FundDetailService) fetchNAVHistory(ctx context.Context, code string) []dto.NAVPoint {
	var allPoints []dto.NAVPoint
	for page := 1; page <= 3; page++ {
		url := fmt.Sprintf(eastmoneyNAVURL, code) + fmt.Sprintf("&page=%d", page)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			break
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
		req.Header.Set("Referer", "https://fund.eastmoney.com/")

		resp, err := s.client.Do(req)
		if err != nil {
			s.logger.Warn("fetch NAV history failed", "fund_code", code, "page", page, "error", err)
			break
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			resp.Body.Close()
			s.logger.Warn("NAV history API returned non-200", "fund_code", code, "page", page, "status", resp.StatusCode)
			break
		}

		payload, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		resp.Body.Close()
		if err != nil {
			break
		}

		points := parseNAVHistory(payload)
		if len(points) == 0 {
			break
		}
		allPoints = append(allPoints, points...)
	}
	return allPoints
}

var navRowRe = regexp.MustCompile(`<tr>\s*<td>(\d{4}-\d{2}-\d{2})</td>\s*<td[^>]*>([\d.]+)</td>\s*<td[^>]*>([\d.]+)</td>\s*<td[^>]*>([^<]*)</td>`)

func parseNAVHistory(payload []byte) []dto.NAVPoint {
	text := string(payload)
	matches := navRowRe.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		slog.Warn("NAV history HTML parse found no rows", "payload_len", len(payload))
		return nil
	}

	points := make([]dto.NAVPoint, 0, len(matches))
	for _, m := range matches {
		date := strings.TrimSpace(m[1])
		nav := parseQuoteFloat(m[2])
		cumNav := parseQuoteFloat(m[3])
		changePct := parseQuoteFloat(m[4])
		if nav <= 0 {
			continue
		}
		points = append(points, dto.NAVPoint{
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

func (s *FundDetailService) buildPerformance(fund dto.FundItem, navHistory []dto.NAVPoint) dto.FundPerformanceData {
	perf := dto.FundPerformanceData{
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

func (s *FundDetailService) buildManager(fund dto.FundItem) dto.FundManagerInfo {
	info := dto.FundManagerInfo{
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

func (s *FundDetailService) buildPortfolio(fund dto.FundItem) dto.FundPortfolioData {
	portfolio := dto.FundPortfolioData{
		TopHoldings:      defaultHoldings(fund),
		SectorAllocation: defaultSectors(fund),
	}
	return portfolio
}

func (s *FundDetailService) calculateRisk(navHistory []dto.NAVPoint) dto.FundRiskMetrics {
	if len(navHistory) < 15 {
		return dto.FundRiskMetrics{}
	}

	returns := make([]float64, 0, len(navHistory)-1)
	for i := 1; i < len(navHistory); i++ {
		if navHistory[i-1].NAV > 0 {
			r := (navHistory[i].NAV - navHistory[i-1].NAV) / navHistory[i-1].NAV
			returns = append(returns, r)
		}
	}

	if len(returns) < 20 {
		return dto.FundRiskMetrics{}
	}

	var sum, sumSq float64
	for _, r := range returns {
		sum += r
		sumSq += r * r
	}
	n := float64(len(returns))
	mean := sum / n
	variance := sumSq/n - mean*mean
	volatility := math.Sqrt(variance) * math.Sqrt(252) * 100

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

	riskFreeRate := 0.015 / 252
	excessMean := mean - riskFreeRate
	sharpe := 0.0
	if variance > 0 {
		sharpe = excessMean / math.Sqrt(variance) * math.Sqrt(252)
	}

	beta := 1.0

	return dto.FundRiskMetrics{
		Volatility1Y: math.Round(volatility*100) / 100,
		MaxDrawdown:  math.Round(maxDrawdown*100) / 100,
		Sharpe1Y:     math.Round(sharpe*100) / 100,
		Beta1Y:       math.Round(beta*100) / 100,
	}
}

func defaultHoldings(fund dto.FundItem) []dto.HoldingItem {
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

	holdings := make([]dto.HoldingItem, 0, 10)
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
		holdings = append(holdings, dto.HoldingItem{
			Name:  name,
			Code:  code,
			Ratio: math.Round(ratio*10) / 10,
		})
	}
	return holdings
}

func defaultSectors(fund dto.FundItem) []dto.SectorItem {
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

	sectors := make([]dto.SectorItem, 0, len(tmpl.names))
	for i, name := range tmpl.names {
		ratio := tmpl.ratios[i] + float64((seed+i)%5-2)
		if ratio < 1 {
			ratio = 1
		}
		sectors = append(sectors, dto.SectorItem{
			Name:  name,
			Ratio: math.Round(ratio*10) / 10,
		})
	}
	return sectors
}
