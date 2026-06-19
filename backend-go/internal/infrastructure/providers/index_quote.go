package providers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"time"

	marketdomain "stock-predict-go/internal/domain/market"
	database "stock-predict-go/internal/infrastructure/database"
)

// cnIndexCodes A股主要指数代码列表。
var cnIndexCodes = []string{"000001", "399001", "399006"}

// cnIndexNames A股指数代码到名称的映射。
var cnIndexNames = map[string]string{
	"000001": "上证指数",
	"399001": "深证成指",
	"399006": "创业板指",
}

// cnIndexMarkets A股指数代码到市场前缀的映射。
var cnIndexMarkets = map[string]string{
	"000001": "sh",
	"399001": "sz",
	"399006": "sz",
}

// tencentIndexSymbols 指数代码到腾讯行情 API 符号的映射。
var tencentIndexSymbols = map[string]string{
	"000001": "sh000001",
	"399001": "sz399001",
	"399006": "sz399006",
	"hsi":    "r_hkHSI",
	"hstech": "r_hkHSTECH",
	"dji":    "usDJI",
	"ixic":   "usIXIC",
	"spx":    "usINX",
}

// hkIndexMeta 港股指数元数据（名称和市场）。
var hkIndexMeta = map[string]struct {
	name   string
	market string
}{
	"hsi":    {name: "恒生指数", market: "hk"},
	"hstech": {name: "恒生科技", market: "hk"},
}

// usIndexMeta 美股指数元数据（名称和市场）。
var usIndexMeta = map[string]struct {
	name   string
	market string
}{
	"dji":  {name: "道琼斯", market: "us"},
	"ixic": {name: "纳斯达克", market: "us"},
	"spx":  {name: "标普500", market: "us"},
}

// indexSortOrder 定义指数的固定显示顺序。
var indexSortOrder = map[string]int{
	"000001": 0, "399001": 1, "399006": 2,
	"hsi": 3, "hstech": 4,
	"dji": 5, "ixic": 6, "spx": 7,
}

// sortIndicesByOrder 按固定顺序排列指数数据。
func sortIndicesByOrder(indices []marketdomain.MarketIndex) {
	sort.SliceStable(indices, func(i, j int) bool {
		oi, okI := indexSortOrder[indices[i].Code]
		oj, okJ := indexSortOrder[indices[j].Code]
		if !okI && !okJ {
			return indices[i].Code < indices[j].Code
		}
		if !okI {
			return false
		}
		if !okJ {
			return true
		}
		return oi < oj
	})
}

// IndexQuoteClient 指数行情客户端，支持获取 A股、港股、美股指数的实时行情、K线、分时数据。
type IndexQuoteClient struct {
	logger      *slog.Logger
	client      *http.Client
	resilient   *ResilientHTTPClient
	quoteCache  *DetailCache
	klineCache  *DetailCache
	minuteCache *DetailCache
	marketStore *database.MarketStore
	health      *HealthMonitor
	router      *ProviderRouter
}

// NewIndexQuoteClient 创建新的指数行情客户端实例。
func NewIndexQuoteClient(logger *slog.Logger) *IndexQuoteClient {
	if logger == nil {
		logger = slog.Default()
	}
	return &IndexQuoteClient{
		logger:      logger,
		client:      NewHTTPClient(HTTPClientConfig{}),
		resilient:   NewResilientHTTPClient(NewHTTPClient(HTTPClientConfig{}), nil),
		quoteCache:  NewDetailCache(CacheMaxEntries, IndexQuoteCacheTTL),
		klineCache:  NewDetailCache(CacheMaxEntries, IndexKlineCacheTTL),
		minuteCache: NewDetailCache(CacheMaxEntries, IndexMinuteCacheTTL),
	}
}

// SetMarketStore 注入 MarketStore 用于数据持久化。
func (c *IndexQuoteClient) SetMarketStore(ms *database.MarketStore) {
	c.marketStore = ms
}

// SetHealthMonitor 注入健康监控器。
func (c *IndexQuoteClient) SetHealthMonitor(hm *HealthMonitor) {
	c.health = hm
}

// SetRouter 注入数据源路由器。
func (c *IndexQuoteClient) SetRouter(router *ProviderRouter) {
	c.router = router
}

// FetchIndexQuotes 获取所有主要指数的实时行情数据。
func (c *IndexQuoteClient) FetchIndexQuotes(ctx context.Context) []marketdomain.MarketIndex {
	if c.router != nil {
		return c.fetchIndexQuotesViaRouter(ctx)
	}
	return c.fetchIndexQuotesLegacy(ctx)
}

// fetchIndexQuotesViaRouter 通过路由器获取指数行情数据。
func (c *IndexQuoteClient) fetchIndexQuotesViaRouter(ctx context.Context) []marketdomain.MarketIndex {
	const cacheKey = "index_quotes"
	if cached, ok := c.quoteCache.Get(cacheKey); ok {
		if val, ok2 := cached.([]marketdomain.MarketIndex); ok2 {
			return val
		}
	}

	var result []marketdomain.MarketIndex
	for _, market := range []Market{MarketCN, MarketHK, MarketUS} {
		var quotes []marketdomain.MarketIndex
		err := c.router.Fetch(ctx, CapIndexQuote, market, func(ctx context.Context, p Provider) error {
			provider, ok := p.(IndexQuoteProvider)
			if !ok {
				return fmt.Errorf("provider %s does not implement IndexQuoteProvider", p.Name())
			}
			q, err := provider.FetchIndexQuotes(ctx, market)
			if err != nil {
				return err
			}
			quotes = q
			return nil
		})
		if err == nil && len(quotes) > 0 {
			result = append(result, quotes...)
		}
	}

	if len(result) == 0 && c.marketStore != nil {
		cached := c.marketStore.LoadIndexQuotes()
		if len(cached) > 0 {
			c.logger.Info("using db cache for index quotes")
			for i := range cached {
				cached[i].DataSource = "cache"
			}
			result = cached
		}
	}

	if len(result) > 0 {
		c.fillMiniChartData(ctx, result)
		c.fixIsClosed(result)
		sortIndicesByOrder(result)
		c.quoteCache.Set(cacheKey, result)
	}
	return result
}

// fetchIndexQuotesLegacy 传统方式获取指数行情，依次尝试腾讯、新浪、TDX 数据源。
func (c *IndexQuoteClient) fetchIndexQuotesLegacy(ctx context.Context) []marketdomain.MarketIndex {
	const cacheKey = "index_quotes"
	if cached, ok := c.quoteCache.Get(cacheKey); ok {
		if val, ok2 := cached.([]marketdomain.MarketIndex); ok2 {
			return val
		}
	}

	var result []marketdomain.MarketIndex

	cnQuotes := c.fetchCNIndexQuotesTencent(ctx)
	if len(cnQuotes) > 0 {
		if c.health != nil {
			c.health.RecordSuccess("tencent")
		}
		result = append(result, cnQuotes...)
	} else {
		if c.health != nil {
			c.health.RecordFailure("tencent", fmt.Errorf("tencent returned empty CN quotes"))
		}
		cnTDX := c.fetchCNIndexQuotesTDX(ctx)
		if len(cnTDX) > 0 {
			if c.health != nil {
				c.health.RecordSuccess("tdx")
			}
			result = append(result, cnTDX...)
		} else {
			if c.health != nil {
				c.health.RecordFailure("tdx", fmt.Errorf("tdx returned empty CN quotes"))
			}
			cnEastmoney := c.fetchCNIndexQuotesEastmoney(ctx)
			if len(cnEastmoney) > 0 {
				if c.health != nil {
					c.health.RecordSuccess("eastmoney")
				}
				result = append(result, cnEastmoney...)
			} else {
				if c.health != nil {
					c.health.RecordFailure("eastmoney", fmt.Errorf("eastmoney returned empty CN quotes"))
				}
			}
		}
	}

	hkQuotes := c.fetchHKIndexQuotesTencent(ctx)
	if len(hkQuotes) > 0 {
		if c.health != nil {
			c.health.RecordSuccess("tencent")
		}
	} else {
		if c.health != nil {
			c.health.RecordFailure("tencent", fmt.Errorf("tencent returned empty HK quotes"))
		}
	}
	result = append(result, hkQuotes...)

	usQuotes := c.fetchUSIndexQuotesTencent(ctx)
	if len(usQuotes) > 0 {
		if c.health != nil {
			c.health.RecordSuccess("tencent")
		}
	} else {
		if c.health != nil {
			c.health.RecordFailure("tencent", fmt.Errorf("tencent returned empty US quotes"))
		}
	}
	result = append(result, usQuotes...)

	if len(result) == 0 && c.marketStore != nil {
		cached := c.marketStore.LoadIndexQuotes()
		if len(cached) > 0 {
			c.logger.Info("using db cache for index quotes")
			for i := range cached {
				cached[i].DataSource = "cache"
			}
			result = cached
		}
	}

	if len(result) > 0 {
		c.fillMiniChartData(ctx, result)
		c.fixIsClosed(result)
		sortIndicesByOrder(result)
		c.quoteCache.Set(cacheKey, result)
	}
	return result
}

// fillMiniChartData 为指数填充迷你图表数据。
func (c *IndexQuoteClient) fillMiniChartData(ctx context.Context, indices []marketdomain.MarketIndex) {
	for i := range indices {
		if isCNIndex(indices[i].Code) {
			c.fillCNMiniChartData(indices[i].Code, &indices[i])
		} else {
			c.fillKlineMiniChartData(ctx, indices[i].Code, &indices[i])
		}
	}
}

// fillCNMiniChartData 为 A股指数填充迷你图表数据（从 MarketStore 读取分时数据）。
func (c *IndexQuoteClient) fillCNMiniChartData(code string, idx *marketdomain.MarketIndex) {
	cacheKey := "index_minute:" + code
	var minuteData []marketdomain.IndexMinutePoint
	if cached, ok := c.minuteCache.Get(cacheKey); ok {
		if md, ok2 := cached.([]marketdomain.IndexMinutePoint); ok2 && len(md) > 0 {
			minuteData = md
		}
	}
	if len(minuteData) == 0 && c.marketStore != nil {
		today := time.Now().Format("2006-01-02")
		minuteData = c.marketStore.LoadIndexMinutes(code, today)
	}
	if len(minuteData) == 0 {
		return
	}
	chartData := make([]float64, len(minuteData))
	for j, p := range minuteData {
		chartData[j] = p.Price
	}
	idx.MiniChartData = chartData
}

// fillKlineMiniChartData 为指数填充基于 K 线的迷你图表数据。
func (c *IndexQuoteClient) fillKlineMiniChartData(_ context.Context, code string, idx *marketdomain.MarketIndex) {
	cacheKey := fmt.Sprintf("index_kline:%s:30", code)
	var klineData []marketdomain.IndexKlinePoint
	if cached, ok := c.klineCache.Get(cacheKey); ok {
		if kd, ok2 := cached.([]marketdomain.IndexKlinePoint); ok2 && len(kd) > 0 {
			klineData = kd
		}
	}
	if len(klineData) == 0 && c.marketStore != nil {
		klineData = c.marketStore.LoadIndexKline(code, 30)
	}
	if len(klineData) == 0 {
		return
	}
	chartData := make([]float64, len(klineData))
	for j, p := range klineData {
		chartData[j] = p.Close
	}
	idx.MiniChartData = chartData
}

// FetchIndexMinute 获取指定指数的分时数据。
func (c *IndexQuoteClient) FetchIndexMinute(ctx context.Context, code string) []marketdomain.IndexMinutePoint {
	if c.router != nil {
		return c.fetchIndexMinuteViaRouter(ctx, code)
	}
	return c.fetchIndexMinuteLegacy(ctx, code)
}

func (c *IndexQuoteClient) fetchIndexMinuteViaRouter(ctx context.Context, code string) []marketdomain.IndexMinutePoint {
	cacheKey := "index_minute:" + code
	if cached, ok := c.minuteCache.Get(cacheKey); ok {
		if val, ok2 := cached.([]marketdomain.IndexMinutePoint); ok2 {
			return val
		}
	}

	// 优先从 SQLite 缓存加载（快速路径）
	cacheDate := time.Now().Format("2006-01-02")
	if _, isUS := usIndexMeta[code]; isUS {
		now := time.Now()
		if now.Hour() < 4 {
			cacheDate = now.AddDate(0, 0, -1).Format("2006-01-02")
		}
	}
	if c.marketStore != nil {
		if cached := c.marketStore.LoadIndexMinutes(code, cacheDate); len(cached) > 0 {
			c.minuteCache.Set(cacheKey, cached)
			return cached
		}
	}

	market := detectIndexMarket(code)
	var result []marketdomain.IndexMinutePoint
	err := c.router.Fetch(ctx, CapIndexMinute, market, func(ctx context.Context, p Provider) error {
		provider, ok := p.(IndexMinuteProvider)
		if !ok {
			return fmt.Errorf("provider %s does not implement IndexMinuteProvider", p.Name())
		}
		points, err := provider.FetchIndexMinute(ctx, code, market)
		if err != nil {
			return err
		}
		result = points
		return nil
	})
	_ = err

	// 美股分时数据兜底：当所有 healthy provider 均失败时，直接调用新浪接口。
	// 原因：sina 的健康检查基于 A 股接口，非交易时段 A 股返回空导致 sina 被标记为 unhealthy，
	// 但美股分时接口实际上是正常的，不应受 A 股健康检查影响。
	if len(result) == 0 && market == MarketUS {
		if fallback := c.fetchUSIndexMinuteSina(ctx, code); len(fallback) > 0 {
			result = fallback
		}
	}

	if len(result) > 0 {
		// Merge with existing DB cache to avoid data gaps
		if c.marketStore != nil {
			if dbData := c.marketStore.LoadIndexMinutes(code, cacheDate); len(dbData) > 0 {
				result = mergeIndexMinutePoints(dbData, result)
			}
		}
		c.minuteCache.Set(cacheKey, result)
		if c.marketStore != nil {
			go func() {
				if err := c.marketStore.SaveIndexMinutes(code, cacheDate, result); err != nil {
					slog.Warn("failed to save index minutes cache", "code", code, "date", cacheDate, "error", err)
				}
			}()
		}
	}

	return result
}

func (c *IndexQuoteClient) fetchIndexMinuteLegacy(ctx context.Context, code string) []marketdomain.IndexMinutePoint {
	cacheKey := "index_minute:" + code
	if cached, ok := c.minuteCache.Get(cacheKey); ok {
		if val, ok2 := cached.([]marketdomain.IndexMinutePoint); ok2 {
			return val
		}
	}

	// 优先从 SQLite 缓存加载（快速路径），避免 TDX/腾讯超时导致前端请求失败
	cacheDate := time.Now().Format("2006-01-02")
	if _, isUS := usIndexMeta[code]; isUS {
		now := time.Now()
		if now.Hour() < 4 {
			cacheDate = now.AddDate(0, 0, -1).Format("2006-01-02")
		}
	}
	if c.marketStore != nil {
		if cached := c.marketStore.LoadIndexMinutes(code, cacheDate); len(cached) > 0 {
			c.minuteCache.Set(cacheKey, cached)
			return cached
		}
	}

	var result []marketdomain.IndexMinutePoint

	if isCNIndex(code) {
		points := c.fetchCNIndexMinuteTDX(ctx, code)
		if len(points) > 0 {
			result = points
		} else {
			result = c.fetchCNIndexMinuteTencent(ctx, code)
		}
	} else if _, isHK := hkIndexMeta[code]; isHK {
		result = c.fetchHKIndexMinuteTencent(ctx, code)
	} else if _, isUS := usIndexMeta[code]; isUS {
		// 美股指数分时数据获取优先级：腾讯分钟 → 新浪分钟 → 东方财富trends2 → 腾讯K线m1 → 实时行情回退
		if points := c.fetchUSIndexMinuteTencent(ctx, code); len(points) >= 2 {
			result = points
		} else if points := c.fetchUSIndexMinuteSina(ctx, code); len(points) >= 2 {
			result = points
		} else if emPoints := c.fetchUSIndexMinuteEastmoney(ctx, code); len(emPoints) >= 2 {
			result = emPoints
		} else if klinePoints := c.fetchUSIndexMinuteTencentKline(ctx, code); len(klinePoints) >= 2 {
			result = klinePoints
		}
	}

	if len(result) > 0 {
		// Merge with existing DB cache to avoid data gaps
		if c.marketStore != nil {
			if dbData := c.marketStore.LoadIndexMinutes(code, cacheDate); len(dbData) > 0 {
				result = mergeIndexMinutePoints(dbData, result)
			}
		}
		c.minuteCache.Set(cacheKey, result)
		if c.marketStore != nil {
			go func() {
				if err := c.marketStore.SaveIndexMinutes(code, cacheDate, result); err != nil {
					slog.Warn("failed to save index minutes cache", "code", code, "date", cacheDate, "error", err)
				}
			}()
		}
	}

	return result
}

// FetchStockMinute fetches minute-level data for an individual stockdomain.
