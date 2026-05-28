package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"stock-predict-go/internal/api"
	"stock-predict-go/internal/config"
	"stock-predict-go/internal/service"
	"stock-predict-go/internal/store"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel(),
	}))

	mem, err := store.NewPersistentStore(cfg.FundStorePath)
	if err != nil {
		logger.Error("failed to initialize fund store", "path", cfg.FundStorePath, "error", err)
		os.Exit(1)
	}
	if cfg.FundAutoSyncOnStart && (mem.CountFunds() < cfg.FundAutoSyncMinCount || (cfg.FundMetricsURL != "" && mem.CountQuotedFunds() == 0)) {
		result, err := service.NewFundService(mem).SyncFromSources(cfg.FundUniverseURL, cfg.FundMetricsURL, cfg.FundSyncCSVPath)
		if err != nil {
			logger.Warn("fund auto sync skipped", "error", err)
		} else {
			logger.Info("fund auto sync completed", "imported", result.Imported, "total", result.Total, "source", result.Source)
		}
	}
	services := service.NewRegistry(mem, cfg, logger)
	handler := api.NewRouter(cfg, services, logger)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: cfg.ReadTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	go func() {
		logger.Info("go backend listening", "addr", server.Addr, "env", cfg.Env)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	logger.Info("server stopped")
}
