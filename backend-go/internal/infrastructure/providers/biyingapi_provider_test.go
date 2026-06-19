package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBiyingProviderCapabilitiesOnlyDeclareImplementedFeatures(t *testing.T) {
	provider := NewBiyingApiProvider("http://example.test", "test-licence", discardLogger())
	caps := provider.Capabilities()

	if _, ok := caps[CapStockRanking]; ok {
		t.Fatalf("BiyingAPI must not declare stock ranking until it is implemented")
	}
	if _, ok := caps[CapSectorRank]; ok {
		t.Fatalf("BiyingAPI must not declare sector ranking until it is implemented")
	}
}

func TestBiyingSafeErrorRedactsLicence(t *testing.T) {
	provider := NewBiyingApiProvider("https://api.example.test", "secret-licence", discardLogger())
	err := provider.safeError(fmt.Errorf("Get \"https://api.example.test/path/secret-licence\": connection refused"))
	if strings.Contains(err.Error(), "secret-licence") {
		t.Fatalf("safe error leaked licence: %v", err)
	}
}

func TestBiyingFetchIndexQuotesUsesMarketSuffixedCodes(t *testing.T) {
	expectedPaths := map[string]bool{
		"/hsindex/real/time/000001.SH/test-licence": false,
		"/hsindex/real/time/399001.SZ/test-licence": false,
		"/hsindex/real/time/399006.SZ/test-licence": false,
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen, ok := expectedPaths[r.URL.Path]
		if !ok {
			t.Fatalf("unexpected Biying index quote path: %s", r.URL.Path)
		}
		if seen {
			t.Fatalf("duplicate Biying index quote path: %s", r.URL.Path)
		}
		expectedPaths[r.URL.Path] = true
		_ = json.NewEncoder(w).Encode(map[string]any{
			"p": 3210.5, "o": 3200.0, "h": 3220.0, "l": 3190.0,
			"yc": 3180.0, "v": 12345, "pv": 30.5, "ud": 0.0096, "t": "2026-06-07 15:00:00",
		})
	}))
	defer server.Close()
	provider := newBiyingProviderForTest(server)

	quotes, err := provider.FetchIndexQuotes(context.Background(), MarketCN)

	if err != nil {
		t.Fatalf("FetchIndexQuotes returned error: %v", err)
	}
	if len(quotes) != len(expectedPaths) {
		t.Fatalf("FetchIndexQuotes returned %d quotes, want %d", len(quotes), len(expectedPaths))
	}
	for path, seen := range expectedPaths {
		if !seen {
			t.Fatalf("expected Biying index quote path was not requested: %s", path)
		}
	}
}

func TestBiyingFetchIndexMinuteUsesLatestEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/hsindex/latest/000001.SH/5/test-licence" {
			t.Fatalf("unexpected Biying index minute path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("lt"); got != "5" {
			t.Fatalf("unexpected Biying index minute lt query: %q", got)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"t": "2026-06-07 09:35:00", "c": 3210.5, "v": 12345},
		})
	}))
	defer server.Close()
	provider := newBiyingProviderForTest(server)

	points, err := provider.FetchIndexMinute(context.Background(), "000001", MarketCN)

	if err != nil {
		t.Fatalf("FetchIndexMinute returned error: %v", err)
	}
	if len(points) != 1 || points[0].Price != 3210.5 || points[0].Volume != 12345 {
		t.Fatalf("unexpected Biying index minute points: %+v", points)
	}
}

func TestBiyingFetchStockQuotesUsesPlainStockCodes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/hsstock/real/time/600519/test-licence" {
			t.Fatalf("unexpected Biying stock quote path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"p": 1700.5, "o": 1690.0, "h": 1710.0, "l": 1680.0,
			"yc": 1675.0, "cje": 2000000, "v": 12000, "pv": 25.5, "ud": 0.0152, "tr": 0.36, "t": "2026-06-07 15:00:00",
		})
	}))
	defer server.Close()
	provider := newBiyingProviderForTest(server)

	quotes, err := provider.FetchStockQuotes(context.Background(), []string{"sh600519"})

	if err != nil {
		t.Fatalf("FetchStockQuotes returned error: %v", err)
	}
	quote, ok := quotes["600519"]
	if !ok {
		t.Fatalf("expected quote keyed by plain code 600519, got %+v", quotes)
	}
	if quote.Price != 1700.5 || quote.ChangePct != 1.52 {
		t.Fatalf("unexpected Biying stock quote: %+v", quote)
	}
}

func TestBiyingFetchStockQuotesRejectsBiyingErrorPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 404,
			"msg":  "resource not found",
		})
	}))
	defer server.Close()
	provider := newBiyingProviderForTest(server)

	_, err := provider.FetchStockQuotes(context.Background(), []string{"600519"})

	if err == nil {
		t.Fatalf("expected Biying error payload to fail")
	}
	var providerErr *providerError
	if !errors.As(err, &providerErr) || providerErr.provider != "biyingapi" {
		t.Fatalf("expected provider error from Biying payload, got %T %v", err, err)
	}
}

func TestBiyingFetchIndexMinuteRejectsNonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Licence missing", http.StatusForbidden)
	}))
	defer server.Close()
	provider := newBiyingProviderForTest(server)

	_, err := provider.FetchIndexMinute(context.Background(), "000001", MarketCN)

	if err == nil {
		t.Fatalf("expected non-OK Biying index minute response to fail")
	}
	var providerErr *providerError
	if !errors.As(err, &providerErr) || providerErr.provider != "biyingapi" {
		t.Fatalf("expected provider error from Biying status, got %T %v", err, err)
	}
}

func newBiyingProviderForTest(server *httptest.Server) *BiyingApiProvider {
	provider := NewBiyingApiProvider(server.URL, "test-licence", discardLogger())
	provider.client = server.Client()
	provider.resilient = NewResilientHTTPClient(server.Client(), []SourcePolicy{{Source: SourceBiyingAPI}})
	return provider
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
