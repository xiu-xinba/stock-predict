package database

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestMigrateDatabasePreservesExistingUpdatedAt(t *testing.T) {
	db := InitTestDB(t)
	want := time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC)
	fund := Fund{
		FundCode:  "999999",
		FundName:  "迁移保留测试",
		UpdatedAt: want,
	}
	if err := db.Create(&fund).Error; err != nil {
		t.Fatalf("create fund: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	if err := MigrateDatabase(db, logger); err != nil {
		t.Fatalf("rerun migrations: %v", err)
	}

	var got Fund
	if err := db.First(&got, "fund_code = ?", fund.FundCode).Error; err != nil {
		t.Fatalf("load fund after migration: %v", err)
	}
	if !got.UpdatedAt.Equal(want) {
		t.Fatalf("updated_at changed during migration: got %s want %s", got.UpdatedAt, want)
	}
}

func TestDatabaseLogTargetOmitsCredentials(t *testing.T) {
	target := databaseLogTarget("postgres://stock:super-secret@db.example.com:5432/stock_predict?sslmode=require")
	if strings.Contains(target, "stock@") || strings.Contains(target, "super-secret") {
		t.Fatalf("database log target leaked credentials: %q", target)
	}
	if target != "db.example.com/stock_predict" {
		t.Fatalf("unexpected database log target: %q", target)
	}
}
