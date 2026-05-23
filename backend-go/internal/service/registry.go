package service

import (
	"log/slog"

	"stock-predict-go/internal/config"
	"stock-predict-go/internal/store"
)

type Registry struct {
	Funds      *FundService
	Market     *MarketService
	Prediction *PredictionService
}

func NewRegistry(store *store.MemoryStore, cfg config.Config, logger *slog.Logger) *Registry {
	market := NewMarketService(logger)
	funds := NewFundService(store)
	prediction := NewPredictionService(store, market, cfg, logger)
	return &Registry{
		Funds:      funds,
		Market:     market,
		Prediction: prediction,
	}
}
