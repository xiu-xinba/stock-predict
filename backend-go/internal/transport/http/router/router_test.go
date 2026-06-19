package router_test

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	funddomain "stock-predict-go/internal/domain/fund"
	marketdomain "stock-predict-go/internal/domain/market"
	stockdomain "stock-predict-go/internal/domain/stock"
	database "stock-predict-go/internal/infrastructure/database"
	providers "stock-predict-go/internal/infrastructure/providers"
	"stock-predict-go/internal/platform/config"
	transporthttp "stock-predict-go/internal/transport/http/router"

	"gorm.io/gorm"
)

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	cfg := config.Config{
		Port:            "0",
		Env:             "test",
		AdminToken:      "test-admin-token",
		CORSOrigins:     []string{"http://localhost:5173"},
		ReadTimeout:     1,
		WriteTimeout:    1,
		ShutdownTimeout: 1,
	}
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	db := database.InitTestDB(t)
	fundStore := database.NewFundStore(db)
	stockStore := database.NewStockStore(db)
	database.SeedFunds(db)
	searchIdx := database.NewSearchStore(db)
	services := providers.NewRegistry(fundStore, stockStore, cfg, logger, searchIdx, nil, db)
	return transporthttp.NewRouter(cfg, services, fundStore, logger, searchIdx)
}

func newTestHandlerWithConfig(cfg config.Config, fundStore *database.FundStore, stockStore *database.StockStore, db *gorm.DB) http.Handler {
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	searchIdx := database.NewSearchStore(db)
	services := providers.NewRegistry(fundStore, stockStore, cfg, logger, searchIdx, nil, db)
	return transporthttp.NewRouter(cfg, services, fundStore, logger, searchIdx)
}

func TestHealth(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)

	newTestHandler(t).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var body struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Status  string `json:"status"`
			Runtime string `json:"runtime"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if body.Code != 0 || body.Data.Status != "ok" || body.Data.Runtime != "go" {
		t.Fatalf("unexpected health response: %+v", body)
	}
}

func TestReadinessFailsWhenDatabaseIsUnavailable(t *testing.T) {
	cfg := config.Config{
		Port:            "0",
		Env:             "test",
		CORSOrigins:     []string{"http://localhost:5173"},
		ReadTimeout:     1,
		WriteTimeout:    1,
		ShutdownTimeout: 1,
	}
	db := database.InitTestDB(t)
	fundStore := database.NewFundStore(db)
	stockStore := database.NewStockStore(db)
	handler := newTestHandlerWithConfig(cfg, fundStore, stockStore, db)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql database: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql database: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/ready", nil)
	handler.ServeHTTP(rec, req)

	response := decodeAPIResponse(t, rec, http.StatusServiceUnavailable)
	if response.Code != -1 {
		t.Fatalf("expected readiness error code, got %+v", response)
	}
}

func TestSearchEndpointsFailWhenDatabaseIsUnavailable(t *testing.T) {
	cfg := config.Config{
		Port:            "0",
		Env:             "test",
		AdminToken:      "test-admin-token",
		CORSOrigins:     []string{"http://localhost:5173"},
		ReadTimeout:     1,
		WriteTimeout:    1,
		ShutdownTimeout: 1,
	}
	db := database.InitTestDB(t)
	fundStore := database.NewFundStore(db)
	stockStore := database.NewStockStore(db)
	handler := newTestHandlerWithConfig(cfg, fundStore, stockStore, db)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql database: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close sql database: %v", err)
	}

	for _, path := range []string{
		"/api/v1/funds/search?keyword=000001",
		"/api/v1/stocks/search",
		"/api/v1/search?q=000001",
	} {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			handler.ServeHTTP(rec, req)

			decodeAPIResponse(t, rec, http.StatusInternalServerError)
		})
	}
}

func TestMetricsCountsRequestsAndErrors(t *testing.T) {
	handler := newTestHandler(t)

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/missing", nil))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	handler.ServeHTTP(rec, req)

	response := decodeAPIResponse(t, rec, http.StatusOK)
	if response.Code != 0 {
		t.Fatalf("expected API code 0, got %d: %s", response.Code, response.Message)
	}
	data, err := remarshal[struct {
		RequestCount int            `json:"request_count"`
		ErrorCount   int            `json:"error_count"`
		InFlight     int            `json:"in_flight"`
		StatusCounts map[string]int `json:"status_counts"`
		UptimeSec    int64          `json:"uptime_seconds"`
	}](response.Data)
	if err != nil {
		t.Fatalf("decode metrics data: %v", err)
	}
	if data.RequestCount < 2 {
		t.Fatalf("expected at least two recorded requests, got %+v", data)
	}
	if data.ErrorCount < 1 {
		t.Fatalf("expected at least one recorded error, got %+v", data)
	}
	if data.StatusCounts["200"] < 1 || data.StatusCounts["404"] < 1 {
		t.Fatalf("expected status counts for 200 and 404, got %+v", data.StatusCounts)
	}
	if data.InFlight < 1 {
		t.Fatalf("expected metrics request to be counted in-flight, got %+v", data)
	}
	if data.UptimeSec < 0 {
		t.Fatalf("expected non-negative uptime, got %+v", data)
	}
}

func TestMetricsRequiresAdminToken(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics", nil)

	newTestHandler(t).ServeHTTP(rec, req)

	decodeAPIResponse(t, rec, http.StatusUnauthorized)
}

func TestMarketIndicesIncludeSP500(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/indices", nil)

	newTestHandler(t).ServeHTTP(rec, req)

	var body transporthttp.APIResponse
	decodeResponse(t, rec, &body)
	items, err := remarshal[[]marketdomain.MarketIndex](body.Data)
	if err != nil {
		t.Fatalf("decode market indices data: %v", err)
	}
	if len(items) < 3 {
		t.Fatalf("expected at least 3 market indices (CN), got %d", len(items))
	}
	hasCN := false
	for _, item := range items {
		if item.Market == "sh" || item.Market == "sz" {
			hasCN = true
			break
		}
	}
	if !hasCN {
		t.Fatalf("expected at least one CN index in response: %+v", items)
	}
}

func TestWatchlistQuotes(t *testing.T) {
	body := bytes.NewBufferString(`{"codes":["000001","110011"]}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/watchlist/quotes", body)
	req.Header.Set("Content-Type", "application/json")

	newTestHandler(t).ServeHTTP(rec, req)

	var response transporthttp.APIResponse
	decodeResponse(t, rec, &response)
	items, err := remarshal[[]funddomain.WatchlistItem](response.Data)
	if err != nil {
		t.Fatalf("decode watchlist data: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected two watchlist quotes, got %d", len(items))
	}
}

func TestSyncFundsImportsConfiguredCSV(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "funds.csv")
	if err := os.WriteFile(csvPath, []byte("fund_code,fund_name,fund_type,latest_nav\n999999,测试同步基金,指数型,1.5\n"), 0o644); err != nil {
		t.Fatalf("write csv: %v", err)
	}
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`var r = [["999998","CSYCJJ","测试远程基金","债券型","CESHIYUANCHENGJIJIN"]];`))
	}))
	defer remote.Close()
	cfg := config.Config{
		Port:            "0",
		Env:             "test",
		AdminToken:      "test-admin-token",
		CORSOrigins:     []string{"http://localhost:5173"},
		FundUniverseURL: remote.URL,
		FundSyncCSVPath: csvPath,
		ReadTimeout:     1,
		WriteTimeout:    1,
		ShutdownTimeout: 1,
	}
	db := database.InitTestDB(t)
	fundStore := database.NewFundStore(db)
	stockStore := database.NewStockStore(db)
	database.SeedFunds(db)
	handler := newTestHandlerWithConfig(cfg, fundStore, stockStore, db)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/funds/sync", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	handler.ServeHTTP(rec, req)

	response := decodeAPIResponse(t, rec, http.StatusOK)
	if response.Code != 0 {
		t.Fatalf("expected API code 0, got %d: %s", response.Code, response.Message)
	}
	result, err := remarshal[funddomain.FundSyncResult](response.Data)
	if err != nil {
		t.Fatalf("decode sync result: %v", err)
	}
	if result.Imported != 2 || result.Total <= 2 {
		t.Fatalf("unexpected sync result: %+v", result)
	}
	if _, ok := fundStore.FindFund("999999"); !ok {
		t.Fatalf("expected csv synced fund in store")
	}
	if _, ok := fundStore.FindFund("999998"); !ok {
		t.Fatalf("expected remote synced fund in store")
	}
	if _, ok := fundStore.FindFund("000001"); !ok {
		t.Fatalf("expected sync to preserve seed fund 000001")
	}
}

func TestInvalidMarketRankingType(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/market/ranking/bad", nil)

	newTestHandler(t).ServeHTTP(rec, req)

	response := decodeAPIResponse(t, rec, http.StatusBadRequest)
	if response.Code != -1 {
		t.Fatalf("expected API code -1, got %d", response.Code)
	}
}

func TestWatchlistQuotesRejectsUnknownFields(t *testing.T) {
	body := bytes.NewBufferString(`{"codes":["000001"],"extra":true}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/watchlist/quotes", body)
	req.Header.Set("Content-Type", "application/json")

	newTestHandler(t).ServeHTTP(rec, req)

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

	newTestHandler(t).ServeHTTP(rec, req)

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

	newTestHandler(t).ServeHTTP(rec, req)

	response := decodeAPIResponse(t, rec, http.StatusBadRequest)
	if response.Code != -1 {
		t.Fatalf("expected API code -1, got %d", response.Code)
	}
}

func TestStockQuotesAcceptsRealtimeFreshness(t *testing.T) {
	body := bytes.NewBufferString(`{"codes":[],"freshness":"realtime"}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/stocks/quotes", body)
	req.Header.Set("Content-Type", "application/json")

	newTestHandler(t).ServeHTTP(rec, req)

	response := decodeAPIResponse(t, rec, http.StatusOK)
	if response.Code != 0 {
		t.Fatalf("expected API code 0, got %d: %s", response.Code, response.Message)
	}
}

func TestStockQuotesRejectsInvalidFreshness(t *testing.T) {
	body := bytes.NewBufferString(`{"codes":[],"freshness":"aggressive"}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/stocks/quotes", body)
	req.Header.Set("Content-Type", "application/json")

	newTestHandler(t).ServeHTTP(rec, req)

	response := decodeAPIResponse(t, rec, http.StatusBadRequest)
	if response.Message != "无效的行情新鲜度参数" {
		t.Fatalf("unexpected error message: %q", response.Message)
	}
}

func TestCORSPreflight(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/health", nil)
	req.Header.Set("Origin", "http://localhost:5173")

	newTestHandler(t).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d: %s", rec.Code, rec.Body.String())
	}
	if origin := rec.Header().Get("Access-Control-Allow-Origin"); origin != "http://localhost:5173" {
		t.Fatalf("unexpected CORS origin header: %q", origin)
	}
	if exposed := rec.Header().Get("Access-Control-Expose-Headers"); exposed != "X-CSRF-Token, X-Request-ID" {
		t.Fatalf("unexpected exposed headers: %q", exposed)
	}
}

func TestCSRFHeaderAuthorizesBrowserMutation(t *testing.T) {
	cfg := config.Config{
		Port:            "0",
		Env:             "development",
		CORSOrigins:     []string{"http://localhost:5173"},
		ReadTimeout:     1,
		WriteTimeout:    1,
		ShutdownTimeout: 1,
	}
	db := database.InitTestDB(t)
	fundStore := database.NewFundStore(db)
	stockStore := database.NewStockStore(db)
	handler := newTestHandlerWithConfig(cfg, fundStore, stockStore, db)

	bootstrap := httptest.NewRecorder()
	bootstrapRequest := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	bootstrapRequest.Header.Set("Origin", "http://localhost:5173")
	handler.ServeHTTP(bootstrap, bootstrapRequest)

	token := bootstrap.Header().Get("X-CSRF-Token")
	if token == "" {
		t.Fatal("expected CSRF token response header")
	}
	cookies := bootstrap.Result().Cookies()
	if len(cookies) == 0 || !cookies[0].HttpOnly {
		t.Fatal("expected HttpOnly CSRF cookie")
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/watchlist/quotes", bytes.NewBufferString(`{"codes":[]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", token)
	req.AddCookie(cookies[0])
	handler.ServeHTTP(rec, req)

	decodeAPIResponse(t, rec, http.StatusOK)
}

func TestNoRouteReturnsJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/not-found", nil)

	newTestHandler(t).ServeHTTP(rec, req)

	response := decodeAPIResponse(t, rec, http.StatusNotFound)
	if response.Code != -1 || response.Message != "not found" {
		t.Fatalf("unexpected not-found response: %+v", response)
	}
}

func TestGzipMiddlewareReturnsValidCompressedJSON(t *testing.T) {
	cfg := config.Config{
		Port:            "0",
		Env:             "test",
		CORSOrigins:     []string{"http://localhost:5173"},
		ReadTimeout:     1,
		WriteTimeout:    1,
		ShutdownTimeout: 1,
	}
	db := database.InitTestDB(t)
	fundStore := database.NewFundStore(db)
	stockStore := database.NewStockStore(db)
	stockStore.ReplaceStocks([]stockdomain.StockItem{
		{StockCode: "600519", StockName: "贵州茅台", Market: "SH", Industry: "白酒"},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stocks/search?size=1", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	newTestHandlerWithConfig(cfg, fundStore, stockStore, db).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("expected gzip content encoding, got %q", got)
	}

	reader, err := gzip.NewReader(bytes.NewReader(rec.Body.Bytes()))
	if err != nil {
		t.Fatalf("expected gzip body, got invalid gzip stream: %v; raw=%q", err, rec.Body.String())
	}
	defer reader.Close()

	var response transporthttp.APIResponse
	if err := json.NewDecoder(reader).Decode(&response); err != nil {
		t.Fatalf("decode gzipped API response: %v", err)
	}
	if response.Code != 0 {
		t.Fatalf("expected API code 0, got %d: %s", response.Code, response.Message)
	}
	data, err := remarshal[stockdomain.StockSearchData](response.Data)
	if err != nil {
		t.Fatalf("decode stock search data: %v", err)
	}
	if len(data.Items) == 0 {
		t.Fatalf("expected stock items in gzipped response")
	}
}

func TestMarketHealthMutationsRequireAdminToken(t *testing.T) {
	cfg := config.Config{
		Port:            "0",
		Env:             "test",
		AdminToken:      "test-admin-token",
		CORSOrigins:     []string{"http://localhost:5173"},
		ReadTimeout:     1,
		WriteTimeout:    1,
		ShutdownTimeout: 1,
	}
	db := database.InitTestDB(t)
	fundStore := database.NewFundStore(db)
	stockStore := database.NewStockStore(db)
	handler := newTestHandlerWithConfig(cfg, fundStore, stockStore, db)

	for _, path := range []string{
		"/api/v1/market/health/tencent/simulate?status=unhealthy",
		"/api/v1/market/health/reset",
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, path, nil)
		handler.ServeHTTP(rec, req)

		response := decodeAPIResponse(t, rec, http.StatusUnauthorized)
		if response.Code != -1 {
			t.Fatalf("expected unauthorized API code -1 for %s, got %d", path, response.Code)
		}
	}
}

func TestMarketHealthMutationsRejectGet(t *testing.T) {
	handler := newTestHandler(t)
	for _, path := range []string{
		"/api/v1/market/health/tencent/simulate?status=unhealthy",
		"/api/v1/market/health/reset",
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected GET %s to be unregistered, got %d", path, rec.Code)
		}
	}
}

func TestDeprecatedPredictionEndpointsReturnGone(t *testing.T) {
	handler := newTestHandler(t)

	for _, path := range []string{
		"/api/v1/predict/000001",
		"/api/v1/stock/600519/predict",
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		handler.ServeHTTP(rec, req)

		response := decodeAPIResponse(t, rec, http.StatusGone)
		if response.Code != -1 || response.Message != "预测服务已迁移，此接口已废弃" || response.Data != nil {
			t.Fatalf("unexpected deprecated prediction response for %s: %+v", path, response)
		}
	}
}

func TestPredictionPlaceholdersRejectInvalidCodes(t *testing.T) {
	handler := newTestHandler(t)

	for _, path := range []string{
		"/api/v1/predict/abc",
		"/api/v1/stock/abc/predict",
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		handler.ServeHTTP(rec, req)

		response := decodeAPIResponse(t, rec, http.StatusBadRequest)
		if response.Code != -1 {
			t.Fatalf("expected API code -1 for %s, got %d", path, response.Code)
		}
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

func decodeAPIResponse(t *testing.T, rec *httptest.ResponseRecorder, expectedStatus int) transporthttp.APIResponse {
	t.Helper()
	if rec.Code != expectedStatus {
		t.Fatalf("expected status %d, got %d: %s", expectedStatus, rec.Code, rec.Body.String())
	}
	var response transporthttp.APIResponse
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
