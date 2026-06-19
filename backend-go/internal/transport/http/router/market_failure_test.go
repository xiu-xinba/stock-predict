package router_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	database "stock-predict-go/internal/infrastructure/database"
	providers "stock-predict-go/internal/infrastructure/providers"
	"stock-predict-go/internal/platform/config"
	transporthttp "stock-predict-go/internal/transport/http/router"
)

func newUnavailableMarketHandler(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Config{
		Port:            "0",
		Env:             "test",
		CORSOrigins:     []string{"http://localhost:5173"},
		ReadTimeout:     1,
		WriteTimeout:    1,
		ShutdownTimeout: 1,
	}
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	db := database.InitTestDB(t)
	fundStore := database.NewFundStore(db)
	stockStore := database.NewStockStore(db)
	searchIdx := database.NewSearchStore(db)
	services := providers.NewRegistry(fundStore, stockStore, cfg, logger, searchIdx, nil, db)
	services.Market = providers.NewMarketService(nil, logger)
	return transporthttp.NewRouter(cfg, services, fundStore, logger, searchIdx)
}

func TestMarketIndicesReturnsUnavailableWhenProvidersFail(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/indices", nil)

	newUnavailableMarketHandler(t).ServeHTTP(rec, req)

	response := decodeAPIResponse(t, rec, http.StatusServiceUnavailable)
	if response.Code != -1 || response.Message != "行情数据暂不可用" {
		t.Fatalf("unexpected unavailable response: %+v", response)
	}
}

func TestIndexMinuteReturnsUnavailableWhenProvidersFail(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/index/000001/minute", nil)

	newUnavailableMarketHandler(t).ServeHTTP(rec, req)

	response := decodeAPIResponse(t, rec, http.StatusServiceUnavailable)
	if response.Code != -1 || response.Message != "指数分时数据暂不可用" {
		t.Fatalf("unexpected unavailable response: %+v", response)
	}
}

func TestIndexKlineReturnsUnavailableWhenProvidersFail(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/index/000001/kline", nil)

	newUnavailableMarketHandler(t).ServeHTTP(rec, req)

	response := decodeAPIResponse(t, rec, http.StatusServiceUnavailable)
	if response.Code != -1 || response.Message != "指数历史数据暂不可用" {
		t.Fatalf("unexpected unavailable response: %+v", response)
	}
}
