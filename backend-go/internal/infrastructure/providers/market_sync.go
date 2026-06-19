package providers

import (
	"context"
	"log/slog"
	"math"
	"math/rand"
	"sync"
	"time"

	marketdomain "stock-predict-go/internal/domain/market"
	database "stock-predict-go/internal/infrastructure/database"
)

// MarketSyncService 市场数据同步服务，负责定时同步指数行情、K线、分时数据到数据库。
type MarketSyncService struct {
	quote          *IndexQuoteClient
	market         *MarketService
	marketStore    *database.MarketStore
	preloader      *TDXPreloader
	health         *HealthMonitor
	stocks         *StockService
	logger         *slog.Logger
	mu             sync.Mutex
	running        bool
	cancel         context.CancelFunc
	lastSyncTime   time.Time
	lastSyncSource string
	lastSyncResult string
	syncCount      int
}

// NewMarketSyncService 创建新的市场数据同步服务实例。
func NewMarketSyncService(quote *IndexQuoteClient, market *MarketService, marketStore *database.MarketStore, preloader *TDXPreloader, health *HealthMonitor, stocks *StockService, logger *slog.Logger) *MarketSyncService {
	if logger == nil {
		logger = slog.Default()
	}
	return &MarketSyncService{
		quote:       quote,
		market:      market,
		marketStore: marketStore,
		preloader:   preloader,
		health:      health,
		stocks:      stocks,
		logger:      logger,
	}
}

// Start 启动市场数据同步服务的后台循环。
func (s *MarketSyncService) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.mu.Unlock()
	go s.run(ctx)
	s.logger.Info("market sync service started")
}

// Stop 停止市场数据同步服务。
func (s *MarketSyncService) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.cancel()
	s.mu.Unlock()
	s.logger.Info("market sync service stopped")
}

// Status 返回市场数据同步服务的当前状态信息。
func (s *MarketSyncService) Status() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	status := map[string]any{
		"running":          s.running,
		"last_sync_time":   s.lastSyncTime.Format("2006-01-02 15:04:05"),
		"last_sync_source": s.lastSyncSource,
		"last_sync_result": s.lastSyncResult,
		"sync_count":       s.syncCount,
	}
	if s.health != nil {
		status["sources"] = s.health.GetAllStatus()
	}
	if s.preloader != nil {
		status["tdx_preloaded"] = s.preloader.IsLoaded()
		status["cache_stats"] = s.preloader.CacheStats()
	}
	return status
}

// run 同步服务的主循环，根据交易时段动态调整同步间隔。
func (s *MarketSyncService) run(ctx context.Context) {
	s.syncOnce(ctx)
	tradingTicker := time.NewTicker(MarketSyncTradingInterval)
	idleTicker := time.NewTicker(MarketSyncIdleInterval)
	cleanupTicker := time.NewTicker(MarketSyncCleanupInterval)
	defer tradingTicker.Stop()
	defer idleTicker.Stop()
	defer cleanupTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tradingTicker.C:
			if s.isAnyMarketTrading() {
				s.syncOnce(ctx)
			}
		case <-idleTicker.C:
			if !s.isAnyMarketTrading() {
				s.syncOnce(ctx)
			}
		case <-cleanupTicker.C:
			s.cleanup()
		}
	}
}

// syncOnce 执行一次完整的市场数据同步，包括指数行情、K线、分时数据。
func (s *MarketSyncService) syncOnce(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	syncCtx, syncCancel := context.WithTimeout(ctx, 60*time.Second)
	defer syncCancel()

	indices := s.quote.FetchIndexQuotes(syncCtx)
	if len(indices) > 0 && s.marketStore != nil {
		s.marketStore.SaveIndexQuotes(indices)
	}
	if len(indices) > 0 {
		s.lastSyncSource = indices[0].DataSource
		s.lastSyncResult = "success"
	} else {
		s.lastSyncSource = ""
		s.lastSyncResult = "empty"
	}

	today := time.Now().Format("2006-01-02")
	for _, code := range cnIndexCodes {
		points := s.quote.FetchIndexMinute(syncCtx, code)
		if len(points) > 0 && s.marketStore != nil {
			s.marketStore.SaveIndexMinutes(code, today, points)
		}
	}

	// Sync HK index minute data
	for code := range hkIndexMeta {
		points := s.quote.FetchIndexMinute(syncCtx, code)
		if len(points) > 0 && s.marketStore != nil {
			s.marketStore.SaveIndexMinutes(code, today, points)
		}
	}

	// Incremental K-line update via preloader
	if s.preloader != nil {
		s.preloader.IncrementalUpdate(syncCtx)
	}

	// Persist HK/US index kline to database
	if s.marketStore != nil {
		for code := range hkIndexMeta {
			points := s.quote.FetchIndexKline(syncCtx, code, 120)
			if len(points) > 0 {
				s.marketStore.SaveIndexKline(code, points)
			}
		}
		for code := range usIndexMeta {
			points := s.quote.FetchIndexKline(syncCtx, code, 120)
			if len(points) > 0 {
				s.marketStore.SaveIndexKline(code, points)
			}
		}
	}

	// Persist CN index kline for incremental updates (MarketStore now handles this internally via SaveKlineDaily)
	if s.marketStore != nil {
		for _, code := range cnIndexCodes {
			points := s.quote.FetchIndexKline(syncCtx, code, 120)
			if len(points) > 0 {
				if err := s.marketStore.SaveKlineDaily(code, points); err != nil {
					s.logger.Warn("failed to save CN index kline", "code", code, "error", err)
				}
			}
		}
	}

	s.validateData(syncCtx, indices)

	// Sync stock ranking data during trading hours
	if s.stocks != nil && s.isAnyMarketTrading() {
		for _, rankingType := range []string{"gainers", "losers", "volume"} {
			items, err := s.stocks.Ranking(syncCtx, rankingType, DefaultRankingSize)
			if err == nil && len(items) > 0 {
				// Already saved inside Ranking method when fetched from API
				_ = items
			}
		}
	}

	s.mu.Lock()
	s.lastSyncTime = time.Now()
	s.syncCount++
	s.mu.Unlock()

	s.logger.Info("market sync completed", "indices", len(indices), "source", s.lastSyncSource, "result", s.lastSyncResult)
}

// validateData 校验同步数据的一致性，检测异常涨跌幅。
func (s *MarketSyncService) validateData(ctx context.Context, indices []marketdomain.MarketIndex) {
	if len(indices) == 0 {
		return
	}
	// Pick a random index for validation
	idx := indices[rand.Intn(len(indices))]

	// Only validate A-share and HK indices (have cross-reference data)
	if !isCNIndex(idx.Code) {
		if _, isHK := hkIndexMeta[idx.Code]; !isHK {
			return
		}
	}

	// For A-share, cross-validate with Eastmoney
	if isCNIndex(idx.Code) {
		eastmoneyData := s.quote.fetchCNIndexQuotesEastmoney(ctx)
		if len(eastmoneyData) == 0 {
			return
		}
		var ref *marketdomain.MarketIndex
		for i := range eastmoneyData {
			if eastmoneyData[i].Code == idx.Code {
				ref = &eastmoneyData[i]
				break
			}
		}
		if ref == nil {
			return
		}
		s.validateAndCorrect(indices, idx.Code, ref.ChangePct)
	}

	// For HK indices, cross-validate with Tencent (re-fetch and compare)
	if _, isHK := hkIndexMeta[idx.Code]; isHK {
		hkQuotes := s.quote.fetchHKIndexQuotesTencent(ctx)
		if len(hkQuotes) == 0 {
			return
		}
		var ref *marketdomain.MarketIndex
		for i := range hkQuotes {
			if hkQuotes[i].Code == idx.Code {
				ref = &hkQuotes[i]
				break
			}
		}
		if ref == nil {
			return
		}
		s.validateAndCorrect(indices, idx.Code, ref.ChangePct)
	}
}

// validateAndCorrect 校验并修正单个指数的涨跌幅数据。
func (s *MarketSyncService) validateAndCorrect(indices []marketdomain.MarketIndex, code string, refChangePct float64) {
	var current *marketdomain.MarketIndex
	for i := range indices {
		if indices[i].Code == code {
			current = &indices[i]
			break
		}
	}
	if current == nil {
		return
	}
	diff := math.Abs(current.ChangePct - refChangePct)
	if diff > MarketSyncValidationThreshold {
		s.logger.Warn("index data validation failed, deviation too large",
			"code", code,
			"cached_pct", current.ChangePct,
			"ref_pct", refChangePct,
			"diff", diff,
		)
		current.ChangePct = refChangePct
		current.DataSource = "corrected"
	}
}

// cleanup 清理过期的市场数据。
func (s *MarketSyncService) cleanup() {
	if s.marketStore != nil {
		s.marketStore.CleanExpiredData(MarketDataRetentionDays, MarketMinuteRetentionDays)
	}
}

// isAnyMarketTrading 检查是否有任何市场当前处于交易时段。
func (s *MarketSyncService) isAnyMarketTrading() bool {
	now := time.Now()
	return IsMarketOpenAt(MarketCN, now) ||
		IsMarketOpenAt(MarketHK, now) ||
		IsMarketOpenAt(MarketUS, now)
}
