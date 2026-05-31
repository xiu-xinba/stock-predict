package api

import (
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

type metricsCollector struct {
	startedAt       time.Time
	requestCount    atomic.Uint64
	errorCount      atomic.Uint64
	inFlight        atomic.Int64
	totalDurationMs atomic.Uint64
	mu              sync.Mutex
	statusCounts    map[int]uint64
}

type metricsSnapshot struct {
	RequestCount  uint64            `json:"request_count"`
	ErrorCount    uint64            `json:"error_count"`
	InFlight      int64             `json:"in_flight"`
	AvgDurationMs float64           `json:"avg_duration_ms"`
	StatusCounts  map[string]uint64 `json:"status_counts"`
	UptimeSeconds int64             `json:"uptime_seconds"`
	CollectedAt   string            `json:"collected_at"`
}

func newMetricsCollector() *metricsCollector {
	return &metricsCollector{
		startedAt:    time.Now(),
		statusCounts: make(map[int]uint64),
	}
}

func metricsMiddleware(metrics *metricsCollector) gin.HandlerFunc {
	return func(c *gin.Context) {
		metrics.inFlight.Add(1)
		startedAt := time.Now()
		defer func() {
			status := c.Writer.Status()
			metrics.record(status, time.Since(startedAt))
			metrics.inFlight.Add(-1)
		}()
		c.Next()
	}
}

func (m *metricsCollector) record(status int, duration time.Duration) {
	m.requestCount.Add(1)
	if status >= http.StatusBadRequest {
		m.errorCount.Add(1)
	}
	m.totalDurationMs.Add(uint64(duration.Milliseconds()))

	m.mu.Lock()
	m.statusCounts[status]++
	m.mu.Unlock()
}

func (m *metricsCollector) snapshot() metricsSnapshot {
	requestCount := m.requestCount.Load()
	avgDuration := 0.0
	if requestCount > 0 {
		avgDuration = float64(m.totalDurationMs.Load()) / float64(requestCount)
	}

	m.mu.Lock()
	statusCounts := make(map[string]uint64, len(m.statusCounts))
	for status, count := range m.statusCounts {
		statusCounts[strconv.Itoa(status)] = count
	}
	m.mu.Unlock()

	return metricsSnapshot{
		RequestCount:  requestCount,
		ErrorCount:    m.errorCount.Load(),
		InFlight:      m.inFlight.Load(),
		AvgDurationMs: avgDuration,
		StatusCounts:  statusCounts,
		UptimeSeconds: int64(time.Since(m.startedAt).Seconds()),
		CollectedAt:   time.Now().UTC().Format(time.RFC3339),
	}
}

func (r *Router) metrics(c *gin.Context) {
	writeSuccess(c, r.metricsCollector.snapshot())
}
