package providers

import (
	"testing"
	"time"
)

func TestSearchSizeConstants(t *testing.T) {
	if DefaultSearchSize != 20 {
		t.Fatalf("expected DefaultSearchSize=20, got %d", DefaultSearchSize)
	}
	if MaxSearchSize != 50 {
		t.Fatalf("expected MaxSearchSize=50, got %d", MaxSearchSize)
	}
}

func TestBatchSizeConstants(t *testing.T) {
	if MaxStockQuoteBatch != 50 {
		t.Fatalf("expected MaxStockQuoteBatch=50, got %d", MaxStockQuoteBatch)
	}
	if MaxWatchlistBatch != 50 {
		t.Fatalf("expected MaxWatchlistBatch=50, got %d", MaxWatchlistBatch)
	}
}

func TestHTTPClientConstants(t *testing.T) {
	if HTTPClientTimeout != 8*time.Second {
		t.Fatalf("expected HTTPClientTimeout=8s, got %v", HTTPClientTimeout)
	}
	if MaxHTTPPayloadBytes != 2<<20 {
		t.Fatalf("expected MaxHTTPPayloadBytes=%d, got %d", 2<<20, MaxHTTPPayloadBytes)
	}
}

func TestCacheConstants(t *testing.T) {
	if CacheMaxEntries != 1000 {
		t.Fatalf("expected CacheMaxEntries=1000, got %d", CacheMaxEntries)
	}
	if CacheTTL != 5*time.Minute {
		t.Fatalf("expected CacheTTL=5m, got %v", CacheTTL)
	}
}

func TestRiskFreeRate(t *testing.T) {
	if RiskFreeRate != 0.015 {
		t.Fatalf("expected RiskFreeRate=0.015, got %f", RiskFreeRate)
	}
}

func TestTradingDaysPerYear(t *testing.T) {
	if TradingDaysPerYear != 252 {
		t.Fatalf("expected TradingDaysPerYear=252, got %d", TradingDaysPerYear)
	}
}
