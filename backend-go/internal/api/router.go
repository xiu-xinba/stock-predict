package api

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"stock-predict-go/internal/config"
	"stock-predict-go/internal/service"
	"stock-predict-go/internal/store"
	"stock-predict-go/internal/util"
)

type Router struct {
	cfg       config.Config
	services  *service.Registry
	store     store.FundRepository
	searchIdx *store.SearchIndex
	logger    *slog.Logger
	stopCh    chan struct{}
}

func NewRouter(cfg config.Config, services *service.Registry, fundRepo store.FundRepository, logger *slog.Logger, searchIdx *store.SearchIndex) http.Handler {
	router := &Router{cfg: cfg, services: services, store: fundRepo, searchIdx: searchIdx, logger: logger, stopCh: make(chan struct{})}

	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.RedirectTrailingSlash = false
	engine.RedirectFixedPath = false
	_ = engine.SetTrustedProxies(nil)
	engine.Use(
		recoverer(logger),
		requestID(),
		requestLogger(logger),
		securityHeaders(),
		cors(cfg, logger),
		csrfProtection(cfg, router.stopCh),
		gzipMiddleware(),
		maxBody(),
		rateLimiter(logger, router.stopCh),
	)

	v1 := engine.Group("/api/v1")
	v1.GET("/health", router.health)
	v1.GET("/search", router.unifiedSearch)
	v1.GET("/funds/search", router.searchFunds)
	v1.GET("/funds/filters", router.fundFilters)
	v1.GET("/funds/coverage", router.fundCoverage)
	v1.POST("/funds/sync", router.requireAdminToken, router.syncFunds)
	v1.GET("/market/indices", router.marketIndices)
	v1.GET("/market/ranking/:type", router.marketRanking)
	v1.GET("/predict/:fundCode", router.predict)
	v1.POST("/watchlist/quotes", router.watchlistQuotes)
	v1.GET("/fund/:fundCode/detail", router.fundDetail)
	v1.GET("/stocks/search", router.searchStocks)
	v1.GET("/stocks/filters", router.stockFilters)
	v1.GET("/stock/:stockCode/detail", router.stockDetail)
	v1.GET("/stock/:stockCode/predict", router.predictStock)
	v1.POST("/stocks/quotes", router.stockQuotes)
	v1.GET("/market/stock-ranking/:type", router.stockRanking)
	v1.POST("/stocks/sync", router.requireAdminToken, router.syncStocks)

	engine.NoRoute(func(c *gin.Context) {
		writeError(c, http.StatusNotFound, -1, "not found")
	})

	return engine
}

func (r *Router) health(c *gin.Context) {
	writeJSON(c, http.StatusOK, map[string]any{
		"status":        "ok",
		"model_loaded":  r.services.Prediction.ModelLoaded(),
		"funds_loaded":  r.services.Funds.Count() > 0,
		"stocks_loaded": r.services.Stocks.IsLoaded(),
		"runtime":       "go",
	})
}

func (r *Router) Close() {
	close(r.stopCh)
}

func isSixDigitCode(value string) bool {
	return len(value) == 6 && util.IsAllDigits(value)
}
