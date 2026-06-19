package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"stock-predict-go/internal/infrastructure/database"
	"stock-predict-go/internal/platform/config"
)

// HSGTScraper 从 AKShare 服务获取沪深港通资金流向数据
type HSGTScraper struct {
	httpClient *http.Client
	store      *database.HSGTFlowDailyStore
	cfg        config.Config
	logger     *slog.Logger
}

// NewHSGTScraper 创建爬虫实例
func NewHSGTScraper(store *database.HSGTFlowDailyStore, cfg config.Config, logger *slog.Logger) *HSGTScraper {
	return &HSGTScraper{
		httpClient: NewHTTPClient(HTTPClientConfig{Timeout: 30 * time.Second}),
		store:      store,
		cfg:        cfg,
		logger:     logger,
	}
}

// hsgtHistResponse AKShare HSGT 历史数据 API 响应
type hsgtHistResponse struct {
	Code int `json:"code"`
	Data []struct {
		Date       string  `json:"date"`
		NetBuy     float64 `json:"net_buy"`
		BuyAmount  float64 `json:"buy_amount"`
		SellAmount float64 `json:"sell_amount"`
		AccNetBuy  float64 `json:"acc_net_buy"`
		CashIn     float64 `json:"cash_in"`
		Balance    float64 `json:"balance"`
	} `json:"data"`
}

// FetchAndSaveAll 获取北向和南向资金数据并保存到数据库
func (s *HSGTScraper) FetchAndSaveAll(ctx context.Context) error {
	s.logger.Info("fetching HSGT data from AKShare service")

	// 获取北向资金整体数据（沪股通+深股通合计）
	northData, err := s.fetchHistData(ctx, "北向资金")
	if err != nil {
		s.logger.Error("failed to fetch 北向资金 data", "error", err)
		return fmt.Errorf("fetch 北向资金: %w", err)
	}

	// 获取南向资金整体数据（港股通沪+港股通深合计）
	southData, err := s.fetchHistData(ctx, "南向资金")
	if err != nil {
		s.logger.Error("failed to fetch 南向资金 data", "error", err)
		return fmt.Errorf("fetch 南向资金: %w", err)
	}

	// 同时获取细分数据用于填充沪股通/深股通/港股通沪/港股通深字段
	shData, _ := s.fetchHistData(ctx, "沪股通")
	szData, _ := s.fetchHistData(ctx, "深股通")
	hkSHData, _ := s.fetchHistData(ctx, "港股通沪")
	hkSZData, _ := s.fetchHistData(ctx, "港股通深")

	// 从东方财富 DataCenter API 获取北向成交额数据
	amtData, err := s.fetchNorthDealAmt(ctx, 2500)
	if err != nil {
		s.logger.Warn("failed to fetch north deal amount data, continuing without it", "error", err)
	}

	// 按日期合并数据
	merged := s.mergeData(northData, southData, shData, szData, hkSHData, hkSZData, amtData)

	// 保存到数据库
	saved := 0
	for date, flow := range merged {
		// 检查是否已存在
		existing, _ := s.store.GetByDate(date)
		if existing != nil {
			// 更新已有数据
			existing.NorthSHBuy = flow.NorthSHBuy
			existing.NorthSZBuy = flow.NorthSZBuy
			existing.NorthTotalBuy = flow.NorthTotalBuy
			existing.NorthTotalAmt = flow.NorthTotalAmt
			existing.NorthSHAmt = flow.NorthSHAmt
			existing.NorthSZAmt = flow.NorthSZAmt
			existing.SouthHKBuy = flow.SouthHKBuy
			existing.SouthSHBuy = flow.SouthSHBuy
			existing.SouthSZBuy = flow.SouthSZBuy
			existing.SouthTotalBuy = flow.SouthTotalBuy
			existing.Source = "akshare+eastmoney"
			existing.Status = "completed"
			if err := s.store.SaveDaily(existing); err != nil {
				s.logger.Warn("failed to update HSGT data", "date", date, "error", err)
				continue
			}
		} else {
			if err := s.store.SaveDaily(flow); err != nil {
				s.logger.Warn("failed to save HSGT data", "date", date, "error", err)
				continue
			}
		}
		saved++
	}

	s.logger.Info("HSGT data fetch completed", "saved", saved, "totalDates", len(merged))
	return nil
}

// FetchAndSaveToday 仅获取并保存今天的数据
func (s *HSGTScraper) FetchAndSaveToday(ctx context.Context) error {
	today := time.Now().Format("2006-01-02")

	// 检查今天是否已有数据
	existing, _ := s.store.GetByDate(today)
	if existing != nil && existing.Status == "completed" {
		s.logger.Debug("today's HSGT data already exists", "date", today)
		return nil
	}

	// 获取北向和南向整体数据
	northData, err := s.fetchHistData(ctx, "北向资金")
	if err != nil {
		return fmt.Errorf("fetch 北向资金: %w", err)
	}
	southData, err := s.fetchHistData(ctx, "南向资金")
	if err != nil {
		return fmt.Errorf("fetch 南向资金: %w", err)
	}

	// 同时获取细分数据
	shData, _ := s.fetchHistData(ctx, "沪股通")
	szData, _ := s.fetchHistData(ctx, "深股通")
	hkSHData, _ := s.fetchHistData(ctx, "港股通沪")
	hkSZData, _ := s.fetchHistData(ctx, "港股通深")

	// 从东方财富获取北向成交额
	amtData, err := s.fetchNorthDealAmt(ctx, 10)
	if err != nil {
		s.logger.Warn("failed to fetch north deal amount data for today", "error", err)
	}

	merged := s.mergeData(northData, southData, shData, szData, hkSHData, hkSZData, amtData)

	// 只保存最近 5 天的数据（覆盖今天和可能的遗漏）
	cutoff := time.Now().AddDate(0, 0, -5).Format("2006-01-02")
	saved := 0
	for date, flow := range merged {
		if date < cutoff {
			continue
		}
		if err := s.store.SaveDaily(flow); err != nil {
			s.logger.Warn("failed to save HSGT data", "date", date, "error", err)
			continue
		}
		saved++
	}

	s.logger.Info("today's HSGT data fetch completed", "saved", saved)
	return nil
}

// fetchHistData 从 AKShare 服务获取指定类型的 HSGT 历史数据
func (s *HSGTScraper) fetchHistData(ctx context.Context, symbol string) (map[string]*hsgtHistItem, error) {
	baseURL := s.cfg.AKShareURL
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8900"
	}

	reqURL := fmt.Sprintf("%s/api/v1/hsgt/hist?symbol=%s&days=370", baseURL, url.QueryEscape(symbol))

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// 设置认证头
	if s.cfg.AKShareToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.AKShareToken)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result hsgtHistResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("API returned code %d", result.Code)
	}

	items := make(map[string]*hsgtHistItem, len(result.Data))
	for i := range result.Data {
		d := &result.Data[i]
		items[d.Date] = &hsgtHistItem{
			Date:       d.Date,
			NetBuy:     d.NetBuy,
			BuyAmount:  d.BuyAmount,
			SellAmount: d.SellAmount,
			AccNetBuy:  d.AccNetBuy,
			CashIn:     d.CashIn,
			Balance:    d.Balance,
		}
	}

	s.logger.Debug("fetched HSGT hist data", "symbol", symbol, "count", len(items))
	return items, nil
}

// hsgtHistItem 单条 HSGT 历史数据
type hsgtHistItem struct {
	Date       string
	NetBuy     float64
	BuyAmount  float64
	SellAmount float64
	AccNetBuy  float64
	CashIn     float64
	Balance    float64
}

// northDealAmtItem 东方财富 DataCenter API 返回的北向成交额数据
type northDealAmtItem struct {
	Date      string  // 交易日期
	NFTAmt    float64 // 北向合计成交额（万元）
	SSCAmt    float64 // 沪股通成交额（万元）
	STAmt     float64 // 深股通成交额（万元）
}

// fetchNorthDealAmt 从东方财富 DataCenter API 获取北向资金成交额数据
func (s *HSGTScraper) fetchNorthDealAmt(ctx context.Context, pageSize int) (map[string]*northDealAmtItem, error) {
	// 东方财富 DataCenter API
	apiURL := fmt.Sprintf(
		"https://datacenter-web.eastmoney.com/api/data/v1/get?reportName=RPT_MUTUAL_DEALAMT&columns=TRADE_DATE,NF_DEAL_AMT,SSC_DEAL_AMT,ST_DEAL_AMT&pageSize=%d&sortColumns=TRADE_DATE&sortTypes=-1",
		pageSize,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
	req.Header.Set("Referer", "https://data.eastmoney.com/hsgtV2/hsgtDetail/scgk.html")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Success bool `json:"success"`
		Result  struct {
			Data []struct {
				TradeDate string  `json:"TRADE_DATE"`
				NFTAmt    float64 `json:"NF_DEAL_AMT"`
				SSCAmt    float64 `json:"SSC_DEAL_AMT"`
				STAmt     float64 `json:"ST_DEAL_AMT"`
			} `json:"data"`
		} `json:"result"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("API returned error: %s", result.Message)
	}

	items := make(map[string]*northDealAmtItem, len(result.Result.Data))
	for i := range result.Result.Data {
		d := &result.Result.Data[i]
		// 日期格式: "2026-06-16 00:00:00" -> "2026-06-16"
		date := d.TradeDate
		if len(date) > 10 {
			date = date[:10]
		}
		items[date] = &northDealAmtItem{
			Date:   date,
			NFTAmt: d.NFTAmt,
			SSCAmt: d.SSCAmt,
			STAmt:  d.STAmt,
		}
	}

	s.logger.Debug("fetched north deal amount data", "count", len(items))
	return items, nil
}

// mergeData 将北向/南向整体数据和细分数据按日期合并为 HSGTFlowDaily 记录
func (s *HSGTScraper) mergeData(
	northData, southData, shData, szData, hkSHData, hkSZData map[string]*hsgtHistItem,
	amtData map[string]*northDealAmtItem,
) map[string]*database.HSGTFlowDaily {
	// 收集所有日期
	allDates := make(map[string]bool)
	for date := range northData {
		allDates[date] = true
	}
	for date := range southData {
		allDates[date] = true
	}
	for date := range amtData {
		allDates[date] = true
	}

	result := make(map[string]*database.HSGTFlowDaily, len(allDates))
	for date := range allDates {
		var northTotal, southTotal float64
		var northSH, northSZ, southSH, southSZ float64
		var northTotalAmt, northSHAmt, northSZAmt float64

		// 北向资金整体净买额
		if item, ok := northData[date]; ok {
			northTotal = item.NetBuy
		}
		// 南向资金整体净买额
		if item, ok := southData[date]; ok {
			southTotal = item.NetBuy
		}

		// 细分数据（可能为空或NaN=0，仅在有值时覆盖）
		if item, ok := shData[date]; ok && item.NetBuy != 0 {
			northSH = item.NetBuy
		}
		if item, ok := szData[date]; ok && item.NetBuy != 0 {
			northSZ = item.NetBuy
		}
		if item, ok := hkSHData[date]; ok && item.NetBuy != 0 {
			southSH = item.NetBuy
		}
		if item, ok := hkSZData[date]; ok && item.NetBuy != 0 {
			southSZ = item.NetBuy
		}

		// 如果细分数据有值但整体为0，用细分合计替代
		if northTotal == 0 && (northSH != 0 || northSZ != 0) {
			northTotal = northSH + northSZ
		}
		if southTotal == 0 && (southSH != 0 || southSZ != 0) {
			southTotal = southSH + southSZ
		}

		// 北向成交额数据（来自东方财富 DataCenter API）
		if item, ok := amtData[date]; ok {
			northTotalAmt = item.NFTAmt
			northSHAmt = item.SSCAmt
			northSZAmt = item.STAmt
		}

		// 跳过全部为零且无成交额的日期（非交易日或数据缺失）
		if northTotal == 0 && southTotal == 0 && northTotalAmt == 0 {
			continue
		}

		result[date] = &database.HSGTFlowDaily{
			Date:          date,
			NorthSHBuy:    northSH,
			NorthSZBuy:    northSZ,
			NorthTotalBuy: northTotal,
			NorthTotalAmt: northTotalAmt,
			NorthSHAmt:    northSHAmt,
			NorthSZAmt:    northSZAmt,
			SouthHKBuy:    0, // 南向资金合计已包含在SouthTotalBuy中
			SouthSHBuy:    southSH,
			SouthSZBuy:    southSZ,
			SouthTotalBuy: southTotal,
			Source:        "akshare+eastmoney",
			Status:        "completed",
		}
	}

	return result
}

// CleanupOldData 清理超过一年的数据
func (s *HSGTScraper) CleanupOldData(ctx context.Context) (int64, error) {
	count, err := s.store.DeleteOlderThanOneYear()
	if err != nil {
		s.logger.Error("failed to cleanup old data", "error", err)
		return 0, err
	}
	if count > 0 {
		s.logger.Info("cleaned up old HSGT data", "deletedRows", count)
	}
	return count, nil
}
