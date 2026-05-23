package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"stock-predict-go/internal/dto"
)

func writeSuccess(c *gin.Context, data any) {
	writeJSON(c, http.StatusOK, dto.APIResponse{Code: 0, Message: "success", Data: data})
}

func writeError(c *gin.Context, status int, code int, message string) {
	writeJSON(c, status, dto.APIResponse{Code: code, Message: message, Data: nil})
}

func writeJSON(c *gin.Context, status int, payload any) {
	c.JSON(status, payload)
}
