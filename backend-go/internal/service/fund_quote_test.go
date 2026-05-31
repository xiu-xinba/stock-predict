package service

import (
	"context"
	"testing"

	"stock-predict-go/internal/config"
	"stock-predict-go/internal/dto"
)

func TestParseEastmoneyFundGZQuote(t *testing.T) {
	payload := []byte(`jsonpgz({"fundcode":"000001","name":"华夏成长混合","jzrq":"2026-05-27","dwjz":"1.3330","gsz":"1.3307","gszzl":"-0.18","gztime":"2026-05-28 13:17"});`)

	quote, ok := parseEastmoneyFundGZQuote(payload)
	if !ok {
		t.Fatalf("expected fundgz quote to parse")
	}

	if quote.FundCode != "000001" || quote.FundName != "华夏成长混合" {
		t.Fatalf("unexpected quote identity: %+v", quote)
	}
	if quote.LatestNAV != 1.333 || quote.EstimatedNAV != 1.3307 || quote.ChangePct != -0.18 {
		t.Fatalf("unexpected quote values: %+v", quote)
	}
	if quote.QuoteDate != "2026-05-28 13:17" || quote.QuoteSource != "eastmoney_fundgz" {
		t.Fatalf("unexpected quote metadata: %+v", quote)
	}
}

func TestParseTencentFundQuotes(t *testing.T) {
	payload := []byte(`v_sh510300="1~沪深300ETF华泰柏瑞~510300~4.879~4.932~4.917~2908259~1339421~1568755~4.879~303~4.878~1140~4.877~2749~4.876~1054~4.875~2875~4.880~3502~4.881~1650~4.882~2827~4.883~1487~4.884~9722~~20260528132015~-0.053~-1.07~4.918~4.866~4.879/2908259/1421651933~2908259~142165~1.04~~~4.918~4.866~1.05~1361.09~1361.09~0.00~5.425~4.439~0.43~-11067~4.888~~~~~~142165.1933~0.0000~0~ ~ETF~5.38~1.48~~~~5.041~3.721~-1.25~2.03~3.26~27896887700~27896887700~-40.53~9.84~27896887700~-0.03~4.8804~30.77~0.10~4.9294~CNY~0~___D__F__N~4.872~17710~";`)

	quotes := parseTencentFundQuotes(payload)
	quote, ok := quotes["510300"]
	if !ok {
		t.Fatalf("expected 510300 quote, got %+v", quotes)
	}

	if quote.FundCode != "510300" || quote.FundName != "沪深300ETF华泰柏瑞" {
		t.Fatalf("unexpected quote identity: %+v", quote)
	}
	if quote.LatestNAV != 4.879 || quote.EstimatedNAV != 4.879 || quote.ChangePct != -1.07 {
		t.Fatalf("unexpected quote values: %+v", quote)
	}
	if quote.QuoteDate != "20260528132015" || quote.QuoteSource != "tencent_quote" {
		t.Fatalf("unexpected quote metadata: %+v", quote)
	}
}

type fakeQuoteProvider struct {
	quotes map[string]dto.FundItem
}

func (p fakeQuoteProvider) RefreshQuotes(context.Context, []dto.FundItem) map[string]dto.FundItem {
	return p.quotes
}

func TestWatchlistQuotesUsesRealtimeProvider(t *testing.T) {
	service := NewWatchlistService(
		fakeFundRepository{funds: []dto.FundItem{{
			FundCode:     "510300",
			FundName:     "沪深300ETF",
			FundType:     "ETF",
			LatestNAV:    3.68,
			EstimatedNAV: 3.69,
			ChangePct:    0.22,
			QuoteSource:  "eastmoney_rank",
		}}},
		config.Config{ReadTimeout: 1},
		nil,
	)
	service.quoteProvider = fakeQuoteProvider{quotes: map[string]dto.FundItem{
		"510300": {
			FundCode:     "510300",
			FundName:     "沪深300ETF华泰柏瑞",
			LatestNAV:    4.879,
			EstimatedNAV: 4.879,
			ChangePct:    -1.07,
			QuoteDate:    "20260528132015",
			QuoteSource:  "tencent_quote",
		},
	}}

	items := service.Quotes([]string{"510300"})

	if len(items) != 1 {
		t.Fatalf("expected one watchlist item, got %+v", items)
	}
	if items[0].ChangePct != -1.07 || items[0].EstimatedNAV != 4.879 || items[0].Direction != dto.DirectionDown {
		t.Fatalf("expected realtime quote to override stored quote, got %+v", items[0])
	}
	if items[0].FundName != "沪深300ETF" {
		t.Fatalf("expected realtime quote to keep store fund name, got %+v", items[0])
	}
	if items[0].QuoteSource != "tencent_quote" || items[0].QuoteDate != "20260528132015" {
		t.Fatalf("expected quote metadata from provider, got %+v", items[0])
	}
}
