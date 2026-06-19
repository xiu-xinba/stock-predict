package handler

import (
	"errors"
	"net/http"
	"stock-predict-go/internal/infrastructure/database"
	"stock-predict-go/internal/transport/http/response"
	"strconv"

	"github.com/gin-gonic/gin"

	marketdomain "stock-predict-go/internal/domain/market"
	providers "stock-predict-go/internal/infrastructure/providers"
)

// MarketIndices 返回主要市场指数的实时行情数据。
func (h *Handler) MarketIndices(c *gin.Context) {
	items, err := h.services.Market.Indices(c.Request.Context())
	if errors.Is(err, providers.ErrMarketUnavailable) {
		response.WriteError(c, http.StatusServiceUnavailable, -1, "行情数据暂不可用")
		return
	}
	if err != nil {
		response.WriteError(c, http.StatusInternalServerError, -1, "服务器内部错误")
		return
	}
	response.WriteSuccess(c, items)
}

// MarketRanking 返回指定类型的市场排行榜数据。
func (h *Handler) MarketRanking(c *gin.Context) {
	var path marketdomain.MarketRankingPath
	var query marketdomain.MarketRankingQuery
	if err := c.ShouldBindUri(&path); err != nil {
		response.WriteError(c, http.StatusBadRequest, -1, "路径参数格式错误")
		return
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		response.WriteError(c, http.StatusBadRequest, -1, "查询参数格式错误")
		return
	}
	items, err := h.services.Funds.Ranking(path.Type, query.Size)
	if errors.Is(err, providers.ErrInvalidRankingType) {
		response.WriteError(c, http.StatusBadRequest, -1, "无效的排名类型")
		return
	}
	if err != nil {
		response.WriteError(c, http.StatusInternalServerError, -1, "服务器内部错误")
		return
	}
	response.WriteSuccess(c, items)
}

// IndexKline 返回指定指数的 K 线（蜡烛图）历史数据，支持通过 count 参数控制返回条数（默认 120，最大 500）。
func (h *Handler) IndexKline(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		response.WriteError(c, http.StatusBadRequest, -1, "缺少指数代码")
		return
	}
	count := 120
	if v := c.Query("count"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			count = n
		}
	}
	items, err := h.services.Market.IndexKline(c.Request.Context(), code, count)
	if errors.Is(err, providers.ErrMarketUnavailable) {
		response.WriteError(c, http.StatusServiceUnavailable, -1, "指数历史数据暂不可用")
		return
	}
	if err != nil {
		response.WriteError(c, http.StatusInternalServerError, -1, "服务器内部错误")
		return
	}
	response.WriteSuccess(c, items)
}

// IndexMinute 返回指定指数的当日分时行情数据。
func (h *Handler) IndexMinute(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		response.WriteError(c, http.StatusBadRequest, -1, "缺少指数代码")
		return
	}
	items, err := h.services.Market.IndexMinute(c.Request.Context(), code)
	if errors.Is(err, providers.ErrMarketUnavailable) {
		response.WriteError(c, http.StatusServiceUnavailable, -1, "指数分时数据暂不可用")
		return
	}
	if err != nil {
		response.WriteError(c, http.StatusInternalServerError, -1, "服务器内部错误")
		return
	}
	response.WriteSuccess(c, items)
}

// SectorRanking 返回板块涨跌排名数据。
func (h *Handler) SectorRanking(c *gin.Context) {
	items := h.services.Market.SectorRanking(c.Request.Context())
	response.WriteSuccess(c, items)
}

// NorthboundFlow 返回北向资金流向数据。
func (h *Handler) NorthboundFlow(c *gin.Context) {
	flow := h.services.Market.NorthboundFlow(c.Request.Context())
	response.WriteSuccess(c, flow)
}

// HSGTHist 返回沪深港通历史资金流向数据。
func (h *Handler) HSGTHist(c *gin.Context) {
	if h.services.HSGTScheduler == nil {
		response.WriteError(c, http.StatusServiceUnavailable, -1, "HSGT 服务未初始化")
		return
	}

	startDate := c.DefaultQuery("start_date", "")
	endDate := c.DefaultQuery("end_date", "")

	store := h.services.HSGTStore
	if store == nil {
		response.WriteError(c, http.StatusServiceUnavailable, -1, "HSGT 数据库未初始化")
		return
	}

	var flows []database.HSGTFlowDaily
	var err error

	if startDate != "" && endDate != "" {
		result, e := store.ListRange(startDate, endDate)
		if e != nil {
			err = e
		} else {
			flows = result
		}
	} else {
		days := 30
		if d := c.Query("days"); d != "" {
			if n, e := strconv.Atoi(d); e == nil && n > 0 && n <= 730 {
				days = n
			}
		}
		result, e := store.ListRecent(days)
		if e != nil {
			err = e
		} else {
			// ListRecent 返回倒序（最新在前），前端图表需要升序（最旧在前）
			flows = result
			for i, j := 0, len(flows)-1; i < j; i, j = i+1, j-1 {
				flows[i], flows[j] = flows[j], flows[i]
			}
		}
	}

	if err != nil {
		response.WriteError(c, http.StatusInternalServerError, -1, "查询HSGT数据失败")
		return
	}

	response.WriteSuccess(c, flows)
}

// HSGTSync 手动触发沪深港通历史数据同步。
func (h *Handler) HSGTSync(c *gin.Context) {
	if h.services.HSGTScheduler == nil {
		response.WriteError(c, http.StatusServiceUnavailable, -1, "HSGT 服务未初始化")
		return
	}

	ctx := c.Request.Context()
	if err := h.services.HSGTScheduler.SyncHistoricalData(ctx); err != nil {
		response.WriteError(c, http.StatusInternalServerError, -1, "同步HSGT数据失败: "+err.Error())
		return
	}

	stats, _ := h.services.HSGTScheduler.GetStats()
	response.WriteSuccess(c, stats)
}

// SyncStatus 返回市场数据同步的当前状态。
func (h *Handler) SyncStatus(c *gin.Context) {
	status := h.services.MarketSync.Status()
	response.WriteSuccess(c, status)
}

// MarketHealth 返回市场数据源的健康状态及缓存统计信息。
func (h *Handler) MarketHealth(c *gin.Context) {
	if h.services.Health == nil {
		response.WriteSuccess(c, nil)
		return
	}
	result := map[string]any{
		"sources": h.services.Health.GetPublicStatus(),
	}
	// Add cache stats if available
	if h.services.Cache != nil {
		result["cache_stats"] = h.services.Cache.GetCacheStats()
	}
	response.WriteSuccess(c, result)
}

// SimulateHealth 模拟指定数据源的健康状态，用于测试和调试。
func (h *Handler) SimulateHealth(c *gin.Context) {
	if h.services.Health == nil {
		response.WriteError(c, http.StatusServiceUnavailable, -1, "健康监控未启用")
		return
	}
	source := c.Param("source")
	status := c.Query("status")
	if source == "" || status == "" {
		response.WriteError(c, http.StatusBadRequest, -1, "缺少 source 或 status 参数")
		return
	}
	errMsg := c.Query("error")
	h.services.Health.SimulateStatus(source, status, errMsg)
	response.WriteSuccess(c, h.services.Health.GetAllStatus())
}

// ResetHealth 重置所有数据源的健康状态模拟，恢复为真实状态。
func (h *Handler) ResetHealth(c *gin.Context) {
	if h.services.Health == nil {
		response.WriteError(c, http.StatusServiceUnavailable, -1, "健康监控未启用")
		return
	}
	h.services.Health.ResetSimulation()
	response.WriteSuccess(c, h.services.Health.GetAllStatus())
}
