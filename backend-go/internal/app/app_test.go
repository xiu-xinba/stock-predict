package app

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	stockdomain "stock-predict-go/internal/domain/stock"
	database "stock-predict-go/internal/infrastructure/database"
	"stock-predict-go/internal/platform/config"
)

func TestNewServerRespondsToHealth(t *testing.T) {
	cfg := config.Config{
		Port:                  "0",
		Env:                   "test",
		CORSOrigins:           []string{"http://localhost:5173"},
		FundAutoSyncOnStart:   false,
		StockAutoSyncOnStart:  false,
		MarketSyncEnabled:     false,
		DatabaseURL:           "postgres://stock:stock123@localhost:5432/stock_predict_test?sslmode=disable",
		RunDatabaseMigrations: true,
		ReadTimeout:           time.Second,
		WriteTimeout:          time.Second,
		ShutdownTimeout:       time.Second,
	}
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

	server, cleanup, err := NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	defer cleanup()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	server.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected health 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSeedDefaultStocksPreservesExistingUniverse(t *testing.T) {
	db := database.InitTestDB(t)
	stockStore := database.NewStockStore(db)
	if err := stockStore.SaveStockList([]stockdomain.StockItem{{
		StockCode: "999999",
		StockName: "自定义股票",
	}}); err != nil {
		t.Fatalf("seed custom stock: %v", err)
	}

	if err := seedDefaultStocks(stockStore); err != nil {
		t.Fatalf("seed default stocks: %v", err)
	}

	if _, ok := stockStore.FindStock("999999"); !ok {
		t.Fatal("existing stock was removed while seeding defaults")
	}
	if got := stockStore.CountStocks(); got != 1 {
		t.Fatalf("expected existing universe to remain unchanged, got %d stocks", got)
	}
}
