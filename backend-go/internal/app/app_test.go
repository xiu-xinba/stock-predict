package app

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"stock-predict-go/internal/config"
)

func TestNewServerRespondsToHealth(t *testing.T) {
	cfg := config.Config{
		Port:            "0",
		Env:             "test",
		CORSOrigins:     []string{"http://localhost:5173"},
		FundStorePath:   "",
		ReadTimeout:     time.Second,
		WriteTimeout:    time.Second,
		ShutdownTimeout: time.Second,
	}
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

	server, cleanup, err := NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}
	defer cleanup()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	server.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected health 200, got %d: %s", rec.Code, rec.Body.String())
	}
}
