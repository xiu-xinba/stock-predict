package api

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"stock-predict-go/internal/config"
	"stock-predict-go/internal/dto"
	"stock-predict-go/internal/service"
)

type Router struct {
	cfg      config.Config
	services *service.Registry
	logger   *slog.Logger
}

func NewRouter(cfg config.Config, services *service.Registry, logger *slog.Logger) http.Handler {
	router := &Router{cfg: cfg, services: services, logger: logger}

	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.RedirectTrailingSlash = false
	engine.RedirectFixedPath = false
	_ = engine.SetTrustedProxies(nil)
	engine.Use(
		recoverer(logger),
		requestLogger(logger),
		securityHeaders(),
		cors(cfg),
		maxBody(),
	)

	v1 := engine.Group("/api/v1")
	v1.GET("/health", router.health)
	v1.GET("/funds/search", router.searchFunds)
	v1.GET("/funds/filters", router.fundFilters)
	v1.POST("/funds/sync", router.syncFunds)
	v1.GET("/market/indices", router.marketIndices)
	v1.GET("/market/ranking/:type", router.marketRanking)
	v1.GET("/predict/:fundCode", router.predict)
	v1.POST("/watchlist/quotes", router.watchlistQuotes)

	engine.NoRoute(func(c *gin.Context) {
		writeError(c, http.StatusNotFound, -1, "not found")
	})

	return engine
}

func (r *Router) health(c *gin.Context) {
	writeJSON(c, http.StatusOK, map[string]any{
		"status":       "ok",
		"model_loaded": r.services.Prediction.ModelLoaded(),
		"runtime":      "go",
	})
}

func (r *Router) searchFunds(c *gin.Context) {
	var query dto.FundSearchRequest
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, -1, "查询参数格式错误")
		return
	}
	writeSuccess(c, r.services.Funds.Search(query))
}

func (r *Router) fundFilters(c *gin.Context) {
	writeSuccess(c, r.services.Funds.Filters())
}

func (r *Router) syncFunds(c *gin.Context) {
	if !r.cfg.IsDevelopment() {
		if r.cfg.AdminToken == "" {
			writeError(c, http.StatusForbidden, -1, "未配置管理员令牌，禁止触发同步")
			return
		}
		if c.GetHeader("X-Admin-Token") != r.cfg.AdminToken {
			writeError(c, http.StatusUnauthorized, -1, "无权触发同步")
			return
		}
	}
	writeSuccess(c, r.services.Funds.Count())
}

func (r *Router) marketIndices(c *gin.Context) {
	writeSuccess(c, r.services.Market.Indices())
}

func (r *Router) marketRanking(c *gin.Context) {
	var path dto.MarketRankingPath
	var query dto.MarketRankingQuery
	if err := c.ShouldBindUri(&path); err != nil {
		writeError(c, http.StatusBadRequest, -1, "路径参数格式错误")
		return
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, -1, "查询参数格式错误")
		return
	}
	items, err := r.services.Funds.Ranking(path.Type, query.Size)
	if errors.Is(err, service.ErrInvalidRankingType) {
		writeJSON(c, http.StatusOK, dto.APIResponse{Code: -1, Message: "type 必须为 gainers 或 losers", Data: nil})
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, -1, "服务器繁忙，请稍后重试")
		return
	}
	writeSuccess(c, items)
}

func (r *Router) predict(c *gin.Context) {
	var path dto.PredictPath
	if err := c.ShouldBindUri(&path); err != nil {
		writeError(c, http.StatusBadRequest, -1, "路径参数格式错误")
		return
	}
	result, err := r.services.Prediction.PredictByFundCode(path.FundCode)
	if errors.Is(err, service.ErrInvalidFundCode) {
		writeJSON(c, http.StatusOK, dto.APIResponse{Code: -1, Message: "基金代码必须为6位数字", Data: nil})
		return
	}
	if errors.Is(err, service.ErrFundNotFound) {
		writeJSON(c, http.StatusOK, dto.APIResponse{Code: -1, Message: "未找到基金 " + path.FundCode, Data: nil})
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, -1, "服务器繁忙，请稍后重试")
		return
	}
	writeSuccess(c, result)
}

func (r *Router) watchlistQuotes(c *gin.Context) {
	var payload dto.WatchlistQuoteRequest
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		writeError(c, http.StatusBadRequest, -1, "请求体格式错误")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(c, http.StatusBadRequest, -1, "请求体格式错误")
		return
	}
	if len(payload.Codes) == 0 {
		writeSuccess(c, []dto.WatchlistItem{})
		return
	}
	if len(payload.Codes) > 50 {
		writeError(c, http.StatusBadRequest, -1, "最多支持50个基金代码")
		return
	}
	for _, code := range payload.Codes {
		if !isFundCode(code) {
			writeError(c, http.StatusBadRequest, -1, "基金代码必须为6位数字")
			return
		}
	}
	writeSuccess(c, r.services.Prediction.WatchlistQuotes(payload.Codes))
}

func isFundCode(value string) bool {
	if len(value) != 6 {
		return false
	}
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}
