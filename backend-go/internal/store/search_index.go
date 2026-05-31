package store

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	_ "modernc.org/sqlite"

	"stock-predict-go/internal/dto"
)

type SearchIndex struct {
	mu   sync.RWMutex
	db   *sql.DB
	path string
}

func NewSearchIndex(dbPath string, logger *slog.Logger) (*SearchIndex, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	idx := &SearchIndex{db: db, path: dbPath}

	if err := idx.createTables(); err != nil {
		db.Close()
		return nil, fmt.Errorf("create tables: %w", err)
	}

	logger.Info("search index initialized", "path", dbPath)
	return idx, nil
}

func (si *SearchIndex) createTables() error {
	_, err := si.db.Exec(`
		CREATE TABLE IF NOT EXISTS funds (
			fund_code   TEXT PRIMARY KEY,
			fund_name   TEXT NOT NULL,
			fund_type   TEXT NOT NULL DEFAULT '',
			pinyin_abbr TEXT NOT NULL DEFAULT '',
			pinyin_full TEXT NOT NULL DEFAULT '',
			company     TEXT NOT NULL DEFAULT '',
			manager     TEXT NOT NULL DEFAULT '',
			risk_level  TEXT NOT NULL DEFAULT ''
		);

		CREATE TABLE IF NOT EXISTS stocks (
			stock_code TEXT PRIMARY KEY,
			stock_name TEXT NOT NULL,
			market     TEXT NOT NULL DEFAULT '',
			industry   TEXT NOT NULL DEFAULT '',
			pinyin     TEXT NOT NULL DEFAULT ''
		);

		CREATE VIRTUAL TABLE IF NOT EXISTS funds_fts USING fts5(
			fund_code,
			pinyin_abbr,
			pinyin_full,
			content='funds',
			content_rowid='rowid'
		);

		CREATE VIRTUAL TABLE IF NOT EXISTS stocks_fts USING fts5(
			stock_code,
			pinyin,
			content='stocks',
			content_rowid='rowid'
		);

		CREATE TRIGGER IF NOT EXISTS funds_ai AFTER INSERT ON funds BEGIN
			INSERT INTO funds_fts(rowid, fund_code, pinyin_abbr, pinyin_full)
			VALUES (new.rowid, new.fund_code, new.pinyin_abbr, new.pinyin_full);
		END;

		CREATE TRIGGER IF NOT EXISTS funds_ad AFTER DELETE ON funds BEGIN
			INSERT INTO funds_fts(funds_fts, rowid, fund_code, pinyin_abbr, pinyin_full)
			VALUES ('delete', old.rowid, old.fund_code, old.pinyin_abbr, old.pinyin_full);
		END;

		CREATE TRIGGER IF NOT EXISTS funds_au AFTER UPDATE ON funds BEGIN
			INSERT INTO funds_fts(funds_fts, rowid, fund_code, pinyin_abbr, pinyin_full)
			VALUES ('delete', old.rowid, old.fund_code, old.pinyin_abbr, old.pinyin_full);
			INSERT INTO funds_fts(rowid, fund_code, pinyin_abbr, pinyin_full)
			VALUES (new.rowid, new.fund_code, new.pinyin_abbr, new.pinyin_full);
		END;

		CREATE TRIGGER IF NOT EXISTS stocks_ai AFTER INSERT ON stocks BEGIN
			INSERT INTO stocks_fts(rowid, stock_code, pinyin)
			VALUES (new.rowid, new.stock_code, new.pinyin);
		END;

		CREATE TRIGGER IF NOT EXISTS stocks_ad AFTER DELETE ON stocks BEGIN
			INSERT INTO stocks_fts(stocks_fts, rowid, stock_code, pinyin)
			VALUES ('delete', old.rowid, old.stock_code, old.pinyin);
		END;

		CREATE TRIGGER IF NOT EXISTS stocks_au AFTER UPDATE ON stocks BEGIN
			INSERT INTO stocks_fts(stocks_fts, rowid, stock_code, pinyin)
			VALUES ('delete', old.rowid, old.stock_code, old.pinyin);
			INSERT INTO stocks_fts(rowid, stock_code, pinyin)
			VALUES (new.rowid, new.stock_code, new.pinyin);
		END;
	`)
	return err
}

func (si *SearchIndex) Close() error {
	if si.db != nil {
		return si.db.Close()
	}
	return nil
}

func (si *SearchIndex) SyncFunds(funds []dto.FundItem) error {
	si.mu.Lock()
	defer si.mu.Unlock()

	tx, err := si.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM funds")
	if err != nil {
		return fmt.Errorf("clear funds: %w", err)
	}

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO funds (fund_code, fund_name, fund_type, pinyin_abbr, pinyin_full, company, manager, risk_level)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare fund insert: %w", err)
	}
	defer stmt.Close()

	for _, f := range funds {
		_, err = stmt.Exec(f.FundCode, f.FundName, f.FundType, f.PinyinAbbr, f.PinyinFull, f.Company, f.Manager, f.RiskLevel)
		if err != nil {
			return fmt.Errorf("insert fund %s: %w", f.FundCode, err)
		}
	}

	return tx.Commit()
}

func (si *SearchIndex) SyncStocks(stocks []dto.StockItem) error {
	si.mu.Lock()
	defer si.mu.Unlock()

	tx, err := si.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM stocks")
	if err != nil {
		return fmt.Errorf("clear stocks: %w", err)
	}

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO stocks (stock_code, stock_name, market, industry, pinyin)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare stock insert: %w", err)
	}
	defer stmt.Close()

	for _, s := range stocks {
		pinyinStr := s.Pinyin
		if s.PinyinAlt != "" {
			pinyinStr = s.Pinyin + " " + s.PinyinAlt
		}
		_, err = stmt.Exec(s.StockCode, s.StockName, s.Market, s.Industry, pinyinStr)
		if err != nil {
			return fmt.Errorf("insert stock %s: %w", s.StockCode, err)
		}
	}

	return tx.Commit()
}

func (si *SearchIndex) SearchFundsByCodeOrPinyin(keyword string, limit int) ([]string, error) {
	si.mu.RLock()
	defer si.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}

	query := strings.TrimSpace(keyword)
	if query == "" {
		return nil, nil
	}

	escaped := escapeFTSQuery(query)
	if escaped == "" {
		return nil, nil
	}

	matchExpr := `"` + escaped + `" OR "` + escaped + `*"`

	sqlStr := `
		SELECT f.fund_code
		FROM funds_fts ft
		JOIN funds f ON f.fund_code = ft.fund_code
		WHERE funds_fts MATCH ?
		ORDER BY rank
		LIMIT ?
	`

	rows, err := si.db.Query(sqlStr, matchExpr, limit)
	if err != nil {
		return nil, fmt.Errorf("query funds fts: %w", err)
	}
	defer rows.Close()

	var codes []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, fmt.Errorf("scan fund code: %w", err)
		}
		codes = append(codes, code)
	}

	return codes, rows.Err()
}

func (si *SearchIndex) SearchStocksByCodeOrPinyin(keyword string, limit int) ([]string, error) {
	si.mu.RLock()
	defer si.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}

	query := strings.TrimSpace(keyword)
	if query == "" {
		return nil, nil
	}

	escaped := escapeFTSQuery(query)
	if escaped == "" {
		return nil, nil
	}

	matchExpr := `"` + escaped + `" OR "` + escaped + `*"`

	sqlStr := `
		SELECT s.stock_code
		FROM stocks_fts st
		JOIN stocks s ON s.stock_code = st.stock_code
		WHERE stocks_fts MATCH ?
		ORDER BY rank
		LIMIT ?
	`

	rows, err := si.db.Query(sqlStr, matchExpr, limit)
	if err != nil {
		return nil, fmt.Errorf("query stocks fts: %w", err)
	}
	defer rows.Close()

	var codes []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, fmt.Errorf("scan stock code: %w", err)
		}
		codes = append(codes, code)
	}

	return codes, rows.Err()
}

func (si *SearchIndex) FundCount() (int, error) {
	var count int
	err := si.db.QueryRow("SELECT COUNT(*) FROM funds").Scan(&count)
	return count, err
}

func (si *SearchIndex) StockCount() (int, error) {
	var count int
	err := si.db.QueryRow("SELECT COUNT(*) FROM stocks").Scan(&count)
	return count, err
}

func escapeFTSQuery(query string) string {
	var b strings.Builder
	for _, r := range query {
		switch r {
		case '"', '*', '(', ')', '^', '+', '-', ':':
			continue
		case 'A', 'N', 'D', 'O', 'R', 'T':
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	result := strings.TrimSpace(b.String())
	if result == "" {
		return ""
	}
	return strings.ReplaceAll(result, `"`, `""`)
}
