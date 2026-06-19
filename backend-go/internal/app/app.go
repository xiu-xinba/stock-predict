// Package app 负责应用程序的组装和启动，协调各基础设施组件的初始化和生命周期管理。
package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	database "stock-predict-go/internal/infrastructure/database"
	seed "stock-predict-go/internal/infrastructure/database/seed"
	providers "stock-predict-go/internal/infrastructure/providers"
	"stock-predict-go/internal/platform/config"
	transporthttp "stock-predict-go/internal/transport/http/router"
)

// CleanupFunc 是应用关闭时需要执行的资源清理函数，用于释放数据库连接、停止后台同步任务等。
type CleanupFunc func()

// NewServer 创建并初始化 HTTP 服务器，按以下顺序完成组装：
//  1. 连接数据库并根据配置执行迁移或校验 Schema
//  2. 初始化基金和股票的数据存储层
//  3. 执行基金种子数据写入及自动同步（若配置启用）
//  4. 初始化搜索索引和市场数据存储
//  5. 写入默认股票种子数据
//  6. 注册服务提供者（Registry），包含基金、股票、搜索等全部领域服务
//  7. 执行股票自动同步（若配置启用）
//  8. 启动市场数据同步和预加载等后台任务
//  9. 将基金和股票数据同步到搜索索引
//  10. 构建 HTTP 路由并返回就绪的 http.Server 及清理函数
func NewServer(cfg config.Config, logger *slog.Logger) (*http.Server, CleanupFunc, error) {
	// 步骤 1：连接数据库
	db, err := database.OpenDB(cfg.DatabaseURL, logger)
	if err != nil {
		logger.Warn("database init failed", "error", err)
		return nil, nil, fmt.Errorf("init database: %w", err)
	}
	// 根据配置决定是执行迁移还是仅校验 Schema
	if cfg.RunDatabaseMigrations {
		if err := database.MigrateDatabase(db, logger); err != nil {
			return nil, nil, fmt.Errorf("migrate database: %w", err)
		}
	} else if err := database.VerifyDatabaseSchema(db); err != nil {
		return nil, nil, err
	}

	// 步骤 2：初始化基金和股票的数据存储层
	fundStore := database.NewFundStore(db)
	stockStore := database.NewStockStore(db)

	// 步骤 3：写入基金种子数据
	if err := database.SeedFunds(db); err != nil {
		logger.Warn("seed funds failed", "error", err)
	}

	// 当配置启用自动同步且基金数量不足时，从外部数据源同步基金数据
	if cfg.FundAutoSyncOnStart && (fundStore.CountFunds() < cfg.FundAutoSyncMinCount || (cfg.FundMetricsURL != "" && fundStore.CountQuotedFunds() == 0)) {
		result, err := providers.NewFundService(fundStore).SyncFromSources(cfg.FundUniverseURL, cfg.FundMetricsURL, cfg.FundSyncCSVPath)
		if err != nil {
			logger.Warn("fund auto sync skipped", "error", err)
		} else {
			logger.Info("fund auto sync completed", "imported", result.Imported, "total", result.Total, "source", result.Source)
		}
	}

	// 步骤 4：初始化搜索索引和市场数据存储
	searchIdx := database.NewSearchStore(db)

	var marketStore *database.MarketStore
	if cfg.MarketSyncEnabled {
		marketStore = database.NewMarketStore(db, logger)
	}

	// 步骤 5：写入默认股票种子数据
	if err := seedDefaultStocks(stockStore); err != nil {
		logger.Warn("failed to load default stocks", "error", err)
	}

	// 步骤 6：注册服务提供者，组装全部领域服务
	services := providers.NewRegistry(fundStore, stockStore, cfg, logger, searchIdx, marketStore, db)

	// 步骤 7：执行股票自动同步（若配置启用）
	if cfg.StockAutoSyncOnStart {
		logger.Info("starting stock auto sync from eastmoney API...")
		syncCtx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		result, err := services.Stocks.SyncStocks(syncCtx)
		cancel()
		if err != nil {
			logger.Warn("stock auto sync failed, using default stocks", "error", err)
		} else {
			logger.Info("stock auto sync completed", "imported", result.Imported, "total", result.Total, "errors", result.Errors)
		}
	}

	// 步骤 8：启动市场数据同步和预加载等后台任务
	if cfg.MarketSyncEnabled && services.MarketSync != nil {
		services.MarketSync.Start()
	}

	if services.Preloader != nil {
		services.Preloader.Start()
	}

	// 步骤 8.5：启动北向/南向资金爬虫
	if services.HSGTScheduler != nil {
		// 异步同步历史数据（不阻塞启动）
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			if err := services.HSGTScheduler.SyncHistoricalData(ctx); err != nil {
				logger.Warn("HSGT historical sync failed", "error", err)
			}
		}()
		services.HSGTScheduler.Start()
		logger.Info("HSGT scheduler started")
	}

	// 步骤 9：将基金和股票数据同步到搜索索引
	if err := searchIdx.SyncFunds(fundStore.ListFunds()); err != nil {
		logger.Warn("failed to sync funds to search index", "error", err)
	}
	if err := searchIdx.SyncStocks(services.Stocks.ListStocks()); err != nil {
		logger.Warn("failed to sync stocks to search index", "error", err)
	}

	// 步骤 10：构建 HTTP 路由
	router := transporthttp.NewRouter(cfg, services, fundStore, logger, searchIdx)
	// 组装资源清理函数，在服务关闭时依次释放各组件
	cleanup := func() {
		if services.HSGTScheduler != nil {
			services.HSGTScheduler.Stop()
		}
		if services.MarketSync != nil {
			services.MarketSync.Stop()
		}
		if services.Preloader != nil {
			services.Preloader.Stop()
		}
		if services.Health != nil {
			services.Health.StopRecoveryProbe()
		}
		router.Close()
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}

	return &http.Server{
		Addr:              net.JoinHostPort(cfg.Host, cfg.Port),
		Handler:           router,
		ReadHeaderTimeout: cfg.ReadTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}, cleanup, nil
}

// seedDefaultStocks 将内置的默认股票列表写入数据库，若数据库中已存在股票数据则跳过。
func seedDefaultStocks(stockStore *database.StockStore) error {
	if stockStore.IsLoaded() {
		return nil
	}
	return stockStore.SaveStockList(seed.LoadDefaultStocks())
}
