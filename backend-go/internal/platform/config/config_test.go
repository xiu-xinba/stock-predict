// Package config 提供应用程序配置的加载、解析和校验功能。
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReadsFundSyncSources(t *testing.T) {
	t.Setenv("FUND_UNIVERSE_URL", "https://example.test/funds.js")
	t.Setenv("FUND_METRICS_URL", "https://example.test/rank.js")
	t.Setenv("FUND_SYNC_CSV_PATH", "tmp/funds.csv")
	t.Setenv("FUND_AUTO_SYNC_ON_START", "false")
	t.Setenv("FUND_REALTIME_QUOTES_ENABLED", "false")

	cfg := Load()

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

func TestLoadDisablesStockAutoSyncByDefault(t *testing.T) {
	cfg := Load()

	if cfg.StockAutoSyncOnStart {
		t.Fatalf("expected stock auto sync to be disabled by default")
	}
}

func TestLoadDefaultsExternalProviderURLs(t *testing.T) {
	unsetEnv(t, "BIYING_API_URL", "BIYING_API_TOKEN", "AKSHARE_URL")
	chdir(t, t.TempDir())

	cfg := Load()

	if cfg.BiyingAPIURL != "https://api.biyingapi.com" {
		t.Fatalf("unexpected BiyingAPI default URL: %q", cfg.BiyingAPIURL)
	}
	if cfg.BiyingAPIToken != "" {
		t.Fatalf("expected empty BiyingAPI token by default")
	}
	if cfg.AKShareURL != "http://localhost:8900" {
		t.Fatalf("unexpected AKShare default URL: %q", cfg.AKShareURL)
	}
}

func TestLoadReadsDotEnvWithoutOverridingEnvironment(t *testing.T) {
	unsetEnv(t, "BIYING_API_URL", "BIYING_API_TOKEN", "AKSHARE_URL")
	t.Setenv("AKSHARE_URL", "http://env-akshare:8900")
	dir := t.TempDir()
	chdir(t, dir)
	dotEnv := []byte(`
BIYING_API_URL="http://biying.test"
BIYING_API_TOKEN='local-token'
AKSHARE_URL=http://dotenv-akshare:8900
`)
	if err := os.WriteFile(filepath.Join(dir, ".env"), dotEnv, 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	cfg := Load()

	if cfg.BiyingAPIURL != "http://biying.test" {
		t.Fatalf("unexpected BiyingAPI URL from .env: %q", cfg.BiyingAPIURL)
	}
	if cfg.BiyingAPIToken != "local-token" {
		t.Fatalf("unexpected BiyingAPI token from .env: %q", cfg.BiyingAPIToken)
	}
	if cfg.AKShareURL != "http://env-akshare:8900" {
		t.Fatalf("expected environment AKShare URL to win, got %q", cfg.AKShareURL)
	}
}

func TestSplitCSVReturnsNilWhenEmpty(t *testing.T) {
	got := splitCSV(" , ")

	if len(got) != 0 {
		t.Fatalf("expected nil for empty input, got: %+v", got)
	}
}

func TestValidateRejectsWeakProductionAdminToken(t *testing.T) {
	for _, token := range []string{"short-token", "dev-admin-token"} {
		cfg := Config{
			Port:        "5070",
			Env:         "production",
			AdminToken:  token,
			CORSOrigins: []string{"https://stock.example.com"},
		}

		if err := cfg.Validate(); err == nil {
			t.Fatalf("expected production config to reject weak admin token %q", token)
		}
	}
}

func TestValidateRejectsInvalidTrustedProxy(t *testing.T) {
	cfg := Config{
		Port:           "5070",
		Env:            "development",
		DatabaseURL:    "postgres://stock:secret@localhost:5432/stock_predict",
		TrustedProxies: []string{"not-a-proxy"},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid trusted proxy to be rejected")
	}
}

func TestValidateRequiresExplicitEnvironment(t *testing.T) {
	cfg := Config{
		Port:        "5070",
		DatabaseURL: "postgres://stock:secret@localhost:5432/stock_predict",
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing APP_ENV to be rejected")
	}
}

func TestValidateRequiresDatabaseURL(t *testing.T) {
	cfg := Config{
		Port: "5070",
		Env:  "development",
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing DATABASE_URL to be rejected")
	}
}

func TestLoadDisablesRuntimeMigrationsInProductionByDefault(t *testing.T) {
	unsetEnv(t, "RUN_DATABASE_MIGRATIONS")
	t.Setenv("APP_ENV", "production")
	chdir(t, t.TempDir())

	cfg := Load()

	if cfg.RunDatabaseMigrations {
		t.Fatal("production must not run schema migrations during API startup by default")
	}
}

// unsetEnv 在测试中临时清除指定环境变量，测试结束后自动恢复原值。
func unsetEnv(t *testing.T, keys ...string) {
	t.Helper()
	previous := make(map[string]string, len(keys))
	present := make(map[string]bool, len(keys))
	for _, key := range keys {
		value, ok := os.LookupEnv(key)
		previous[key] = value
		present[key] = ok
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
	}
	t.Cleanup(func() {
		for _, key := range keys {
			if present[key] {
				_ = os.Setenv(key, previous[key])
			} else {
				_ = os.Unsetenv(key)
			}
		}
	})
}

// chdir 在测试中临时切换工作目录，测试结束后自动恢复原目录。
func chdir(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(old)
	})
}
