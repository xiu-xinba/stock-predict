// Package router 实现了 HTTP 路由注册与引擎初始化，负责将请求路径映射到对应的处理器。
package router

import (
	"context"
	"log/slog"
	"net/http"
	"stock-predict-go/internal/transport/http/response"
	"time"

	"github.com/gin-gonic/gin"

	funddomain "stock-predict-go/internal/domain/fund"
	database "stock-predict-go/internal/infrastructure/database"
	providers "stock-predict-go/internal/infrastructure/providers"
	"stock-predict-go/internal/platform/config"
	"stock-predict-go/internal/platform/observability"
	httphandler "stock-predict-go/internal/transport/http/handler"
	"stock-predict-go/internal/transport/http/middleware"
)

// Router 是 HTTP 路由的核心结构，持有配置、服务注册表、Gin 引擎及指标收集器等依赖。
type Router struct {
	cfg              config.Config
	services         *providers.Registry
	store            funddomain.CoverageRepository
	searchIdx        *database.SearchStore
	logger           *slog.Logger
	stopCh           chan struct{}
	metricsCollector *observability.HTTPMetrics
	engine           *gin.Engine
}

// NewRouter 创建并初始化路由引擎，注册所有中间件和 API 路由，返回可用的 Router 实例。
func NewRouter(cfg config.Config, services *providers.Registry, fundRepo funddomain.CoverageRepository, logger *slog.Logger, searchIdx *database.SearchStore) *Router {
	metrics := observability.NewHTTPMetrics()
	router := &Router{cfg: cfg, services: services, store: fundRepo, searchIdx: searchIdx, logger: logger, stopCh: make(chan struct{}), metricsCollector: metrics}
	handlers := httphandler.New(cfg, services, fundRepo, searchIdx, logger)
	requireAdminToken := middleware.RequireAdminToken(cfg)

	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.RedirectTrailingSlash = false
	engine.RedirectFixedPath = false
	if err := engine.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		logger.Error("invalid trusted proxy configuration", "error", err)
		_ = engine.SetTrustedProxies(nil)
	}
	engine.Use(
		middleware.Recoverer(logger),
		middleware.RequestID(),
		middleware.RequestLogger(logger),
		middleware.SecurityHeaders(cfg),
		middleware.CORS(cfg, logger),
		middleware.CSRFProtection(cfg),
		middleware.Gzip(),
		middleware.MaxBody(),
		metrics.Middleware(),
		middleware.RateLimiter(logger, router.stopCh),
	)

	v1 := engine.Group("/api/v1")
	v1.GET("/health", router.readiness)
	v1.GET("/health/live", router.liveness)
	v1.GET("/health/ready", router.readiness)
	v1.GET("/metrics", requireAdminToken, router.metrics)
	v1.GET("/search", handlers.UnifiedSearch)
	v1.GET("/funds/search", handlers.SearchFunds)
	v1.GET("/funds/filters", handlers.FundFilters)
	v1.GET("/funds/coverage", handlers.FundCoverage)
	v1.POST("/funds/sync", requireAdminToken, handlers.SyncFunds)
	v1.GET("/predict/:fundCode", handlers.PredictFund)
	v1.GET("/market/indices", handlers.MarketIndices)
	v1.GET("/market/ranking/:type", handlers.MarketRanking)
	v1.GET("/market/index/:code/kline", handlers.IndexKline)
	v1.GET("/market/index/:code/minute", handlers.IndexMinute)
	v1.GET("/market/sectors", handlers.SectorRanking)
	v1.GET("/market/northbound", handlers.NorthboundFlow)
	v1.GET("/market/hsgt/hist", handlers.HSGTHist)
	v1.POST("/market/hsgt/sync", requireAdminToken, handlers.HSGTSync)
	v1.GET("/market/sync-status", handlers.SyncStatus)
	v1.GET("/market/health", handlers.MarketHealth)
	v1.POST("/market/health/:source/simulate", requireAdminToken, handlers.SimulateHealth)
	v1.POST("/market/health/reset", requireAdminToken, handlers.ResetHealth)
	v1.POST("/watchlist/quotes", handlers.WatchlistQuotes)
	v1.GET("/fund/:fundCode/detail", handlers.FundDetail)
	v1.GET("/stocks/search", handlers.SearchStocks)
	v1.GET("/stocks/filters", handlers.StockFilters)
	v1.GET("/stock/:stockCode/detail", handlers.StockDetail)
	v1.GET("/stock/:stockCode/minute", handlers.StockMinute)
	v1.GET("/stock/:stockCode/kline", handlers.StockKline)
	v1.GET("/stock/:stockCode/predict", handlers.PredictStock)
	v1.POST("/stocks/quotes", handlers.StockQuotes)
	v1.GET("/market/stock-ranking/:type", handlers.StockRanking)
	v1.POST("/stocks/sync", requireAdminToken, handlers.SyncStocks)
	v1.POST("/admin/restart", requireAdminToken, handlers.RestartBackend)
	
	// HSGT 北向南向资金流向 API
	v1.GET("/hsgt/latest", handlers.GetHSGTLatest)
	v1.GET("/hsgt/recent", handlers.GetHSGTRecent)
	v1.GET("/hsgt/range", handlers.GetHSGTRange)
	v1.GET("/hsgt/date/:date", handlers.GetHSGTByDate)
	v1.GET("/hsgt/stats", handlers.GetHSGTStatistics)

	engine.NoRoute(func(c *gin.Context) {
		response.WriteError(c, http.StatusNotFound, -1, "not found")
	})

	router.engine = engine
	return router
}

// liveness 返回存活探针响应，表示服务进程正在运行。
func (r *Router) liveness(c *gin.Context) {
	response.WriteSuccess(c, map[string]any{
		"status":  "ok",
		"runtime": "go",
	})
}

func (r *Router) readiness(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	sqlDB, err := r.services.DB.DB()
	if err != nil || sqlDB.PingContext(ctx) != nil {
		response.WriteError(c, http.StatusServiceUnavailable, -1, "数据库未就绪")
		return
	}
	response.WriteSuccess(c, map[string]any{
		"status":        "ok",
		"model_loaded":  false,
		"funds_loaded":  r.services.Funds.Count() > 0,
		"stocks_loaded": r.services.Stocks.IsLoaded(),
		"runtime":       "go",
	})
}

// Close 关闭路由器，停止后台协程（如限流清理器）。
func (r *Router) Close() {
	close(r.stopCh)
}

// ServeHTTP 实现 http.Handler 接口，将请求委托给内部 Gin 引擎处理。
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.engine.ServeHTTP(w, req)
}
