package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"stock-predict-go/internal/transport/http/response"

	"github.com/gin-gonic/gin"

	funddomain "stock-predict-go/internal/domain/fund"
	providers "stock-predict-go/internal/infrastructure/providers"
)

// SearchFunds 处理基金搜索请求，根据查询参数返回匹配的基金列表。
func (h *Handler) SearchFunds(c *gin.Context) {
	var query funddomain.FundSearchRequest
	if err := c.ShouldBindQuery(&query); err != nil {
		response.WriteError(c, http.StatusBadRequest, -1, "查询参数格式错误")
		return
	}
	result, err := h.services.Funds.Search(query)
	if err != nil {
		h.logger.Error("fund search failed", "error", err)
		response.WriteError(c, http.StatusInternalServerError, -1, "基金查询失败")
		return
	}
	response.WriteSuccess(c, result)
}

// FundFilters 返回基金筛选条件的可选值列表（基金类型等）。
func (h *Handler) FundFilters(c *gin.Context) {
	result, err := h.services.Funds.Filters()
	if err != nil {
		h.logger.Error("fund filters failed", "error", err)
		response.WriteError(c, http.StatusInternalServerError, -1, "基金筛选条件查询失败")
		return
	}
	response.WriteSuccess(c, result)
}

// FundCoverage 返回基金数据的覆盖率报告。
func (h *Handler) FundCoverage(c *gin.Context) {
	report := h.store.CoverageReport()
	response.WriteSuccess(c, report)
}

// SyncFunds 触发基金数据同步，从配置的外部数据源拉取基金列表和指标数据并更新本地存储。
func (h *Handler) SyncFunds(c *gin.Context) {
	result, err := h.services.Funds.SyncFromSources(h.cfg.FundUniverseURL, h.cfg.FundMetricsURL, h.cfg.FundSyncCSVPath)
	if errors.Is(err, providers.ErrSyncSourceRequired) {
		response.WriteError(c, http.StatusBadRequest, -1, "未配置基金同步来源，无法同步基金数据")
		return
	}
	if errors.Is(err, providers.ErrSyncUnsupported) {
		response.WriteError(c, http.StatusInternalServerError, -1, "同步功能暂不可用")
		return
	}
	if err != nil {
		h.logger.Warn("fund sync failed", "error", err)
		response.WriteError(c, http.StatusInternalServerError, -1, "基金同步失败")
		return
	}
	response.WriteSuccess(c, result)
}

// WatchlistQuotes 根据请求体中的基金代码列表批量获取自选基金的实时报价，最多支持 50 个代码。
func (h *Handler) WatchlistQuotes(c *gin.Context) {
	var payload funddomain.WatchlistQuoteRequest
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		response.WriteError(c, http.StatusBadRequest, -1, "请求体格式错误")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		response.WriteError(c, http.StatusBadRequest, -1, "请求体格式错误")
		return
	}
	if len(payload.Codes) == 0 {
		response.WriteSuccess(c, []funddomain.WatchlistItem{})
		return
	}
	if len(payload.Codes) > providers.MaxWatchlistBatch {
		response.WriteError(c, http.StatusBadRequest, -1, "最多支持50个基金代码")
		return
	}
	for _, code := range payload.Codes {
		if !isSixDigitCode(code) {
			response.WriteError(c, http.StatusBadRequest, -1, "基金代码必须为6位数字")
			return
		}
	}
	response.WriteSuccess(c, h.services.Watchlist.Quotes(payload.Codes))
}

// FundDetail 根据基金代码返回该基金的详细信息。
func (h *Handler) FundDetail(c *gin.Context) {
	var path funddomain.FundDetailPath
	if err := c.ShouldBindUri(&path); err != nil {
		response.WriteError(c, http.StatusBadRequest, -1, "路径参数格式错误")
		return
	}
	result, err := h.services.Detail.GetDetail(c.Request.Context(), path.FundCode)
	if errors.Is(err, providers.ErrInvalidFundCode) {
		response.WriteError(c, http.StatusBadRequest, -1, "基金代码必须为6位数字")
		return
	}
	if errors.Is(err, providers.ErrFundNotFound) {
		response.WriteError(c, http.StatusNotFound, -1, "未找到该基金")
		return
	}
	if err != nil {
		response.WriteError(c, http.StatusInternalServerError, -1, "服务器内部错误")
		return
	}
	response.WriteSuccess(c, result)
}
