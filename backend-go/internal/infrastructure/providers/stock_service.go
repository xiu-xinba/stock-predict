package providers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	stockdomain "stock-predict-go/internal/domain/stock"
	database "stock-predict-go/internal/infrastructure/database"
	seed "stock-predict-go/internal/infrastructure/database/seed"
	httpclient "stock-predict-go/internal/platform/httpclient"
)

// rankingCacheEntry 股票排行缓存条目
type rankingCacheEntry struct {
	items    []stockdomain.StockRankingItem
	cachedAt time.Time
}

// StockService 股票服务，提供股票搜索、排行、同步等功能。
type StockService struct {
	client         *http.Client
	eastmoney      *EastmoneyClient
	stockQuote     *StockQuoteClient
	logger         *slog.Logger
	store          stockdomain.Repository
	marketStore    *database.MarketStore
	stockStore     *database.StockStore
	health         *HealthMonitor
	rankingCache   map[string]*rankingCacheEntry
	rankingCacheMu sync.RWMutex
	router         *ProviderRouter
}

// SetMarketStore 注入 MarketStore 用于数据持久化。
func (s *StockService) SetMarketStore(ms *database.MarketStore) {
	s.marketStore = ms
}

// SetStockStore 注入 StockStore 用于股票列表缓存。
func (s *StockService) SetStockStore(ss *database.StockStore) {
	s.stockStore = ss
}

// SetHealthMonitor 注入健康监控器。
func (s *StockService) SetHealthMonitor(hm *HealthMonitor) {
	s.health = hm
}

// SetStockQuoteClient 注入股票行情客户端。
func (s *StockService) SetStockQuoteClient(sq *StockQuoteClient) {
	s.stockQuote = sq
}

// SetRouter 注入数据源路由器。
func (s *StockService) SetRouter(router *ProviderRouter) {
	s.router = router
}

// NewStockService 创建新的股票服务实例。
func NewStockService(stockRepo stockdomain.Repository, logger *slog.Logger) *StockService {
	if logger == nil {
		logger = slog.Default()
	}
	svc := &StockService{
		client: NewHTTPClient(HTTPClientConfig{
			Timeout:     15 * time.Second,
			DialTimeout: HTTPDialTimeout,
			KeepAlive:   HTTPKeepAlive,
		}),
		eastmoney: newEastmoneyClient(NewHTTPClient(HTTPClientConfig{
			Timeout:     15 * time.Second,
			DialTimeout: HTTPDialTimeout,
			KeepAlive:   HTTPKeepAlive,
		})),
		logger:       logger,
		store:        stockRepo,
		rankingCache: make(map[string]*rankingCacheEntry),
	}
	return svc
}

// Search 根据关键词和筛选条件搜索股票，支持分页。
func (s *StockService) Search(ctx context.Context, q stockdomain.StockSearchRequest) (stockdomain.StockSearchData, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Size < 1 {
		q.Size = DefaultSearchSize
	}
	if q.Size > MaxSearchSize {
		q.Size = MaxSearchSize
	}

	keyword := strings.TrimSpace(q.Keyword)

	if keyword != "" {
		if s.router != nil {
			var searchItems []stockdomain.StockSearchItem
			err := s.router.Fetch(ctx, CapStockSearch, MarketCN, func(ctx context.Context, p Provider) error {
				provider, ok := p.(StockSearchProvider)
				if !ok {
					return fmt.Errorf("provider %s does not implement StockSearchProvider", p.Name())
				}
				items, err := provider.SearchStocks(ctx, keyword)
				if err != nil {
					return err
				}
				searchItems = items
				return nil
			})
			if err == nil && len(searchItems) > 0 {
				items := s.convertSearchItems(searchItems)
				filtered := s.filterItems(items, q)
				return s.paginate(filtered, q.Page, q.Size), nil
			}
		}

		items := s.searchFromAPI(ctx, keyword)
		if len(items) > 0 {
			filtered := s.filterItems(items, q)
			return s.paginate(filtered, q.Page, q.Size), nil
		}
	}

	localStocks, err := s.listStocks()
	if err != nil {
		return stockdomain.StockSearchData{}, err
	}

	filtered := s.filterItems(localStocks, q)
	return s.paginate(filtered, q.Page, q.Size), nil
}

// Filters 返回当前股票列表中所有可用的行业和市场筛选项。
func (s *StockService) Filters() (stockdomain.StockFilters, error) {
	stocksCopy, err := s.listStocks()
	if err != nil {
		return stockdomain.StockFilters{}, err
	}

	industries := map[string]bool{}
	markets := map[string]bool{}
	for _, stock := range stocksCopy {
		if stock.Industry != "" {
			industries[stock.Industry] = true
		}
		if stock.Market != "" {
			markets[stock.Market] = true
		}
	}
	return stockdomain.StockFilters{
		Industries: keys(industries),
		Markets:    keys(markets),
	}, nil
}

// Ranking 获取股票排行榜，支持涨幅、跌幅、成交量三种类型。
// 优先通过路由器获取，失败时回退到本地排行。
func (s *StockService) Ranking(ctx context.Context, rankingType string, size int) ([]stockdomain.StockRankingItem, error) {
	if rankingType != "gainers" && rankingType != "losers" && rankingType != "volume" {
		return nil, ErrInvalidRankingType
	}
	if size < 1 {
		size = DefaultRankingSize
	}
	if size > MaxRankingSize {
		size = MaxRankingSize
	}

	if s.router != nil {
		var result []stockdomain.StockRankingItem
		err := s.router.Fetch(ctx, CapStockRanking, MarketCN, func(ctx context.Context, p Provider) error {
			provider, ok := p.(StockRankingProvider)
			if !ok {
				return fmt.Errorf("provider %s does not implement StockRankingProvider", p.Name())
			}
			items, err := provider.FetchStockRanking(ctx, rankingType, size)
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

	return s.rankingLegacy(ctx, rankingType, size)
}

// rankingLegacy 传统排行获取逻辑，依次尝试腾讯行情、东方财富API、SQLite缓存、本地排行。
func (s *StockService) rankingLegacy(ctx context.Context, rankingType string, size int) ([]stockdomain.StockRankingItem, error) {
	cacheKey := rankingType + strconv.Itoa(size)
	s.rankingCacheMu.RLock()
	if entry, ok := s.rankingCache[cacheKey]; ok && time.Since(entry.cachedAt) < RankingCacheTTL {
		items := entry.items
		s.rankingCacheMu.RUnlock()
		return items, nil
	}
	s.rankingCacheMu.RUnlock()

	// Primary: Tencent stock quote API
	items := s.fetchRankingFromTencent(ctx, rankingType, size)
	if len(items) > 0 {
		if s.health != nil {
			s.health.RecordSuccess("tencent")
		}
		now := time.Now().Format("2006-01-02 15:04:05")
		for i := range items {
			items[i].DataSource = "tencent"
			items[i].UpdateTime = now
		}
		if s.marketStore != nil {
			s.marketStore.SaveStockRanking(rankingType, items, "tencent")
		}
		s.rankingCacheMu.Lock()
		s.rankingCache[cacheKey] = &rankingCacheEntry{items: items, cachedAt: time.Now()}
		s.rankingCacheMu.Unlock()
		return items, nil
	}

	// Secondary: Eastmoney API
	if s.health != nil {
		s.health.RecordFailure("tencent", fmt.Errorf("tencent ranking returned empty for %s", rankingType))
	}
	items = s.fetchRankingFromAPI(ctx, rankingType, size)
	if len(items) > 0 {
		if s.health != nil {
			s.health.RecordSuccess("eastmoney")
		}
		now := time.Now().Format("2006-01-02 15:04:05")
		for i := range items {
			items[i].DataSource = "eastmoney"
			items[i].UpdateTime = now
		}
		if s.marketStore != nil {
			s.marketStore.SaveStockRanking(rankingType, items, "eastmoney")
		}
		s.rankingCacheMu.Lock()
		s.rankingCache[cacheKey] = &rankingCacheEntry{items: items, cachedAt: time.Now()}
		s.rankingCacheMu.Unlock()
		return items, nil
	}

	// Tertiary: SQLite cache
	if s.health != nil {
		s.health.RecordFailure("eastmoney", fmt.Errorf("stock ranking API returned empty for %s", rankingType))
	}
	if s.marketStore != nil {
		cached, _, err := s.marketStore.LoadStockRanking(rankingType)
		if err == nil && len(cached) > 0 {
			for i := range cached {
				cached[i].DataSource = "cache"
			}
			return cached, nil
		}
	}

	// Final fallback: local ranking from stock list
	localItems := s.localRanking(rankingType, size)
	if len(localItems) > 0 {
		for i := range localItems {
			localItems[i].DataSource = "local"
		}
		return localItems, nil
	}

	return nil, ErrMarketUnavailable
}

// FindStock 根据股票代码查找股票信息。
func (s *StockService) FindStock(code string) (stockdomain.StockItem, error) {
	item, ok := s.store.FindStock(code)
	if !ok {
		return stockdomain.StockItem{}, ErrStockNotFound
	}
	return item, nil
}

// IsLoaded 返回股票列表是否已加载。
func (s *StockService) IsLoaded() bool {
	return s.store.IsLoaded()
}

// ListStocks 返回所有已加载的股票列表。
func (s *StockService) ListStocks() []stockdomain.StockItem {
	return s.store.ListStocks()
}

func (s *StockService) listStocks() ([]stockdomain.StockItem, error) {
	type stockListRepositoryWithError interface {
		ListStocksWithError() ([]stockdomain.StockItem, error)
	}
	if repository, ok := s.store.(stockListRepositoryWithError); ok {
		return repository.ListStocksWithError()
	}
	return s.store.ListStocks(), nil
}

// SyncStocks 从远程数据源同步股票列表，依次尝试数据中心API、clist API、路由器、缓存、默认列表。
func (s *StockService) SyncStocks(ctx context.Context) (stockdomain.StockSyncResult, error) {
	var items []stockdomain.StockItem
	source := ""

	// Prefer data center API for stock sync: it returns 500 items per page
	// and is more reliable than the clist API which caps at 100 items/page
	// and is prone to rate-limiting (EOF errors) when fetching many pages.
	items = s.fetchStocksFromDataCenter(ctx)
	if len(items) > 0 {
		source = "datacenter"
	}

	if len(items) == 0 {
		s.logger.Warn("data center API failed, falling back to clist API")
		items = s.fetchAllStocksFromAPI(ctx)
		source = "clist"
	}

	if len(items) == 0 {
		s.logger.Warn("clist API failed, trying provider router")
		if s.router != nil {
			_ = s.router.Fetch(ctx, CapStockSync, MarketCN, func(ctx context.Context, p Provider) error {
				provider, ok := p.(StockSyncProvider)
				if !ok {
					return fmt.Errorf("provider %s does not implement StockSyncProvider", p.Name())
				}
				result, err := provider.SyncStocks(ctx)
				if err != nil {
					return err
				}
				items = result
				return nil
			})
			if len(items) > 0 {
				source = "provider"
			}
		}
	}

	if len(items) == 0 {
		s.logger.Warn("remote stock sync failed, trying cache store")
		if s.stockStore != nil {
			if cached, err := s.stockStore.GetStockList(); err == nil && len(cached) > 0 {
				items = cached
				source = "cache"
			}
		}
	}

	if len(items) == 0 {
		s.logger.Warn("remote stock sync failed, falling back to default stock universe")
		items = seed.LoadDefaultStocks()
		source = "default"
	}

	if len(items) == 0 {
		return stockdomain.StockSyncResult{}, fmt.Errorf("failed to load default stocks")
	}

	seen := make(map[string]bool)
	errCount := 0
	validItems := make([]stockdomain.StockItem, 0, len(items))
	for _, item := range items {
		if len(item.StockCode) != 6 || !httpclient.IsAllDigits(item.StockCode) {
			errCount++
			continue
		}
		if item.StockName == "" {
			errCount++
			continue
		}
		if seen[item.StockCode] {
			continue
		}
		seen[item.StockCode] = true
		validItems = append(validItems, item)
	}

	defaults := seed.LoadDefaultStocks()
	merged := 0
	for _, d := range defaults {
		if !seen[d.StockCode] && d.StockName != "" {
			seen[d.StockCode] = true
			validItems = append(validItems, d)
			merged++
		}
	}

	if err := s.store.ReplaceStocks(validItems); err != nil {
		return stockdomain.StockSyncResult{}, fmt.Errorf("failed to store synced stocks: %w", err)
	}

	// Async save to cache store for offline fallback
	if s.stockStore != nil && len(validItems) > 0 && source != "cache" {
		go func() {
			if err := s.stockStore.SaveStockList(validItems); err != nil {
				s.logger.Warn("failed to save stock list to cache store", "error", err)
			}
		}()
	}

	s.logger.Info("stock sync completed", "source", source, "total", len(items), "imported", len(validItems), "errors", errCount, "merged_from_defaults", merged)
	return stockdomain.StockSyncResult{
		Total:    len(items),
		Imported: len(validItems),
		Errors:   errCount,
	}, nil
}

// convertSearchItems 将搜索结果项转换为股票列表项。
func (s *StockService) convertSearchItems(searchItems []stockdomain.StockSearchItem) []stockdomain.StockItem {
	items := make([]stockdomain.StockItem, len(searchItems))
	for i, si := range searchItems {
		items[i] = stockdomain.StockItem{
			StockCode: si.StockCode,
			StockName: si.StockName,
			Market:    si.Market,
			Pinyin:    si.Pinyin,
		}
	}
	return items
}

// filterItems 根据关键词、行业、市场等条件过滤股票列表。
func (s *StockService) filterItems(items []stockdomain.StockItem, q stockdomain.StockSearchRequest) []stockdomain.StockItem {
	keyword := strings.TrimSpace(strings.ToLower(q.Keyword))
	filtered := make([]stockdomain.StockItem, 0)
	for _, stock := range items {
		if keyword != "" && !stockMatchesKeyword(stock, keyword) {
			continue
		}
		if q.Industry != "" && stock.Industry != q.Industry {
			continue
		}
		if q.Market != "" && stock.Market != q.Market {
			continue
		}
		filtered = append(filtered, stock)
	}
	sortStockItems(filtered, q.SortBy, q.SortOrder, keyword)
	return filtered
}

// paginate 对过滤后的股票列表进行分页。
func (s *StockService) paginate(items []stockdomain.StockItem, page, size int) stockdomain.StockSearchData {
	total := len(items)
	start := (page - 1) * size
	if start > total {
		start = total
	}
	end := start + size
	if end > total {
		end = total
	}
	return stockdomain.StockSearchData{
		Items: items[start:end],
		Total: total,
		Page:  page,
		Size:  size,
	}
}
