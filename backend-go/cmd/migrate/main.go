// Package main 是数据库迁移工具的入口。
package main

import (
	"fmt"
	"log/slog"
	"os"

	database "stock-predict-go/internal/infrastructure/database"
	"stock-predict-go/internal/platform/config"
)

// main 是数据库迁移工具的入口函数，负责加载配置、连接数据库并执行全部迁移。
func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	db, err := database.OpenDB(cfg.DatabaseURL, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "database connection failed: %v\n", err)
		os.Exit(1)
	}
	if err := database.MigrateDatabase(db, logger); err != nil {
		fmt.Fprintf(os.Stderr, "database migration failed: %v\n", err)
		os.Exit(1)
	}
	logger.Info("database migration completed")
}
