package service

import (
	"context"
	"log/slog"
	"time"

	"stock-predict-go/internal/config"
	"stock-predict-go/internal/dto"
)

type WatchlistService struct {
	store         FundRepository
	cfg           config.Config
	quoteProvider fundQuoteProvider
}

func NewWatchlistService(store FundRepository, cfg config.Config, logger *slog.Logger) *WatchlistService {
	if logger == nil {
		logger = slog.Default()
	}
	var quoteProvider fundQuoteProvider
	if cfg.FundRealtimeQuotesEnabled {
		quoteProvider = NewFundQuoteClient(cfg.ReadTimeout, logger)
	}
	return &WatchlistService{
		store:         store,
		cfg:           cfg,
		quoteProvider: quoteProvider,
	}
}

func (s *WatchlistService) Quotes(codes []string) []dto.WatchlistItem {
	now := time.Now().UnixMilli()
	items := make([]dto.WatchlistItem, 0, len(codes))
	funds := make([]dto.FundItem, 0, len(codes))
	for _, code := range codes {
		fund, ok := s.store.FindFund(code)
		if !ok {
			continue
		}
		funds = append(funds, fund)
	}
	quotes := map[string]dto.FundItem{}
	if s.quoteProvider != nil && len(funds) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), s.cfg.ReadTimeout)
		defer cancel()
		quotes = s.quoteProvider.RefreshQuotes(ctx, funds)
	}
	for _, fund := range funds {
		if quote, ok := quotes[fund.FundCode]; ok {
			fund = mergeRealtimeQuote(fund, quote)
		}
		items = append(items, dto.WatchlistItem{
			FundCode:     fund.FundCode,
			FundName:     fund.FundName,
			FundType:     fund.FundType,
			EstimatedNAV: fund.EstimatedNAV,
			ChangePct:    fund.ChangePct,
			Direction:    direction(fund.ChangePct, 0),
			AddedAt:      now,
			QuoteDate:    fund.QuoteDate,
			QuoteSource:  fund.QuoteSource,
		})
	}
	return items
}

func mergeRealtimeQuote(fund, quote dto.FundItem) dto.FundItem {
	if quote.LatestNAV != 0 {
		fund.LatestNAV = quote.LatestNAV
	}
	if quote.EstimatedNAV != 0 {
		fund.EstimatedNAV = quote.EstimatedNAV
	}
	fund.ChangePct = quote.ChangePct
	if quote.QuoteDate != "" {
		fund.QuoteDate = quote.QuoteDate
	}
	if quote.QuoteSource != "" {
		fund.QuoteSource = quote.QuoteSource
	}
	return fund
}

func direction(value, flatThreshold float64) dto.Direction {
	if value > flatThreshold {
		return dto.DirectionUp
	}
	if value < -flatThreshold {
		return dto.DirectionDown
	}
	return dto.DirectionFlat
}
