package database

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	stockdomain "stock-predict-go/internal/domain/stock"
)

// StockStore 基于 GORM + PostgreSQL 实现的股票数据仓库
type StockStore struct {
	db *gorm.DB
}

// NewStockStore 创建 StockStore 实例
func NewStockStore(db *gorm.DB) *StockStore {
	return &StockStore{db: db}
}

// ListStocks 返回所有股票，按 stock_code 排序
func (s *StockStore) ListStocks() []stockdomain.StockItem {
	items, _ := s.ListStocksWithError()
	return items
}

// ListStocksWithError 返回所有股票并保留数据库错误
func (s *StockStore) ListStocksWithError() ([]stockdomain.StockItem, error) {
	var models []Stock
	if err := s.db.Order("stock_code").Find(&models).Error; err != nil {
		return nil, err
	}
	items := make([]stockdomain.StockItem, 0, len(models))
	for _, m := range models {
		items = append(items, stockModelToDTO(m))
	}
	return items, nil
}

// FindStock 根据股票代码查找股票
func (s *StockStore) FindStock(code string) (stockdomain.StockItem, bool) {
	var m Stock
	if err := s.db.Where("stock_code = ?", code).First(&m).Error; err != nil {
		return stockdomain.StockItem{}, false
	}
	return stockModelToDTO(m), true
}

// CountStocks 返回股票总数
func (s *StockStore) CountStocks() int {
	var count int64
	s.db.Model(&Stock{}).Count(&count)
	return int(count)
}

// ReplaceStocks 在事务中替换所有股票（先删除再批量插入）
func (s *StockStore) ReplaceStocks(stocks []stockdomain.StockItem) error {
	if len(stocks) == 0 {
		return fmt.Errorf("no valid stocks to store")
	}

	models := make([]Stock, 0, len(stocks))
	for _, s := range stocks {
		if s.StockCode == "" {
			continue
		}
		models = append(models, stockDTOToModel(s))
	}
	if len(models) == 0 {
		return fmt.Errorf("no valid stocks to store")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("1 = 1").Delete(&Stock{}).Error; err != nil {
			return fmt.Errorf("delete stocks: %w", err)
		}
		if err := tx.CreateInBatches(models, 500).Error; err != nil {
			return fmt.Errorf("insert stocks: %w", err)
		}
		return nil
	})
}

// IsLoaded 判断数据库中是否存在股票数据
func (s *StockStore) IsLoaded() bool {
	var count int64
	s.db.Model(&Stock{}).Count(&count)
	return count > 0
}

// SaveStockList 使用 OnConflict 更新插入股票列表（主键冲突时更新）
func (s *StockStore) SaveStockList(stocks []stockdomain.StockItem) error {
	if len(stocks) == 0 {
		return nil
	}

	models := make([]Stock, 0, len(stocks))
	for _, s := range stocks {
		if s.StockCode == "" {
			continue
		}
		models = append(models, stockDTOToModel(s))
	}
	if len(models) == 0 {
		return nil
	}

	return s.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "stock_code"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"stock_name", "market", "industry", "list_date",
			"total_shares", "float_shares", "current_price", "change_pct",
			"volume", "amount", "turnover_rate", "pe_ratio", "pb_ratio",
			"total_mv", "pinyin", "pinyin_alt",
		}),
	}).CreateInBatches(models, 500).Error
}

// GetStockList 返回所有股票，按 stock_code 排序
func (s *StockStore) GetStockList() ([]stockdomain.StockItem, error) {
	var models []Stock
	if err := s.db.Order("stock_code").Find(&models).Error; err != nil {
		return nil, err
	}
	items := make([]stockdomain.StockItem, 0, len(models))
	for _, m := range models {
		items = append(items, stockModelToDTO(m))
	}
	return items, nil
}

// stockDTOToModel 将领域层 StockItem 转换为 GORM Stock 模型。
// Pinyin 和 PinyinAlt 会合并：若 PinyinAlt 非空，则 Pinyin 设为 "Pinyin PinyinAlt"。
func stockDTOToModel(s stockdomain.StockItem) Stock {
	pinyin := s.Pinyin
	if s.PinyinAlt != "" {
		pinyin = strings.TrimSpace(pinyin + " " + s.PinyinAlt)
	}
	return Stock{
		StockCode:    s.StockCode,
		StockName:    s.StockName,
		Market:       s.Market,
		Industry:     s.Industry,
		ListDate:     s.ListDate,
		TotalShares:  s.TotalShares,
		FloatShares:  s.FloatShares,
		CurrentPrice: s.CurrentPrice,
		ChangePct:    s.ChangePct,
		Volume:       s.Volume,
		Amount:       s.Amount,
		TurnoverRate: s.TurnoverRate,
		PERatio:      s.PERatio,
		PBRatio:      s.PBRatio,
		TotalMV:      s.TotalMV,
		Pinyin:       pinyin,
		PinyinAlt:    s.PinyinAlt,
	}
}

// stockModelToDTO 将 GORM Stock 模型转换为领域层 StockItem
func stockModelToDTO(m Stock) stockdomain.StockItem {
	return stockdomain.StockItem{
		StockCode:    m.StockCode,
		StockName:    m.StockName,
		Market:       m.Market,
		Industry:     m.Industry,
		ListDate:     m.ListDate,
		TotalShares:  m.TotalShares,
		FloatShares:  m.FloatShares,
		CurrentPrice: m.CurrentPrice,
		ChangePct:    m.ChangePct,
		Volume:       m.Volume,
		Amount:       m.Amount,
		TurnoverRate: m.TurnoverRate,
		PERatio:      m.PERatio,
		PBRatio:      m.PBRatio,
		TotalMV:      m.TotalMV,
		Pinyin:       m.Pinyin,
		PinyinAlt:    m.PinyinAlt,
	}
}
