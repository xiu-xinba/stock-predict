package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port            string
	Env             string
	CORSOrigins     []string
	AdminToken      string
	ModelServiceURL string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

func Load() Config {
	return Config{
		Port:            env("PORT", "5070"),
		Env:             env("APP_ENV", "development"),
		CORSOrigins:     splitCSV(env("CORS_ORIGINS", "http://localhost:5173")),
		AdminToken:      env("ADMIN_TOKEN", ""),
		ModelServiceURL: env("MODEL_SERVICE_URL", ""),
		ReadTimeout:     seconds("READ_TIMEOUT_SECONDS", 8),
		WriteTimeout:    seconds("WRITE_TIMEOUT_SECONDS", 12),
		ShutdownTimeout: seconds("SHUTDOWN_TIMEOUT_SECONDS", 8),
	}
}

func (c Config) IsDevelopment() bool {
	return strings.EqualFold(c.Env, "development") || strings.EqualFold(c.Env, "dev")
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
	raw := env(key, strconv.Itoa(fallback))
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		value = fallback
	}
	return time.Duration(value) * time.Second
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return []string{"http://localhost:5173"}
	}
	return out
}
