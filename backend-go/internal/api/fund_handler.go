package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"stock-predict-go/internal/dto"
	"stock-predict-go/internal/service"
)

func (r *Router) searchFunds(c *gin.Context) {
	var query dto.FundSearchRequest
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, -1, "查询参数格式错误")
		return
	}
	writeSuccess(c, r.services.Funds.Search(query))
}

func (r *Router) fundFilters(c *gin.Context) {
	writeSuccess(c, r.services.Funds.Filters())
}

func (r *Router) fundCoverage(c *gin.Context) {
	report := r.store.CoverageReport()
	writeSuccess(c, report)
}

func (r *Router) syncFunds(c *gin.Context) {
	result, err := r.services.Funds.SyncFromSources(r.cfg.FundUniverseURL, r.cfg.FundMetricsURL, r.cfg.FundSyncCSVPath)
	if errors.Is(err, service.ErrSyncSourceRequired) {
		writeError(c, http.StatusBadRequest, -1, "未配置基金同步来源，无法同步基金数据")
		return
	}
	if errors.Is(err, service.ErrSyncUnsupported) {
		writeError(c, http.StatusInternalServerError, -1, "同步功能暂不可用")
		return
	}
	if err != nil {
		r.logger.Warn("fund sync failed", "error", err)
		writeError(c, http.StatusInternalServerError, -1, "基金同步失败")
		return
	}
	writeSuccess(c, result)
}

func (r *Router) predict(c *gin.Context) {
	fundCode := c.Param("fundCode")
	if !isSixDigitCode(fundCode) {
		writeError(c, http.StatusBadRequest, -1, "基金代码必须为6位数字")
		return
	}
	writeError(c, http.StatusNotImplemented, -2, "预测模型已拆分为独立项目，当前主项目仅保留入口。")
}

func (r *Router) watchlistQuotes(c *gin.Context) {
	var payload dto.WatchlistQuoteRequest
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		writeError(c, http.StatusBadRequest, -1, "请求体格式错误")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(c, http.StatusBadRequest, -1, "请求体格式错误")
		return
	}
	if len(payload.Codes) == 0 {
		writeSuccess(c, []dto.WatchlistItem{})
		return
	}
	if len(payload.Codes) > 50 {
		writeError(c, http.StatusBadRequest, -1, "最多支持50个基金代码")
		return
	}
	for _, code := range payload.Codes {
		if !isSixDigitCode(code) {
			writeError(c, http.StatusBadRequest, -1, "基金代码必须为6位数字")
			return
		}
	}
	writeSuccess(c, r.services.Watchlist.Quotes(payload.Codes))
}

func (r *Router) fundDetail(c *gin.Context) {
	var path dto.FundDetailPath
	if err := c.ShouldBindUri(&path); err != nil {
		writeError(c, http.StatusBadRequest, -1, "路径参数格式错误")
		return
	}
	result, err := r.services.Detail.GetDetail(c.Request.Context(), path.FundCode)
	if errors.Is(err, service.ErrInvalidFundCode) {
		writeError(c, http.StatusBadRequest, -1, "基金代码必须为6位数字")
		return
	}
	if errors.Is(err, service.ErrFundNotFound) {
		writeError(c, http.StatusNotFound, -1, "未找到该基金")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, -1, "服务器内部错误")
		return
	}
	writeSuccess(c, result)
}
