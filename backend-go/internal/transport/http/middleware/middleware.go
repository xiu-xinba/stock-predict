// Package middleware 实现了 HTTP 中间件，提供限流、CORS、安全头、CSRF 防护、请求日志、压缩等横切关注点。
package middleware

import (
	"compress/gzip"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"log/slog"
	"net/http"
	"runtime/debug"
	"slices"
	"stock-predict-go/internal/transport/http/response"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"stock-predict-go/internal/platform/config"
)

const (
	// rateLimitPerMinute 是每个 IP 每分钟允许的最大请求数。
	rateLimitPerMinute = 120
	// csrfCookieMaxAge 是 CSRF Cookie 的最大存活时间（秒）。
	csrfCookieMaxAge = 86400
	// maxBodyBytes 是请求体的最大字节数限制（1 MB）。
	maxBodyBytes = 1 << 20
)

// rateLimitEntry 记录单个 IP 的请求计数与过期时间。
type rateLimitEntry struct {
	count    int
	expireAt time.Time
}

// RateLimiter 返回基于 IP 的请求限流中间件，每分钟超过 rateLimitPerMinute 次请求将返回 429。
// stopCh 用于优雅关闭时停止后台清理协程。
func RateLimiter(_ *slog.Logger, stopCh chan struct{}) gin.HandlerFunc {
	var mu sync.Mutex
	entries := make(map[string]*rateLimitEntry)

	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				mu.Lock()
				now := time.Now()
				for ip, e := range entries {
					if now.After(e.expireAt) {
						delete(entries, ip)
					}
				}
				mu.Unlock()
			case <-stopCh:
				return
			}
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()
		mu.Lock()
		now := time.Now()
		e, ok := entries[ip]
		if !ok || now.After(e.expireAt) {
			entries[ip] = &rateLimitEntry{
				count:    1,
				expireAt: now.Add(time.Minute),
			}
			mu.Unlock()
			c.Next()
			return
		}
		e.count++
		if e.count > rateLimitPerMinute {
			mu.Unlock()
			response.WriteError(c, http.StatusTooManyRequests, -1, "请求过于频繁，请稍后再试")
			c.Abort()
			return
		}
		mu.Unlock()
		c.Next()
	}
}

// MaxBody 返回请求体大小限制中间件，将请求体限制为 maxBodyBytes（1 MB），防止过大的请求体消耗服务器资源。
func MaxBody() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)
		c.Next()
	}
}

// CORS 返回跨域资源共享中间件，根据配置的允许来源列表设置响应头。
// 开发模式下未配置来源时默认允许 localhost:5173。
func CORS(cfg config.Config, logger *slog.Logger) gin.HandlerFunc {
	if slices.Contains(cfg.CORSOrigins, "*") && !cfg.IsDevelopment() {
		logger.Warn("CORS AllowOrigins 包含通配符 '*' 且非开发模式，存在安全风险，请配置明确的允许来源")
	}
	if len(cfg.CORSOrigins) == 0 && cfg.IsDevelopment() {
		logger.Warn("CORS_ORIGINS 未配置，开发模式默认允许 localhost:5173")
	}
	return func(c *gin.Context) {
		origins := cfg.CORSOrigins
		if len(origins) == 0 && cfg.IsDevelopment() {
			origins = []string{"http://localhost:5173"}
		}
		origin := c.GetHeader("Origin")
		if origin != "" && (slices.Contains(origins, "*") || slices.Contains(origins, origin)) {
			headers := c.Writer.Header()
			headers.Set("Access-Control-Allow-Origin", origin)
			headers.Set("Vary", "Origin")
			headers.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRFToken, X-CSRF-Token")
			headers.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			headers.Set("Access-Control-Allow-Credentials", "true")
			headers.Set("Access-Control-Expose-Headers", "X-CSRF-Token, X-Request-ID")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// SecurityHeaders 返回安全响应头中间件，设置 X-Content-Type-Options、CSP、HSTS 等安全相关头部。
// 生产环境额外启用 Strict-Transport-Security 和 Content-Security-Policy。
func SecurityHeaders(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		headers := c.Writer.Header()
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("X-Frame-Options", "DENY")
		headers.Set("Referrer-Policy", "no-referrer")
		headers.Set("Cache-Control", "no-store")
		headers.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		headers.Set("X-Permitted-Cross-Domain-Policies", "none")
		headers.Set("X-XSS-Protection", "0")
		if !cfg.IsDevelopment() {
			headers.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			headers.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:")
		}
		c.Next()
	}
}

// RequestID 返回请求 ID 中间件，为每个请求生成唯一的十六进制标识符，
// 并通过 Gin 上下文和 X-Request-ID 响应头传递，便于请求链路追踪。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := generateRequestID()
		c.Set("request_id", id)
		c.Writer.Header().Set("X-Request-ID", id)
		c.Next()
	}
}

// generateRequestID 使用加密随机数生成 16 位十六进制请求标识符。
func generateRequestID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// RequestLogger 返回请求日志中间件，记录每个请求的方法、路径、状态码和耗时。
func RequestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		reqID, _ := c.Get("request_id")
		logger.Info("request",
			"request_id", reqID,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"remote", scrubRemote(c.Request.RemoteAddr),
		)
	}
}

// Recoverer 返回 panic 恢复中间件，捕获处理器中的 panic 并返回 500 错误，防止服务崩溃。
func Recoverer(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if value := recover(); value != nil {
				logger.Error("panic recovered", "panic", value, "stack", string(debug.Stack()))
				response.WriteError(c, http.StatusInternalServerError, -1, "服务器内部错误")
				c.Abort()
			}
		}()
		c.Next()
	}
}

// scrubRemote 去除远程地址中的端口号，仅保留 IP 部分。
func scrubRemote(remote string) string {
	if idx := strings.LastIndex(remote, ":"); idx > 0 {
		return remote[:idx]
	}
	return remote
}

// gzipResponseWriter 封装 gin.ResponseWriter，在内容类型为 JSON 或文本时自动启用 gzip 压缩。
type gzipResponseWriter struct {
	gin.ResponseWriter
	gw      *gzip.Writer
	gzipped bool
}

func (w *gzipResponseWriter) Write(data []byte) (int, error) {
	if !w.gzipped {
		ct := w.Header().Get("Content-Type")
		ce := w.Header().Get("Content-Encoding")
		if ce != "" || !(strings.Contains(ct, "json") || strings.Contains(ct, "text")) {
			return w.ResponseWriter.Write(data)
		}
		w.Header().Del("Content-Length")
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")
		w.gw = gzip.NewWriter(w.ResponseWriter)
		w.gzipped = true
	}
	return w.gw.Write(data)
}

// WriteString 将字符串写入响应，内部调用 Write 方法以复用 gzip 逻辑。
func (w *gzipResponseWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

// Close 关闭 gzip 写入器，刷新剩余压缩数据。
func (w *gzipResponseWriter) Close() error {
	if w.gw == nil {
		return nil
	}
	return w.gw.Close()
}

// CSRFProtection 返回 CSRF 防护中间件，采用 Double Submit Cookie 模式：
// 对安全方法（GET/HEAD/OPTIONS）生成或复用 CSRF Token 并写入 Cookie 和响应头；
// 对状态变更方法（POST 等）校验 Cookie 与 X-CSRF-Token 请求头是否一致。
// 测试环境下直接放行所有请求。
func CSRFProtection(cfg config.Config) gin.HandlerFunc {
	if cfg.Env == "test" {
		return func(c *gin.Context) { c.Next() }
	}

	return func(c *gin.Context) {
		method := c.Request.Method

		if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
			b := make([]byte, 32)
			_, _ = rand.Read(b)
			token := hex.EncodeToString(b)

			cookieKey := "csrf_token"
			existingToken, _ := c.Cookie(cookieKey)
			if existingToken == "" {
				existingToken = token
				c.SetSameSite(http.SameSiteLaxMode)
				c.SetCookie(cookieKey, existingToken, csrfCookieMaxAge, "/", "", !cfg.IsDevelopment(), true)
			}

			c.Header("X-CSRF-Token", existingToken)
			c.Next()
			return
		}

		cookieToken, _ := c.Cookie("csrf_token")
		headerToken := c.GetHeader("X-CSRF-Token")

		if cookieToken == "" || headerToken == "" || cookieToken != headerToken {
			response.WriteError(c, http.StatusForbidden, -1, "CSRF token 验证失败")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAdminToken 返回管理员 Token 验证中间件，从 Authorization 请求头提取 Bearer Token，
// 使用恒定时间比较与配置的 AdminToken 进行校验，防止时序攻击。
func RequireAdminToken(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		if token == "" {
			response.WriteError(c, http.StatusUnauthorized, -1, "未提供管理员令牌")
			c.Abort()
			return
		}
		if cfg.AdminToken == "" {
			response.WriteError(c, http.StatusUnauthorized, -1, "管理员令牌无效")
			c.Abort()
			return
		}
		if subtle.ConstantTimeCompare([]byte(token), []byte(cfg.AdminToken)) == 1 {
			c.Next()
			return
		}
		response.WriteError(c, http.StatusUnauthorized, -1, "管理员令牌无效")
		c.Abort()
	}
}

// Gzip 返回 gzip 压缩中间件，对支持 gzip 的客户端自动压缩 JSON/文本响应。
func Gzip() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}
		writer := &gzipResponseWriter{ResponseWriter: c.Writer}
		c.Writer = writer
		c.Next()
		if err := writer.Close(); err != nil && !c.Writer.Written() {
			response.WriteError(c, http.StatusInternalServerError, -1, "服务器内部错误")
		}
	}
}
