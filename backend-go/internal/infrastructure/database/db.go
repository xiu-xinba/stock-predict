// Package database 提供基于 GORM + PostgreSQL 的数据持久化基础设施，
// 包含数据库连接管理、Schema 迁移、GORM 模型定义以及各业务领域的 Store 实现。
package database

import (
	"fmt"
	"log/slog"
	"net/url"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// OpenDB 初始化 PostgreSQL 连接，不修改数据库 Schema。
// 设置最大打开连接数为 25，最大空闲连接数为 10。
func OpenDB(databaseURL string, appLogger *slog.Logger) (*gorm.DB, error) {
	gormLogger := logger.New(
		&slogWriterAdapter{logger: appLogger},
		logger.Config{
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)

	appLogger.Info("database initialized", "target", databaseLogTarget(databaseURL))
	return db, nil
}

// InitDB 打开数据库连接并执行 Schema 迁移，用于兼容需要自动迁移的工具。
func InitDB(databaseURL string, appLogger *slog.Logger) (*gorm.DB, error) {
	db, err := OpenDB(databaseURL, appLogger)
	if err != nil {
		return nil, err
	}
	if err := MigrateDatabase(db, appLogger); err != nil {
		return nil, err
	}
	return db, nil
}

func databaseLogTarget(databaseURL string) string {
	parsed, err := url.Parse(databaseURL)
	if err != nil || parsed.Host == "" {
		return "configured"
	}
	database := parsed.Path
	if database == "" {
		database = "/"
	}
	return parsed.Hostname() + database
}

// slogWriterAdapter 将 slog 适配为 gorm logger.Writer 接口
type slogWriterAdapter struct {
	logger *slog.Logger
}

// Printf 实现 gorm logger.Writer 接口，将日志以 Warn 级别输出到 slog。
func (w *slogWriterAdapter) Printf(format string, args ...interface{}) {
	w.logger.Warn(fmt.Sprintf(format, args...))
}
