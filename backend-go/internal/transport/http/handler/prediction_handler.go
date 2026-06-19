package handler

import (
	"net/http"
	"stock-predict-go/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// predictionGoneMessage 是预测接口废弃后返回的提示信息。
const predictionGoneMessage = "预测服务已迁移，此接口已废弃"

// PredictFund 返回 410 Gone，表示基金预测接口已废弃并迁移至独立服务。
func (h *Handler) PredictFund(c *gin.Context) {
	if !isSixDigitCode(c.Param("fundCode")) {
		response.WriteError(c, http.StatusBadRequest, -1, "基金代码必须为6位数字")
		return
	}
	response.WriteError(c, http.StatusGone, -1, predictionGoneMessage)
}

// PredictStock 返回 410 Gone，表示股票预测接口已废弃并迁移至独立服务。
func (h *Handler) PredictStock(c *gin.Context) {
	if !isSixDigitCode(c.Param("stockCode")) {
		response.WriteError(c, http.StatusBadRequest, -1, "股票代码必须为6位数字")
		return
	}
	response.WriteError(c, http.StatusGone, -1, predictionGoneMessage)
}
