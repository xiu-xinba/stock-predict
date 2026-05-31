package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"stock-predict-go/internal/dto"
)

func (r *Router) unifiedSearch(c *gin.Context) {
	var query dto.UnifiedSearchRequest
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, -1, "查询参数格式错误")
		return
	}
	if query.Query == "" {
		writeError(c, http.StatusBadRequest, -1, "搜索关键词不能为空")
		return
	}
	if len(query.Query) > 100 {
		writeError(c, http.StatusBadRequest, -1, "搜索关键词过长")
		return
	}
	result := r.services.Search.Search(query)
	writeSuccess(c, result)
}
