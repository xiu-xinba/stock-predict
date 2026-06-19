package database

import (
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	funddomain "stock-predict-go/internal/domain/fund"
)

// FundStore 基于 GORM 实现的基金数据仓库
type FundStore struct {
	db *gorm.DB
}

// NewFundStore 创建 FundStore 实例
func NewFundStore(db *gorm.DB) *FundStore {
	return &FundStore{db: db}
}

// LoadFunds 返回所有基金，按 fund_code 排序
func (s *FundStore) LoadFunds() []funddomain.FundItem {
	return s.ListFunds()
}

// SaveFunds 用给定列表替换所有基金数据
func (s *FundStore) SaveFunds(funds []funddomain.FundItem) error {
	return s.ReplaceFunds(funds)
}

// GetFunds 根据基金代码列表批量查询基金
func (s *FundStore) GetFunds(codes []string) []funddomain.FundItem {
	if len(codes) == 0 {
		return nil
	}
	var models []Fund
	if err := s.db.Where("fund_code IN ?", codes).Find(&models).Error; err != nil {
		return nil
	}
	items := make([]funddomain.FundItem, len(models))
	for i, m := range models {
		items[i] = fundModelToDTO(m)
	}
	return items
}

// AddFund 添加单只基金（upsert）
func (s *FundStore) AddFund(item funddomain.FundItem) error {
	return s.MergeFunds([]funddomain.FundItem{item})
}

// RemoveFund 根据基金代码删除基金
func (s *FundStore) RemoveFund(code string) error {
	code = normalizeFundCode(code)
	if code == "" {
		return fmt.Errorf("invalid fund code")
	}
	return s.db.Where("fund_code = ?", code).Delete(&Fund{}).Error
}

// FindFund 根据基金代码查找单只基金
func (s *FundStore) FindFund(code string) (funddomain.FundItem, bool) {
	code = normalizeFundCode(code)
	if code == "" {
		return funddomain.FundItem{}, false
	}
	var m Fund
	if err := s.db.Where("fund_code = ?", code).First(&m).Error; err != nil {
		return funddomain.FundItem{}, false
	}
	return fundModelToDTO(m), true
}

// CountFunds 返回基金总数
func (s *FundStore) CountFunds() int {
	var count int64
	s.db.Model(&Fund{}).Count(&count)
	return int(count)
}

// CountQuotedFunds 返回拥有行情数据的基金数量
func (s *FundStore) CountQuotedFunds() int {
	var count int64
	s.db.Model(&Fund{}).Where("quote_source != '' AND quote_source IS NOT NULL").Count(&count)
	return int(count)
}

// ListFunds 返回所有基金，按 fund_code 排序
func (s *FundStore) ListFunds() []funddomain.FundItem {
	items, _ := s.ListFundsWithError()
	return items
}

// ListFundsWithError 返回所有基金并保留数据库错误
func (s *FundStore) ListFundsWithError() ([]funddomain.FundItem, error) {
	var models []Fund
	if err := s.db.Order("fund_code").Find(&models).Error; err != nil {
		return nil, err
	}
	items := make([]funddomain.FundItem, len(models))
	for i, m := range models {
		items[i] = fundModelToDTO(m)
	}
	return items, nil
}

// CoverageReport 使用 GROUP BY 查询生成基金覆盖率报告
func (s *FundStore) CoverageReport() *funddomain.CoverageReport {
	report := &funddomain.CoverageReport{
		CountsByFundType:    make(map[string]int),
		CountsByQuoteSource: make(map[string]int),
	}

	var totalFunds int64
	s.db.Model(&Fund{}).Count(&totalFunds)
	report.TotalFunds = int(totalFunds)

	var fundsWithQuote int64
	s.db.Model(&Fund{}).Where("quote_source != '' AND quote_source IS NOT NULL").Count(&fundsWithQuote)
	report.FundsWithQuote = int(fundsWithQuote)

	type groupResult struct {
		Key   string
		Count int
	}

	var typeResults []groupResult
	s.db.Model(&Fund{}).Select("fund_type as key, COUNT(*) as count").
		Where("fund_type != '' AND fund_type IS NOT NULL").
		Group("fund_type").
		Find(&typeResults)
	for _, r := range typeResults {
		report.CountsByFundType[r.Key] = r.Count
	}

	var sourceResults []groupResult
	s.db.Model(&Fund{}).Select("quote_source as key, COUNT(*) as count").
		Where("quote_source != '' AND quote_source IS NOT NULL").
		Group("quote_source").
		Find(&sourceResults)
	for _, r := range sourceResults {
		report.CountsByQuoteSource[r.Key] = r.Count
	}

	return report
}

// ReplaceFunds 在事务中替换所有基金（先删除再批量插入）
func (s *FundStore) ReplaceFunds(funds []funddomain.FundItem) error {
	models := make([]Fund, 0, len(funds))
	for _, f := range funds {
		f.FundCode = normalizeFundCode(f.FundCode)
		if f.FundCode == "" {
			continue
		}
		models = append(models, fundDTOToModel(f))
	}
	if len(models) == 0 {
		return fmt.Errorf("no valid funds to store")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("1 = 1").Delete(&Fund{}).Error; err != nil {
			return err
		}
		return tx.CreateInBatches(models, 500).Error
	})
}

// MergeFunds 使用 ON CONFLICT DO UPDATE 合并基金数据，
// 非零值覆盖已有记录，零值保留原值。
func (s *FundStore) MergeFunds(funds []funddomain.FundItem) error {
	models := make([]Fund, 0, len(funds))
	for _, f := range funds {
		f.FundCode = normalizeFundCode(f.FundCode)
		if f.FundCode == "" {
			continue
		}
		models = append(models, fundDTOToModel(f))
	}
	if len(models) == 0 {
		return fmt.Errorf("no valid funds to store")
	}

	return s.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "fund_code"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"fund_name":      gorm.Expr("COALESCE(NULLIF(EXCLUDED.fund_name, ''), funds.fund_name)"),
			"fund_type":      gorm.Expr("COALESCE(NULLIF(EXCLUDED.fund_type, ''), funds.fund_type)"),
			"pinyin_abbr":    gorm.Expr("COALESCE(NULLIF(EXCLUDED.pinyin_abbr, ''), funds.pinyin_abbr)"),
			"pinyin_full":    gorm.Expr("COALESCE(NULLIF(EXCLUDED.pinyin_full, ''), funds.pinyin_full)"),
			"company":        gorm.Expr("COALESCE(NULLIF(EXCLUDED.company, ''), funds.company)"),
			"manager":        gorm.Expr("COALESCE(NULLIF(EXCLUDED.manager, ''), funds.manager)"),
			"latest_nav":     gorm.Expr("CASE WHEN EXCLUDED.latest_nav = 0 THEN funds.latest_nav ELSE EXCLUDED.latest_nav END"),
			"cumulative_nav": gorm.Expr("CASE WHEN EXCLUDED.cumulative_nav = 0 THEN funds.cumulative_nav ELSE EXCLUDED.cumulative_nav END"),
			"return1_m":      gorm.Expr("CASE WHEN EXCLUDED.return1_m = 0 THEN funds.return1_m ELSE EXCLUDED.return1_m END"),
			"return3_m":      gorm.Expr("CASE WHEN EXCLUDED.return3_m = 0 THEN funds.return3_m ELSE EXCLUDED.return3_m END"),
			"return6_m":      gorm.Expr("CASE WHEN EXCLUDED.return6_m = 0 THEN funds.return6_m ELSE EXCLUDED.return6_m END"),
			"return1_y":      gorm.Expr("CASE WHEN EXCLUDED.return1_y = 0 THEN funds.return1_y ELSE EXCLUDED.return1_y END"),
			"return3_y":      gorm.Expr("CASE WHEN EXCLUDED.return3_y = 0 THEN funds.return3_y ELSE EXCLUDED.return3_y END"),
			"risk_level":     gorm.Expr("COALESCE(NULLIF(EXCLUDED.risk_level, ''), funds.risk_level)"),
			"inception_date": gorm.Expr("COALESCE(NULLIF(EXCLUDED.inception_date, ''), funds.inception_date)"),
			"estimated_nav":  gorm.Expr("CASE WHEN EXCLUDED.estimated_nav = 0 THEN funds.estimated_nav ELSE EXCLUDED.estimated_nav END"),
			"change_pct":     gorm.Expr("CASE WHEN EXCLUDED.change_pct = 0 THEN funds.change_pct ELSE EXCLUDED.change_pct END"),
			"quote_date":     gorm.Expr("COALESCE(NULLIF(EXCLUDED.quote_date, ''), funds.quote_date)"),
			"quote_source":   gorm.Expr("COALESCE(NULLIF(EXCLUDED.quote_source, ''), funds.quote_source)"),
			"updated_at":     gorm.Expr("NOW()"),
		}),
	}).CreateInBatches(models, 500).Error
}

// MergeFundUniverse 合并基金基础信息（委托给 MergeFunds）
func (s *FundStore) MergeFundUniverse(funds []funddomain.FundItem) error {
	return s.MergeFunds(funds)
}

// SyncFundsFromCSV 从 CSV 文件读取基金数据并合并到数据库
func (s *FundStore) SyncFundsFromCSV(path string) (int, error) {
	funds, err := ReadFundsCSV(path)
	if err != nil {
		return 0, err
	}
	if err := s.MergeFunds(funds); err != nil {
		return 0, err
	}
	return len(funds), nil
}

// SyncFundsFromEastmoneyURL 从单个东方财富 URL 同步基金数据
func (s *FundStore) SyncFundsFromEastmoneyURL(sourceURL string) (int, error) {
	return s.SyncFundsFromEastmoneySources(sourceURL, "")
}

// SyncFundsFromEastmoneySources 从东方财富基础信息 URL 和/或行情指标 URL 同步基金数据
func (s *FundStore) SyncFundsFromEastmoneySources(sourceURL, metricsURL string) (int, error) {
	imported := 0
	if sourceURL != "" {
		payload, err := fetchEastmoneyPayload(sourceURL, 10<<20)
		if err != nil {
			return imported, fmt.Errorf("fund universe request failed: %w", err)
		}
		funds, err := ReadEastmoneyFundCodeSearchJS(payload)
		if err != nil {
			return imported, err
		}
		if err := s.MergeFundUniverse(funds); err != nil {
			return imported, err
		}
		imported += len(funds)
	}
	if metricsURL != "" {
		payload, err := fetchEastmoneyPayload(metricsURL, 50<<20)
		if err != nil {
			return imported, fmt.Errorf("fund metrics request failed: %w", err)
		}
		funds, err := ReadEastmoneyFundRankHandlerJS(payload)
		if err != nil {
			return imported, err
		}
		if err := s.MergeFunds(funds); err != nil {
			return imported, err
		}
		imported += len(funds)
	}
	return imported, nil
}

// DataPath 返回空字符串（GORM 仓库无文件路径）
func (s *FundStore) DataPath() string {
	return ""
}

// fundDTOToModel 将领域层 FundItem 转换为 GORM Fund 模型
func fundDTOToModel(f funddomain.FundItem) Fund {
	return Fund{
		FundCode:      f.FundCode,
		FundName:      f.FundName,
		FundType:      f.FundType,
		PinyinAbbr:    f.PinyinAbbr,
		PinyinFull:    f.PinyinFull,
		Company:       f.Company,
		Manager:       f.Manager,
		LatestNAV:     f.LatestNAV,
		CumulativeNAV: f.CumulativeNAV,
		Return1M:      f.Return1M,
		Return3M:      f.Return3M,
		Return6M:      f.Return6M,
		Return1Y:      f.Return1Y,
		Return3Y:      f.Return3Y,
		RiskLevel:     f.RiskLevel,
		InceptionDate: f.InceptionDate,
		EstimatedNAV:  f.EstimatedNAV,
		ChangePct:     f.ChangePct,
		QuoteDate:     f.QuoteDate,
		QuoteSource:   f.QuoteSource,
	}
}

// fundModelToDTO 将 GORM Fund 模型转换为领域层 FundItem
func fundModelToDTO(m Fund) funddomain.FundItem {
	return funddomain.FundItem{
		FundCode:      m.FundCode,
		FundName:      m.FundName,
		FundType:      m.FundType,
		PinyinAbbr:    m.PinyinAbbr,
		PinyinFull:    m.PinyinFull,
		Company:       m.Company,
		Manager:       m.Manager,
		LatestNAV:     m.LatestNAV,
		CumulativeNAV: m.CumulativeNAV,
		Return1M:      m.Return1M,
		Return3M:      m.Return3M,
		Return6M:      m.Return6M,
		Return1Y:      m.Return1Y,
		Return3Y:      m.Return3Y,
		RiskLevel:     m.RiskLevel,
		InceptionDate: m.InceptionDate,
		EstimatedNAV:  m.EstimatedNAV,
		ChangePct:     m.ChangePct,
		QuoteDate:     m.QuoteDate,
		QuoteSource:   m.QuoteSource,
	}
}
