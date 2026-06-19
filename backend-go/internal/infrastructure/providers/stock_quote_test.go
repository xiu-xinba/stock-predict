package providers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	stockdomain "stock-predict-go/internal/domain/stock"
	database "stock-predict-go/internal/infrastructure/database"
)

func tencentStockPayload(code string, price float64) string {
	fields := make([]string, 39)
	fields[1] = "贵州茅台"
	fields[2] = code
	fields[3] = fmt.Sprintf("%.2f", price)
	fields[4] = "1790.00"
	fields[5] = "1795.00"
	fields[6] = "1000"
	fields[9] = "1800.00"
	fields[18] = "1801.00"
	fields[30] = "20260604103000"
	fields[31] = "10.00"
	fields[32] = "0.56"
	fields[33] = "1810.00"
	fields[34] = "1788.00"
	fields[37] = "1800000"
	fields[38] = "0.12"
	return fmt.Sprintf(`v_sh%s="%s";`, code, strings.Join(fields, "~"))
}

func newTestStockQuoteClient(base *http.Client, cacheTTL time.Duration) *StockQuoteClient {
	client := NewStockQuoteClient(time.Second)
	client.client = base
	client.resilient = NewResilientHTTPClient(base, []SourcePolicy{{
		Source:          SourceTencent,
		UserAgent:       "StockPredict-Test/1.0",
		Referer:         "https://gu.qq.com/",
		MinInterval:     0,
		CooldownOnLimit: 0,
		CooldownOnError: 0,
	}})
	client.quoteCache = NewDetailCache(CacheMaxEntries, cacheTTL)
	client.marketOpen = func([]string) bool { return true }
	return client
}

func ageStockQuoteCache(t *testing.T, client *StockQuoteClient, code string, age time.Duration) {
	t.Helper()
	if !client.quoteCache.Backdate(code, age) {
		t.Fatalf("expected quote cache entry for %s", code)
	}
}

func waitForCachedQuotePrice(t *testing.T, client *StockQuoteClient, code string, price float64) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		value, ok := client.quoteCache.Peek(code)
		var got float64
		if ok {
			if quote, ok := value.(stockdomain.StockQuote); ok {
				got = quote.Price
			}
		}
		if got == price {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected cached quote %s to refresh to %.2f", code, price)
}

func TestStockQuoteClientReturnsStaleCacheWhenProviderUnavailable(t *testing.T) {
	callCount := 0
	base := &http.Client{Transport: resilientRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		callCount++
		if callCount == 1 {
			return testResponse(http.StatusOK, tencentStockPayload("600519", 1800), nil), nil
		}
		return testResponse(http.StatusTooManyRequests, `{}`, nil), nil
	})}
	client := newTestStockQuoteClient(base, 0)

	first := client.FetchQuotes(context.Background(), []string{"600519"})
	if first["600519"].Price != 1800 {
		t.Fatalf("expected first quote from provider, got %+v", first)
	}
	second := client.FetchQuotes(context.Background(), []string{"600519"})
	if second["600519"].Price != 1800 {
		t.Fatalf("expected stale cached quote after provider failure, got %+v", second)
	}
	if callCount != 2 {
		t.Fatalf("expected provider to be called twice, got %d", callCount)
	}
}

func TestStockQuoteClientCoalescesIdenticalBatchRequests(t *testing.T) {
	var mu sync.Mutex
	callCount := 0
	release := make(chan struct{})
	base := &http.Client{Transport: resilientRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		mu.Lock()
		callCount++
		mu.Unlock()
		<-release
		return testResponse(http.StatusOK, tencentStockPayload("600519", 1800), nil), nil
	})}
	client := newTestStockQuoteClient(base, StockQuoteCacheTTL)

	results := make(chan map[string]float64, 2)
	for i := 0; i < 2; i++ {
		go func() {
			quotes := client.FetchQuotes(context.Background(), []string{"600519"})
			results <- map[string]float64{"600519": quotes["600519"].Price}
		}()
	}

	time.Sleep(20 * time.Millisecond)
	close(release)
	for i := 0; i < 2; i++ {
		result := <-results
		if result["600519"] != 1800 {
			t.Fatalf("expected coalesced quote result, got %+v", result)
		}
	}
	if callCount != 1 {
		t.Fatalf("expected one upstream request for concurrent identical batch, got %d", callCount)
	}
}

func TestStockQuoteClientBalancedUsesLongerFreshTTL(t *testing.T) {
	var callCount atomic.Int32
	base := &http.Client{Transport: resilientRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		callCount.Add(1)
		return testResponse(http.StatusOK, tencentStockPayload("600519", 1900), nil), nil
	})}
	client := newTestStockQuoteClient(base, StockQuoteCacheTTL)
	client.quoteCache.Set("600519", stockdomain.StockQuote{Price: 1800})
	ageStockQuoteCache(t, client, "600519", 10*time.Second)

	quotes := client.FetchQuotesWithOptions(context.Background(), []string{"600519"}, StockQuoteOptions{Freshness: StockQuoteFreshnessBalanced})

	if quotes["600519"].Price != 1800 {
		t.Fatalf("expected balanced freshness to use 10s cached quote, got %+v", quotes["600519"])
	}
	if got := callCount.Load(); got != 0 {
		t.Fatalf("expected no upstream call for balanced fresh cache, got %d", got)
	}
}

func TestStockQuoteClientRealtimeReturnsStaleAndRefreshesInBackground(t *testing.T) {
	var callCount atomic.Int32
	refreshed := make(chan struct{}, 1)
	base := &http.Client{Transport: resilientRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		callCount.Add(1)
		select {
		case refreshed <- struct{}{}:
		default:
		}
		return testResponse(http.StatusOK, tencentStockPayload("600519", 1900), nil), nil
	})}
	client := newTestStockQuoteClient(base, StockQuoteCacheTTL)
	client.quoteCache.Set("600519", stockdomain.StockQuote{Price: 1800})
	ageStockQuoteCache(t, client, "600519", 5*time.Second)

	first := client.FetchQuotesWithOptions(context.Background(), []string{"600519"}, StockQuoteOptions{Freshness: StockQuoteFreshnessRealtime})
	if first["600519"].Price != 1800 {
		t.Fatalf("expected realtime request to return stale quote immediately, got %+v", first["600519"])
	}

	select {
	case <-refreshed:
	case <-time.After(time.Second):
		t.Fatal("expected stale realtime quote to trigger background refresh")
	}
	waitForCachedQuotePrice(t, client, "600519", 1900)

	second := client.FetchQuotesWithOptions(context.Background(), []string{"600519"}, StockQuoteOptions{Freshness: StockQuoteFreshnessRealtime})
	if second["600519"].Price != 1900 {
		t.Fatalf("expected refreshed quote on next realtime request, got %+v", second["600519"])
	}
	if got := callCount.Load(); got != 1 {
		t.Fatalf("expected one background upstream call, got %d", got)
	}
}

func TestSyncStocksFallsBackToDefaultStocksWhenRemoteSourcesFail(t *testing.T) {
	base := &http.Client{Transport: resilientRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("remote unavailable")
	})}
	db := database.InitTestDB(t)
	repo := database.NewStockStore(db)
	service := NewStockService(repo, nil)
	service.eastmoney = newEastmoneyClient(base)
	service.eastmoney.minInterval = 0
	service.eastmoney.jitter = nil
	service.eastmoney.resilient = NewResilientHTTPClient(base, []SourcePolicy{{
		Source:          SourceEastmoney,
		MinInterval:     0,
		CooldownOnError: 0,
	}})

	result, err := service.SyncStocks(context.Background())

	if err != nil {
		t.Fatalf("expected default stock fallback, got error: %v", err)
	}
	if result.Imported == 0 || result.Total == 0 {
		t.Fatalf("expected imported default stocks, got %+v", result)
	}
	if _, ok := repo.FindStock("600519"); !ok {
		t.Fatalf("expected default stock 600519 to be stored")
	}
}
