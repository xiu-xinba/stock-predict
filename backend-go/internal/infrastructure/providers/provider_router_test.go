package providers

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"
)

type mockProvider struct {
	name       string
	caps       map[Capability][]Market
	priorities map[string]int
	healthy    bool
	fetchErr   error
}

func (m *mockProvider) Name() string                          { return m.name }
func (m *mockProvider) Capabilities() map[Capability][]Market { return m.caps }
func (m *mockProvider) Priority(cap Capability, market Market) int {
	key := string(cap) + ":" + string(market)
	if p, ok := m.priorities[key]; ok {
		return p
	}
	return 99
}
func (m *mockProvider) HealthCheck(_ context.Context) error {
	if m.healthy {
		return nil
	}
	return errors.New("unhealthy")
}

func newMockProvider(name string, caps map[Capability][]Market, priorities map[string]int) *mockProvider {
	return &mockProvider{name: name, caps: caps, priorities: priorities, healthy: true}
}

func TestRouter_Fallback(t *testing.T) {
	p1 := newMockProvider("p1", map[Capability][]Market{CapIndexQuote: {MarketCN}}, map[string]int{"index_quote:cn": 1})
	p1.fetchErr = errors.New("fail")
	p2 := newMockProvider("p2", map[Capability][]Market{CapIndexQuote: {MarketCN}}, map[string]int{"index_quote:cn": 2})

	logger := slog.Default()
	health := NewHealthMonitor(logger, "p1", "p2")
	router := NewProviderRouter([]Provider{p1, p2}, health, RouterConfig{DefaultStrategy: StrategyFallback, RaceTimeout: 2 * time.Second}, logger)

	var resultProvider Provider
	err := router.FetchWithFallback(context.Background(), CapIndexQuote, MarketCN, func(ctx context.Context, p Provider) error {
		if p.Name() == "p2" {
			resultProvider = p
			return nil
		}
		return p.(*mockProvider).fetchErr
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if resultProvider == nil || resultProvider.Name() != "p2" {
		t.Fatalf("expected p2, got %v", resultProvider)
	}
}

func TestRouter_Race(t *testing.T) {
	p1 := newMockProvider("p1", map[Capability][]Market{CapIndexQuote: {MarketCN}}, map[string]int{"index_quote:cn": 1})
	p2 := newMockProvider("p2", map[Capability][]Market{CapIndexQuote: {MarketCN}}, map[string]int{"index_quote:cn": 2})

	logger := slog.Default()
	health := NewHealthMonitor(logger, "p1", "p2")
	router := NewProviderRouter([]Provider{p1, p2}, health, RouterConfig{DefaultStrategy: StrategyRace, RaceTimeout: 2 * time.Second}, logger)

	err := router.FetchWithRace(context.Background(), CapIndexQuote, MarketCN, func(ctx context.Context, p Provider) error {
		return nil
	}, 2)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestRouter_SkipUnhealthy(t *testing.T) {
	p1 := newMockProvider("p1", map[Capability][]Market{CapIndexQuote: {MarketCN}}, map[string]int{"index_quote:cn": 1})
	p2 := newMockProvider("p2", map[Capability][]Market{CapIndexQuote: {MarketCN}}, map[string]int{"index_quote:cn": 2})

	logger := slog.Default()
	health := NewHealthMonitor(logger, "p1", "p2")
	health.RecordFailure("p1", errors.New("fail"))
	health.RecordFailure("p1", errors.New("fail"))
	health.RecordFailure("p1", errors.New("fail"))

	router := NewProviderRouter([]Provider{p1, p2}, health, RouterConfig{DefaultStrategy: StrategyFallback, RaceTimeout: 2 * time.Second}, logger)

	var resultProvider Provider
	err := router.FetchWithFallback(context.Background(), CapIndexQuote, MarketCN, func(ctx context.Context, p Provider) error {
		resultProvider = p
		return nil
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if resultProvider.Name() != "p2" {
		t.Fatalf("expected p2 (p1 unhealthy), got %s", resultProvider.Name())
	}
}

func TestRouter_NoProviders(t *testing.T) {
	logger := slog.Default()
	health := NewHealthMonitor(logger)
	router := NewProviderRouter(nil, health, RouterConfig{DefaultStrategy: StrategyFallback, RaceTimeout: 2 * time.Second}, logger)

	err := router.FetchWithFallback(context.Background(), CapIndexQuote, MarketCN, func(ctx context.Context, p Provider) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error for no providers")
	}
}

func TestRouter_PerCapabilityStrategy(t *testing.T) {
	logger := slog.Default()
	health := NewHealthMonitor(logger, "p1")
	p1 := newMockProvider("p1", map[Capability][]Market{CapIndexQuote: {MarketCN}}, map[string]int{"index_quote:cn": 1})

	router := NewProviderRouter([]Provider{p1}, health, RouterConfig{
		DefaultStrategy:        StrategyFallback,
		RaceTimeout:            2 * time.Second,
		PerCapabilityOverrides: map[Capability]FetchStrategy{CapIndexQuote: StrategyRace},
	}, logger)

	if s := router.strategyFor(CapIndexQuote); s != StrategyRace {
		t.Fatalf("expected race, got %s", s)
	}
	if s := router.strategyFor(CapIndexKline); s != StrategyFallback {
		t.Fatalf("expected fallback, got %s", s)
	}
}
