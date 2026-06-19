package router

import (
	"github.com/gin-gonic/gin"

	"stock-predict-go/internal/transport/http/response"
)

// metrics 返回 HTTP 请求指标的快照数据，需要管理员权限访问。
func (r *Router) metrics(c *gin.Context) {
	response.WriteSuccess(c, r.metricsCollector.Snapshot())
}
