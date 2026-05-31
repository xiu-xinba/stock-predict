package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port                      string
	Env                       string
	CORSOrigins               []string
	AdminToken                string
	FundStorePath             string
	FundUniverseURL           string
	FundMetricsURL            string
	FundSyncCSVPath           string
	FundAutoSyncOnStart       bool
	FundAutoSyncMinCount      int
	FundRealtimeQuotesEnabled bool
	StockAutoSyncOnStart      bool
	ModelServiceURL           string
	WeeklyModelServiceURL     string
	IntradayModelServiceURL   string
	CacheTTLMinutes           int
	EastMoneyBaseURL          string
	TencentQuoteBaseURL       string
	ReadTimeout               time.Duration
	WriteTimeout              time.Duration
	ShutdownTimeout           time.Duration
}

func Load() Config {
	return Config{
		Port:                      env("PORT", "5070"),
		Env:                       env("APP_ENV", "development"),
		CORSOrigins:               splitCSV(envCORS()),
		AdminToken:                env("ADMIN_TOKEN", ""),
		FundStorePath:             env("FUND_STORE_PATH", "data/funds.json"),
		FundUniverseURL:           env("FUND_UNIVERSE_URL", "https://fund.eastmoney.com/js/fundcode_search.js"),
		FundMetricsURL:            env("FUND_METRICS_URL", defaultFundMetricsURL(time.Now())),
		FundSyncCSVPath:           env("FUND_SYNC_CSV_PATH", ""),
		FundAutoSyncOnStart:       boolEnv("FUND_AUTO_SYNC_ON_START", true),
		FundAutoSyncMinCount:      intEnv("FUND_AUTO_SYNC_MIN_COUNT", 1000),
		FundRealtimeQuotesEnabled: boolEnv("FUND_REALTIME_QUOTES_ENABLED", true),
		StockAutoSyncOnStart:      boolEnv("STOCK_AUTO_SYNC_ON_START", true),
		ModelServiceURL:           env("MODEL_SERVICE_URL", ""),
		WeeklyModelServiceURL:     env("WEEKLY_MODEL_SERVICE_URL", ""),
		IntradayModelServiceURL:   env("INTRADAY_MODEL_SERVICE_URL", ""),
		CacheTTLMinutes:           intEnv("CACHE_TTL_MINUTES", 5),
		EastMoneyBaseURL:          env("EASTMONEY_BASE_URL", "https://push2his.eastmoney.com"),
		TencentQuoteBaseURL:       env("TENCENT_QUOTE_BASE_URL", "https://qt.gtimg.cn"),
		ReadTimeout:               seconds("READ_TIMEOUT_SECONDS", 8),
		WriteTimeout:              seconds("WRITE_TIMEOUT_SECONDS", 12),
		ShutdownTimeout:           seconds("SHUTDOWN_TIMEOUT_SECONDS", 8),
	}
}

func (c Config) IsDevelopment() bool {
	return strings.EqualFold(c.Env, "development") || strings.EqualFold(c.Env, "dev")
}

func (c Config) CacheTTL() time.Duration {
	if c.CacheTTLMinutes <= 0 {
		return 5 * time.Minute
	}
	return time.Duration(c.CacheTTLMinutes) * time.Minute
}

func (c Config) LogLevel() slog.Level {
	if c.IsDevelopment() {
		return slog.LevelDebug
	}
	return slog.LevelInfo
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func seconds(key string, fallback int) time.Duration {
	return time.Duration(intEnv(key, fallback)) * time.Second
}

func intEnv(key string, fallback int) int {
	raw := env(key, strconv.Itoa(fallback))
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func boolEnv(key string, fallback bool) bool {
	raw := strings.ToLower(env(key, strconv.FormatBool(fallback)))
	switch raw {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return fallback
	}
}

func defaultFundMetricsURL(now time.Time) string {
	end := now.Format("2006-01-02")
	start := now.AddDate(-1, 0, 0).Format("2006-01-02")
	return fmt.Sprintf("https://fund.eastmoney.com/data/rankhandler.aspx?op=ph&dt=kf&ft=all&rs=&gs=0&sc=dm&st=asc&sd=%s&ed=%s&qdii=&tabSubtype=,,,,,&pi=1&pn=50000&dx=1&v=%d", start, end, now.UnixMilli())
}

func envCORS() string {
	if v := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS")); v != "" {
		return v
	}
	return env("CORS_ORIGINS", "")
}

func splitCSV(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
