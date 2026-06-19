package handler

import (
	"net/http"
	"stock-predict-go/internal/transport/http/response"

	"github.com/gin-gonic/gin"

	searchdomain "stock-predict-go/internal/domain/search"
	providers "stock-predict-go/internal/infrastructure/providers"
)

// UnifiedSearch 处理统一搜索请求，支持跨股票和基金的联合搜索。
func (h *Handler) UnifiedSearch(c *gin.Context) {
	var query searchdomain.UnifiedSearchRequest
	if err := c.ShouldBindQuery(&query); err != nil {
		response.WriteError(c, http.StatusBadRequest, -1, "查询参数格式错误")
		return
	}
	if query.Query == "" {
		response.WriteError(c, http.StatusBadRequest, -1, "搜索关键词不能为空")
		return
	}
	if len(query.Query) > providers.MaxSearchKeywordLen {
		response.WriteError(c, http.StatusBadRequest, -1, "搜索关键词过长")
		return
	}
	result, err := h.services.Search.Search(query)
	if err != nil {
		h.logger.Error("unified search failed", "error", err)
		response.WriteError(c, http.StatusInternalServerError, -1, "搜索失败")
		return
	}
	response.WriteSuccess(c, result)
}
