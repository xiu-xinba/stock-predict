package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	marketdomain "stock-predict-go/internal/domain/market"
	stockdomain "stock-predict-go/internal/domain/stock"
	database "stock-predict-go/internal/infrastructure/database"
	httpclient "stock-predict-go/internal/platform/httpclient"
)

const (
	eastmoneyCapitalFlowURL = "https://push2his.eastmoney.com/api/qt/stock/fflow/daykline/get" // 东方财富资金流向 API
	eastmoneyFinancialURL   = "https://datacenter-web.eastmoney.com/api/data/v1/get"           // 东方财富财务数据 API（带 -web）
	eastmoneyShareholderURL = "https://datacenter-web.eastmoney.com/api/data/v1/get"           // 东方财富股东数据 API（带 -web）
	eastmoneyReportURL      = "https://reportapi.eastmoney.com/report/list"                    // 东方财富研报 API
)

// StockDetailService 股票详情服务，提供股票详情、K线、资金流向、财务数据、股东信息等。
type StockDetailService struct {
	stocks      *StockService
	quote       *StockQuoteClient
	indexQuote  *IndexQuoteClient
	logger      *slog.Logger
	client      *http.Client
	resilient   *ResilientHTTPClient
	cache       *DetailCache
	marketStore *database.MarketStore
}

// NewStockDetailService 创建新的股票详情服务实例。
func NewStockDetailService(stocks *StockService, quote *StockQuoteClient, indexQuote *IndexQuoteClient, logger *slog.Logger) *StockDetailService {
	if logger == nil {
		logger = slog.Default()
	}
	return &StockDetailService{
		stocks:     stocks,
		quote:      quote,
		indexQuote: indexQuote,
		logger:     logger,
		client:     NewHTTPClient(HTTPClientConfig{}),
		resilient:  NewResilientHTTPClient(NewHTTPClient(HTTPClientConfig{}), nil),
		cache:      NewDetailCache(CacheMaxEntries, CacheTTL),
	}
}

// SetMarketStore 注入 MarketStore 用于财务数据缓存。
func (s *StockDetailService) SetMarketStore(ms *database.MarketStore) {
	s.marketStore = ms
}

// GetDetail 获取股票详情，包含基本信息、实时行情、K线、资金流向、财务数据、股东信息等。
func (s *StockDetailService) GetDetail(ctx context.Context, code string) (stockdomain.StockDetailData, error) {
	if len(code) != 6 || !httpclient.IsAllDigits(code) {
		return stockdomain.StockDetailData{}, ErrInvalidStockCode
	}

	if cached, ok := s.cache.Get(code); ok {
		if val, ok2 := cached.(stockdomain.StockDetailData); ok2 {
			return val, nil
		}
	}

	stock, err := s.stocks.FindStock(code)
	if err != nil {
		return stockdomain.StockDetailData{}, fmt.Errorf("find stock: %w", err)
	}

	basic := s.buildBasicInfo(stock, code)

	quoteMap := s.quote.FetchQuotesWithOptions(ctx, []string{code}, StockQuoteOptions{Freshness: StockQuoteFreshnessRealtime})
	quoteData := stockdomain.StockQuote{}
	if q, ok := quoteMap[code]; ok {
		quoteData = q
	}

	var (
		wg               sync.WaitGroup
		kline            stockdomain.StockKlineData
		capitalFlow      stockdomain.StockCapitalFlow
		financials       stockdomain.StockFinancials
		shareholders     stockdomain.StockShareholders
		minuteData       []marketdomain.IndexMinutePoint
		research         stockdomain.StockResearch
		dividends        stockdomain.StockDividends
		margin           stockdomain.StockMargin
		shareholderTrend stockdomain.StockShareholderTrend
		restricted       stockdomain.StockRestricted
	)

	wg.Add(10)
	go func() { defer wg.Done(); kline = s.fetchKlineData(ctx, code, "daily", 1) }()
	go func() { defer wg.Done(); capitalFlow = s.fetchCapitalFlow(ctx, code) }()
	go func() { defer wg.Done(); financials = s.fetchFinancials(ctx, code) }()
	go func() { defer wg.Done(); shareholders = s.fetchShareholders(ctx, code) }()
	go func() { defer wg.Done(); minuteData = s.indexQuote.FetchStockMinute(ctx, code) }()
	go func() { defer wg.Done(); research = s.fetchResearch(ctx, code) }()
	go func() { defer wg.Done(); dividends = s.fetchDividends(ctx, code) }()
	go func() { defer wg.Done(); margin = s.fetchMargin(ctx, code) }()
	go func() { defer wg.Done(); shareholderTrend = s.fetchShareholderTrend(ctx, code) }()
	go func() { defer wg.Done(); restricted = s.fetchRestricted(ctx, code) }()
	wg.Wait()

	data := stockdomain.StockDetailData{
		Basic:            basic,
		Quote:            quoteData,
		Kline:            kline,
		MinuteData:       minuteData,
		CapitalFlow:      capitalFlow,
		Financials:       financials,
		Shareholders:     shareholders,
		Research:         research,
		Dividends:        dividends,
		Margin:           margin,
		ShareholderTrend: shareholderTrend,
		Restricted:       restricted,
	}

	s.cache.Set(code, data)

	return data, nil
}

// GetKline 单独获取股票 K 线数据，支持指定周期和复权类型。
// period: "daily"/"weekly"/"monthly"; fq: 0=不复权, 1=前复权, 2=后复权
func (s *StockDetailService) GetKline(ctx context.Context, code, period string, fq int) (stockdomain.StockKlineData, error) {
	if len(code) != 6 || !httpclient.IsAllDigits(code) {
		return stockdomain.StockKlineData{}, ErrInvalidStockCode
	}
	if period != "daily" && period != "weekly" && period != "monthly" {
		period = "daily"
	}
	if fq < 0 || fq > 2 {
		fq = 1
	}
	return s.fetchKlineData(ctx, code, period, fq), nil
}

// buildBasicInfo 根据股票信息构建基本信息。
func (s *StockDetailService) buildBasicInfo(stock stockdomain.StockItem, code string) stockdomain.StockBasicInfo {
	market := stock.Market
	if market == "" {
		market = stockMarketPrefix(code)
	}
	return stockdomain.StockBasicInfo{
		StockCode:   code,
		StockName:   stock.StockName,
		Market:      market,
		Industry:    stock.Industry,
		ListDate:    stock.ListDate,
		TotalShares: stock.TotalShares,
		FloatShares: stock.FloatShares,
	}
}

// fetchJSON 通过 HTTP GET 请求获取 JSON 数据。
func (s *StockDetailService) fetchJSON(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Referer", "https://quote.eastmoney.com/")
	resp, err := s.resilient.Do(ctx, SourceEastmoney, req)
	if err != nil {
		s.logger.Warn("HTTP fetch failed", "url", url, "error", err)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.logger.Warn("HTTP fetch non-2xx", "url", url, "status", resp.StatusCode)
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, int64(MaxHTTPPayloadBytes)))
}

// fetchKlineData 从东方财富获取股票 K 线数据。
// fq: 0=不复权, 1=前复权(默认), 2=后复权
func (s *StockDetailService) fetchKlineData(ctx context.Context, code, period string, fq int) stockdomain.StockKlineData {
	market := stockMarketPrefix(code)
	if market == "" {
		return stockdomain.StockKlineData{Period: period}
	}

	secid := fmt.Sprintf("%d.%s", marketToSecID(market), code)
	klineURL := "https://push2his.eastmoney.com" + "/api/qt/stock/kline/get"
	// klt: 101=日K, 102=周K, 103=月K; fqt: 0=不复权, 1=前复权, 2=后复权
	klt := 101
	switch period {
	case "weekly":
		klt = 102
	case "monthly":
		klt = 103
	}
	if fq < 0 || fq > 2 {
		fq = 1
	}
	url := fmt.Sprintf("%s?secid=%s&fields1=f1,f2,f3,f4,f5,f6&fields2=f51,f52,f53,f54,f55,f56,f57&klt=%d&fqt=%d&end=20500101&lmt=120", klineURL, secid, klt, fq)

	if !isAllowedURL(url) {
		s.logger.Warn("URL not in whitelist", "url", url)
		return stockdomain.StockKlineData{Period: period}
	}

	payload, err := s.fetchJSON(ctx, url)
	if err != nil {
		return stockdomain.StockKlineData{Period: period}
	}

	var result struct {
		Data struct {
			Klines []string `json:"klines"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &result); err != nil {
		return stockdomain.StockKlineData{Period: period}
	}
	if len(result.Data.Klines) == 0 {
		return stockdomain.StockKlineData{Period: period}
	}

	klines := make([]stockdomain.KlinePoint, 0, len(result.Data.Klines))
	for _, line := range result.Data.Klines {
		parts := strings.Split(line, ",")
		if len(parts) < 7 {
			continue
		}
		klines = append(klines, stockdomain.KlinePoint{
			Date:   parts[0],
			Open:   httpclient.ParseQuoteFloat(parts[1]),
			Close:  httpclient.ParseQuoteFloat(parts[2]),
			High:   httpclient.ParseQuoteFloat(parts[3]),
			Low:    httpclient.ParseQuoteFloat(parts[4]),
			Volume: httpclient.ParseQuoteFloat(parts[5]),
			Amount: httpclient.ParseQuoteFloat(parts[6]),
		})
	}

	return stockdomain.StockKlineData{
		Period: period,
		Klines: klines,
	}
}

// fetchCapitalFlow 从东方财富获取股票资金流向数据。
func (s *StockDetailService) fetchCapitalFlow(ctx context.Context, code string) stockdomain.StockCapitalFlow {
	market := stockMarketPrefix(code)
	if market == "" {
		return stockdomain.StockCapitalFlow{}
	}

	secid := fmt.Sprintf("%d.%s", marketToSecID(market), code)
	url := fmt.Sprintf("%s?secid=%s&fields1=f1,f2,f3&fields2=f51,f52,f53,f54,f55,f56&lmt=30", eastmoneyCapitalFlowURL, secid)

	if !isAllowedURL(url) {
		s.logger.Warn("URL not in whitelist", "url", url)
		return stockdomain.StockCapitalFlow{}
	}

	payload, err := s.fetchJSON(ctx, url)
	if err != nil {
		return stockdomain.StockCapitalFlow{}
	}

	var result struct {
		Data struct {
			Klines []string `json:"klines"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &result); err != nil {
		return stockdomain.StockCapitalFlow{}
	}

	history := make([]stockdomain.CapitalFlowPoint, 0, len(result.Data.Klines))
	var mainNetInflow, retailNetInflow float64
	for i, line := range result.Data.Klines {
		parts := strings.Split(line, ",")
		if len(parts) < 6 {
			continue
		}
		mainIn := httpclient.ParseQuoteFloat(parts[1])
		mainOut := httpclient.ParseQuoteFloat(parts[2])
		retailIn := httpclient.ParseQuoteFloat(parts[3])
		retailOut := httpclient.ParseQuoteFloat(parts[4])
		netIn := httpclient.ParseQuoteFloat(parts[5])
		history = append(history, stockdomain.CapitalFlowPoint{
			Date:          parts[0],
			MainInflow:    mainIn,
			MainOutflow:   mainOut,
			RetailInflow:  retailIn,
			RetailOutflow: retailOut,
			NetInflow:     netIn,
		})
		if i == len(result.Data.Klines)-1 {
			mainNetInflow = mainIn - mainOut
			retailNetInflow = retailIn - retailOut
		}
	}

	return stockdomain.StockCapitalFlow{
		MainNetInflow:   mainNetInflow,
		RetailNetInflow: retailNetInflow,
		FlowHistory:     history,
	}
}

// fetchFinancials 从东方财富获取股票财务数据。
func (s *StockDetailService) fetchFinancials(ctx context.Context, code string) stockdomain.StockFinancials {
	// Check database cache first (financial data updates quarterly)
	if s.marketStore != nil {
		cached, err := s.marketStore.GetFinancials(code)
		if err == nil && len(cached) > 0 {
			latest := cached[0]
			// Check if cache is recent enough (within 90 days = one quarter)
			if latest.ReportDate != "" {
				return stockdomain.StockFinancials{
					ROE:         latest.ROE,
					Revenue:     latest.Revenue,
					NetProfit:   latest.NetProfit,
					EPS:         latest.EPS,
					GrossMargin: latest.GrossMargin,
					NetMargin:   latest.NetMargin,
					Quarterly:   cached,
				}
			}
		}
	}

	market := stockMarketPrefix(code)
	if market == "" {
		return stockdomain.StockFinancials{}
	}

	url := fmt.Sprintf("%s?reportName=RPT_LICO_FN_CPD&columns=REPORT_DATE,TOTAL_OPERATE_INCOME,PARENT_NETPROFIT,BASIC_EPS,GROSS_PROFIT_RATIO,NETPROFIT_MARGIN,ROE_JQK&filter=(SECURITY_CODE%%3D%%22%s%%22)&pageSize=4&sortColumns=REPORT_DATE&sortTypes=-1", eastmoneyFinancialURL, code)

	if !isAllowedURL(url) {
		s.logger.Warn("URL not in whitelist", "url", url)
		return stockdomain.StockFinancials{}
	}

	payload, err := s.fetchJSON(ctx, url)
	if err != nil {
		return stockdomain.StockFinancials{}
	}

	var rawResp struct {
		Result struct {
			Data []struct {
				ReportDate  string  `json:"REPORT_DATE"`
				Revenue     float64 `json:"TOTAL_OPERATE_INCOME"`
				NetProfit   float64 `json:"PARENT_NETPROFIT"`
				EPS         float64 `json:"BASIC_EPS"`
				GrossMargin float64 `json:"GROSS_PROFIT_RATIO"`
				NetMargin   float64 `json:"NETPROFIT_MARGIN"`
				ROE         float64 `json:"ROE_JQK"`
			} `json:"data"`
		} `json:"result"`
	}
	if err := json.Unmarshal(payload, &rawResp); err != nil {
		return stockdomain.StockFinancials{}
	}

	quarterly := make([]stockdomain.FinancialQuarter, 0, len(rawResp.Result.Data))
	var latest stockdomain.FinancialQuarter
	for i, d := range rawResp.Result.Data {
		reportDate := d.ReportDate
		if idx := strings.Index(reportDate, "T"); idx > 0 {
			reportDate = reportDate[:idx]
		}
		q := stockdomain.FinancialQuarter{
			ReportDate:  reportDate,
			Revenue:     d.Revenue,
			NetProfit:   d.NetProfit,
			EPS:         d.EPS,
			GrossMargin: d.GrossMargin,
			NetMargin:   d.NetMargin,
			ROE:         d.ROE,
		}
		quarterly = append(quarterly, q)
		if i == 0 {
			latest = q
		}
	}

	result := stockdomain.StockFinancials{
		ROE:         latest.ROE,
		Revenue:     latest.Revenue,
		NetProfit:   latest.NetProfit,
		EPS:         latest.EPS,
		GrossMargin: latest.GrossMargin,
		NetMargin:   latest.NetMargin,
		Quarterly:   quarterly,
	}

	// Async save to database cache
	if s.marketStore != nil && len(quarterly) > 0 {
		go func() {
			if err := s.marketStore.SaveFinancials(code, quarterly); err != nil {
				s.logger.Warn("failed to save financials to cache store", "code", code, "error", err)
			}
		}()
	}

	return result
}

// fetchShareholders 从东方财富获取股票股东信息。
func (s *StockDetailService) fetchShareholders(ctx context.Context, code string) stockdomain.StockShareholders {
	url := fmt.Sprintf("%s?reportName=RPT_F10_EH_HOLDERSNUM&columns=END_DATE,HOLDER_NUM&filter=(SECURITY_CODE%%3D%%22%s%%22)&pageSize=1&sortColumns=END_DATE&sortTypes=-1", eastmoneyShareholderURL, code)

	if !isAllowedURL(url) {
		s.logger.Warn("URL not in whitelist", "url", url)
		return stockdomain.StockShareholders{}
	}

	payload, err := s.fetchJSON(ctx, url)
	if err != nil {
		return stockdomain.StockShareholders{}
	}

	var result struct {
		Result struct {
			Data []struct {
				EndDate   string `json:"END_DATE"`
				HolderNum int    `json:"HOLDER_NUM"`
			} `json:"data"`
		} `json:"result"`
	}
	if err := json.Unmarshal(payload, &result); err != nil {
		return stockdomain.StockShareholders{}
	}

	institutionCount := 0
	if len(result.Result.Data) > 0 {
		institutionCount = result.Result.Data[0].HolderNum
	}

	top10 := s.fetchTop10Shareholders(ctx, code)

	return stockdomain.StockShareholders{
		Top10:            top10,
		InstitutionCount: institutionCount,
		InstitutionRatio: 0,
	}
}

// fetchTop10Shareholders 从东方财富获取前十大股东数据。
func (s *StockDetailService) fetchTop10Shareholders(ctx context.Context, code string) []stockdomain.ShareholderItem {
	url := fmt.Sprintf("%s?reportName=RPT_F10_EH_TOP10&columns=HOLDER_NAME,HOLD_RATIO,HOLD_CHANGE,HOLD_TYPE&filter=(SECURITY_CODE%%3D%%22%s%%22)&pageSize=10&sortColumns=HOLD_RATIO&sortTypes=-1", eastmoneyShareholderURL, code)

	if !isAllowedURL(url) {
		s.logger.Warn("URL not in whitelist", "url", url)
		return nil
	}

	payload, err := s.fetchJSON(ctx, url)
	if err != nil {
		return nil
	}

	var result struct {
		Result struct {
			Data []struct {
				Name      string  `json:"HOLDER_NAME"`
				Ratio     float64 `json:"HOLD_RATIO"`
				Change    float64 `json:"HOLD_CHANGE"`
				ShareType string  `json:"HOLD_TYPE"`
			} `json:"data"`
		} `json:"result"`
	}
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil
	}

	items := make([]stockdomain.ShareholderItem, 0, len(result.Result.Data))
	for _, d := range result.Result.Data {
		items = append(items, stockdomain.ShareholderItem{
			Name:      d.Name,
			Ratio:     d.Ratio,
			Change:    d.Change,
			ShareType: d.ShareType,
		})
	}
	return items
}

// fetchResearch 从东方财富获取研报评级数据。
func (s *StockDetailService) fetchResearch(ctx context.Context, code string) stockdomain.StockResearch {
	// reportapi.eastmoney.com 使用不同的响应结构（直接 {data: [...]}）
	url := fmt.Sprintf("%s?industryCode=*&pageSize=10&industry=*&rating=*&ratingChange=*&beginTime=2024-01-01&endTime=2026-12-31&pageNo=1&qType=0&code=%s", eastmoneyReportURL, code)

	if !isAllowedURL(url) {
		s.logger.Warn("URL not in whitelist", "url", url)
		return stockdomain.StockResearch{}
	}

	payload, err := s.fetchJSON(ctx, url)
	if err != nil {
		return stockdomain.StockResearch{}
	}

	var result struct {
		Data []struct {
			Title              string `json:"title"`
			OrgName            string `json:"orgName"`
			PublishDate        string `json:"publishDate"`
			EmRatingName       string `json:"emRatingName"`
			ResearchName       string `json:"researchName"`
			PredictThisYearPe  string `json:"predictThisYearPe"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &result); err != nil {
		return stockdomain.StockResearch{}
	}

	reports := make([]stockdomain.ResearchReport, 0, len(result.Data))
	for _, d := range result.Data {
		dateStr := d.PublishDate
		if len(dateStr) > 10 {
			dateStr = dateStr[:10]
		}
		// predictThisYearPe 是字符串，尝试解析为 float64
		var targetPrice float64
		if d.PredictThisYearPe != "" {
			fmt.Sscanf(d.PredictThisYearPe, "%f", &targetPrice)
		}
		reports = append(reports, stockdomain.ResearchReport{
			Date:        dateStr,
			OrgName:     d.OrgName,
			Rating:      d.EmRatingName,
			TargetPrice: targetPrice,
			Researcher:  d.ResearchName,
		})
	}

	latestRating := ""
	if len(reports) > 0 {
		latestRating = reports[0].Rating
	}

	return stockdomain.StockResearch{
		LatestRating: latestRating,
		RatingCount:  len(reports),
		Reports:      reports,
	}
}

// fetchDividends 从东方财富获取分红送配数据。
func (s *StockDetailService) fetchDividends(ctx context.Context, code string) stockdomain.StockDividends {
	url := fmt.Sprintf("%s?reportName=RPT_SHAREBONUS_DET&columns=EX_DIVIDEND_DATE,BONUS_RATIO,IT_RATIO,PRETAX_BONUS_RMB,ASSIGN_PROGRESS&filter=(SECURITY_CODE%%3D%%22%s%%22)&pageSize=20&sortColumns=EX_DIVIDEND_DATE&sortTypes=-1", eastmoneyFinancialURL, code)

	if !isAllowedURL(url) {
		s.logger.Warn("URL not in whitelist", "url", url)
		return stockdomain.StockDividends{}
	}

	payload, err := s.fetchJSON(ctx, url)
	if err != nil {
		return stockdomain.StockDividends{}
	}

	var result struct {
		Result struct {
			Data []struct {
				ExDividendDate string  `json:"EX_DIVIDEND_DATE"`
				BonusRatio     float64 `json:"BONUS_RATIO"`
				ITRatio        float64 `json:"IT_RATIO"`
				PretaxBonusRmb float64 `json:"PRETAX_BONUS_RMB"`
				AssignProgress string  `json:"ASSIGN_PROGRESS"`
			} `json:"data"`
		} `json:"result"`
	}
	if err := json.Unmarshal(payload, &result); err != nil {
		return stockdomain.StockDividends{}
	}

	records := make([]stockdomain.DividendRecord, 0, len(result.Result.Data))
	totalDividend := 0.0
	for _, d := range result.Result.Data {
		dateStr := d.ExDividendDate
		if len(dateStr) > 10 {
			dateStr = dateStr[:10]
		}
		records = append(records, stockdomain.DividendRecord{
			Date:     dateStr,
			Bonus:    d.BonusRatio,
			Transfer: d.ITRatio,
			Dividend: d.PretaxBonusRmb,
			Progress: d.AssignProgress,
		})
		totalDividend += d.PretaxBonusRmb
	}

	return stockdomain.StockDividends{
		TotalDividend: totalDividend,
		Records:       records,
	}
}

// fetchMargin 从东方财富获取融资融券数据。
func (s *StockDetailService) fetchMargin(ctx context.Context, code string) stockdomain.StockMargin {
	url := fmt.Sprintf("%s?reportName=RPTA_WEB_RZRQ_GGMX&columns=DATE,RZYE,RZMRE,RQYE,RQYL&filter=(SCODE%%3D%%22%s%%22)&pageSize=30&sortColumns=DATE&sortTypes=-1", eastmoneyFinancialURL, code)

	if !isAllowedURL(url) {
		s.logger.Warn("URL not in whitelist", "url", url)
		return stockdomain.StockMargin{}
	}

	payload, err := s.fetchJSON(ctx, url)
	if err != nil {
		return stockdomain.StockMargin{}
	}

	var result struct {
		Result struct {
			Data []struct {
				Date  string  `json:"DATE"`
				RZYE  float64 `json:"RZYE"`
				RZMRE float64 `json:"RZMRE"`
				RQYE  float64 `json:"RQYE"`
				RQYL  float64 `json:"RQYL"`
			} `json:"data"`
		} `json:"result"`
	}
	if err := json.Unmarshal(payload, &result); err != nil {
		return stockdomain.StockMargin{}
	}

	history := make([]stockdomain.MarginData, 0, len(result.Result.Data))
	latestBalance := 0.0
	for i, d := range result.Result.Data {
		dateStr := d.Date
		if len(dateStr) > 10 {
			dateStr = dateStr[:10]
		}
		history = append(history, stockdomain.MarginData{
			Date:          dateStr,
			MarginBalance: d.RZYE,
			MarginBuy:     d.RZMRE,
			ShortBalance:  d.RQYE,
			ShortVolume:   d.RQYL,
		})
		if i == 0 {
			latestBalance = d.RZYE
		}
	}

	return stockdomain.StockMargin{
		LatestMarginBalance: latestBalance,
		History:             history,
	}
}

// fetchShareholderTrend 从东方财富获取股东人数变化趋势数据。
func (s *StockDetailService) fetchShareholderTrend(ctx context.Context, code string) stockdomain.StockShareholderTrend {
	url := fmt.Sprintf("%s?reportName=RPT_HOLDERNUM_DET&columns=END_DATE,HOLDER_NUM,AVG_HOLD_NUM,HOLDER_NUM_CHANGE&filter=(SECURITY_CODE%%3D%%22%s%%22)&pageSize=8&sortColumns=END_DATE&sortTypes=-1", eastmoneyShareholderURL, code)

	if !isAllowedURL(url) {
		s.logger.Warn("URL not in whitelist", "url", url)
		return stockdomain.StockShareholderTrend{}
	}

	payload, err := s.fetchJSON(ctx, url)
	if err != nil {
		return stockdomain.StockShareholderTrend{}
	}

	var result struct {
		Result struct {
			Data []struct {
				EndDate         string  `json:"END_DATE"`
				HolderNum       float64 `json:"HOLDER_NUM"`
				AvgHoldNum      float64 `json:"AVG_HOLD_NUM"`
				HolderNumChange float64 `json:"HOLDER_NUM_CHANGE"`
			} `json:"data"`
		} `json:"result"`
	}
	if err := json.Unmarshal(payload, &result); err != nil {
		return stockdomain.StockShareholderTrend{}
	}

	trend := make([]stockdomain.ShareholderTrendPoint, 0, len(result.Result.Data))
	latestCount := 0.0
	for i, d := range result.Result.Data {
		dateStr := d.EndDate
		if len(dateStr) > 10 {
			dateStr = dateStr[:10]
		}
		trend = append(trend, stockdomain.ShareholderTrendPoint{
			Date:       dateStr,
			Count:      d.HolderNum,
			AvgHolding: d.AvgHoldNum,
			Change:     d.HolderNumChange,
		})
		if i == 0 {
			latestCount = d.HolderNum
		}
	}

	return stockdomain.StockShareholderTrend{
		LatestCount: latestCount,
		Trend:       trend,
	}
}

// fetchRestricted 从 AKShare 服务获取限售解禁数据。
func (s *StockDetailService) fetchRestricted(ctx context.Context, code string) stockdomain.StockRestricted {
	akshareURL := os.Getenv("AKSHARE_URL")
	if akshareURL == "" {
		akshareURL = "http://localhost:8900"
	}
	akshareToken := os.Getenv("AKSHARE_SERVICE_TOKEN")

	url := fmt.Sprintf("%s/api/v1/stock/restricted?code=%s", akshareURL, code)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return stockdomain.StockRestricted{}
	}
	if akshareToken != "" {
		req.Header.Set("Authorization", "Bearer "+akshareToken)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Warn("fetchRestricted HTTP failed", "url", url, "error", err)
		return stockdomain.StockRestricted{}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.logger.Warn("fetchRestricted non-2xx", "url", url, "status", resp.StatusCode)
		return stockdomain.StockRestricted{}
	}

	payload, err := io.ReadAll(io.LimitReader(resp.Body, int64(MaxHTTPPayloadBytes)))
	if err != nil {
		return stockdomain.StockRestricted{}
	}

	var apiResp struct {
		Code int `json:"code"`
		Data []struct {
			Date         string  `json:"date"`
			Volume       float64 `json:"volume"`
			MarketValue  float64 `json:"market_value"`
			Batch        int     `json:"batch"`
			AnnounceDate string  `json:"announce_date"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &apiResp); err != nil {
		return stockdomain.StockRestricted{}
	}

	history := make([]stockdomain.RestrictedRelease, 0, len(apiResp.Data))
	var nextRelease *stockdomain.RestrictedRelease
	now := time.Now().Format("2006-01-02")
	for _, d := range apiResp.Data {
		item := stockdomain.RestrictedRelease{
			Date:   d.Date,
			Volume: d.Volume,
			Ratio:  d.MarketValue, // 复用 Ratio 字段存储市值（亿元）
			Type:   fmt.Sprintf("第%d批", d.Batch),
		}
		if d.Date >= now && nextRelease == nil {
			nextRelease = &item
		} else {
			history = append(history, item)
		}
	}

	return stockdomain.StockRestricted{
		NextRelease: nextRelease,
		History:     history,
	}
}
