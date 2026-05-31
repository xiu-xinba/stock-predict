package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"stock-predict-go/internal/dto"
	"stock-predict-go/internal/util"
)

const (
	eastmoneyCapitalFlowURL = "https://push2.eastmoney.com/api/qt/stock/fflow/daykline/get"
	eastmoneyFinancialURL   = "https://datacenter.eastmoney.com/securities/api/data/v1/get"
	eastmoneyShareholderURL = "https://datacenter.eastmoney.com/api/data/v1/get"
)

type StockDetailService struct {
	stocks *StockService
	quote  *StockQuoteClient
	logger *slog.Logger
	client *http.Client
	cache  *DetailCache
}

func NewStockDetailService(stocks *StockService, quote *StockQuoteClient, logger *slog.Logger) *StockDetailService {
	if logger == nil {
		logger = slog.Default()
	}
	return &StockDetailService{
		stocks: stocks,
		quote:  quote,
		logger: logger,
		client: &http.Client{Timeout: 10 * time.Second},
		cache:  NewDetailCache(1000, 5*time.Minute),
	}
}

func (s *StockDetailService) GetDetail(ctx context.Context, code string) (dto.StockDetailData, error) {
	if len(code) != 6 || !util.IsAllDigits(code) {
		return dto.StockDetailData{}, ErrInvalidStockCode
	}

	if cached, ok := s.cache.Get(code); ok {
		if val, ok2 := cached.(dto.StockDetailData); ok2 {
			return val, nil
		}
	}

	stock, err := s.stocks.FindStock(code)
	if err != nil {
		return dto.StockDetailData{}, fmt.Errorf("find stock: %w", err)
	}

	basic := s.buildBasicInfo(stock, code)

	quoteMap := s.quote.FetchQuotes(ctx, []string{code})
	quoteData := dto.StockQuote{}
	if q, ok := quoteMap[code]; ok {
		quoteData = q
	}

	kline := s.fetchKlineData(ctx, code, "daily")
	capitalFlow := s.fetchCapitalFlow(ctx, code)
	financials := s.fetchFinancials(ctx, code)
	shareholders := s.fetchShareholders(ctx, code)

	data := dto.StockDetailData{
		Basic:        basic,
		Quote:        quoteData,
		Kline:        kline,
		CapitalFlow:  capitalFlow,
		Financials:   financials,
		Shareholders: shareholders,
	}

	s.cache.Set(code, data)

	return data, nil
}

func (s *StockDetailService) buildBasicInfo(stock dto.StockItem, code string) dto.StockBasicInfo {
	market := stock.Market
	if market == "" {
		market = stockMarketPrefix(code)
	}
	return dto.StockBasicInfo{
		StockCode:   code,
		StockName:   stock.StockName,
		Market:      market,
		Industry:    stock.Industry,
		ListDate:    stock.ListDate,
		TotalShares: stock.TotalShares,
		FloatShares: stock.FloatShares,
	}
}

func (s *StockDetailService) fetchKlineData(ctx context.Context, code, period string) dto.StockKlineData {
	market := stockMarketPrefix(code)
	if market == "" {
		return dto.StockKlineData{Period: period}
	}

	secid := fmt.Sprintf("%d.%s", marketToSecID(market), code)
	klineURL := "https://push2his.eastmoney.com" + "/api/qt/stock/kline/get"
	url := fmt.Sprintf("%s?secid=%s&fields1=f1,f2,f3,f4,f5,f6&fields2=f51,f52,f53,f54,f55,f56,f57&klt=101&fqt=1&end=20500101&lmt=120", klineURL, secid)

	if !isAllowedURL(url) {
		s.logger.Warn("URL not in whitelist", "url", url)
		return dto.StockKlineData{Period: period}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return dto.StockKlineData{Period: period}
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
	req.Header.Set("Referer", "https://quote.eastmoney.com/")

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Warn("fetch kline failed", "code", code, "error", err)
		return dto.StockKlineData{Period: period}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return dto.StockKlineData{Period: period}
	}
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return dto.StockKlineData{Period: period}
	}

	var result struct {
		Data struct {
			Klines []string `json:"klines"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &result); err != nil {
		return dto.StockKlineData{Period: period}
	}
	if len(result.Data.Klines) == 0 {
		return dto.StockKlineData{Period: period}
	}

	klines := make([]dto.KlinePoint, 0, len(result.Data.Klines))
	for _, line := range result.Data.Klines {
		parts := strings.Split(line, ",")
		if len(parts) < 7 {
			continue
		}
		klines = append(klines, dto.KlinePoint{
			Date:   parts[0],
			Open:   util.ParseQuoteFloat(parts[1]),
			Close:  util.ParseQuoteFloat(parts[2]),
			High:   util.ParseQuoteFloat(parts[3]),
			Low:    util.ParseQuoteFloat(parts[4]),
			Volume: util.ParseQuoteFloat(parts[5]),
			Amount: util.ParseQuoteFloat(parts[6]),
		})
	}

	return dto.StockKlineData{
		Period: period,
		Klines: klines,
	}
}

func (s *StockDetailService) fetchCapitalFlow(ctx context.Context, code string) dto.StockCapitalFlow {
	market := stockMarketPrefix(code)
	if market == "" {
		return dto.StockCapitalFlow{}
	}

	secid := fmt.Sprintf("%d.%s", marketToSecID(market), code)
	url := fmt.Sprintf("%s?secid=%s&fields1=f1,f2,f3&fields2=f51,f52,f53,f54,f55,f56&lmt=30", eastmoneyCapitalFlowURL, secid)

	if !isAllowedURL(url) {
		s.logger.Warn("URL not in whitelist", "url", url)
		return dto.StockCapitalFlow{}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return dto.StockCapitalFlow{}
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
	req.Header.Set("Referer", "https://quote.eastmoney.com/")

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Warn("fetch capital flow failed", "code", code, "error", err)
		return dto.StockCapitalFlow{}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return dto.StockCapitalFlow{}
	}
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return dto.StockCapitalFlow{}
	}

	var result struct {
		Data struct {
			Klines []string `json:"klines"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &result); err != nil {
		return dto.StockCapitalFlow{}
	}

	history := make([]dto.CapitalFlowPoint, 0, len(result.Data.Klines))
	var mainNetInflow, retailNetInflow float64
	for i, line := range result.Data.Klines {
		parts := strings.Split(line, ",")
		if len(parts) < 6 {
			continue
		}
		mainIn := util.ParseQuoteFloat(parts[1])
		mainOut := util.ParseQuoteFloat(parts[2])
		retailIn := util.ParseQuoteFloat(parts[3])
		retailOut := util.ParseQuoteFloat(parts[4])
		netIn := util.ParseQuoteFloat(parts[5])
		history = append(history, dto.CapitalFlowPoint{
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

	return dto.StockCapitalFlow{
		MainNetInflow:   mainNetInflow,
		RetailNetInflow: retailNetInflow,
		FlowHistory:     history,
	}
}

func (s *StockDetailService) fetchFinancials(ctx context.Context, code string) dto.StockFinancials {
	market := stockMarketPrefix(code)
	if market == "" {
		return dto.StockFinancials{}
	}

	url := fmt.Sprintf("%s?reportName=RPT_LICO_FN_CPD&columns=REPORT_DATE,TOTAL_OPERATE_INCOME,PARENT_NETPROFIT,BASIC_EPS,GROSS_PROFIT_RATIO,NETPROFIT_MARGIN,ROE_JQK&filter=(SECURITY_CODE%%3D%%22%s%%22)&pageSize=4&sortColumns=REPORT_DATE&sortTypes=-1", eastmoneyFinancialURL, code)

	if !isAllowedURL(url) {
		s.logger.Warn("URL not in whitelist", "url", url)
		return dto.StockFinancials{}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return dto.StockFinancials{}
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
	req.Header.Set("Referer", "https://data.eastmoney.com/")

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Warn("fetch financials failed", "code", code, "error", err)
		return dto.StockFinancials{}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return dto.StockFinancials{}
	}
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return dto.StockFinancials{}
	}

	var result struct {
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
	if err := json.Unmarshal(payload, &result); err != nil {
		return dto.StockFinancials{}
	}

	quarterly := make([]dto.FinancialQuarter, 0, len(result.Result.Data))
	var latest dto.FinancialQuarter
	for i, d := range result.Result.Data {
		reportDate := d.ReportDate
		if idx := strings.Index(reportDate, "T"); idx > 0 {
			reportDate = reportDate[:idx]
		}
		q := dto.FinancialQuarter{
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

	return dto.StockFinancials{
		ROE:         latest.ROE,
		Revenue:     latest.Revenue,
		NetProfit:   latest.NetProfit,
		EPS:         latest.EPS,
		GrossMargin: latest.GrossMargin,
		NetMargin:   latest.NetMargin,
		Quarterly:   quarterly,
	}
}

func (s *StockDetailService) fetchShareholders(ctx context.Context, code string) dto.StockShareholders {
	url := fmt.Sprintf("%s?reportName=RPT_F10_EH_HOLDERSNUM&columns=END_DATE,HOLDER_NUM&filter=(SECURITY_CODE%%3D%%22%s%%22)&pageSize=1&sortColumns=END_DATE&sortTypes=-1", eastmoneyShareholderURL, code)

	if !isAllowedURL(url) {
		s.logger.Warn("URL not in whitelist", "url", url)
		return dto.StockShareholders{}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return dto.StockShareholders{}
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
	req.Header.Set("Referer", "https://data.eastmoney.com/")

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Warn("fetch shareholders failed", "code", code, "error", err)
		return dto.StockShareholders{}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return dto.StockShareholders{}
	}
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return dto.StockShareholders{}
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
		return dto.StockShareholders{}
	}

	institutionCount := 0
	if len(result.Result.Data) > 0 {
		institutionCount = result.Result.Data[0].HolderNum
	}

	top10 := s.fetchTop10Shareholders(ctx, code)

	return dto.StockShareholders{
		Top10:            top10,
		InstitutionCount: institutionCount,
		InstitutionRatio: 0,
	}
}

func (s *StockDetailService) fetchTop10Shareholders(ctx context.Context, code string) []dto.ShareholderItem {
	url := fmt.Sprintf("%s?reportName=RPT_F10_EH_TOP10&columns=HOLDER_NAME,HOLD_RATIO,HOLD_CHANGE,HOLD_TYPE&filter=(SECURITY_CODE%%3D%%22%s%%22)&pageSize=10&sortColumns=HOLD_RATIO&sortTypes=-1", eastmoneyShareholderURL, code)

	if !isAllowedURL(url) {
		s.logger.Warn("URL not in whitelist", "url", url)
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
	req.Header.Set("Referer", "https://data.eastmoney.com/")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil
	}
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
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

	items := make([]dto.ShareholderItem, 0, len(result.Result.Data))
	for _, d := range result.Result.Data {
		items = append(items, dto.ShareholderItem{
			Name:      d.Name,
			Ratio:     d.Ratio,
			Change:    d.Change,
			ShareType: d.ShareType,
		})
	}
	return items
}

func marketToSecID(market string) int {
	switch market {
	case "sh":
		return 1
	case "sz":
		return 0
	case "bj":
		return 0
	default:
		return 1
	}
}
