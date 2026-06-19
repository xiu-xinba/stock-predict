package database

import (
	"fmt"
	"strings"

	funddomain "stock-predict-go/internal/domain/fund"
	stockdomain "stock-predict-go/internal/domain/stock"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SearchStore 基于 PostgreSQL pg_trgm + GIN 索引提供全文搜索功能
type SearchStore struct {
	db *gorm.DB
}

// NewSearchStore 创建 SearchStore 实例
func NewSearchStore(db *gorm.DB) *SearchStore {
	return &SearchStore{db: db}
}

// Close 空操作（GORM 连接池由中央管理）
func (s *SearchStore) Close() error {
	return nil
}

// SyncFunds 同步基金数据用于搜索（更新插入到 funds 表）
func (s *SearchStore) SyncFunds(funds []funddomain.FundItem) error {
	models := make([]Fund, 0, len(funds))
	for _, f := range funds {
		f.FundCode = normalizeFundCode(f.FundCode)
		if f.FundCode == "" {
			continue
		}
		models = append(models, fundDTOToModel(f))
	}
	if len(models) == 0 {
		return nil
	}
	return s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "fund_code"}},
		DoUpdates: clause.AssignmentColumns([]string{"fund_name", "fund_type", "pinyin_abbr", "pinyin_full", "company", "manager", "risk_level"}),
	}).CreateInBatches(models, 100).Error
}

// SyncStocks 同步股票数据用于搜索（更新插入到 stocks 表）
func (s *SearchStore) SyncStocks(stocks []stockdomain.StockItem) error {
	models := make([]Stock, 0, len(stocks))
	for _, st := range stocks {
		if st.StockCode == "" {
			continue
		}
		models = append(models, stockDTOToModel(st))
	}
	if len(models) == 0 {
		return nil
	}
	return s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "stock_code"}},
		DoUpdates: clause.AssignmentColumns([]string{"stock_name", "market", "industry", "pinyin"}),
	}).CreateInBatches(models, 100).Error
}

// SearchFundsByCodeOrPinyin 通过基金代码或拼音搜索基金，使用 pg_trgm 模糊匹配
func (s *SearchStore) SearchFundsByCodeOrPinyin(keyword string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 50
	}
	query := strings.TrimSpace(keyword)
	if query == "" {
		return nil, nil
	}

	var codes []string
	q := s.db.Model(&Fund{}).Select("fund_code").Limit(limit).
		Where("fund_code LIKE ? OR pinyin_abbr % ? OR pinyin_full LIKE ?",
			query+"%", query, query+"%")
	if err := q.Pluck("fund_code", &codes).Error; err != nil {
		return nil, fmt.Errorf("search funds: %w", err)
	}
	return codes, nil
}

// SearchStocksByCodeOrPinyin 通过股票代码或拼音搜索股票，使用 pg_trgm 模糊匹配
func (s *SearchStore) SearchStocksByCodeOrPinyin(keyword string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 50
	}
	query := strings.TrimSpace(keyword)
	if query == "" {
		return nil, nil
	}

	var codes []string
	q := s.db.Model(&Stock{}).Select("stock_code").Limit(limit).
		Where("stock_code LIKE ? OR pinyin % ?",
			query+"%", query)
	if err := q.Pluck("stock_code", &codes).Error; err != nil {
		return nil, fmt.Errorf("search stocks: %w", err)
	}
	return codes, nil
}

// FundCount 返回基金总数
func (s *SearchStore) FundCount() (int, error) {
	var count int64
	err := s.db.Model(&Fund{}).Count(&count).Error
	return int(count), err
}

// StockCount 返回股票总数
func (s *SearchStore) StockCount() (int, error) {
	var count int64
	err := s.db.Model(&Stock{}).Count(&count).Error
	return int(count), err
}
