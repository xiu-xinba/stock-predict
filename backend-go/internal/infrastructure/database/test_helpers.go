package database

import (
	"fmt"
	"net/url"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var testSchemaSequence atomic.Uint64

// InitTestDB creates an isolated PostgreSQL schema for one test.
func InitTestDB(test ...testing.TB) *gorm.DB {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://stock:stock123@localhost:5432/stock_predict_test?sslmode=disable"
	}

	adminDB, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		panic("failed to connect test database: " + err.Error())
	}

	schema := fmt.Sprintf("test_%d_%d_%d", os.Getpid(), time.Now().UnixNano(), testSchemaSequence.Add(1))
	if err := adminDB.Exec(`CREATE SCHEMA "` + schema + `"`).Error; err != nil {
		panic("failed to create test schema: " + err.Error())
	}

	scopedURL, err := withSearchPath(databaseURL, schema)
	if err != nil {
		panic("failed to configure test schema: " + err.Error())
	}
	db, err := gorm.Open(postgres.Open(scopedURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		panic("failed to connect isolated test database: " + err.Error())
	}
	if err := MigrateDatabase(db, nil); err != nil {
		panic("failed to migrate test database: " + err.Error())
	}

	if len(test) > 0 && test[0] != nil {
		test[0].Helper()
		test[0].Cleanup(func() {
			if sqlDB, dbErr := db.DB(); dbErr == nil {
				_ = sqlDB.Close()
			}
			_ = adminDB.Exec(`DROP SCHEMA IF EXISTS "` + schema + `" CASCADE`).Error
			if sqlDB, dbErr := adminDB.DB(); dbErr == nil {
				_ = sqlDB.Close()
			}
		})
	}
	return db
}

func withSearchPath(databaseURL, schema string) (string, error) {
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	query.Set("search_path", schema+",public")
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}
