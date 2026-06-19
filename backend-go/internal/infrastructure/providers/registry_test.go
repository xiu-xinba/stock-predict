package providers

import (
	"io"
	"log/slog"
	"testing"
	"time"

	database "stock-predict-go/internal/infrastructure/database"
	"stock-predict-go/internal/platform/config"
)

func TestNewRegistryRegistersBiyingAndAKShareWhenConfigured(t *testing.T) {
	cfg := testRegistryConfig()
	cfg.BiyingAPIURL = "https://api.biyingapi.com"
	cfg.BiyingAPIToken = "test-licence"
	cfg.AKShareURL = "http://localhost:8900"
	cfg.AKShareToken = "test-service-token"

	registry := newTestRegistry(t, cfg)

	if !hasProvider(registry.Providers, "biyingapi") {
		t.Fatalf("expected BiyingAPI provider to be registered when URL and token are configured")
	}
	if !hasProvider(registry.Providers, "akshare") {
		t.Fatalf("expected AKShare provider to be registered when URL is configured")
	}
}

func TestNewRegistrySkipsProvidersWithoutTokens(t *testing.T) {
	cfg := testRegistryConfig()
	cfg.BiyingAPIURL = "https://api.biyingapi.com"
	cfg.AKShareURL = "http://localhost:8900"

	registry := newTestRegistry(t, cfg)

	if hasProvider(registry.Providers, "biyingapi") {
		t.Fatalf("expected BiyingAPI provider to be skipped without a token")
	}
	if hasProvider(registry.Providers, "akshare") {
		t.Fatalf("expected AKShare provider to be skipped without a service token")
	}
}

func testRegistryConfig() config.Config {
	return config.Config{
		Port:            "0",
		Env:             "test",
		ReadTimeout:     time.Second,
		WriteTimeout:    time.Second,
		ShutdownTimeout: time.Second,
	}
}

func newTestRegistry(t *testing.T, cfg config.Config) *Registry {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	db := database.InitTestDB(t)
	fundStore := database.NewFundStore(db)
	stockStore := database.NewStockStore(db)
	searchIdx := database.NewSearchStore(db)
	return NewRegistry(fundStore, stockStore, cfg, logger, searchIdx, nil, db)
}

func hasProvider(providers []Provider, name string) bool {
	for _, provider := range providers {
		if provider.Name() == name {
			return true
		}
	}
	return false
}
