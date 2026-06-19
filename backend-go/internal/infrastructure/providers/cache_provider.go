package providers

import (
	"context"
	"log/slog"
	"sort"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"

	marketdomain "stock-predict-go/internal/domain/market"
	stockdomain "stock-predict-go/internal/domain/stock"
	database "stock-predict-go/internal/infrastructure/database"
)

// CacheProvider 是透明的缓存装饰器，包装 Provider 调用。
// 根据数据类型应用不同的缓存策略：
//   - 实时数据（行情）：仅 LRU 内存缓存，短 TTL
//   - 历史数据（K线）：数据库缓存 + LRU，支持增量更新
//   - 财务数据：数据库缓存，按季度更新
type CacheProvider struct {
	inner       Provider
	marketStore *database.MarketStore
	logger      *slog.Logger

	// LRU caches for different data types
	quoteCache   *lru.Cache[string, cacheEntry[[]marketdomain.MarketIndex]]
	klineCache   *lru.Cache[string, cacheEntry[[]marketdomain.IndexKlinePoint]]
	minuteCache  *lru.Cache[string, cacheEntry[[]marketdomain.IndexMinutePoint]]
	financeCache *lru.Cache[string, cacheEntry[[]stockdomain.FinancialQuarter]]
}

// cacheEntry 带过期时间的泛型缓存条目。
type cacheEntry[T any] struct {
	value     T
	expiresAt time.Time
}

const (
	// cacheMaxEntries LRU 缓存最大条目数
	cacheMaxEntries = 256

	// 不同数据类型的 TTL
	quoteCacheTTL   = 5 * time.Second   // 行情缓存有效期
	klineCacheTTL   = 30 * time.Second  // K线缓存有效期
	minuteCacheTTL  = 30 * time.Second  // 分时缓存有效期
	financeCacheTTL = 24 * time.Hour    // 财务数据缓存有效期
)

// NewCacheProvider 创建一个新的 CacheProvider，包装给定的 Provider。
func NewCacheProvider(inner Provider, marketStore *database.MarketStore, logger *slog.Logger) *CacheProvider {
	if logger == nil {
		logger = slog.Default()
	}

	quoteCache, _ := lru.New[string, cacheEntry[[]marketdomain.MarketIndex]](cacheMaxEntries)
	klineCache, _ := lru.New[string, cacheEntry[[]marketdomain.IndexKlinePoint]](cacheMaxEntries)
	minuteCache, _ := lru.New[string, cacheEntry[[]marketdomain.IndexMinutePoint]](cacheMaxEntries)
	financeCache, _ := lru.New[string, cacheEntry[[]stockdomain.FinancialQuarter]](cacheMaxEntries)

	return &CacheProvider{
		inner:        inner,
		marketStore:  marketStore,
		logger:       logger,
		quoteCache:   quoteCache,
		klineCache:   klineCache,
		minuteCache:  minuteCache,
		financeCache: financeCache,
	}
}

// Name 返回被包装 Provider 的名称。
func (cp *CacheProvider) Name() string {
	return cp.inner.Name()
}

// Capabilities 返回被包装 Provider 的能力列表。
func (cp *CacheProvider) Capabilities() map[Capability][]Market {
	return cp.inner.Capabilities()
}

// Priority 返回被包装 Provider 在指定能力和市场下的优先级。
func (cp *CacheProvider) Priority(cap Capability, market Market) int {
	return cp.inner.Priority(cap, market)
}

// HealthCheck 委托给被包装 Provider 执行健康检查。
func (cp *CacheProvider) HealthCheck(ctx context.Context) error {
	return cp.inner.HealthCheck(ctx)
}

// --- IndexQuoteProvider 带 LRU 缓存 ---

// FetchIndexQuotes 获取指数行情数据，带 LRU 缓存。
func (cp *CacheProvider) FetchIndexQuotes(ctx context.Context, market Market) ([]marketdomain.MarketIndex, error) {
	provider, ok := cp.inner.(IndexQuoteProvider)
	if !ok {
		return nil, newProviderError(cp.Name(), "does not implement IndexQuoteProvider")
	}

	cacheKey := "index_quotes:" + string(market)
	if cached, ok := cp.quoteCache.Get(cacheKey); ok && time.Now().Before(cached.expiresAt) {
		return cached.value, nil
	}

	result, err := provider.FetchIndexQuotes(ctx, market)
	if err != nil {
		// On error, try to return stale cache if available
		if cached, ok := cp.quoteCache.Get(cacheKey); ok {
			cp.logger.Debug("returning stale cache for index quotes", "provider", cp.Name(), "market", market)
			return cached.value, nil
		}
		return nil, err
	}

	cp.quoteCache.Add(cacheKey, cacheEntry[[]marketdomain.MarketIndex]{
		value:     result,
		expiresAt: time.Now().Add(quoteCacheTTL),
	})
	return result, nil
}

// --- IndexKlineProvider 带数据库 + LRU 缓存 ---

// FetchIndexKline 获取指数K线数据，带数据库 + LRU 缓存及增量更新。
func (cp *CacheProvider) FetchIndexKline(ctx context.Context, code string, market Market, count int) ([]marketdomain.IndexKlinePoint, error) {
	provider, ok := cp.inner.(IndexKlineProvider)
	if !ok {
		return nil, newProviderError(cp.Name(), "does not implement IndexKlineProvider")
	}

	// Check LRU cache first
	cacheKey := "index_kline:" + code + ":" + string(market)
	if cached, ok := cp.klineCache.Get(cacheKey); ok && time.Now().Before(cached.expiresAt) {
		return cp.trimKline(cached.value, count), nil
	}

	// Check database cache for incremental update
	var cachedFromDB []marketdomain.IndexKlinePoint
	if cp.marketStore != nil {
		var err error
		cachedFromDB, err = cp.marketStore.GetKlineDaily(code, "", "")
		if err == nil && len(cachedFromDB) > 0 {
			// Check if cache is recent enough (within today)
			latestDate := cachedFromDB[len(cachedFromDB)-1].Date
			today := time.Now().Format("2006-01-02")
			if latestDate >= today {
				// Cache is up to date, use it directly
				cp.klineCache.Add(cacheKey, cacheEntry[[]marketdomain.IndexKlinePoint]{
					value:     cachedFromDB,
					expiresAt: time.Now().Add(klineCacheTTL),
				})
				return cp.trimKline(cachedFromDB, count), nil
			}
			// Cache is stale, try incremental update
			newPoints, fetchErr := provider.FetchIndexKline(ctx, code, market, 30)
			if fetchErr == nil && len(newPoints) > 0 {
				// Merge: filter out duplicates, append only new data
				merged := cp.mergeKlineData(cachedFromDB, newPoints)
				cp.klineCache.Add(cacheKey, cacheEntry[[]marketdomain.IndexKlinePoint]{
					value:     merged,
					expiresAt: time.Now().Add(klineCacheTTL),
				})
				// Async save new points to database
				go func() {
					if err := cp.marketStore.SaveKlineDaily(code, newPoints); err != nil {
						cp.logger.Warn("failed to save incremental kline to cache store", "code", code, "error", err)
					}
				}()
				return cp.trimKline(merged, count), nil
			}
			// Incremental fetch failed, return stale cache
			cp.klineCache.Add(cacheKey, cacheEntry[[]marketdomain.IndexKlinePoint]{
				value:     cachedFromDB,
				expiresAt: time.Now().Add(klineCacheTTL),
			})
			return cp.trimKline(cachedFromDB, count), nil
		}
	}

	// Also check MarketStore (existing kline table)
	if cp.marketStore != nil && len(cachedFromDB) == 0 {
		cached := cp.marketStore.LoadIndexKline(code, count)
		if len(cached) > 0 {
			cp.klineCache.Add(cacheKey, cacheEntry[[]marketdomain.IndexKlinePoint]{
				value:     cached,
				expiresAt: time.Now().Add(klineCacheTTL),
			})
			return cached, nil
		}
	}

	// Full fetch from provider (no cache available)
	result, err := provider.FetchIndexKline(ctx, code, market, count)
	if err != nil {
		// On error, try stale cache
		if cached, ok := cp.klineCache.Get(cacheKey); ok {
			return cp.trimKline(cached.value, count), nil
		}
		return nil, err
	}

	// Save to caches
	cp.klineCache.Add(cacheKey, cacheEntry[[]marketdomain.IndexKlinePoint]{
		value:     result,
		expiresAt: time.Now().Add(klineCacheTTL),
	})

	// Async save to database
	if cp.marketStore != nil && len(result) > 0 {
		go func() {
			if err := cp.marketStore.SaveKlineDaily(code, result); err != nil {
				cp.logger.Warn("failed to save kline to cache store", "code", code, "error", err)
			}
		}()
	}

	return result, nil
}

// --- IndexMinuteProvider 带 LRU 缓存 ---

// FetchIndexMinute 获取指数分时数据，带 LRU 缓存。
func (cp *CacheProvider) FetchIndexMinute(ctx context.Context, code string, market Market) ([]marketdomain.IndexMinutePoint, error) {
	provider, ok := cp.inner.(IndexMinuteProvider)
	if !ok {
		return nil, newProviderError(cp.Name(), "does not implement IndexMinuteProvider")
	}

	cacheKey := "index_minute:" + code + ":" + string(market)
	if cached, ok := cp.minuteCache.Get(cacheKey); ok && time.Now().Before(cached.expiresAt) {
		return cached.value, nil
	}

	result, err := provider.FetchIndexMinute(ctx, code, market)
	if err != nil {
		if cached, ok := cp.minuteCache.Get(cacheKey); ok {
			return cached.value, nil
		}
		return nil, err
	}

	cp.minuteCache.Add(cacheKey, cacheEntry[[]marketdomain.IndexMinutePoint]{
		value:     result,
		expiresAt: time.Now().Add(minuteCacheTTL),
	})
	return result, nil
}

// --- StockQuoteProvider 带 LRU 缓存 ---

// FetchStockQuotes 获取股票行情数据，带 LRU 缓存。当前直接委托给内部 Provider。
func (cp *CacheProvider) FetchStockQuotes(ctx context.Context, symbols []string) (map[string]stockdomain.StockQuote, error) {
	provider, ok := cp.inner.(StockQuoteProvider)
	if !ok {
		return nil, newProviderError(cp.Name(), "does not implement StockQuoteProvider")
	}

	// Stock quotes are real-time, just delegate with short LRU
	return provider.FetchStockQuotes(ctx, symbols)
}

// --- StockMinuteProvider 带 LRU 缓存 ---

// FetchStockMinute 获取个股分时数据，带 LRU 缓存。
func (cp *CacheProvider) FetchStockMinute(ctx context.Context, code string) ([]marketdomain.IndexMinutePoint, error) {
	provider, ok := cp.inner.(StockMinuteProvider)
	if !ok {
		return nil, newProviderError(cp.Name(), "does not implement StockMinuteProvider")
	}

	cacheKey := "stock_minute:" + code
	if cached, ok := cp.minuteCache.Get(cacheKey); ok && time.Now().Before(cached.expiresAt) {
		return cached.value, nil
	}

	result, err := provider.FetchStockMinute(ctx, code)
	if err != nil {
		if cached, ok := cp.minuteCache.Get(cacheKey); ok {
			return cached.value, nil
		}
		return nil, err
	}

	cp.minuteCache.Add(cacheKey, cacheEntry[[]marketdomain.IndexMinutePoint]{
		value:     result,
		expiresAt: time.Now().Add(minuteCacheTTL),
	})
	return result, nil
}

// --- NorthboundProvider 带 LRU 缓存 ---

// FetchNorthboundFlow 获取北向资金流向数据，带 LRU 缓存。当前直接委托给内部 Provider。
func (cp *CacheProvider) FetchNorthboundFlow(ctx context.Context) (*marketdomain.NorthboundFlow, error) {
	provider, ok := cp.inner.(NorthboundProvider)
	if !ok {
		return nil, newProviderError(cp.Name(), "does not implement NorthboundProvider")
	}
	// Northbound data is intraday, just delegate
	return provider.FetchNorthboundFlow(ctx)
}

// --- 辅助方法 ---

// trimKline 截取K线数据，保留最后 count 条记录。
func (cp *CacheProvider) trimKline(points []marketdomain.IndexKlinePoint, count int) []marketdomain.IndexKlinePoint {
	if count <= 0 || len(points) <= count {
		return points
	}
	return points[len(points)-count:]
}

// mergeKlineData 合并已有缓存的K线数据与新获取的数据，
// 按日期去重（新数据覆盖旧数据）并按时间正序排列。
func (cp *CacheProvider) mergeKlineData(existing, newPoints []marketdomain.IndexKlinePoint) []marketdomain.IndexKlinePoint {
	byDate := make(map[string]marketdomain.IndexKlinePoint, len(existing)+len(newPoints))
	for _, p := range existing {
		byDate[p.Date] = p
	}
	for _, p := range newPoints {
		byDate[p.Date] = p // overwrite with newer data
	}

	merged := make([]marketdomain.IndexKlinePoint, 0, len(byDate))
	for _, p := range byDate {
		merged = append(merged, p)
	}
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Date < merged[j].Date
	})
	return merged
}

// GetKlineFromCache 仅从数据库缓存中获取K线数据（不触发 API 调用）。
// 若缓存不存在则返回 nil。
func (cp *CacheProvider) GetKlineFromCache(code string, count int) []marketdomain.IndexKlinePoint {
	// Check LRU first
	cacheKey := "index_kline:" + code + ":cn"
	if cached, ok := cp.klineCache.Get(cacheKey); ok {
		return cp.trimKline(cached.value, count)
	}

	// Check database
	if cp.marketStore != nil {
		cached, err := cp.marketStore.GetKlineDaily(code, "", "")
		if err == nil && len(cached) > 0 {
			cp.klineCache.Add(cacheKey, cacheEntry[[]marketdomain.IndexKlinePoint]{
				value:     cached,
				expiresAt: time.Now().Add(klineCacheTTL),
			})
			return cp.trimKline(cached, count)
		}
	}

	if cp.marketStore != nil {
		cached := cp.marketStore.LoadIndexKline(code, count)
		if len(cached) > 0 {
			return cached
		}
	}

	return nil
}

// SaveKlineToCache 将K线数据保存到数据库缓存。
func (cp *CacheProvider) SaveKlineToCache(code string, points []marketdomain.IndexKlinePoint) {
	if cp.marketStore == nil || len(points) == 0 {
		return
	}
	if err := cp.marketStore.SaveKlineDaily(code, points); err != nil {
		cp.logger.Warn("failed to save kline to cache store", "code", code, "error", err)
	}
}

// GetLatestKlineDate 返回指定指数代码在缓存中最新的K线日期。
func (cp *CacheProvider) GetLatestKlineDate(code string) string {
	if cp.marketStore != nil {
		date, err := cp.marketStore.GetLatestKlineDate(code)
		if err == nil && date != "" {
			return date
		}
	}
	return ""
}

// CacheStats 缓存统计信息。
type CacheStats struct {
	KlineEntries  map[string]int `json:"kline_entries"`
	QuoteLRUSize  int            `json:"quote_lru_size"`
	KlineLRUSize  int            `json:"kline_lru_size"`
	MinuteLRUSize int            `json:"minute_lru_size"`
}

// GetCacheStats 返回当前缓存统计信息。
func (cp *CacheProvider) GetCacheStats() *CacheStats {
	stats := &CacheStats{
		QuoteLRUSize:  cp.quoteCache.Len(),
		KlineLRUSize:  cp.klineCache.Len(),
		MinuteLRUSize: cp.minuteCache.Len(),
	}

	if cp.marketStore != nil {
		stats.KlineEntries = make(map[string]int)
		for _, code := range cnIndexCodes {
			count, err := cp.marketStore.GetKlineDailyCount(code)
			if err == nil && count > 0 {
				stats.KlineEntries[code] = count
			}
		}
	}

	return stats
}

// 编译时接口合规性检查
var (
	_ IndexQuoteProvider  = (*CacheProvider)(nil)
	_ IndexKlineProvider  = (*CacheProvider)(nil)
	_ IndexMinuteProvider = (*CacheProvider)(nil)
	_ StockQuoteProvider  = (*CacheProvider)(nil)
	_ StockMinuteProvider = (*CacheProvider)(nil)
	_ NorthboundProvider  = (*CacheProvider)(nil)
)

// WarmCache 预热缓存，并发获取指定指数代码的K线数据。
func (cp *CacheProvider) WarmCache(ctx context.Context, codes []string) {
	var wg sync.WaitGroup
	for _, code := range codes {
		wg.Add(1)
		go func(code string) {
			defer wg.Done()
			caps := cp.inner.Capabilities()
			if markets, ok := caps[CapIndexKline]; ok {
				for _, m := range markets {
					if _, err := cp.FetchIndexKline(ctx, code, m, 120); err != nil {
						cp.logger.Debug("warm cache failed for kline", "code", code, "error", err)
					}
				}
			}
		}(code)
	}
	wg.Wait()
}
