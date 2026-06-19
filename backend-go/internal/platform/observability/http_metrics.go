// Package observability 提供了 HTTP 请求的可观测性指标收集功能。
package observability

import (
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

// HTTPMetrics 收集和统计 HTTP 请求的运行时指标，包括请求数、错误数、
// 在途请求数、响应时间以及状态码分布。
type HTTPMetrics struct {
	startedAt       time.Time         // 指标收集的启动时间
	requestCount    atomic.Uint64     // 总请求计数
	errorCount      atomic.Uint64     // 错误请求计数（状态码 >= 400）
	inFlight        atomic.Int64      // 当前在途请求数
	totalDurationMs atomic.Uint64     // 所有请求的总耗时（毫秒）
	mu              sync.Mutex        // 保护 statusCounts 的互斥锁
	statusCounts    map[int]uint64    // HTTP 状态码到请求计数的映射
}

// HTTPSnapshot 是某一时刻的 HTTP 指标快照，用于序列化为 JSON 响应。
type HTTPSnapshot struct {
	RequestCount  uint64            `json:"request_count"`  // 总请求数
	ErrorCount    uint64            `json:"error_count"`    // 错误请求数
	InFlight      int64             `json:"in_flight"`      // 当前在途请求数
	AvgDurationMs float64           `json:"avg_duration_ms"` // 平均请求耗时（毫秒）
	StatusCounts  map[string]uint64 `json:"status_counts"`  // HTTP 状态码分布
	UptimeSeconds int64             `json:"uptime_seconds"` // 服务运行时长（秒）
	CollectedAt   string            `json:"collected_at"`   // 快照采集时间（RFC3339）
}

// NewHTTPMetrics 创建一个新的 HTTP 指标收集器。
func NewHTTPMetrics() *HTTPMetrics {
	return &HTTPMetrics{
		startedAt:    time.Now(),
		statusCounts: make(map[int]uint64),
	}
}

// Middleware 返回 Gin 中间件，自动记录每个请求的状态码和耗时。
func (m *HTTPMetrics) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		m.inFlight.Add(1)
		startedAt := time.Now()
		defer func() {
			m.record(c.Writer.Status(), time.Since(startedAt))
			m.inFlight.Add(-1)
		}()
		c.Next()
	}
}

func (m *HTTPMetrics) record(status int, duration time.Duration) {
	m.requestCount.Add(1)
	if status >= http.StatusBadRequest {
		m.errorCount.Add(1)
	}
	m.totalDurationMs.Add(uint64(duration.Milliseconds()))

	m.mu.Lock()
	m.statusCounts[status]++
	m.mu.Unlock()
}

// Snapshot 返回当前时刻的 HTTP 指标快照。
func (m *HTTPMetrics) Snapshot() HTTPSnapshot {
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

	return HTTPSnapshot{
		RequestCount:  requestCount,
		ErrorCount:    m.errorCount.Load(),
		InFlight:      m.inFlight.Load(),
		AvgDurationMs: avgDuration,
		StatusCounts:  statusCounts,
		UptimeSeconds: int64(time.Since(m.startedAt).Seconds()),
		CollectedAt:   time.Now().UTC().Format(time.RFC3339),
	}
}
