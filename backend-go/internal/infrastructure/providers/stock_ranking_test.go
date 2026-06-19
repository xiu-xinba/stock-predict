package providers

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	stockdomain "stock-predict-go/internal/domain/stock"
	database "stock-predict-go/internal/infrastructure/database"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestStockRankingFallsBackToLocalWhenAPIsFail(t *testing.T) {
	db := database.InitTestDB(t)
	repo := database.NewStockStore(db)
	repo.ReplaceStocks([]stockdomain.StockItem{
		{
			StockCode:    "600519",
			StockName:    "Local Placeholder",
			CurrentPrice: 100,
			ChangePct:    10,
			Volume:       1000,
			Amount:       10000,
		},
	})
	svc := NewStockService(repo, nil)
	svc.eastmoney = newEastmoneyClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`upstream failed`)),
				Request:    req,
			}, nil
		}),
	})
	svc.eastmoney.minInterval = 0

	items, err := svc.Ranking(context.Background(), "gainers", 5)

	if err != nil {
		t.Fatalf("expected no error with local fallback, got err=%v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected local fallback ranking items, got none")
	}
	if items[0].DataSource != "local" {
		t.Fatalf("expected data_source=local, got %s", items[0].DataSource)
	}
	if items[0].StockCode != "600519" {
		t.Fatalf("expected stock_code=600519, got %s", items[0].StockCode)
	}
}
