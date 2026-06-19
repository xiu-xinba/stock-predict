// Package response 实现了统一的 API 响应构造工具，所有 HTTP 响应均通过此包生成标准格式。
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIResponse 是所有 API 接口的统一响应结构，包含业务状态码、消息和数据载荷。
type APIResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// WriteSuccess 向客户端写入成功的 JSON 响应，业务状态码为 0。
func WriteSuccess(c *gin.Context, data any) {
	WriteJSON(c, http.StatusOK, APIResponse{Code: 0, Message: "success", Data: data})
}

// WriteError 向客户端写入错误的 JSON 响应，包含 HTTP 状态码、业务错误码和错误消息。
func WriteError(c *gin.Context, status int, code int, message string) {
	WriteJSON(c, status, APIResponse{Code: code, Message: message, Data: nil})
}

// WriteJSON 以指定 HTTP 状态码向客户端写入任意 JSON 载荷。
func WriteJSON(c *gin.Context, status int, payload any) {
	c.JSON(status, payload)
}
