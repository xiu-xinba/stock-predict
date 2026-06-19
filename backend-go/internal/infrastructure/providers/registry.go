package providers

import (
	"log/slog"
	"time"

	funddomain "stock-predict-go/internal/domain/fund"
	stockdomain "stock-predict-go/internal/domain/stock"
	database "stock-predict-go/internal/infrastructure/database"
	"stock-predict-go/internal/platform/config"

	"gorm.io/gorm"
)

// Registry 是所有数据源服务的注册中心，持有各服务实例和数据源列表，
// 负责在启动时完成依赖注入和组装。
type Registry struct {
	Funds        *FundService        // 基金服务
	Market       *MarketService      // 市场行情服务
	Watchlist    *WatchlistService   // 自选股服务
	Detail       *FundDetailService  // 基金详情服务
	Stocks       *StockService       // 股票服务
	StockDetail  *StockDetailService // 股票详情服务
	StockQuote   *StockQuoteClient   // 股票行情客户端
	Search       *SearchService      // 统一搜索服务
	MarketSync   *MarketSyncService  // 市场数据同步服务
	Health       *HealthMonitor      // 数据源健康监控
	Preloader    *TDXPreloader       // TDX 预加载器
	Router       *ProviderRouter     // 数据源路由器
	HSGTScheduler *HSGTScheduler                // 北向南向资金爬虫调度器
	HSGTStore     *database.HSGTFlowDailyStore   // 北向南向资金数据库
	Providers    []Provider          // 所有已注册的数据源
	DB           *gorm.DB            // 数据库连接
	Cache        *CacheProvider      // 缓存装饰器
}

// NewRegistry 创建并组装所有数据源服务实例，完成依赖注入。
func NewRegistry(fundRepo funddomain.Repository, stockRepo stockdomain.Repository, cfg config.Config, logger *slog.Logger, searchIdx *database.SearchStore, marketStore *database.MarketStore, db *gorm.DB) *Registry {
	indexQuote := NewIndexQuoteClient(logger)
	if marketStore != nil {
		indexQuote.SetMarketStore(marketStore)
	}

	// Create core clients first (needed by providers)
	quote := NewFundQuoteClient(HTTPClientTimeout, logger)
	stockQuote := NewStockQuoteClient(HTTPClientTimeout)

	// Create providers — only those with valid config are included
	tencentProvider := NewTencentProvider(indexQuote, stockQuote, quote)
	eastmoneyProvider := NewEastmoneyProvider(indexQuote, newEastmoneyClient(NewHTTPClient(HTTPClientConfig{})), nil)
	sinaProvider := NewSinaProvider(indexQuote)
	tdxProvider := NewTDXProvider(indexQuote, logger)
	thsProvider := NewTHSProvider(indexQuote)

	var providers []Provider
	providers = append(providers, tencentProvider, eastmoneyProvider, sinaProvider, tdxProvider, thsProvider)

	if cfg.BiyingAPIURL != "" && cfg.BiyingAPIToken != "" {
		biyingProvider := NewBiyingApiProvider(cfg.BiyingAPIURL, cfg.BiyingAPIToken, logger)
		providers = append(providers, biyingProvider)
	}
	if cfg.AKShareURL != "" && cfg.AKShareToken != "" {
		akshareProvider := NewAKShareProviderWithToken(cfg.AKShareURL, cfg.AKShareToken, logger)
		providers = append(providers, akshareProvider)
	}

	// Register only providers that actually exist in the HealthMonitor
	sourceNames := make([]string, 0, len(providers))
	for _, p := range providers {
		sourceNames = append(sourceNames, p.Name())
	}
	health := NewHealthMonitor(logger, sourceNames...)
	indexQuote.SetHealthMonitor(health)

	// Create remaining services
	market := NewMarketService(indexQuote, logger)
	funds := NewFundService(fundRepo)
	detail := NewFundDetailService(fundRepo, quote, logger)
	stocks := NewStockService(stockRepo, logger)
	stocks.SetMarketStore(marketStore)
	stocks.SetStockStore(database.NewStockStore(db))
	stocks.SetHealthMonitor(health)
	stocks.SetStockQuoteClient(stockQuote)
	watchlist := NewWatchlistService(fundRepo, cfg, logger)
	stockDetail := NewStockDetailService(stocks, stockQuote, indexQuote, logger)
	stockDetail.SetMarketStore(marketStore)
	search := NewSearchService(fundRepo, stockRepo, searchIdx)
	var preloader *TDXPreloader
	if marketStore != nil {
		preloader = NewTDXPreloader(indexQuote, market, marketStore, health, logger)
	}
	marketSync := NewMarketSyncService(indexQuote, market, marketStore, preloader, health, stocks, logger)

	// Inject eastmoney stocks service for stock ranking
	eastmoneyProvider.SetStockService(stocks)

	// Start active health probing
	health.SetProviders(providers)
	health.StartRecoveryProbe()

	router := NewProviderRouter(providers, health, RouterConfig{
		DefaultStrategy: StrategyFallback,
		RaceTimeout:     3 * time.Second,
		PerCapabilityOverrides: map[Capability]FetchStrategy{
			CapIndexQuote:  StrategyRace,
			CapIndexMinute: StrategyRaceThenFallback,
			CapStockQuote:  StrategyRaceThenFallback,
			CapStockMinute: StrategyRace,
		},
	}, logger)

	// Wrap THS provider with CacheProvider for K-line caching
	var cacheProvider *CacheProvider
	if marketStore != nil {
		cacheProvider = NewCacheProvider(thsProvider, marketStore, logger)
	}

	// Initialize HSGT scheduler for daily northbound/southbound fund data
	hsgtStore := database.NewHSGTFlowDailyStore(db)
	hsgtScheduler := NewHSGTScheduler(hsgtStore, cfg, logger)

	// Inject router into services
	indexQuote.SetRouter(router)
	stocks.SetRouter(router)
	market.SetRouter(router)
	quote.SetRouter(router)
	stockQuote.SetRouter(router)

	return &Registry{
		Funds:         funds,
		Market:        market,
		Watchlist:     watchlist,
		Detail:        detail,
		Stocks:        stocks,
		StockDetail:   stockDetail,
		StockQuote:    stockQuote,
		Search:        search,
		MarketSync:    marketSync,
		Health:        health,
		Preloader:     preloader,
		Router:        router,
		HSGTScheduler: hsgtScheduler,
		HSGTStore:     hsgtStore,
		Providers:     providers,
		DB:            db,
		Cache:         cacheProvider,
	}
}
