package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"stock-predict-go/internal/transport/http/response"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	stockdomain "stock-predict-go/internal/domain/stock"
	providers "stock-predict-go/internal/infrastructure/providers"
)

const stockSyncRequestTimeout = 120 * time.Second

// SearchStocks 处理股票搜索请求，根据查询参数返回匹配的股票列表，单次最多返回 100 条。
func (h *Handler) SearchStocks(c *gin.Context) {
	var query stockdomain.StockSearchRequest
	if err := c.ShouldBindQuery(&query); err != nil {
		response.WriteError(c, http.StatusBadRequest, -1, "查询参数格式错误")
		return
	}
	if query.Size > 100 {
		query.Size = 100
	}
	result, err := h.services.Stocks.Search(c.Request.Context(), query)
	if err != nil {
		h.logger.Error("stock search failed", "error", err)
		response.WriteError(c, http.StatusInternalServerError, -1, "股票查询失败")
		return
	}
	response.WriteSuccess(c, result)
}

// StockFilters 返回股票筛选条件的可选值列表（行业、板块等）。
func (h *Handler) StockFilters(c *gin.Context) {
	result, err := h.services.Stocks.Filters()
	if err != nil {
		h.logger.Error("stock filters failed", "error", err)
		response.WriteError(c, http.StatusInternalServerError, -1, "股票筛选条件查询失败")
		return
	}
	response.WriteSuccess(c, result)
}

// StockDetail 根据股票代码返回该股票的详细信息。
func (h *Handler) StockDetail(c *gin.Context) {
	stockCode := c.Param("stockCode")
	if !isSixDigitCode(stockCode) {
		response.WriteError(c, http.StatusBadRequest, -1, "股票代码必须为6位数字")
		return
	}
	result, err := h.services.StockDetail.GetDetail(c.Request.Context(), stockCode)
	if errors.Is(err, providers.ErrInvalidStockCode) {
		response.WriteError(c, http.StatusBadRequest, -1, "股票代码必须为6位数字")
		return
	}
	if errors.Is(err, providers.ErrStockNotFound) {
		response.WriteError(c, http.StatusNotFound, -1, "未找到该股票")
		return
	}
	if err != nil {
		response.WriteError(c, http.StatusInternalServerError, -1, "服务器内部错误")
		return
	}
	response.WriteSuccess(c, result)
}

// StockMinute 根据股票代码返回该股票的分时行情数据。
func (h *Handler) StockMinute(c *gin.Context) {
	stockCode := c.Param("stockCode")
	if stockCode == "" {
		response.WriteError(c, http.StatusBadRequest, -1, "缺少股票代码")
		return
	}
	items, err := h.services.Market.StockMinute(c.Request.Context(), stockCode)
	if errors.Is(err, providers.ErrMarketUnavailable) {
		response.WriteError(c, http.StatusServiceUnavailable, -1, "分时数据暂不可用")
		return
	}
	if err != nil {
		response.WriteError(c, http.StatusInternalServerError, -1, "服务器内部错误")
		return
	}
	response.WriteSuccess(c, items)
}

// StockKline 返回指定股票的 K 线数据，支持周期和复权类型参数。
// period: daily/weekly/monthly（默认 daily）; fq: 0=不复权, 1=前复权, 2=后复权（默认 1）
func (h *Handler) StockKline(c *gin.Context) {
	stockCode := c.Param("stockCode")
	if !isSixDigitCode(stockCode) {
		response.WriteError(c, http.StatusBadRequest, -1, "股票代码必须为6位数字")
		return
	}
	period := c.DefaultQuery("period", "daily")
	fqStr := c.DefaultQuery("fq", "1")
	fq, err := strconv.Atoi(fqStr)
	if err != nil || fq < 0 || fq > 2 {
		fq = 1
	}
	result, err := h.services.StockDetail.GetKline(c.Request.Context(), stockCode, period, fq)
	if errors.Is(err, providers.ErrInvalidStockCode) {
		response.WriteError(c, http.StatusBadRequest, -1, "股票代码必须为6位数字")
		return
	}
	if err != nil {
		response.WriteError(c, http.StatusInternalServerError, -1, "服务器内部错误")
		return
	}
	response.WriteSuccess(c, result)
}

// StockQuotes 批量获取多只股票的实时报价，支持 freshness 参数控制数据新鲜度。
func (h *Handler) StockQuotes(c *gin.Context) {
	var payload stockdomain.StockQuoteRequest
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
	freshness := providers.StockQuoteFreshnessBalanced
	if payload.Freshness != "" {
		switch providers.StockQuoteFreshness(payload.Freshness) {
		case providers.StockQuoteFreshnessBalanced, providers.StockQuoteFreshnessRealtime:
			freshness = providers.StockQuoteFreshness(payload.Freshness)
		default:
			response.WriteError(c, http.StatusBadRequest, -1, "无效的行情新鲜度参数")
			return
		}
	}
	if len(payload.Codes) == 0 {
		response.WriteSuccess(c, map[string]stockdomain.StockQuote{})
		return
	}
	if len(payload.Codes) > providers.MaxStockQuoteBatch {
		response.WriteError(c, http.StatusBadRequest, -1, "最多支持50个股票代码")
		return
	}
	for _, code := range payload.Codes {
		if !isSixDigitCode(code) {
			response.WriteError(c, http.StatusBadRequest, -1, "股票代码必须为6位数字")
			return
		}
	}
	response.WriteSuccess(c, h.services.StockQuote.FetchQuotesWithOptions(c.Request.Context(), payload.Codes, providers.StockQuoteOptions{Freshness: freshness}))
}

// StockRanking 返回指定类型的股票排行榜（涨幅、跌幅、成交量）。
func (h *Handler) StockRanking(c *gin.Context) {
	rankingType := c.Param("type")
	if rankingType != "gainers" && rankingType != "losers" && rankingType != "volume" {
		response.WriteError(c, http.StatusBadRequest, -1, "无效的排名类型")
		return
	}
	size := 10
	if s := c.Query("size"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			size = v
		}
	}
	items, err := h.services.Stocks.Ranking(c.Request.Context(), rankingType, size)
	if errors.Is(err, providers.ErrInvalidRankingType) {
		response.WriteError(c, http.StatusBadRequest, -1, "无效的排名类型")
		return
	}
	if errors.Is(err, providers.ErrMarketUnavailable) {
		response.WriteError(c, http.StatusServiceUnavailable, -1, "股票排行数据暂不可用")
		return
	}
	if err != nil {
		response.WriteError(c, http.StatusInternalServerError, -1, "服务器内部错误")
		return
	}
	response.WriteSuccess(c, items)
}

// SyncStocks 触发股票数据全量同步，同步完成后刷新搜索索引。
func (h *Handler) SyncStocks(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), stockSyncRequestTimeout)
	defer cancel()

	result, err := h.services.Stocks.SyncStocks(ctx)
	if err != nil {
		h.logger.Warn("stock sync failed", "error", err)
		response.WriteError(c, http.StatusInternalServerError, -1, "股票同步失败")
		return
	}
	if h.searchIdx != nil {
		if err := h.searchIdx.SyncStocks(h.services.Stocks.ListStocks()); err != nil {
			h.logger.Warn("failed to update search index after stock sync", "error", err)
		} else {
			count, _ := h.searchIdx.StockCount()
			h.logger.Info("search index updated after stock sync", "count", count)
		}
	}
	response.WriteSuccess(c, stockdomain.StockSyncResult{
		Total:    result.Total,
		Imported: result.Imported,
		Errors:   result.Errors,
	})
}
