// Package config 提供应用程序配置的加载、解析和校验功能。
//
// 支持从环境变量和 .env 文件加载配置，并提供统一的校验机制确保
// 生产环境下的安全性和配置完整性。
package config

import (
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Config 保存应用程序的全部配置项，通过环境变量或 .env 文件加载。
type Config struct {
	Port                      string        // HTTP 服务监听端口
	Host                      string        // HTTP 服务绑定地址
	Env                       string        // 运行环境：development、test 或 production
	CORSOrigins               []string      // 允许的跨域来源列表
	TrustedProxies            []string      // 可信代理 IP 或 CIDR 列表
	AdminToken                string        // 管理接口认证令牌
	FundUniverseURL           string        // 基金全量列表数据源 URL
	FundMetricsURL            string        // 基金排名指标数据源 URL
	FundSyncCSVPath           string        // 基金同步 CSV 文件路径，为空则不从文件同步
	FundAutoSyncOnStart       bool          // 启动时是否自动同步基金数据
	FundAutoSyncMinCount      int           // 自动同步时基金数量最低阈值
	FundRealtimeQuotesEnabled bool          // 是否启用基金实时行情
	StockAutoSyncOnStart      bool          // 启动时是否自动同步股票数据
	CacheTTLMinutes           int           // 缓存默认过期时间（分钟）
	EastMoneyBaseURL          string        // 东方财富 API 基础 URL
	TencentQuoteBaseURL       string        // 腾讯行情 API 基础 URL
	ReadTimeout               time.Duration // HTTP 服务器读超时
	WriteTimeout              time.Duration // HTTP 服务器写超时
	ShutdownTimeout           time.Duration // HTTP 服务器优雅关闭超时
	MarketSyncEnabled         bool          // 是否启用市场数据同步
	DatabaseURL               string        // PostgreSQL 数据库连接字符串
	BiyingAPIURL              string        // 必应 API 基础 URL
	BiyingAPIToken            string        // 必应 API 认证令牌
	AKShareURL                string        // AKShare 服务基础 URL
	AKShareToken              string        // AKShare 服务认证令牌
	RunDatabaseMigrations     bool          // 启动时是否自动执行数据库迁移
}

// Load 从环境变量和 .env 文件加载配置，返回完整的 Config 实例。
// 优先使用系统环境变量，.env 文件仅补充未设置的环境变量。
func Load() Config {
	loadDotEnv()
	environment := env("APP_ENV", "")

	return Config{
		Port:                      env("PORT", "5070"),
		Host:                      env("HOST", "127.0.0.1"),
		Env:                       environment,
		CORSOrigins:               splitCSV(envCORS()),
		TrustedProxies:            splitCSV(env("TRUSTED_PROXIES", "")),
		AdminToken:                env("ADMIN_TOKEN", ""),
		FundUniverseURL:           env("FUND_UNIVERSE_URL", "https://fund.eastmoney.com/js/fundcode_search.js"),
		FundMetricsURL:            env("FUND_METRICS_URL", defaultFundMetricsURL(time.Now())),
		FundSyncCSVPath:           env("FUND_SYNC_CSV_PATH", ""),
		FundAutoSyncOnStart:       boolEnv("FUND_AUTO_SYNC_ON_START", true),
		FundAutoSyncMinCount:      intEnv("FUND_AUTO_SYNC_MIN_COUNT", 1000),
		FundRealtimeQuotesEnabled: boolEnv("FUND_REALTIME_QUOTES_ENABLED", true),
		StockAutoSyncOnStart:      boolEnv("STOCK_AUTO_SYNC_ON_START", false),
		CacheTTLMinutes:           intEnv("CACHE_TTL_MINUTES", 5),
		EastMoneyBaseURL:          env("EASTMONEY_BASE_URL", "https://push2his.eastmoney.com"),
		TencentQuoteBaseURL:       env("TENCENT_QUOTE_BASE_URL", "https://qt.gtimg.cn"),
		ReadTimeout:               seconds("READ_TIMEOUT_SECONDS", 8),
		WriteTimeout:              seconds("WRITE_TIMEOUT_SECONDS", 12),
		ShutdownTimeout:           seconds("SHUTDOWN_TIMEOUT_SECONDS", 8),
		MarketSyncEnabled:         boolEnv("MARKET_SYNC_ENABLED", true),
		DatabaseURL:               env("DATABASE_URL", ""),
		BiyingAPIURL:              env("BIYING_API_URL", "https://api.biyingapi.com"),
		BiyingAPIToken:            env("BIYING_API_TOKEN", ""),
		AKShareURL:                env("AKSHARE_URL", "http://localhost:8900"),
		AKShareToken:              env("AKSHARE_SERVICE_TOKEN", ""),
		RunDatabaseMigrations:     boolEnv("RUN_DATABASE_MIGRATIONS", !strings.EqualFold(environment, "production")),
	}
}

// IsDevelopment 判断当前运行环境是否为开发模式（development 或 dev）。
func (c Config) IsDevelopment() bool {
	return strings.EqualFold(c.Env, "development") || strings.EqualFold(c.Env, "dev")
}

// Validate 校验配置的完整性和合法性，包括环境名称、数据库连接、
// 生产环境安全要求（ADMIN_TOKEN 长度、CORS 通配符限制）、
// 端口范围、可信代理格式以及上游 URL 协议。
func (c Config) Validate() error {
	switch strings.ToLower(strings.TrimSpace(c.Env)) {
	case "development", "dev", "test", "production":
	default:
		return fmt.Errorf("APP_ENV must be explicitly set to development, test, or production")
	}
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return fmt.Errorf("DATABASE_URL must be explicitly set")
	}
	if strings.EqualFold(c.Env, "production") {
		adminToken := strings.TrimSpace(c.AdminToken)
		if adminToken == "" {
			return fmt.Errorf("production environment requires ADMIN_TOKEN to be set")
		}
		if len(adminToken) < 32 || adminToken == "dev-admin-token" {
			return fmt.Errorf("production environment requires ADMIN_TOKEN to be at least 32 characters and not use a development token")
		}
		if slices.Contains(c.CORSOrigins, "*") {
			return fmt.Errorf("production environment must not use wildcard CORS origin \"*\"")
		}
	}
	port, err := strconv.Atoi(c.Port)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("invalid port %q: must be a number between 1 and 65535", c.Port)
	}
	for _, proxy := range c.TrustedProxies {
		if net.ParseIP(proxy) != nil {
			continue
		}
		if _, _, err := net.ParseCIDR(proxy); err != nil {
			return fmt.Errorf("invalid trusted proxy %q", proxy)
		}
	}
	if err := validateUpstreamURL("BIYING_API_URL", c.BiyingAPIURL, false); err != nil {
		return err
	}
	if err := validateUpstreamURL("AKSHARE_URL", c.AKShareURL, true); err != nil {
		return err
	}
	return nil
}

// validateUpstreamURL 校验上游服务 URL 的合法性，确保使用 HTTPS 协议，
// 除非 allowLoopbackHTTP 为 true 且目标为回环地址。
func validateUpstreamURL(name, rawURL string, allowLoopbackHTTP bool) error {
	if strings.TrimSpace(rawURL) == "" {
		return nil
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Hostname() == "" {
		return fmt.Errorf("%s must be a valid URL", name)
	}
	if parsed.Scheme == "https" {
		return nil
	}
	host := parsed.Hostname()
	ip := net.ParseIP(host)
	if allowLoopbackHTTP && parsed.Scheme == "http" && (host == "localhost" || (ip != nil && ip.IsLoopback())) {
		return nil
	}
	return fmt.Errorf("%s must use HTTPS unless it targets a loopback address", name)
}

// CacheTTL 返回缓存过期时间，若 CacheTTLMinutes 小于等于 0 则默认 5 分钟。
func (c Config) CacheTTL() time.Duration {
	if c.CacheTTLMinutes <= 0 {
		return 5 * time.Minute
	}
	return time.Duration(c.CacheTTLMinutes) * time.Minute
}

// LogLevel 根据运行环境返回日志级别：开发环境返回 Debug，其他环境返回 Info。
func (c Config) LogLevel() slog.Level {
	if c.IsDevelopment() {
		return slog.LevelDebug
	}
	return slog.LevelInfo
}

// env 读取指定环境变量，若不存在或为空则返回 fallback 默认值。
func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

// loadDotEnv 从当前目录的 .env 文件加载环境变量，
// 已存在的环境变量不会被覆盖。
func loadDotEnv() {
	data, err := os.ReadFile(".env")
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := parseDotEnvLine(line)
		if !ok {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		_ = os.Setenv(key, value)
	}
}

// parseDotEnvLine 解析 .env 文件的单行内容，支持引号包裹的值，
// 忽略空行和以 # 开头的注释行。
func parseDotEnvLine(line string) (string, string, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false
	}
	key, value, ok := strings.Cut(line, "=")
	if !ok {
		return "", "", false
	}
	key = strings.TrimSpace(key)
	if key == "" || strings.ContainsAny(key, " \t") {
		return "", "", false
	}
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		quote := value[0]
		if (quote == '\'' || quote == '"') && value[len(value)-1] == quote {
			value = value[1 : len(value)-1]
		}
	}
	return key, value, true
}

// seconds 从环境变量读取整数值并转换为 time.Duration（秒）。
func seconds(key string, fallback int) time.Duration {
	return time.Duration(intEnv(key, fallback)) * time.Second
}

// intEnv 从环境变量读取整数值，若解析失败或值非正则返回 fallback。
func intEnv(key string, fallback int) int {
	raw := env(key, strconv.Itoa(fallback))
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

// boolEnv 从环境变量读取布尔值，支持 "1/true/yes/y/on" 为真，
// "0/false/no/n/off" 为假，其他值返回 fallback。
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

// defaultFundMetricsURL 根据给定时间生成东方财富基金排名数据源的默认 URL，
// 查询区间为过去一年至当前日期。
func defaultFundMetricsURL(now time.Time) string {
	end := now.Format("2006-01-02")
	start := now.AddDate(-1, 0, 0).Format("2006-01-02")
	return fmt.Sprintf("https://fund.eastmoney.com/data/rankhandler.aspx?op=ph&dt=kf&ft=all&rs=&gs=0&sc=dm&st=asc&sd=%s&ed=%s&qdii=&tabSubtype=,,,,,&pi=1&pn=50000&dx=1&v=%d", start, end, now.UnixMilli())
}

// envCORS 读取 CORS 允许来源，优先使用 CORS_ALLOWED_ORIGINS，
// 回退到 CORS_ORIGINS 环境变量。
func envCORS() string {
	if v := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS")); v != "" {
		return v
	}
	return env("CORS_ORIGINS", "")
}

// splitCSV 将逗号分隔的字符串拆分为字符串切片，忽略空白部分。
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
