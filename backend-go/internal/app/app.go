package app

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"stock-predict-go/internal/api"
	"stock-predict-go/internal/config"
	"stock-predict-go/internal/data"
	"stock-predict-go/internal/service"
	"stock-predict-go/internal/store"
)

type CleanupFunc func()

func NewServer(cfg config.Config, logger *slog.Logger) (*http.Server, CleanupFunc, error) {
	mem, err := store.NewPersistentStore(cfg.FundStorePath)
	if err != nil {
		return nil, nil, err
	}
	if cfg.FundAutoSyncOnStart && (mem.CountFunds() < cfg.FundAutoSyncMinCount || (cfg.FundMetricsURL != "" && mem.CountQuotedFunds() == 0)) {
		result, err := service.NewFundService(mem).SyncFromSources(cfg.FundUniverseURL, cfg.FundMetricsURL, cfg.FundSyncCSVPath)
		if err != nil {
			logger.Warn("fund auto sync skipped", "error", err)
		} else {
			logger.Info("fund auto sync completed", "imported", result.Imported, "total", result.Total, "source", result.Source)
		}
	}

	searchIdx, err := store.NewSearchIndex("file:search_index?mode=memory", logger)
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() {
		_ = searchIdx.Close()
	}

	if err := mem.ReplaceStocks(data.LoadDefaultStocks()); err != nil {
		logger.Warn("failed to load default stocks", "error", err)
	}

	services := service.NewRegistry(mem, mem, cfg, logger, searchIdx)

	if cfg.StockAutoSyncOnStart {
		logger.Info("starting stock auto sync from eastmoney API...")
		syncCtx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		result, err := services.Stocks.SyncStocks(syncCtx)
		cancel()
		if err != nil {
			logger.Warn("stock auto sync failed, using default stocks", "error", err)
		} else {
			logger.Info("stock auto sync completed", "imported", result.Imported, "total", result.Total, "errors", result.Errors)
		}
	}

	if err := searchIdx.SyncFunds(mem.ListFunds()); err != nil {
		logger.Warn("failed to sync funds to search index", "error", err)
	}
	if err := searchIdx.SyncStocks(services.Stocks.ListStocks()); err != nil {
		logger.Warn("failed to sync stocks to search index", "error", err)
	}

	router := api.NewRouter(cfg, services, mem, logger, searchIdx)
	previousCleanup := cleanup
	cleanup = func() {
		router.Close()
		previousCleanup()
	}

	return &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: cfg.ReadTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}, cleanup, nil
}
