package api

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"stock-predict-go/internal/config"
)

func maxBody() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
		c.Next()
	}
}

func cors(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && (slices.Contains(cfg.CORSOrigins, "*") || slices.Contains(cfg.CORSOrigins, origin)) {
			headers := c.Writer.Header()
			headers.Set("Access-Control-Allow-Origin", origin)
			headers.Set("Vary", "Origin")
			headers.Set("Access-Control-Allow-Headers", "Content-Type, X-Admin-Token, X-CSRFToken")
			headers.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			headers.Set("Access-Control-Allow-Credentials", "true")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		headers := c.Writer.Header()
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("X-Frame-Options", "DENY")
		headers.Set("Referrer-Policy", "no-referrer")
		headers.Set("Cache-Control", "no-store")
		c.Next()
	}
}

func requestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		logger.Info("request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"remote", scrubRemote(c.Request.RemoteAddr),
		)
	}
}

func recoverer(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if value := recover(); value != nil {
				logger.Error("panic recovered", "panic", value, "stack", string(debug.Stack()))
				writeError(c, http.StatusInternalServerError, -1, "服务器繁忙，请稍后重试")
				c.Abort()
			}
		}()
		c.Next()
	}
}

func scrubRemote(remote string) string {
	if idx := strings.LastIndex(remote, ":"); idx > 0 {
		return remote[:idx]
	}
	return remote
}
