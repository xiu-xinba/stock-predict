package providers

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
)

func TestMarketServiceNorthboundFallbackRejectsEmptyPlaceholder(t *testing.T) {
	responseBody := `{"data":{"s2n":["09:30,0,0,0,0,0","09:31,0,0,0,0,0"]}}`
	client := newEastmoneyClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(responseBody)),
				Request:    req,
			}, nil
		}),
	})
	client.minInterval = 0

	service := NewMarketService(nil, slog.Default())
	service.eastmoney = client

	flow := service.NorthboundFlow(context.Background())
	if flow == nil {
		t.Fatal("expected unavailable northbound status object, got nil")
	}
	if flow.Status != "intraday_unavailable" {
		t.Fatalf("expected intraday_unavailable status, got %+v", flow)
	}
	if flow.Notice == "" {
		t.Fatalf("expected disclosure notice, got %+v", flow)
	}

	responseBody = `{"data":{"s2n":["00:01,100,200,100,200,300"]}}`
	flow = service.NorthboundFlow(context.Background())
	if flow == nil {
		t.Fatal("expected second non-empty response to be fetched instead of cached empty placeholder")
	}
	if flow.TotalBuy != 300 || len(flow.Timeline) != 1 {
		t.Fatalf("unexpected non-empty fallback flow: %+v", flow)
	}
}
