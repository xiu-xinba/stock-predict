package providers

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"
)

// RouterConfig 配置数据源路由策略和超时参数。
type RouterConfig struct {
	DefaultStrategy        FetchStrategy               // 默认请求策略
	RaceTimeout            time.Duration                // 竞速模式超时时间
	PerCapabilityOverrides map[Capability]FetchStrategy // 按能力覆盖的策略
}

// ProviderRouter 根据能力和市场选择合适的数据源，并按配置的策略（回退/竞速/混合）执行请求。
type ProviderRouter struct {
	providers []Provider
	health    *HealthMonitor
	config    RouterConfig
	logger    *slog.Logger
}

// NewProviderRouter 创建一个新的数据源路由器。
func NewProviderRouter(providers []Provider, health *HealthMonitor, config RouterConfig, logger *slog.Logger) *ProviderRouter {
	if logger == nil {
		logger = slog.Default()
	}
	return &ProviderRouter{
		providers: providers,
		health:    health,
		config:    config,
		logger:    logger,
	}
}

// resolveProviders 根据能力和市场筛选候选数据源，并按优先级排序。
// 当 skipUnhealthy 为 true 时，跳过不健康的数据源。
func (r *ProviderRouter) resolveProviders(cap Capability, market Market, skipUnhealthy bool) []Provider {
	var candidates []Provider
	for _, p := range r.providers {
		markets, ok := p.Capabilities()[cap]
		if !ok {
			continue
		}
		supported := false
		for _, m := range markets {
			if m == market {
				supported = true
				break
			}
		}
		if !supported {
			continue
		}
		if skipUnhealthy && !r.health.IsHealthy(p.Name()) {
			r.logger.Debug("skipping unhealthy provider", "provider", p.Name(), "capability", string(cap), "market", string(market))
			continue
		}
		candidates = append(candidates, p)
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Priority(cap, market) < candidates[j].Priority(cap, market)
	})
	return candidates
}

// strategyFor 返回指定能力的请求策略，若未覆盖则使用默认策略。
func (r *ProviderRouter) strategyFor(cap Capability) FetchStrategy {
	if override, ok := r.config.PerCapabilityOverrides[cap]; ok {
		return override
	}
	return r.config.DefaultStrategy
}

// Fetch 根据配置的策略执行数据请求，fn 为实际调用数据源的闭包。
func (r *ProviderRouter) Fetch(ctx context.Context, cap Capability, market Market, fn func(ctx context.Context, p Provider) error) error {
	strategy := r.strategyFor(cap)
	switch strategy {
	case StrategyRace:
		return r.FetchWithRace(ctx, cap, market, fn, 2)
	case StrategyRaceThenFallback:
		candidates := r.resolveProviders(cap, market, true)
		if len(candidates) == 0 {
			candidates = r.resolveProviders(cap, market, false)
		}
		if len(candidates) >= 2 {
			if err := r.FetchWithRace(ctx, cap, market, fn, 2); err == nil {
				return nil
			}
			raced := map[string]bool{}
			for _, p := range candidates[:2] {
				raced[p.Name()] = true
			}
			// Race 失败后，重新获取包含不健康 provider 的完整候选列表
			allCandidates := r.resolveProviders(cap, market, false)
			r.logger.Warn("RaceThenFallback: race failed, trying all candidates", "capability", string(cap), "market", string(market), "allCandidates", len(allCandidates))
			for _, p := range allCandidates {
				if raced[p.Name()] {
					continue
				}
				if err := fn(ctx, p); err == nil {
					r.health.RecordSuccess(p.Name())
					return nil
				} else {
					r.health.RecordFailure(p.Name(), err)
				}
			}
			return fmt.Errorf("all providers failed for %s/%s", cap, market)
		}
		return r.FetchWithFallback(ctx, cap, market, fn)
	default:
		return r.FetchWithFallback(ctx, cap, market, fn)
	}
}

// FetchWithFallback 按优先级串行尝试候选数据源，失败则降级到下一个。
func (r *ProviderRouter) FetchWithFallback(ctx context.Context, cap Capability, market Market, fn func(ctx context.Context, p Provider) error) error {
	candidates := r.resolveProviders(cap, market, true)
	if len(candidates) == 0 {
		candidates = r.resolveProviders(cap, market, false)
	}
	if len(candidates) == 0 {
		return fmt.Errorf("no provider available for %s/%s", cap, market)
	}
	var lastErr error
	for _, p := range candidates {
		start := time.Now()
		err := fn(ctx, p)
		elapsed := time.Since(start)
		r.health.RecordLatency(p.Name(), elapsed)
		if err == nil {
			r.health.RecordSuccess(p.Name())
			return nil
		}
		r.health.RecordFailure(p.Name(), err)
		r.logger.Warn("provider failed, trying next", "provider", p.Name(), "capability", string(cap), "market", string(market), "error", err)
		lastErr = err
	}
	return fmt.Errorf("all providers failed for %s/%s: %w", cap, market, lastErr)
}

// FetchWithRace 同时请求前 raceCount 个数据源，取最快返回的成功结果。
func (r *ProviderRouter) FetchWithRace(ctx context.Context, cap Capability, market Market, fn func(ctx context.Context, p Provider) error, raceCount int) error {
	candidates := r.resolveProviders(cap, market, true)
	if len(candidates) == 0 {
		candidates = r.resolveProviders(cap, market, false)
	}
	if len(candidates) == 0 {
		return fmt.Errorf("no provider available for %s/%s", cap, market)
	}
	if len(candidates) > raceCount {
		candidates = candidates[:raceCount]
	}
	type result struct {
		provider Provider
		err      error
	}
	ch := make(chan result, len(candidates))
	raceCtx, cancel := context.WithTimeout(ctx, r.config.RaceTimeout)
	defer cancel()
	for _, p := range candidates {
		go func(provider Provider) {
			start := time.Now()
			err := fn(raceCtx, provider)
			elapsed := time.Since(start)
			r.health.RecordLatency(provider.Name(), elapsed)
			if err == nil {
				r.health.RecordSuccess(provider.Name())
			} else {
				r.health.RecordFailure(provider.Name(), err)
			}
			ch <- result{provider: provider, err: err}
		}(p)
	}
	var lastErr error
	for i := 0; i < len(candidates); i++ {
		select {
		case res := <-ch:
			if res.err == nil {
				return nil
			}
			lastErr = res.err
		case <-raceCtx.Done():
			return fmt.Errorf("race timeout for %s/%s: %w", cap, market, lastErr)
		}
	}
	return fmt.Errorf("all racing providers failed for %s/%s: %w", cap, market, lastErr)
}
