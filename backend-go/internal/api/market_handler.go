package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"stock-predict-go/internal/dto"
	"stock-predict-go/internal/service"
)

func (r *Router) marketIndices(c *gin.Context) {
	writeSuccess(c, r.services.Market.Indices())
}

func (r *Router) marketRanking(c *gin.Context) {
	var path dto.MarketRankingPath
	var query dto.MarketRankingQuery
	if err := c.ShouldBindUri(&path); err != nil {
		writeError(c, http.StatusBadRequest, -1, "路径参数格式错误")
		return
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, -1, "查询参数格式错误")
		return
	}
	items, err := r.services.Funds.Ranking(path.Type, query.Size)
	if errors.Is(err, service.ErrInvalidRankingType) {
		writeJSON(c, http.StatusOK, dto.APIResponse{Code: -1, Message: "type 必须为 gainers 或 losers", Data: nil})
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, -1, "服务器内部错误")
		return
	}
	writeSuccess(c, items)
}
