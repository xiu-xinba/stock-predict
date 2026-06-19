package providers

import (
	"testing"
)

func TestTencentProvider_Name(t *testing.T) {
	p := NewTencentProvider(nil, nil, nil)
	if got := p.Name(); got != "tencent" {
		t.Errorf("Name() = %q, want %q", got, "tencent")
	}
}

func TestTencentProvider_Capabilities(t *testing.T) {
	p := NewTencentProvider(nil, nil, nil)
	caps := p.Capabilities()

	expectedCaps := map[Capability][]Market{
		CapIndexQuote:  {MarketCN, MarketHK, MarketUS},
		CapIndexMinute: {MarketCN, MarketHK, MarketUS},
		CapIndexKline:  {MarketCN, MarketHK, MarketUS},
		CapStockQuote:  {MarketCN, MarketHK, MarketUS},
		CapStockMinute: {MarketCN, MarketHK, MarketUS},
		CapFundQuote:   {MarketCN},
	}

	if len(caps) != len(expectedCaps) {
		t.Fatalf("Capabilities() returned %d entries, want %d", len(caps), len(expectedCaps))
	}

	for cap, wantMarkets := range expectedCaps {
		gotMarkets, ok := caps[cap]
		if !ok {
			t.Errorf("missing capability %q", cap)
			continue
		}
		if len(gotMarkets) != len(wantMarkets) {
			t.Errorf("capability %q: got %d markets, want %d", cap, len(gotMarkets), len(wantMarkets))
			continue
		}
		for i, m := range wantMarkets {
			if gotMarkets[i] != m {
				t.Errorf("capability %q: market[%d] = %q, want %q", cap, i, gotMarkets[i], m)
			}
		}
	}
}

func TestTencentProvider_Priority(t *testing.T) {
	p := NewTencentProvider(nil, nil, nil)

	tests := []struct {
		cap    Capability
		market Market
		want   int
	}{
		{CapIndexQuote, MarketCN, 1},
		{CapIndexQuote, MarketHK, 1},
		{CapIndexQuote, MarketUS, 1},
		{CapIndexMinute, MarketCN, 2},
		{CapIndexMinute, MarketHK, 1},
		{CapIndexMinute, MarketUS, 1},
		{CapIndexKline, MarketCN, 2},
		{CapIndexKline, MarketHK, 1},
		{CapIndexKline, MarketUS, 1},
		{CapStockQuote, MarketCN, 1},
		{CapStockQuote, MarketHK, 1},
		{CapStockQuote, MarketUS, 1},
		{CapStockMinute, MarketCN, 1},
		{CapStockMinute, MarketHK, 1},
		{CapStockMinute, MarketUS, 1},
		{CapFundQuote, MarketCN, 1},
		{CapStockSearch, MarketCN, 99},
	}

	for _, tt := range tests {
		got := p.Priority(tt.cap, tt.market)
		if got != tt.want {
			t.Errorf("Priority(%q, %q) = %d, want %d", tt.cap, tt.market, got, tt.want)
		}
	}
}
