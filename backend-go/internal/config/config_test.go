package config

import "testing"

func TestLoadReadsFundStoreAndSyncPaths(t *testing.T) {
	t.Setenv("FUND_STORE_PATH", "tmp/funds.json")
	t.Setenv("FUND_UNIVERSE_URL", "https://example.test/funds.js")
	t.Setenv("FUND_METRICS_URL", "https://example.test/rank.js")
	t.Setenv("FUND_SYNC_CSV_PATH", "tmp/funds.csv")
	t.Setenv("FUND_AUTO_SYNC_ON_START", "false")
	t.Setenv("FUND_REALTIME_QUOTES_ENABLED", "false")

	cfg := Load()

	if cfg.FundStorePath != "tmp/funds.json" {
		t.Fatalf("unexpected fund store path: %q", cfg.FundStorePath)
	}
	if cfg.FundSyncCSVPath != "tmp/funds.csv" {
		t.Fatalf("unexpected fund sync csv path: %q", cfg.FundSyncCSVPath)
	}
	if cfg.FundUniverseURL != "https://example.test/funds.js" {
		t.Fatalf("unexpected fund universe url: %q", cfg.FundUniverseURL)
	}
	if cfg.FundMetricsURL != "https://example.test/rank.js" {
		t.Fatalf("unexpected fund metrics url: %q", cfg.FundMetricsURL)
	}
	if cfg.FundAutoSyncOnStart {
		t.Fatalf("expected auto sync to be disabled")
	}
	if cfg.FundRealtimeQuotesEnabled {
		t.Fatalf("expected realtime quotes to be disabled")
	}
}

func TestSplitCSVFallsBackWhenEmpty(t *testing.T) {
	got := splitCSV(" , ")

	if len(got) != 1 || got[0] != "http://localhost:5173" {
		t.Fatalf("unexpected fallback origins: %+v", got)
	}
}
