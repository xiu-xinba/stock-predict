package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"stock-predict-go/internal/dto"
	"stock-predict-go/internal/service"
)

func (r *Router) searchStocks(c *gin.Context) {
	var query dto.StockSearchRequest
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, -1, "查询参数格式错误")
		return
	}
	if query.Size > 100 {
		query.Size = 100
	}
	writeSuccess(c, r.services.Stocks.Search(c.Request.Context(), query))
}

func (r *Router) stockFilters(c *gin.Context) {
	writeSuccess(c, r.services.Stocks.Filters())
}

func (r *Router) stockDetail(c *gin.Context) {
	stockCode := c.Param("stockCode")
	if !isSixDigitCode(stockCode) {
		writeJSON(c, http.StatusOK, dto.APIResponse{Code: -1, Message: "股票代码必须为6位数字", Data: nil})
		return
	}
	result, err := r.services.StockDetail.GetDetail(c.Request.Context(), stockCode)
	if errors.Is(err, service.ErrInvalidStockCode) {
		writeJSON(c, http.StatusOK, dto.APIResponse{Code: -1, Message: "股票代码必须为6位数字", Data: nil})
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, -1, "服务器内部错误")
		return
	}
	writeSuccess(c, result)
}

func (r *Router) predictStock(c *gin.Context) {
	stockCode := c.Param("stockCode")
	if !isSixDigitCode(stockCode) {
		writeJSON(c, http.StatusOK, dto.APIResponse{Code: -1, Message: "股票代码必须为6位数字", Data: nil})
		return
	}
	result, err := r.services.Prediction.PredictStock(stockCode)
	if errors.Is(err, service.ErrInvalidStockCode) {
		writeJSON(c, http.StatusOK, dto.APIResponse{Code: -1, Message: "股票代码必须为6位数字", Data: nil})
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, -1, "服务器内部错误")
		return
	}
	writeSuccess(c, result)
}

func (r *Router) stockQuotes(c *gin.Context) {
	var payload dto.StockQuoteRequest
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
		writeSuccess(c, map[string]dto.StockQuote{})
		return
	}
	if len(payload.Codes) > 50 {
		writeError(c, http.StatusBadRequest, -1, "最多支持50个股票代码")
		return
	}
	for _, code := range payload.Codes {
		if !isSixDigitCode(code) {
			writeError(c, http.StatusBadRequest, -1, "股票代码必须为6位数字")
			return
		}
	}
	writeSuccess(c, r.services.StockQuote.FetchQuotes(c.Request.Context(), payload.Codes))
}

func (r *Router) stockRanking(c *gin.Context) {
	rankingType := c.Param("type")
	if rankingType != "gainers" && rankingType != "losers" && rankingType != "volume" {
		writeJSON(c, http.StatusOK, dto.APIResponse{Code: -1, Message: "type 必须为 gainers、losers 或 volume", Data: nil})
		return
	}
	size := 10
	if s := c.Query("size"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			size = v
		}
	}
	items, err := r.services.Stocks.Ranking(c.Request.Context(), rankingType, size)
	if errors.Is(err, service.ErrInvalidRankingType) {
		writeJSON(c, http.StatusOK, dto.APIResponse{Code: -1, Message: "type 必须为 gainers、losers 或 volume", Data: nil})
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, -1, "服务器内部错误")
		return
	}
	writeSuccess(c, items)
}

func (r *Router) syncStocks(c *gin.Context) {
	result, err := r.services.Stocks.SyncStocks(c.Request.Context())
	if err != nil {
		r.logger.Warn("stock sync failed", "error", err)
		writeError(c, http.StatusInternalServerError, -1, "股票同步失败")
		return
	}
	if r.searchIdx != nil {
		if err := r.searchIdx.SyncStocks(r.services.Stocks.ListStocks()); err != nil {
			r.logger.Warn("failed to update search index after stock sync", "error", err)
		} else {
			count, _ := r.searchIdx.StockCount()
			r.logger.Info("search index updated after stock sync", "count", count)
		}
	}
	writeSuccess(c, dto.StockSyncResult{
		Total:    result.Total,
		Imported: result.Imported,
		Errors:   result.Errors,
	})
}
