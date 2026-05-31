package service

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"stock-predict-go/internal/data"
	"stock-predict-go/internal/dto"
	"stock-predict-go/internal/util"
)

var (
	ErrInvalidStockCode = errors.New("invalid stock code")
	ErrStockNotFound    = errors.New("stock not found")
)

type rankingCacheEntry struct {
	items    []dto.StockRankingItem
	cachedAt time.Time
}

type StockService struct {
	client         *http.Client
	logger         *slog.Logger
	stocks         []dto.StockItem
	mu             sync.RWMutex
	rankingCache   map[string]*rankingCacheEntry
	rankingCacheMu sync.RWMutex
}

func NewStockService(logger *slog.Logger) *StockService {
	if logger == nil {
		logger = slog.Default()
	}
	svc := &StockService{
		client: &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					d := net.Dialer{
						Timeout:   10 * time.Second,
						KeepAlive: 30 * time.Second,
					}
					return d.DialContext(ctx, "tcp4", addr)
				},
				TLSClientConfig: &tls.Config{
					Renegotiation: tls.RenegotiateFreelyAsClient,
				},
				TLSNextProto:      make(map[string]func(string, *tls.Conn) http.RoundTripper),
				ForceAttemptHTTP2: false,
			},
		},
		logger:       logger,
		stocks:       data.LoadDefaultStocks(),
		rankingCache: make(map[string]*rankingCacheEntry),
	}
	return svc
}

func (s *StockService) Search(ctx context.Context, q dto.StockSearchRequest) dto.StockSearchData {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Size < 1 {
		q.Size = 20
	}
	if q.Size > 50 {
		q.Size = 50
	}

	keyword := strings.TrimSpace(q.Keyword)

	if keyword != "" {
		items := s.searchFromAPI(ctx, keyword)
		if len(items) > 0 {
			filtered := s.filterItems(items, q)
			return s.paginate(filtered, q.Page, q.Size)
		}
	}

	s.mu.RLock()
	localStocks := make([]dto.StockItem, len(s.stocks))
	copy(localStocks, s.stocks)
	s.mu.RUnlock()

	filtered := s.filterItems(localStocks, q)
	return s.paginate(filtered, q.Page, q.Size)
}

func (s *StockService) Filters() dto.StockFilters {
	s.mu.RLock()
	stocksCopy := make([]dto.StockItem, len(s.stocks))
	copy(stocksCopy, s.stocks)
	s.mu.RUnlock()

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
	return dto.StockFilters{
		Industries: keys(industries),
		Markets:    keys(markets),
	}
}

func (s *StockService) Ranking(ctx context.Context, rankingType string, size int) ([]dto.StockRankingItem, error) {
	if rankingType != "gainers" && rankingType != "losers" && rankingType != "volume" {
		return nil, ErrInvalidRankingType
	}
	if size < 1 {
		size = 10
	}
	if size > 50 {
		size = 50
	}

	cacheKey := rankingType + strconv.Itoa(size)
	s.rankingCacheMu.RLock()
	if entry, ok := s.rankingCache[cacheKey]; ok && time.Since(entry.cachedAt) < 30*time.Second {
		items := entry.items
		s.rankingCacheMu.RUnlock()
		return items, nil
	}
	s.rankingCacheMu.RUnlock()

	items := s.fetchRankingFromAPI(ctx, rankingType, size)
	if len(items) == 0 {
		items = s.localRanking(rankingType, size)
	}

	s.rankingCacheMu.Lock()
	s.rankingCache[cacheKey] = &rankingCacheEntry{items: items, cachedAt: time.Now()}
	s.rankingCacheMu.Unlock()

	return items, nil
}

func (s *StockService) FindStock(code string) (dto.StockItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, stock := range s.stocks {
		if stock.StockCode == code {
			return stock, nil
		}
	}
	return dto.StockItem{}, ErrStockNotFound
}

func (s *StockService) IsLoaded() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.stocks) > 0
}

func (s *StockService) ListStocks() []dto.StockItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]dto.StockItem, len(s.stocks))
	copy(out, s.stocks)
	return out
}

func (s *StockService) SyncStocks(ctx context.Context) (dto.StockSyncResult, error) {
	items := s.fetchAllStocksFromAPI(ctx)
	source := "clist"

	if len(items) == 0 {
		s.logger.Warn("clist API failed, falling back to data center API")
		items = s.fetchStocksFromDataCenter(ctx)
		source = "datacenter"
	}

	if len(items) == 0 {
		return dto.StockSyncResult{}, fmt.Errorf("failed to fetch stocks from all APIs")
	}

	seen := make(map[string]bool)
	errCount := 0
	validItems := make([]dto.StockItem, 0, len(items))
	for _, item := range items {
		if len(item.StockCode) != 6 || !util.IsAllDigits(item.StockCode) {
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

	defaults := data.LoadDefaultStocks()
	merged := 0
	for _, d := range defaults {
		if !seen[d.StockCode] && d.StockName != "" {
			seen[d.StockCode] = true
			validItems = append(validItems, d)
			merged++
		}
	}

	s.mu.Lock()
	s.stocks = validItems
	s.mu.Unlock()

	s.logger.Info("stock sync completed", "source", source, "total", len(items), "imported", len(validItems), "errors", errCount, "merged_from_defaults", merged)
	return dto.StockSyncResult{
		Total:    len(items),
		Imported: len(validItems),
		Errors:   errCount,
	}, nil
}

func (s *StockService) filterItems(items []dto.StockItem, q dto.StockSearchRequest) []dto.StockItem {
	keyword := strings.TrimSpace(strings.ToLower(q.Keyword))
	filtered := make([]dto.StockItem, 0)
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

func (s *StockService) paginate(items []dto.StockItem, page, size int) dto.StockSearchData {
	total := len(items)
	start := (page - 1) * size
	if start > total {
		start = total
	}
	end := start + size
	if end > total {
		end = total
	}
	return dto.StockSearchData{
		Items: items[start:end],
		Total: total,
		Page:  page,
		Size:  size,
	}
}
