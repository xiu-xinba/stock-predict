package providers

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"stock-predict-go/internal/infrastructure/database"
	"stock-predict-go/internal/platform/config"
)

// HSGTScheduler 定时获取北向和南向资金数据
type HSGTScheduler struct {
	scraper   *HSGTScraper
	store     *database.HSGTFlowDailyStore
	logger    *slog.Logger
	stopChan  chan struct{}
	running   bool
	mu        sync.Mutex
	lastRun   time.Time
}

// NewHSGTScheduler 创建调度器
func NewHSGTScheduler(store *database.HSGTFlowDailyStore, cfg config.Config, logger *slog.Logger) *HSGTScheduler {
	scraper := NewHSGTScraper(store, cfg, logger)
	return &HSGTScheduler{
		scraper:  scraper,
		store:    store,
		logger:   logger,
		stopChan: make(chan struct{}),
	}
}

// Start 启动定时任务（每天 20:00 执行）
func (hs *HSGTScheduler) Start() {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	if hs.running {
		hs.logger.Warn("HSGT scheduler already running")
		return
	}

	hs.running = true
	go hs.run()
	hs.logger.Info("HSGT scheduler started, will run daily at 20:00")
}

// Stop 停止定时任务
func (hs *HSGTScheduler) Stop() {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	if !hs.running {
		return
	}

	hs.running = false
	close(hs.stopChan)
	hs.logger.Info("HSGT scheduler stopped")
}

// run 执行定时任务逻辑
func (hs *HSGTScheduler) run() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-hs.stopChan:
			return
		case <-ticker.C:
			if hs.shouldRun() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				hs.executeDaily(ctx)
				cancel()
			}
		}
	}
}

// shouldRun 判断是否应该在当前时间运行（20:00-20:05 之间且今天尚未运行）
func (hs *HSGTScheduler) shouldRun() bool {
	now := time.Now()
	// 只在 20:00-20:05 之间执行
	if now.Hour() != 20 || now.Minute() >= 5 {
		return false
	}
	// 今天已经运行过则跳过
	if hs.lastRun.Format("2006-01-02") == now.Format("2006-01-02") {
		return false
	}
	return true
}

// executeDaily 执行每日数据获取和清理
func (hs *HSGTScheduler) executeDaily(ctx context.Context) {
	startTime := time.Now()
	hs.logger.Info("starting daily HSGT data fetch", "time", startTime)

	// 获取今天的数据
	if err := hs.scraper.FetchAndSaveToday(ctx); err != nil {
		hs.logger.Error("failed to fetch today's HSGT data", "error", err)
		return
	}

	hs.lastRun = startTime

	// 清理超过一年的旧数据
	if _, err := hs.scraper.CleanupOldData(ctx); err != nil {
		hs.logger.Error("failed to cleanup old HSGT data", "error", err)
	}

	duration := time.Since(startTime)
	hs.logger.Info("daily HSGT fetch completed", "duration", duration)
}

// SyncHistoricalData 同步近一年的历史数据（初始化时调用）
func (hs *HSGTScheduler) SyncHistoricalData(ctx context.Context) error {
	hs.logger.Info("syncing historical HSGT data for the past year")

	// 检查现有数据量
	count, err := hs.store.Count()
	if err != nil {
		return fmt.Errorf("count existing data: %w", err)
	}

	// 如果已有足够数据则跳过
	if count >= 200 {
		hs.logger.Info("sufficient HSGT data exists, skipping historical sync", "existingRecords", count)
		return nil
	}

	if err := hs.scraper.FetchAndSaveAll(ctx); err != nil {
		return fmt.Errorf("fetch historical data: %w", err)
	}

	// 清理超过一年的旧数据
	if _, err := hs.scraper.CleanupOldData(ctx); err != nil {
		hs.logger.Warn("failed to cleanup old data after historical sync", "error", err)
	}

	return nil
}

// GetStats 获取数据统计信息
func (hs *HSGTScheduler) GetStats() (map[string]interface{}, error) {
	count, err := hs.store.Count()
	if err != nil {
		return nil, fmt.Errorf("count records: %w", err)
	}

	recent, err := hs.store.ListRecent(1)
	if err != nil {
		return nil, fmt.Errorf("get recent data: %w", err)
	}

	stats := map[string]interface{}{
		"totalRecords": count,
		"running":      hs.running,
		"lastRun":      hs.lastRun.Format("2006-01-02 15:04:05"),
	}

	if len(recent) > 0 {
		stats["latestDate"] = recent[0].Date
		stats["latestNorth"] = recent[0].NorthTotalBuy
		stats["latestSouth"] = recent[0].SouthTotalBuy
	}

	return stats, nil
}
