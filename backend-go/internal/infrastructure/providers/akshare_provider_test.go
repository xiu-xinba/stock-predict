package providers

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAKShareHealthCheckUsesResilientClient(t *testing.T) {
	const userAgent = "akshare-health-test"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("User-Agent"); got != userAgent {
			t.Fatalf("expected resilient AKShare policy user agent %q, got %q", userAgent, got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := NewAKShareProvider(server.URL, slog.Default())
	provider.resilient = NewResilientHTTPClient(server.Client(), []SourcePolicy{{
		Source:      SourceAKShare,
		UserAgent:   userAgent,
		MinInterval: 0,
	}})

	if err := provider.HealthCheck(context.Background()); err != nil {
		t.Fatalf("health check failed: %v", err)
	}
}

func TestAKShareNorthboundFlowRejectsAllZeroPlaceholder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/northbound/flow" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"code": 0,
			"message": "success",
			"data": {
				"sh_net_buy": 0,
				"sz_net_buy": 0,
				"total_net_buy": 0,
				"timeline": [
					{"time": "10:00", "sh_flow": 0, "sz_flow": 0},
					{"time": "10:01", "sh_flow": 0, "sz_flow": 0}
				]
			}
		}`))
	}))
	defer server.Close()

	provider := NewAKShareProvider(server.URL, slog.Default())

	flow, err := provider.FetchNorthboundFlow(context.Background())
	if err == nil {
		t.Fatalf("expected all-zero placeholder to be rejected, got flow %+v", flow)
	}
	if !strings.Contains(err.Error(), "empty result") {
		t.Fatalf("expected empty result error, got %v", err)
	}
}
