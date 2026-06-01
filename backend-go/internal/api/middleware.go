package api

import (
	"compress/gzip"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"log/slog"
	"net/http"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"stock-predict-go/internal/config"
)

const (
	rateLimitPerMinute = 60
	csrfCookieMaxAge   = 86400
	maxBodyBytes       = 1 << 20
)

type rateLimitEntry struct {
	count    int
	expireAt time.Time
}

func rateLimiter(logger *slog.Logger, stopCh chan struct{}) gin.HandlerFunc {
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
			writeError(c, http.StatusTooManyRequests, -1, "请求过于频繁，请稍后再试")
			c.Abort()
			return
		}
		mu.Unlock()
		c.Next()
	}
}

func maxBody() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)
		c.Next()
	}
}

func cors(cfg config.Config, logger *slog.Logger) gin.HandlerFunc {
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
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func securityHeaders(cfg config.Config) gin.HandlerFunc {
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

func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := generateRequestID()
		c.Set("request_id", id)
		c.Writer.Header().Set("X-Request-ID", id)
		c.Next()
	}
}

func generateRequestID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func requestLogger(logger *slog.Logger) gin.HandlerFunc {
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

func recoverer(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if value := recover(); value != nil {
				logger.Error("panic recovered", "panic", value, "stack", string(debug.Stack()))
				writeError(c, http.StatusInternalServerError, -1, "服务器内部错误")
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

func (w *gzipResponseWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func (w *gzipResponseWriter) Close() error {
	if w.gw == nil {
		return nil
	}
	return w.gw.Close()
}

func csrfProtection(cfg config.Config, stopCh chan struct{}) gin.HandlerFunc {
	if cfg.Env == "test" {
		return func(c *gin.Context) { c.Next() }
	}
	type csrfEntry struct {
		token    string
		expireAt time.Time
	}
	var mu sync.Mutex
	entries := make(map[string]*csrfEntry)

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				mu.Lock()
				now := time.Now()
				for k, e := range entries {
					if now.After(e.expireAt) {
						delete(entries, k)
					}
				}
				mu.Unlock()
			case <-stopCh:
				return
			}
		}
	}()

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
				c.SetCookie(cookieKey, existingToken, csrfCookieMaxAge, "/", "", !cfg.IsDevelopment(), true)
				mu.Lock()
				entries[existingToken] = &csrfEntry{
					token:    existingToken,
					expireAt: time.Now().Add(24 * time.Hour),
				}
				mu.Unlock()
			} else {
				mu.Lock()
				if _, ok := entries[existingToken]; !ok {
					entries[existingToken] = &csrfEntry{
						token:    existingToken,
						expireAt: time.Now().Add(24 * time.Hour),
					}
				}
				mu.Unlock()
			}

			c.Header("X-CSRF-Token", existingToken)
			c.Next()
			return
		}

		cookieToken, _ := c.Cookie("csrf_token")
		headerToken := c.GetHeader("X-CSRF-Token")

		if cookieToken == "" || headerToken == "" || cookieToken != headerToken {
			writeError(c, http.StatusForbidden, -1, "CSRF token 验证失败")
			c.Abort()
			return
		}

		mu.Lock()
		entry, ok := entries[cookieToken]
		mu.Unlock()
		if !ok || time.Now().After(entry.expireAt) {
			writeError(c, http.StatusForbidden, -1, "CSRF token 已过期")
			c.Abort()
			return
		}

		c.Next()
	}
}

func (r *Router) requireAdminToken(c *gin.Context) {
	token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	if token == "" {
		writeError(c, http.StatusUnauthorized, -1, "未提供管理员令牌")
		c.Abort()
		return
	}
	if r.cfg.Env == "development" && subtle.ConstantTimeCompare([]byte(token), []byte("dev-admin-token")) == 1 {
		c.Next()
		return
	}
	if r.cfg.AdminToken == "" {
		writeError(c, http.StatusUnauthorized, -1, "管理员令牌无效")
		c.Abort()
		return
	}
	if subtle.ConstantTimeCompare([]byte(token), []byte(r.cfg.AdminToken)) == 1 {
		c.Next()
		return
	}
	writeError(c, http.StatusUnauthorized, -1, "管理员令牌无效")
	c.Abort()
}

func gzipMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}
		writer := &gzipResponseWriter{ResponseWriter: c.Writer}
		c.Writer = writer
		c.Next()
		if err := writer.Close(); err != nil && !c.Writer.Written() {
			writeError(c, http.StatusInternalServerError, -1, "服务器内部错误")
		}
	}
}
