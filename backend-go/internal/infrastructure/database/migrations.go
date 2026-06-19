package database

import (
	"fmt"
	"log/slog"

	"gorm.io/gorm"
)

type schemaMigration struct {
	Version int `gorm:"primaryKey"`
}

func (schemaMigration) TableName() string {
	return "schema_migrations"
}

// MigrateDatabase 在 PostgreSQL advisory lock 保护下执行 Schema 变更。
// 每个迁移必须保留已有数据行，且可安全重复执行。
func MigrateDatabase(db *gorm.DB, appLogger *slog.Logger) error {
	if db == nil {
		return fmt.Errorf("migrate database: nil database")
	}
	if appLogger == nil {
		appLogger = slog.Default()
	}
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS pg_trgm").Error; err != nil {
		return fmt.Errorf("create pg_trgm extension: %w", err)
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext('stock_predict_schema_migrations'))").Error; err != nil {
			return fmt.Errorf("acquire migration lock: %w", err)
		}
		if err := tx.AutoMigrate(&schemaMigration{}); err != nil {
			return fmt.Errorf("create migration table: %w", err)
		}
		if err := applyMigration(tx, 1, migrateLegacyUpdatedAt); err != nil {
			return err
		}
		if err := tx.AutoMigrate(
			&Fund{},
			&Stock{},
			&IndexQuote{},
			&IndexMinute{},
			&IndexKlineDaily{},
			&KlineDaily{},
			&KlineWeekly{},
			&KlineMonthly{},
			&Financial{},
			&CacheMetadata{},
			&StockRanking{},
			&HSGTFlowDaily{},
		); err != nil {
			return fmt.Errorf("auto migrate schema: %w", err)
		}
		if err := createSearchIndexes(tx); err != nil {
			return err
		}
		if err := widenTextColumns(tx); err != nil {
			return err
		}
		return nil
	})
}

// VerifyDatabaseSchema 验证数据库 Schema 是否已初始化，检查核心表是否存在。
func VerifyDatabaseSchema(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("verify database schema: nil database")
	}
	for _, model := range []any{&Fund{}, &Stock{}, &schemaMigration{}} {
		if !db.Migrator().HasTable(model) {
			return fmt.Errorf("database schema is not initialized; run `go run ./cmd/migrate`")
		}
	}
	return nil
}

func applyMigration(tx *gorm.DB, version int, migrate func(*gorm.DB) error) error {
	var count int64
	if err := tx.Model(&schemaMigration{}).Where("version = ?", version).Count(&count).Error; err != nil {
		return fmt.Errorf("check migration %d: %w", version, err)
	}
	if count > 0 {
		return nil
	}
	if err := migrate(tx); err != nil {
		return fmt.Errorf("apply migration %d: %w", version, err)
	}
	if err := tx.Create(&schemaMigration{Version: version}).Error; err != nil {
		return fmt.Errorf("record migration %d: %w", version, err)
	}
	return nil
}

// migrateLegacyUpdatedAt 将 funds 和 cache_metadata 表的 updated_at 列
// 从非标准类型（如 text）迁移为 timestamptz，保留已有数据。
func migrateLegacyUpdatedAt(tx *gorm.DB) error {
	for _, table := range []string{"funds", "cache_metadata"} {
		var dataType string
		err := tx.Raw(`
			SELECT data_type
			FROM information_schema.columns
			WHERE table_schema = current_schema()
			  AND table_name = ?
			  AND column_name = 'updated_at'
		`, table).Scan(&dataType).Error
		if err != nil {
			return err
		}
		if dataType == "" || dataType == "timestamp with time zone" || dataType == "timestamp without time zone" {
			continue
		}
		statement := fmt.Sprintf(`
			ALTER TABLE %s
			ALTER COLUMN updated_at DROP DEFAULT,
			ALTER COLUMN updated_at TYPE timestamptz
			USING CASE
				WHEN updated_at IS NULL OR btrim(updated_at::text) = '' THEN NULL
				ELSE updated_at::timestamptz
			END
		`, table)
		if err := tx.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func createSearchIndexes(tx *gorm.DB) error {
	for _, statement := range []string{
		`CREATE INDEX IF NOT EXISTS idx_funds_pinyin_abbr_gin ON funds USING GIN (pinyin_abbr gin_trgm_ops)`,
		`CREATE INDEX IF NOT EXISTS idx_funds_pinyin_full_gin ON funds USING GIN (pinyin_full gin_trgm_ops)`,
		`CREATE INDEX IF NOT EXISTS idx_stocks_pinyin_gin ON stocks USING GIN (pinyin gin_trgm_ops)`,
	} {
		if err := tx.Exec(statement).Error; err != nil {
			return fmt.Errorf("create search index: %w", err)
		}
	}
	return nil
}

func widenTextColumns(tx *gorm.DB) error {
	for _, statement := range []string{
		`ALTER TABLE funds ALTER COLUMN fund_name TYPE varchar(60)`,
		`ALTER TABLE funds ALTER COLUMN fund_type TYPE varchar(30)`,
		`ALTER TABLE funds ALTER COLUMN company TYPE varchar(60)`,
		`ALTER TABLE funds ALTER COLUMN manager TYPE varchar(40)`,
		`ALTER TABLE funds ALTER COLUMN pinyin_abbr TYPE varchar(40)`,
		`ALTER TABLE stocks ALTER COLUMN stock_name TYPE varchar(60)`,
		`ALTER TABLE stocks ALTER COLUMN industry TYPE varchar(40)`,
	} {
		if err := tx.Exec(statement).Error; err != nil {
			return fmt.Errorf("widen text column: %w", err)
		}
	}
	return nil
}
