package providers

import (
	"log/slog"
	"testing"
)

func TestProvidersImplementDeclaredCapabilityInterfaces(t *testing.T) {
	providers := []Provider{
		NewTencentProvider(nil, nil, nil),
		NewEastmoneyProvider(nil, nil, nil),
		NewSinaProvider(nil),
		NewTDXProvider(nil, slog.Default()),
		NewBiyingApiProvider("http://example.test", "token", slog.Default()),
		NewAKShareProvider("http://example.test", slog.Default()),
	}

	for _, provider := range providers {
		for capability := range provider.Capabilities() {
			if !providerImplementsCapability(provider, capability) {
				t.Fatalf("%s declares %s but does not implement the matching provider interface", provider.Name(), capability)
			}
		}
	}
}

func providerImplementsCapability(provider Provider, capability Capability) bool {
	switch capability {
	case CapIndexQuote:
		_, ok := provider.(IndexQuoteProvider)
		return ok
	case CapIndexMinute:
		_, ok := provider.(IndexMinuteProvider)
		return ok
	case CapIndexKline:
		_, ok := provider.(IndexKlineProvider)
		return ok
	case CapStockQuote:
		_, ok := provider.(StockQuoteProvider)
		return ok
	case CapStockMinute:
		_, ok := provider.(StockMinuteProvider)
		return ok
	case CapStockSearch:
		_, ok := provider.(StockSearchProvider)
		return ok
	case CapStockSync:
		_, ok := provider.(StockSyncProvider)
		return ok
	case CapStockRanking:
		_, ok := provider.(StockRankingProvider)
		return ok
	case CapSectorRank:
		_, ok := provider.(SectorRankingProvider)
		return ok
	case CapNorthbound:
		_, ok := provider.(NorthboundProvider)
		return ok
	case CapFundQuote:
		_, ok := provider.(FundQuoteProvider)
		return ok
	default:
		return false
	}
}
