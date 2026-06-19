package seed

import (
	"testing"

	httpclient "stock-predict-go/internal/platform/httpclient"
)

func TestLoadDefaultStocksReturnsNonEmpty(t *testing.T) {
	stocks := LoadDefaultStocks()
	if len(stocks) == 0 {
		t.Fatalf("expected non-empty stock list")
	}
}

func TestLoadDefaultStocksCodeIsSixDigits(t *testing.T) {
	stocks := LoadDefaultStocks()
	for _, s := range stocks {
		if len(s.StockCode) != 6 || !httpclient.IsAllDigits(s.StockCode) {
			t.Fatalf("expected 6-digit stock code, got %q", s.StockCode)
		}
	}
}

func TestLoadDefaultStocksNameNonEmpty(t *testing.T) {
	stocks := LoadDefaultStocks()
	for _, s := range stocks {
		if s.StockName == "" {
			t.Fatalf("expected non-empty stock name for code %q", s.StockCode)
		}
	}
}
