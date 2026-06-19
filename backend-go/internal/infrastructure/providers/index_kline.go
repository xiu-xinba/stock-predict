package providers

import (
	"context"
	"fmt"

	marketdomain "stock-predict-go/internal/domain/market"
)

// FetchIndexKline 获取指数日K线数据。优先通过路由器选择数据源，否则走遗留逻辑。
func (c *IndexQuoteClient) FetchIndexKline(ctx context.Context, code string, count int) []marketdomain.IndexKlinePoint {
	if c.router != nil {
		return c.fetchIndexKlineViaRouter(ctx, code, count)
	}
	return c.fetchIndexKlineLegacy(ctx, code, count)
}

// fetchIndexKlineViaRouter 通过路由器选择数据源获取指数K线数据。
// 依次检查内存缓存、SQLite 缓存，均未命中则通过路由器调度 Provider 获取。
func (c *IndexQuoteClient) fetchIndexKlineViaRouter(ctx context.Context, code string, count int) []marketdomain.IndexKlinePoint {
	if count <= 0 {
		count = 120
	}
	cacheKey := fmt.Sprintf("index_kline:%s:%d", code, count)
	if cached, ok := c.klineCache.Get(cacheKey); ok {
		if val, ok2 := cached.([]marketdomain.IndexKlinePoint); ok2 {
			return val
		}
	}

	// 优先从 SQLite 缓存加载（快速路径）
	if c.marketStore != nil {
		if cached := c.marketStore.LoadIndexKline(code, count); len(cached) > 0 {
			c.klineCache.Set(cacheKey, cached)
			return cached
		}
	}

	market := detectIndexMarket(code)
	var result []marketdomain.IndexKlinePoint
	err := c.router.Fetch(ctx, CapIndexKline, market, func(ctx context.Context, p Provider) error {
		provider, ok := p.(IndexKlineProvider)
		if !ok {
			return fmt.Errorf("provider %s does not implement IndexKlineProvider", p.Name())
		}
		points, err := provider.FetchIndexKline(ctx, code, market, count)
		if err != nil {
			return err
		}
		result = points
		return nil
	})
	_ = err

	if len(result) > 0 {
		c.klineCache.Set(cacheKey, result)
	}
	return result
}

// fetchIndexKlineLegacy 使用遗留逻辑获取指数K线数据（无路由器时的降级路径）。
// A 股指数优先 TDX，失败后降级到腾讯；港股/美股指数使用腾讯接口。
func (c *IndexQuoteClient) fetchIndexKlineLegacy(ctx context.Context, code string, count int) []marketdomain.IndexKlinePoint {
	if count <= 0 {
		count = 120
	}
	cacheKey := fmt.Sprintf("index_kline:%s:%d", code, count)
	if cached, ok := c.klineCache.Get(cacheKey); ok {
		if val, ok2 := cached.([]marketdomain.IndexKlinePoint); ok2 {
			return val
		}
	}

	// 优先从 SQLite 缓存加载（快速路径），避免 TDX/腾讯超时导致前端请求失败
	if c.marketStore != nil {
		if cached := c.marketStore.LoadIndexKline(code, count); len(cached) > 0 {
			c.klineCache.Set(cacheKey, cached)
			return cached
		}
	}

	if isCNIndex(code) {
		points := c.fetchCNIndexKlineTDX(ctx, code, count)
		if len(points) > 0 {
			if c.health != nil {
				c.health.RecordSuccess("tdx")
			}
			c.klineCache.Set(cacheKey, points)
			return points
		}
		if c.health != nil {
			c.health.RecordFailure("tdx", fmt.Errorf("tdx kline failed for %s", code))
		}
		fallback := c.fetchCNIndexKlineTencent(ctx, code, count)
		if len(fallback) > 0 {
			if c.health != nil {
				c.health.RecordSuccess("tencent")
			}
			c.klineCache.Set(cacheKey, fallback)
			return fallback
		}
		if c.health != nil {
			c.health.RecordFailure("tencent", fmt.Errorf("tencent kline failed for %s", code))
		}
	} else {
		points := c.fetchHKUSIndexKlineTencent(ctx, code, count)
		if len(points) > 0 {
			if c.health != nil {
				c.health.RecordSuccess("tencent")
			}
			c.klineCache.Set(cacheKey, points)
			return points
		}
		if c.health != nil {
			c.health.RecordFailure("tencent", fmt.Errorf("tencent HK/US kline failed for %s", code))
		}
	}

	return nil
}

// Close 释放 IndexQuoteClient 持有的资源。当前为空操作，预留扩展。
func (c *IndexQuoteClient) Close() {}
