package database

import (
	"time"

	"gorm.io/gorm"
)

// HSGTFlowDaily 沪深港通每日资金流向数据表
// 存储北向资金（陆股通）和南向资金（港股通）的每日汇总数据
// 按交易日期保存，支持滚动存储（保留近一年数据）
type HSGTFlowDaily struct {
	ID            int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Date          string    `gorm:"uniqueIndex;size:10;not null" json:"date"` // 交易日期，格式 YYYY-MM-DD
	UpdateTime    string    `gorm:"size:20;default:''" json:"update_time"`    // 数据更新时间，格式 HH:MM:SS
	
	// 北向资金（陆股通）
	NorthSHBuy    float64   `gorm:"default:0" json:"north_sh_buy"`    // 沪股通净买入（万元）
	NorthSZBuy    float64   `gorm:"default:0" json:"north_sz_buy"`    // 深股通净买入（万元）
	NorthTotalBuy float64   `gorm:"default:0" json:"north_total_buy"` // 北向资金合计净买入（万元）
	NorthTotalAmt float64   `gorm:"default:0" json:"north_total_amt"` // 北向资金合计成交额（万元）
	NorthSHAmt    float64   `gorm:"default:0" json:"north_sh_amt"`    // 沪股通成交额（万元）
	NorthSZAmt    float64   `gorm:"default:0" json:"north_sz_amt"`    // 深股通成交额（万元）
	
	// 南向资金（港股通）
	SouthHKBuy    float64   `gorm:"default:0" json:"south_hk_buy"`    // 香港买入沪深港通（万元）
	SouthSHBuy    float64   `gorm:"default:0" json:"south_sh_buy"`    // 香港买入沪股通（万元）
	SouthSZBuy    float64   `gorm:"default:0" json:"south_sz_buy"`    // 香港买入深股通（万元）
	SouthTotalBuy float64   `gorm:"default:0" json:"south_total_buy"` // 南向资金合计（万元）
	
	// 元数据
	Source        string    `gorm:"size:50;default:'eastmoney'" json:"source"` // 数据来源
	Status        string    `gorm:"size:20;default:'completed'" json:"status"` // 数据状态：completed、partial、failed
	Note          string    `gorm:"size:200;default:''" json:"note"`           // 备注信息
	
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (HSGTFlowDaily) TableName() string {
	return "hsgt_flow_daily"
}

// HSGTFlowDailyStore 提供 HSGT 每日流向数据的数据库操作
type HSGTFlowDailyStore struct {
	db *gorm.DB
}

// NewHSGTFlowDailyStore 创建 HSGTFlowDailyStore 实例
func NewHSGTFlowDailyStore(db *gorm.DB) *HSGTFlowDailyStore {
	return &HSGTFlowDailyStore{db: db}
}

// SaveDaily 保存或更新每日数据
func (s *HSGTFlowDailyStore) SaveDaily(flow *HSGTFlowDaily) error {
	if flow == nil {
		return nil
	}
	// 先检查是否存在
	var existing HSGTFlowDaily
	if err := s.db.Where("date = ?", flow.Date).First(&existing).Error; err == nil {
		// 存在则更新
		return s.db.Model(&existing).Updates(flow).Error
	} else if err == gorm.ErrRecordNotFound {
		// 不存在则创建
		return s.db.Create(flow).Error
	} else {
		return err
	}
}

// GetByDate 按日期获取数据
func (s *HSGTFlowDailyStore) GetByDate(date string) (*HSGTFlowDaily, error) {
	var flow HSGTFlowDaily
	if err := s.db.Where("date = ?", date).First(&flow).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &flow, nil
}

// ListRecent 获取最近 N 天的数据，按日期倒序
func (s *HSGTFlowDailyStore) ListRecent(days int) ([]HSGTFlowDaily, error) {
	var flows []HSGTFlowDaily
	err := s.db.Order("date DESC").Limit(days).Find(&flows).Error
	return flows, err
}

// ListRange 按日期范围查询，包含起始日期
func (s *HSGTFlowDailyStore) ListRange(startDate, endDate string) ([]HSGTFlowDaily, error) {
	var flows []HSGTFlowDaily
	err := s.db.Where("date >= ? AND date <= ?", startDate, endDate).Order("date").Find(&flows).Error
	return flows, err
}

// DeleteBefore 删除指定日期之前的数据
func (s *HSGTFlowDailyStore) DeleteBefore(date string) (int64, error) {
	result := s.db.Where("date < ?", date).Delete(&HSGTFlowDaily{})
	return result.RowsAffected, result.Error
}

// Count 获取数据行数
func (s *HSGTFlowDailyStore) Count() (int64, error) {
	var count int64
	err := s.db.Model(&HSGTFlowDaily{}).Count(&count).Error
	return count, err
}

// DeleteOlderThanOneYear 删除超过一年的数据，保持滚动存储
func (s *HSGTFlowDailyStore) DeleteOlderThanOneYear() (int64, error) {
	// 计算一年前的日期
	oneYearAgo := time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	return s.DeleteBefore(oneYearAgo)
}
