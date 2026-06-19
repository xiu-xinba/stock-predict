package providers

import (
	"context"
	"fmt"
	"time"
)

// StartRecoveryProbe 启动后台 goroutine，定期检查不可用的数据源并尝试恢复，
// 同时对所有已注册的 Provider 执行主动健康探测。
func (h *HealthMonitor) StartRecoveryProbe() {
	h.mu.Lock()
	if h.cancel != nil {
		h.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	h.cancel = cancel
	h.mu.Unlock()

	go func() {
		// Perform an initial probe immediately
		h.probeAllProviders(context.WithoutCancel(ctx))

		ticker := time.NewTicker(h.config.RecoveryInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				h.probeAllProviders(context.WithoutCancel(ctx))

				h.mu.RLock()
				unhealthy := make([]string, 0)
				for name, sh := range h.sources {
					sh.mu.RLock()
					if sh.Status == SourceStatusUnhealthy {
						unhealthy = append(unhealthy, name)
					}
					sh.mu.RUnlock()
				}
				h.mu.RUnlock()
				for _, name := range unhealthy {
					h.TryRecovery(name)
				}
			}
		}
	}()
}

// probeAllProviders 对所有已注册的 Provider 调用 HealthCheck 并更新健康状态。
// 确保低优先级的 Provider（很少被路由器选中）也能定期更新健康状态。
func (h *HealthMonitor) probeAllProviders(ctx context.Context) {
	if len(h.providers) == 0 {
		return
	}
	probeCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	for _, p := range h.providers {
		name := p.Name()
		// Only probe sources that are registered in the monitor
		if _, ok := h.sources[name]; !ok {
			continue
		}
		err := p.HealthCheck(probeCtx)
		if err != nil {
			h.RecordFailure(name, err)
		} else {
			h.RecordSuccess(name)
		}
	}
}

// StopRecoveryProbe 停止恢复探测的后台 goroutine。
func (h *HealthMonitor) StopRecoveryProbe() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.cancel != nil {
		h.cancel()
		h.cancel = nil
	}
}

// ── 模拟支持 (测试用，保持向后兼容) ──

// SimulateStatus 为测试目的覆盖指定数据源的健康状态。
func (h *HealthMonitor) SimulateStatus(source, status, errMsg string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if sh, ok := h.sources[source]; ok {
		sh.Status = status
		sh.LastCheck = time.Now()
		sh.LastError = errMsg
		switch status {
		case SourceStatusUnhealthy:
			sh.FailCount = h.config.FailThreshold
		case SourceStatusDegraded:
			sh.FailCount = 1
		default:
			sh.FailCount = 0
		}
	}
}

// ResetSimulation 清除所有模拟覆盖，将数据源重置为真实健康状态。
func (h *HealthMonitor) ResetSimulation() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, sh := range h.sources {
		sh.Status = SourceStatusHealthy
		sh.FailCount = 0
		sh.LastError = ""
		sh.LastCheck = time.Now()
	}
}

// ── 动态注册 ──

// RegisterProvider 动态注册新数据源到监控
func (h *HealthMonitor) RegisterProvider(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, exists := h.sources[name]; !exists {
		h.sources[name] = newSourceHealthInternal(name, h.config)
		h.logger.Info("registered new data source in health monitor", "source", name)
	}
}

// ── 辅助方法 ──

// SourceNames 返回所有已注册的数据源名称
func (h *HealthMonitor) SourceNames() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	names := make([]string, 0, len(h.sources))
	for name := range h.sources {
		names = append(names, name)
	}
	return names
}

// FormatHealthReport 格式化健康报告
func (h *HealthMonitor) FormatHealthReport() string {
	statuses := h.GetAllStatus()
	report := "Data Source Health Report:\n"
	for name, sh := range statuses {
		report += fmt.Sprintf("  %s: %s (fail_count=%d", name, sh.Status, sh.FailCount)
		if sh.LastError != "" {
			report += fmt.Sprintf(", last_error=%s", sh.LastError)
		}
		if sr := h.SuccessRate(name); sr < 1.0 {
			report += fmt.Sprintf(", success_rate=%.0f%%", sr*100)
		}
		if al := h.AvgLatency(name); al > 0 {
			report += fmt.Sprintf(", avg_latency=%s", al.Round(time.Millisecond))
		}
		report += ")\n"
		// per-capability 详情
		for cap, ch := range sh.CapHealth {
			report += fmt.Sprintf("    %s: %s (fail_count=%d", cap, ch.Status, ch.FailCount)
			if ch.LastError != "" {
				report += fmt.Sprintf(", last_error=%s", ch.LastError)
			}
			report += ")\n"
		}
	}
	return report
}
