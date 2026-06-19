package providers

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestEastmoneyNorthboundFlowUsesMinuteTimeline(t *testing.T) {
	var requestedPath string
	client := newEastmoneyClient(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			requestedPath = req.URL.Path
			body := `{"data":{"s2n":["09:30,100,200,100,200,300","09:31,-50,80,50,280,330"]}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    req,
			}, nil
		}),
	})
	client.minInterval = 0
	provider := NewEastmoneyProvider(nil, client, nil)

	flow, err := provider.FetchNorthboundFlow(context.Background())
	if err != nil {
		t.Fatalf("FetchNorthboundFlow returned error: %v", err)
	}

	if requestedPath != "/api/qt/kamtbs.wss" {
		t.Fatalf("expected minute endpoint, got %s", requestedPath)
	}
	if len(flow.Timeline) != 2 {
		t.Fatalf("expected 2 minute points, got %d", len(flow.Timeline))
	}
	if flow.Timeline[0].Time != "09:30" || flow.Timeline[1].Time != "09:31" {
		t.Fatalf("expected HH:mm timeline, got %+v", flow.Timeline)
	}
	if flow.Timeline[0].SHFlow != 100 || flow.Timeline[0].SZFlow != 200 {
		t.Fatalf("unexpected first timeline point: %+v", flow.Timeline[0])
	}
	if flow.SHNetBuy != 50 || flow.SZNetBuy != 280 || flow.TotalBuy != 330 {
		t.Fatalf("unexpected totals: %+v", flow)
	}
}
