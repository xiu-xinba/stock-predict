package providers

import (
	"context"
	"log/slog"
	"time"

	funddomain "stock-predict-go/internal/domain/fund"
	"stock-predict-go/internal/platform/config"
)

// WatchlistService 自选股服务，提供自选基金列表及实时估值。
type WatchlistService struct {
	store         funddomain.Repository
	cfg           config.Config
	quoteProvider fundQuoteProvider
}

// NewWatchlistService 创建新的自选股服务实例。
func NewWatchlistService(store funddomain.Repository, cfg config.Config, logger *slog.Logger) *WatchlistService {
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

// Quotes 获取自选基金列表的实时估值数据。
func (s *WatchlistService) Quotes(codes []string) []funddomain.WatchlistItem {
	now := time.Now().UnixMilli()
	items := make([]funddomain.WatchlistItem, 0, len(codes))
	funds := make([]funddomain.FundItem, 0, len(codes))
	for _, code := range codes {
		fund, ok := s.store.FindFund(code)
		if !ok {
			continue
		}
		funds = append(funds, fund)
	}
	quotes := map[string]funddomain.FundItem{}
	if s.quoteProvider != nil && len(funds) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), s.cfg.ReadTimeout)
		defer cancel()
		quotes = s.quoteProvider.RefreshQuotes(ctx, funds)
	}
	for _, fund := range funds {
		if quote, ok := quotes[fund.FundCode]; ok {
			fund = mergeRealtimeQuote(fund, quote)
		}
		items = append(items, funddomain.WatchlistItem{
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

func mergeRealtimeQuote(fund, quote funddomain.FundItem) funddomain.FundItem {
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

func direction(value, flatThreshold float64) funddomain.Direction {
	if value > flatThreshold {
		return funddomain.DirectionUp
	}
	if value < -flatThreshold {
		return funddomain.DirectionDown
	}
	return funddomain.DirectionFlat
}
