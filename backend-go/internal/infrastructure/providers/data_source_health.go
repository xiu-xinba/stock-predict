package providers

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// ── 健康状态常量（保持向后兼容） ──

const (
	// SourceStatusHealthy 数据源健康状态：正常
	SourceStatusHealthy = "healthy"
	// SourceStatusUnhealthy 数据源健康状态：不可用
	SourceStatusUnhealthy = "unhealthy"
	// SourceStatusDegraded 数据源健康状态：降级
	SourceStatusDegraded = "degraded"
)

// HealthStatus 类型化的健康状态
type HealthStatus string

const (
	// HealthStatusHealthy 健康状态枚举：正常
	HealthStatusHealthy HealthStatus = "healthy"
	// HealthStatusDegraded 健康状态枚举：降级
	HealthStatusDegraded HealthStatus = "degraded"
	// HealthStatusUnhealthy 健康状态枚举：不可用
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// ── 环形缓冲区 ──

// RingBuffer 固定大小的环形缓冲区
type RingBuffer[T any] struct {
	data []T
	cap  int
	head int
	size int
	mu   sync.Mutex
}

// NewRingBuffer 创建指定容量的环形缓冲区。
func NewRingBuffer[T any](cap int) *RingBuffer[T] {
	return &RingBuffer[T]{data: make([]T, cap), cap: cap}
}

// Push 向环形缓冲区追加一个元素。缓冲区满时覆盖最旧的数据。
func (r *RingBuffer[T]) Push(val T) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[r.head] = val
	r.head = (r.head + 1) % r.cap
	if r.size < r.cap {
		r.size++
	}
}

// All 返回缓冲区中所有元素，按插入顺序排列。
func (r *RingBuffer[T]) All() []T {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]T, r.size)
	for i := 0; i < r.size; i++ {
		idx := (r.head - r.size + i + r.cap) % r.cap
		result[i] = r.data[idx]
	}
	return result
}

// ── 健康配置 ──

// HealthConfig 健康监控配置
type HealthConfig struct {
	// FailThreshold 连续失败多少次标记为 unhealthy
	FailThreshold int
	// DegradedThreshold 连续失败多少次标记为 degraded
	DegradedThreshold int
	// RecoveryInterval 自动恢复探测间隔
	RecoveryInterval time.Duration
	// LatencyWarningMs 延迟警告阈值 (毫秒)
	LatencyWarningMs int64
	// SuccessRateWindow 成功率计算窗口大小 (最近N次请求)
	SuccessRateWindow int
}

// DefaultHealthConfig 返回默认的健康监控配置。
func DefaultHealthConfig() HealthConfig {
	return HealthConfig{
		FailThreshold:     HealthCheckFailThreshold,
		DegradedThreshold: 1,
		RecoveryInterval:  time.Duration(HealthRecoveryInterval) * time.Minute,
		LatencyWarningMs:  5000,
		SuccessRateWindow: 20,
	}
}

// ── Per-Capability 健康状态 ──

// CapHealth 单个能力的健康状态
type CapHealth struct {
	Status    HealthStatus `json:"status"`
	FailCount int          `json:"fail_count"`
	LastError string       `json:"last_error"`
	LastCheck time.Time    `json:"last_check"`
}

// ── 数据源健康状态 ──

// SourceHealth 数据源健康状态的导出视图（无锁，可安全拷贝）
type SourceHealth struct {
	Name      string                    `json:"name"`
	Status    string                    `json:"status"`
	FailCount int                       `json:"fail_count"`
	LastCheck time.Time                 `json:"last_check"`
	LastError string                    `json:"last_error"`
	CapHealth map[Capability]*CapHealth `json:"cap_health,omitempty"`
}

// sourceHealthInternal 内部健康状态（含锁和指标，不可拷贝）
type sourceHealthInternal struct {
	mu sync.RWMutex

	Name      string
	Status    string
	FailCount int
	LastCheck time.Time
	LastError string

	CapHealth map[Capability]*CapHealth

	recentResults *RingBuffer[bool]
	recentLatency *RingBuffer[int64]
}

// toPublic 返回可安全拷贝的导出视图
func (s *sourceHealthInternal) toPublic() SourceHealth {
	return SourceHealth{
		Name:      s.Name,
		Status:    s.Status,
		FailCount: s.FailCount,
		LastCheck: s.LastCheck,
		LastError: s.LastError,
		CapHealth: s.CapHealth,
	}
}

// ── 健康监控器 ──

// HealthMonitor 数据源健康监控器，跟踪所有数据源的健康状态。
type HealthMonitor struct {
	mu      sync.RWMutex
	sources map[string]*sourceHealthInternal
	config  HealthConfig
	logger  *slog.Logger
	cancel  context.CancelFunc

	// Active health probing: periodically call Provider.HealthCheck()
	// to detect issues with low-priority providers that are rarely
	// selected by the router and thus never get passive health updates.
	providers []Provider
	probeOnce sync.Once

	// 模拟支持 (测试用)
	simulated map[string]HealthStatus
	simMu     sync.RWMutex
}

// NewHealthMonitor 使用默认配置创建 HealthMonitor，保持向后兼容的签名。
func NewHealthMonitor(logger *slog.Logger, sourceNames ...string) *HealthMonitor {
	return NewHealthMonitorWithConfig(logger, DefaultHealthConfig(), sourceNames...)
}

// NewHealthMonitorWithConfig 使用自定义配置创建 HealthMonitor。
func NewHealthMonitorWithConfig(logger *slog.Logger, config HealthConfig, sourceNames ...string) *HealthMonitor {
	hm := &HealthMonitor{
		sources:   make(map[string]*sourceHealthInternal, len(sourceNames)),
		config:    config,
		logger:    logger,
		simulated: make(map[string]HealthStatus),
	}
	for _, name := range sourceNames {
		hm.sources[name] = newSourceHealthInternal(name, config)
	}
	return hm
}

// newSourceHealthInternal 创建数据源内部健康状态实例。
func newSourceHealthInternal(name string, config HealthConfig) *sourceHealthInternal {
	return &sourceHealthInternal{
		Name:          name,
		Status:        SourceStatusHealthy,
		CapHealth:     make(map[Capability]*CapHealth),
		recentResults: NewRingBuffer[bool](config.SuccessRateWindow),
		recentLatency: NewRingBuffer[int64](config.SuccessRateWindow),
	}
}

// ── 核心方法（向后兼容） ──

// RecordSuccess 记录数据源请求成功（向后兼容，不含能力维度）。
func (h *HealthMonitor) RecordSuccess(source string) {
	h.RecordSuccessWithCap(source, "")
}

// RecordSuccessWithCap 记录请求成功（含能力维度）
func (h *HealthMonitor) RecordSuccessWithCap(source string, cap Capability) {
	s, ok := h.sources[source]
	if !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	s.FailCount = 0
	s.Status = SourceStatusHealthy
	s.LastCheck = now
	s.LastError = ""
	s.recentResults.Push(true)

	// 更新 per-capability 状态
	if cap != "" {
		ch := s.CapHealth[cap]
		if ch != nil {
			ch.Status = HealthStatusHealthy
			ch.FailCount = 0
			ch.LastCheck = now
			ch.LastError = ""
		}
	}
}

// RecordFailure 记录数据源请求失败。连续失败次数达到 FailThreshold 后标记为 unhealthy。
func (h *HealthMonitor) RecordFailure(source string, err error) {
	h.RecordFailureWithCap(source, "", err)
}

// RecordFailureWithCap 记录请求失败（含能力维度）
func (h *HealthMonitor) RecordFailureWithCap(source string, cap Capability, err error) {
	s, ok := h.sources[source]
	if !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	s.FailCount++
	s.LastCheck = now
	if err != nil {
		s.LastError = err.Error()
	}
	s.recentResults.Push(false)

	// 更新整体状态
	if s.FailCount >= h.config.FailThreshold {
		s.Status = SourceStatusUnhealthy
	} else if s.FailCount >= h.config.DegradedThreshold {
		s.Status = SourceStatusDegraded
	}

	// 更新 per-capability 状态
	if cap != "" {
		ch, exists := s.CapHealth[cap]
		if !exists {
			ch = &CapHealth{}
			s.CapHealth[cap] = ch
		}
		ch.FailCount++
		ch.LastCheck = now
		if err != nil {
			ch.LastError = err.Error()
		}
		if ch.FailCount >= h.config.FailThreshold {
			ch.Status = HealthStatusUnhealthy
		} else if ch.FailCount >= h.config.DegradedThreshold {
			ch.Status = HealthStatusDegraded
		}
	}

	h.logger.Warn("data source health degraded",
		"source", source, "capability", string(cap),
		"status", s.Status, "fail_count", s.FailCount, "error", s.LastError)
}

// RecordLatency 记录请求延迟
func (h *HealthMonitor) RecordLatency(source string, duration time.Duration) {
	s, ok := h.sources[source]
	if !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recentLatency.Push(duration.Milliseconds())
}

// IsHealthy 检查数据源是否健康（非 unhealthy 状态）。
func (h *HealthMonitor) IsHealthy(source string) bool {
	// 检查模拟状态
	h.simMu.RLock()
	if sim, ok := h.simulated[source]; ok {
		h.simMu.RUnlock()
		return sim == HealthStatusHealthy || sim == HealthStatusDegraded
	}
	h.simMu.RUnlock()

	s, ok := h.sources[source]
	if !ok {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Status != SourceStatusUnhealthy
}

// IsCapHealthy 检查数据源某能力是否健康
func (h *HealthMonitor) IsCapHealthy(source string, cap Capability) bool {
	s, ok := h.sources[source]
	if !ok {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 如果整体 unhealthy，直接返回
	if s.Status == SourceStatusUnhealthy {
		return false
	}
	// 检查 per-capability 状态
	ch, exists := s.CapHealth[cap]
	if !exists {
		return true // 没有记录说明没出过问题
	}
	return ch.Status != HealthStatusUnhealthy
}

// TryRecovery 尝试将不可用的数据源重置为降级状态。
// 若数据源原状态为 unhealthy 则重置并返回 true，否则返回 false。
func (h *HealthMonitor) TryRecovery(source string) bool {
	s, ok := h.sources[source]
	if !ok {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Status == SourceStatusUnhealthy {
		s.Status = SourceStatusDegraded
		s.FailCount = 0
		// 同时恢复 per-capability 状态
		for _, ch := range s.CapHealth {
			if ch.Status == HealthStatusUnhealthy {
				ch.Status = HealthStatusDegraded
				ch.FailCount = 0
			}
		}
		h.logger.Info("data source recovery attempted", "source", source)
		return true
	}
	return false
}

// ── 指标查询 ──

// SuccessRate 返回数据源最近N次请求的成功率 (0.0 ~ 1.0)
func (h *HealthMonitor) SuccessRate(source string) float64 {
	s, ok := h.sources[source]
	if !ok {
		return 0
	}
	results := s.recentResults.All()
	if len(results) == 0 {
		return 1.0
	}
	success := 0
	for _, r := range results {
		if r {
			success++
		}
	}
	return float64(success) / float64(len(results))
}

// AvgLatency 返回数据源最近N次请求的平均延迟
func (h *HealthMonitor) AvgLatency(source string) time.Duration {
	s, ok := h.sources[source]
	if !ok {
		return 0
	}
	latencies := s.recentLatency.All()
	if len(latencies) == 0 {
		return 0
	}
	var sum int64
	for _, l := range latencies {
		sum += l
	}
	return time.Duration(sum/int64(len(latencies))) * time.Millisecond
}

// ── 状态查询 ──

// GetAllStatus 返回所有数据源健康状态的副本。
// 返回的 SourceHealth 不包含 mu/recentResults/recentLatency 等内部字段。
func (h *HealthMonitor) GetAllStatus() map[string]SourceHealth {
	h.mu.RLock()
	defer h.mu.RUnlock()
	result := make(map[string]SourceHealth, len(h.sources))
	for k, v := range h.sources {
		v.mu.RLock()
		health := v.toPublic()
		// 深拷贝 cap health
		if len(v.CapHealth) > 0 {
			health.CapHealth = make(map[Capability]*CapHealth, len(v.CapHealth))
			for ck, cv := range v.CapHealth {
				cc := *cv
				health.CapHealth[ck] = &cc
			}
		}
		v.mu.RUnlock()
		result[k] = health
	}
	return result
}

// GetPublicStatus 返回对外展示的运营状态，不暴露上游错误详情。
func (h *HealthMonitor) GetPublicStatus() map[string]SourceHealth {
	result := h.GetAllStatus()
	for name, health := range result {
		health.LastError = ""
		for _, capability := range health.CapHealth {
			capability.LastError = ""
		}
		result[name] = health
	}
	return result
}

// ── 自动恢复探测 ──

// SetProviders 注册 Provider 列表用于主动健康探测。
// 必须在 StartRecoveryProbe 之前调用。
func (h *HealthMonitor) SetProviders(providers []Provider) {
	h.providers = providers
}

// StartRecoveryProbe starts a background goroutine that periodically checks
// unhealthy data sources and attempts recovery, and also performs active
// health probing on all registered providers.
