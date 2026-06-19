package providers

import (
	"errors"
	"io"
	"log/slog"
	"testing"
)

func TestGetPublicStatusOmitsProviderErrors(t *testing.T) {
	monitor := NewHealthMonitor(slog.New(slog.NewTextHandler(io.Discard, nil)), "biyingapi")
	monitor.RecordFailure("biyingapi", errors.New("request failed: https://api.example.test/token/secret-value"))

	status := monitor.GetPublicStatus()["biyingapi"]
	if status.LastError != "" {
		t.Fatalf("public health leaked provider error: %q", status.LastError)
	}
	for _, capability := range status.CapHealth {
		if capability.LastError != "" {
			t.Fatalf("public capability health leaked provider error: %q", capability.LastError)
		}
	}
}
