package service

import (
	"log/slog"
	"time"

	"stock-predict-go/internal/config"
	"stock-predict-go/internal/store"
)

type Registry struct {
	Funds       *FundService
	Market      *MarketService
	Watchlist   *WatchlistService
	Detail      *FundDetailService
	Stocks      *StockService
	StockDetail *StockDetailService
	StockQuote  *StockQuoteClient
	Search      *SearchService
}

func NewRegistry(fundRepo store.FundRepository, cfg config.Config, logger *slog.Logger, searchIdx *store.SearchIndex) *Registry {
	market := NewMarketService(logger)
	quote := NewFundQuoteClient(8*time.Second, logger)
	funds := NewFundService(fundRepo)
	detail := NewFundDetailService(fundRepo, quote, logger)
	stockQuote := NewStockQuoteClient(8 * time.Second)
	stocks := NewStockService(logger)
	watchlist := NewWatchlistService(fundRepo, cfg, logger)
	stockDetail := NewStockDetailService(stocks, stockQuote, logger)
	search := NewSearchService(fundRepo, stocks, searchIdx)
	return &Registry{
		Funds:       funds,
		Market:      market,
		Watchlist:   watchlist,
		Detail:      detail,
		Stocks:      stocks,
		StockDetail: stockDetail,
		StockQuote:  stockQuote,
		Search:      search,
	}
}
