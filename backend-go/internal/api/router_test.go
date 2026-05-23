package api_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"stock-predict-go/internal/api"
	"stock-predict-go/internal/config"
	"stock-predict-go/internal/dto"
	"stock-predict-go/internal/service"
	"stock-predict-go/internal/store"
)

func newTestHandler() http.Handler {
	cfg := config.Config{
		Port:            "0",
		Env:             "test",
		CORSOrigins:     []string{"http://localhost:5173"},
		ReadTimeout:     1,
		WriteTimeout:    1,
		ShutdownTimeout: 1,
	}
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	services := service.NewRegistry(store.NewMemoryStore(), cfg, logger)
	return api.NewRouter(cfg, services, logger)
}

func TestHealth(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)

	newTestHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var body struct {
		Status  string `json:"status"`
		Runtime string `json:"runtime"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if body.Status != "ok" || body.Runtime != "go" {
		t.Fatalf("unexpected health response: %+v", body)
	}
}

func TestMarketIndicesIncludeSP500(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/indices", nil)

	newTestHandler().ServeHTTP(rec, req)

	var body dto.APIResponse
	decodeResponse(t, rec, &body)
	items, err := remarshal[[]dto.MarketIndex](body.Data)
	if err != nil {
		t.Fatalf("decode market indices data: %v", err)
	}
	if len(items) < 8 {
		t.Fatalf("expected at least 8 market indices, got %d", len(items))
	}
	for _, item := range items {
		if item.Code == "SPX" && item.Name == "标普500" && item.Market == "us" {
			return
		}
	}
	t.Fatalf("SPX/标普500 index missing from response: %+v", items)
}

func TestPredictFund(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/predict/000001", nil)

	newTestHandler().ServeHTTP(rec, req)

	var body dto.APIResponse
	decodeResponse(t, rec, &body)
	data, err := remarshal[dto.PredictionData](body.Data)
	if err != nil {
		t.Fatalf("decode prediction data: %v", err)
	}
	if data.FundCode != "000001" {
		t.Fatalf("expected fund 000001, got %q", data.FundCode)
	}
	if data.NextDayPrediction.Horizon != "next_day" {
		t.Fatalf("unexpected next-day horizon: %q", data.NextDayPrediction.Horizon)
	}
	if data.IntradayPrediction.Horizon != "intraday_5m" {
		t.Fatalf("unexpected intraday horizon: %q", data.IntradayPrediction.Horizon)
	}
}

func TestWatchlistQuotes(t *testing.T) {
	body := bytes.NewBufferString(`{"codes":["000001","110011"]}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/watchlist/quotes", body)
	req.Header.Set("Content-Type", "application/json")

	newTestHandler().ServeHTTP(rec, req)

	var response dto.APIResponse
	decodeResponse(t, rec, &response)
	items, err := remarshal[[]dto.WatchlistItem](response.Data)
	if err != nil {
		t.Fatalf("decode watchlist data: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected two watchlist quotes, got %d", len(items))
	}
}

func TestInvalidMarketRankingType(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/ranking/bad", nil)

	newTestHandler().ServeHTTP(rec, req)

	response := decodeAPIResponse(t, rec, http.StatusOK)
	if response.Code != -1 {
		t.Fatalf("expected API code -1, got %d", response.Code)
	}
}

func TestInvalidPredictCode(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/predict/abc", nil)

	newTestHandler().ServeHTTP(rec, req)

	response := decodeAPIResponse(t, rec, http.StatusOK)
	if response.Code != -1 {
		t.Fatalf("expected API code -1, got %d", response.Code)
	}
}

func TestWatchlistQuotesRejectsUnknownFields(t *testing.T) {
	body := bytes.NewBufferString(`{"codes":["000001"],"extra":true}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/watchlist/quotes", body)
	req.Header.Set("Content-Type", "application/json")

	newTestHandler().ServeHTTP(rec, req)

	response := decodeAPIResponse(t, rec, http.StatusBadRequest)
	if response.Code != -1 {
		t.Fatalf("expected API code -1, got %d", response.Code)
	}
}

func TestWatchlistQuotesRejectsInvalidFundCode(t *testing.T) {
	body := bytes.NewBufferString(`{"codes":["000001","abc"]}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/watchlist/quotes", body)
	req.Header.Set("Content-Type", "application/json")

	newTestHandler().ServeHTTP(rec, req)

	response := decodeAPIResponse(t, rec, http.StatusBadRequest)
	if response.Code != -1 {
		t.Fatalf("expected API code -1, got %d", response.Code)
	}
}

func TestWatchlistQuotesRejectsTrailingJSON(t *testing.T) {
	body := bytes.NewBufferString(`{"codes":["000001"]}{}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/watchlist/quotes", body)
	req.Header.Set("Content-Type", "application/json")

	newTestHandler().ServeHTTP(rec, req)

	response := decodeAPIResponse(t, rec, http.StatusBadRequest)
	if response.Code != -1 {
		t.Fatalf("expected API code -1, got %d", response.Code)
	}
}

func TestCORSPreflight(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/health", nil)
	req.Header.Set("Origin", "http://localhost:5173")

	newTestHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d: %s", rec.Code, rec.Body.String())
	}
	if origin := rec.Header().Get("Access-Control-Allow-Origin"); origin != "http://localhost:5173" {
		t.Fatalf("unexpected CORS origin header: %q", origin)
	}
}

func TestNoRouteReturnsJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/not-found", nil)

	newTestHandler().ServeHTTP(rec, req)

	response := decodeAPIResponse(t, rec, http.StatusNotFound)
	if response.Code != -1 || response.Message != "not found" {
		t.Fatalf("unexpected not-found response: %+v", response)
	}
}

func decodeResponse(t *testing.T, rec *httptest.ResponseRecorder, target any) {
	t.Helper()
	response := decodeAPIResponse(t, rec, http.StatusOK)
	raw, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("remarshal response: %v", err)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Code != 0 {
		t.Fatalf("expected API code 0, got %d: %s", response.Code, response.Message)
	}
}

func decodeAPIResponse(t *testing.T, rec *httptest.ResponseRecorder, expectedStatus int) dto.APIResponse {
	t.Helper()
	if rec.Code != expectedStatus {
		t.Fatalf("expected status %d, got %d: %s", expectedStatus, rec.Code, rec.Body.String())
	}
	var response dto.APIResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return response
}

func remarshal[T any](value any) (T, error) {
	var out T
	raw, err := json.Marshal(value)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, err
	}
	return out, nil
}
