package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTdxIndexSymbol(t *testing.T) {
	tests := []struct {
		code     string
		expected string
	}{
		{"000001", "sh000001"},
		{"399001", "sz399001"},
		{"399006", "sz399006"},
		{"999999", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := tdxIndexSymbol(tt.code)
		if got != tt.expected {
			t.Errorf("tdxIndexSymbol(%q) = %q, want %q", tt.code, got, tt.expected)
		}
	}
}

func TestIsCNIndex(t *testing.T) {
	tests := []struct {
		code     string
		expected bool
	}{
		{"000001", true},
		{"399001", true},
		{"399006", true},
		{"600000", false},
		{"hsi", false},
		{"dji", false},
		{"", false},
	}
	for _, tt := range tests {
		got := isCNIndex(tt.code)
		if got != tt.expected {
			t.Errorf("isCNIndex(%q) = %v, want %v", tt.code, got, tt.expected)
		}
	}
}

func TestTencentIndexToMarketIndexUsesFallbackUpdateTime(t *testing.T) {
	fields := make([]string, 41)
	fields[1] = "上证指数"
	fields[3] = "3100"
	fields[4] = "3069"
	fields[6] = "100000"
	fields[31] = "31"
	fields[32] = "1.01"
	fields[33] = "3120"
	fields[34] = "3060"

	index := tencentIndexToMarketIndex("000001", fields)

	if index.UpdateTime == "" {
		t.Fatalf("expected fallback update time when Tencent field 30 is empty")
	}
}

func TestFetchEastmoneyURLPreservesParentCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(250 * time.Millisecond)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := NewIndexQuoteClient(nil)
	client.resilient = NewResilientHTTPClient(server.Client(), nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	_, err := client.fetchEastmoneyURL(ctx, server.URL)
	if err == nil {
		t.Fatal("expected canceled parent context to stop request")
	}
	if elapsed := time.Since(start); elapsed >= 100*time.Millisecond {
		t.Fatalf("expected prompt cancellation, request took %s", elapsed)
	}
}
