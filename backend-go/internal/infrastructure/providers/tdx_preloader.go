package providers

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	marketdomain "stock-predict-go/internal/domain/market"
	database "stock-predict-go/internal/infrastructure/database"
)

// TDXPreloader 异步预加载器，将 A 股指数历史K线数据（通过 TDX）
// 及港股/美股指数K线数据（通过东方财富）预加载到 SQLite 缓存中。
type TDXPreloader struct {
	quote       *IndexQuoteClient
	market      *MarketService
	marketStore *database.MarketStore
	health      *HealthMonitor
	logger      *slog.Logger
	mu          sync.Mutex
	running     bool
	loaded      bool
	cancel      context.CancelFunc
}

// NewTDXPreloader 创建新的 TDXPreloader 实例。
func NewTDXPreloader(
	quote *IndexQuoteClient,
	market *MarketService,
	marketStore *database.MarketStore,
	health *HealthMonitor,
	logger *slog.Logger,
) *TDXPreloader {
	return &TDXPreloader{
		quote:       quote,
		market:      market,
		marketStore: marketStore,
		health:      health,
		logger:      logger,
	}
}

// Start 启动后台预加载过程，非阻塞。
func (p *TDXPreloader) Start() {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return
	}
	p.running = true
	p.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel

	go p.run(ctx)
}

// Stop 取消预加载过程。
func (p *TDXPreloader) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cancel != nil {
		p.cancel()
	}
	p.running = false
}

// IsLoaded 返回初始预加载是否已完成。
func (p *TDXPreloader) IsLoaded() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.loaded
}

// run 执行预加载主循环，依次加载 A 股和港股/美股指数K线数据。
func (p *TDXPreloader) run(ctx context.Context) {
	p.logger.Info("TDX preloader started")
	preloadCtx, preloadCancel := context.WithTimeout(ctx, time.Duration(TDXPreloadTimeout)*time.Second)
	defer preloadCancel()

	// Preload A-share index K-line (3 years)
	cnCodes := []string{"000001", "399001", "399006"}
	for _, code := range cnCodes {
		select {
		case <-preloadCtx.Done():
			p.logger.Warn("TDX preload cancelled", "error", preloadCtx.Err())
			return
		default:
		}
		p.preloadIndexKline(preloadCtx, code, "tdx")
	}

	// Preload HK/US index K-line (3 years) via Eastmoney
	hkusCodes := []string{"hsi", "hstech", "dji", "ixic", "spx"}
	for _, code := range hkusCodes {
		select {
		case <-preloadCtx.Done():
			p.logger.Warn("TDX preload cancelled", "error", preloadCtx.Err())
			return
		default:
		}
		p.preloadIndexKline(preloadCtx, code, "eastmoney")
	}

	p.mu.Lock()
	p.loaded = true
	p.mu.Unlock()
	p.logger.Info("TDX preload completed")
}

// preloadIndexKline 预加载单个指数的3年K线数据。
// 若缓存已足够新（2天内更新且记录数≥700），则跳过加载。
func (p *TDXPreloader) preloadIndexKline(ctx context.Context, code, source string) {
	// Check if we already have enough data
	if p.marketStore != nil {
		meta, err := p.marketStore.LoadCacheMetadata(code, "kline")
		if err == nil && meta != nil {
			endDate, err := time.Parse("2006-01-02", meta.EndDate)
			if err == nil {
				daysSinceUpdate := time.Since(endDate).Hours() / 24
				if daysSinceUpdate < 2 && meta.RecordCount >= 700 {
					p.logger.Info("kline cache already up-to-date, skipping preload",
						"code", code, "records", meta.RecordCount, "end_date", meta.EndDate)
					return
				}
			}
		}
	}

	// Fetch 3 years of K-line data (approximately 750 trading days)
	count := 750
	points := p.quote.FetchIndexKline(ctx, code, count)
	if len(points) == 0 {
		p.logger.Warn("preload kline failed, no data returned", "code", code, "source", source)
		p.health.RecordFailure(source, fmt.Errorf("preload kline failed for %s", code))
		return
	}

	// Save to SQLite
	if p.marketStore != nil {
		if err := p.marketStore.SaveIndexKline(code, points); err != nil {
			p.logger.Warn("preload kline save failed", "code", code, "error", err)
		}

		// Update cache metadata
		startDate := points[0].Date
		endDate := points[len(points)-1].Date
		p.marketStore.SaveCacheMetadata(database.CacheMetadata{
			Code:        code,
			DataType:    "kline",
			Source:      source,
			StartDate:   startDate,
			EndDate:     endDate,
			RecordCount: len(points),
		})
	}

	p.health.RecordSuccess(source)
	p.logger.Info("preload kline completed", "code", code, "source", source,
		"count", len(points), "start", points[0].Date, "end", points[len(points)-1].Date)
}

// IncrementalUpdate 对所有指数执行增量K线更新。
func (p *TDXPreloader) IncrementalUpdate(ctx context.Context) {
	allCodes := []string{"000001", "399001", "399006", "hsi", "hstech", "dji", "ixic", "spx"}
	for _, code := range allCodes {
		select {
		case <-ctx.Done():
			return
		default:
		}
		source := "tdx"
		if !isCNIndex(code) {
			source = "eastmoney"
		}
		p.incrementalUpdateKline(ctx, code, source)
	}
}

// incrementalUpdateKline 对单个指数执行增量K线更新，获取最新数据并保存到 SQLite。
func (p *TDXPreloader) incrementalUpdateKline(ctx context.Context, code, _ string) {
	points := p.quote.FetchIndexKline(ctx, code, KlineIncrementalDays)
	if len(points) == 0 {
		return
	}
	if p.marketStore != nil {
		p.marketStore.SaveIndexKline(code, points)
		// Update metadata end_date
		endDate := points[len(points)-1].Date
		meta, _ := p.marketStore.LoadCacheMetadata(code, "kline")
		if meta != nil {
			meta.EndDate = endDate
			meta.RecordCount += len(points)
			p.marketStore.SaveCacheMetadata(*meta)
		}
	}
	p.logger.Info("incremental kline update completed", "code", code, "new_records", len(points))
}

// CacheStats 返回所有指数的缓存统计信息。
func (p *TDXPreloader) CacheStats() map[string]marketdomain.CacheStat {
	if p.marketStore == nil {
		return nil
	}
	metas, err := p.marketStore.LoadAllCacheMetadata()
	if err != nil {
		return nil
	}
	stats := make(map[string]marketdomain.CacheStat, len(metas))
	for _, m := range metas {
		if m.DataType == "kline" {
			stats[m.Code+"_kline"] = marketdomain.CacheStat{
				Start: m.StartDate,
				End:   m.EndDate,
				Count: m.RecordCount,
			}
		}
	}
	return stats
}
