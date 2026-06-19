package database

import (
	"encoding/json"
	"log/slog"
	"sort"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	funddomain "stock-predict-go/internal/domain/fund"
	marketdomain "stock-predict-go/internal/domain/market"
	stockdomain "stock-predict-go/internal/domain/stock"
)

// MarketStore 基于 GORM 实现的市场数据持久化仓库，
// 合并了原 MarketStore 和 MarketCacheStore 的功能。
type MarketStore struct {
	db     *gorm.DB
	logger *slog.Logger
}

// NewMarketStore 创建 MarketStore 实例
func NewMarketStore(db *gorm.DB, logger *slog.Logger) *MarketStore {
	return &MarketStore{db: db, logger: logger}
}

// Close 关闭底层数据库连接
func (ms *MarketStore) Close() error {
	sqlDB, err := ms.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// --- Index Quotes ---

// SaveIndexQuotes 使用 OnConflict 按 code 更新插入指数行情数据
func (ms *MarketStore) SaveIndexQuotes(indices []marketdomain.MarketIndex) error {
	if len(indices) == 0 {
		return nil
	}

	models := make([]IndexQuote, 0, len(indices))
	for _, idx := range indices {
		models = append(models, IndexQuote{
			Code:       idx.Code,
			Name:       idx.Name,
			Market:     idx.Market,
			Value:      idx.Value,
			Change:     idx.Change,
			ChangePct:  idx.ChangePct,
			High:       idx.High,
			Low:        idx.Low,
			PrevClose:  idx.PrevClose,
			Volume:     int64(idx.Volume),
			DataSource: idx.DataSource,
		})
	}

	return ms.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "code"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"name", "market", "value", "change", "change_pct",
			"high", "low", "prev_close", "volume", "data_source",
		}),
	}).CreateInBatches(models, 500).Error
}

// LoadIndexQuotes 加载所有指数行情并转换为领域层 MarketIndex
func (ms *MarketStore) LoadIndexQuotes() []marketdomain.MarketIndex {
	var models []IndexQuote
	if err := ms.db.Find(&models).Error; err != nil {
		ms.logger.Warn("load index quotes failed", "error", err)
		return nil
	}

	result := make([]marketdomain.MarketIndex, 0, len(models))
	for _, m := range models {
		result = append(result, marketdomain.MarketIndex{
			Code:       m.Code,
			Name:       m.Name,
			Market:     m.Market,
			Value:      m.Value,
			Change:     m.Change,
			ChangePct:  m.ChangePct,
			High:       m.High,
			Low:        m.Low,
			PrevClose:  m.PrevClose,
			Volume:     float64(m.Volume),
			UpdateTime: m.UpdatedAt.Format("2006-01-02 15:04:05"),
			DataSource: m.DataSource,
		})
	}
	return result
}

// --- Index Minutes ---

// SaveIndexMinutes 使用 OnConflict 按唯一索引 (code, trade_date, time) 更新插入指数分钟线数据
func (ms *MarketStore) SaveIndexMinutes(code string, tradeDate string, points []marketdomain.IndexMinutePoint) error {
	if len(points) == 0 {
		return nil
	}

	models := make([]IndexMinute, 0, len(points))
	for _, p := range points {
		models = append(models, IndexMinute{
			Code:      code,
			TradeDate: tradeDate,
			Time:      p.Time,
			Price:     p.Price,
			AvgPrice:  p.AvgPrice,
			Volume:    p.Volume,
		})
	}

	return ms.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "code"}, {Name: "trade_date"}, {Name: "time"}},
		DoUpdates: clause.AssignmentColumns([]string{"price", "avg_price", "volume"}),
	}).CreateInBatches(models, 500).Error
}

// LoadIndexMinutes 根据指数代码和交易日期加载分钟线数据，按交易时间排序。
// 美股数据跨午夜（21:30-04:00），需特殊排序：21:xx 排在 00:xx 之前。
func (ms *MarketStore) LoadIndexMinutes(code string, tradeDate string) []marketdomain.IndexMinutePoint {
	var models []IndexMinute
	if err := ms.db.Where("code = ? AND trade_date = ?", code, tradeDate).
		Find(&models).Error; err != nil {
		ms.logger.Warn("load index minutes failed", "error", err)
		return nil
	}

	result := make([]marketdomain.IndexMinutePoint, 0, len(models))
	for _, m := range models {
		result = append(result, marketdomain.IndexMinutePoint{
			Time:     m.Time,
			Price:    m.Price,
			AvgPrice: m.AvgPrice,
			Volume:   m.Volume,
		})
	}

	// 按交易时间排序：跨午夜数据（如美股 21:30-04:00）需将 21:xx 排在 00:xx 之前
	sort.SliceStable(result, func(i, j int) bool {
		mi := minuteTotalFromStr(result[i].Time)
		mj := minuteTotalFromStr(result[j].Time)
		if mi >= 21*60 && mj < 21*60 {
			return true
		}
		if mi < 21*60 && mj >= 21*60 {
			return false
		}
		return mi < mj
	})

	return result
}

// minuteTotalFromStr 将 "HH:MM" 转换为当天的分钟总数。
func minuteTotalFromStr(s string) int {
	if len(s) < 5 {
		return 0
	}
	return int(s[0]-'0')*600 + int(s[1]-'0')*60 + int(s[3]-'0')*10 + int(s[4]-'0')
}

// --- Index Kline ---

// SaveIndexKline 使用 OnConflict 按唯一索引 (code, date) 更新插入指数K线数据
func (ms *MarketStore) SaveIndexKline(code string, points []marketdomain.IndexKlinePoint) error {
	if len(points) == 0 {
		return nil
	}

	models := make([]IndexKlineDaily, 0, len(points))
	for _, p := range points {
		models = append(models, IndexKlineDaily{
			Code:   code,
			Date:   p.Date,
			Open:   p.Open,
			Close:  p.Close,
			High:   p.High,
			Low:    p.Low,
			Volume: p.Volume,
			Amount: p.Amount,
		})
	}

	return ms.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "code"}, {Name: "date"}},
		DoUpdates: clause.AssignmentColumns([]string{"open", "close", "high", "low", "volume", "amount"}),
	}).CreateInBatches(models, 500).Error
}

// LoadIndexKline 根据指数代码加载K线数据，按日期降序取 limit 条后反转为升序
func (ms *MarketStore) LoadIndexKline(code string, limit int) []marketdomain.IndexKlinePoint {
	var models []IndexKlineDaily
	if err := ms.db.Where("code = ?", code).
		Order("date DESC").
		Limit(limit).
		Find(&models).Error; err != nil {
		ms.logger.Warn("load index kline failed", "error", err)
		return nil
	}

	// Reverse to ascending order
	for i, j := 0, len(models)-1; i < j; i, j = i+1, j-1 {
		models[i], models[j] = models[j], models[i]
	}

	result := make([]marketdomain.IndexKlinePoint, 0, len(models))
	for _, m := range models {
		result = append(result, marketdomain.IndexKlinePoint{
			Date:   m.Date,
			Open:   m.Open,
			Close:  m.Close,
			High:   m.High,
			Low:    m.Low,
			Volume: m.Volume,
			Amount: m.Amount,
		})
	}
	return result
}

// LoadIndexKlineRange 根据指数代码和日期范围加载K线数据，按日期升序排列
func (ms *MarketStore) LoadIndexKlineRange(code, startDate, endDate string) []marketdomain.IndexKlinePoint {
	var models []IndexKlineDaily
	query := ms.db.Where("code = ?", code)
	if startDate != "" {
		query = query.Where("date >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("date <= ?", endDate)
	}
	if err := query.Order("date ASC").Find(&models).Error; err != nil {
		return nil
	}

	result := make([]marketdomain.IndexKlinePoint, 0, len(models))
	for _, m := range models {
		result = append(result, marketdomain.IndexKlinePoint{
			Date:   m.Date,
			Open:   m.Open,
			Close:  m.Close,
			High:   m.High,
			Low:    m.Low,
			Volume: m.Volume,
			Amount: m.Amount,
		})
	}
	return result
}

// --- Cache Metadata ---

// SaveCacheMetadata 按 (code, data_type) 更新插入缓存元数据
func (ms *MarketStore) SaveCacheMetadata(meta CacheMetadata) error {
	return ms.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "code"}, {Name: "data_type"}},
		DoUpdates: clause.AssignmentColumns([]string{"source", "start_date", "end_date", "updated_at", "record_count"}),
	}).Create(&meta).Error
}

// LoadCacheMetadata 根据代码和数据类型加载缓存元数据
func (ms *MarketStore) LoadCacheMetadata(code, dataType string) (*CacheMetadata, error) {
	var meta CacheMetadata
	if err := ms.db.Where("code = ? AND data_type = ?", code, dataType).First(&meta).Error; err != nil {
		return nil, err
	}
	return &meta, nil
}

// LoadAllCacheMetadata 加载所有缓存元数据记录
func (ms *MarketStore) LoadAllCacheMetadata() ([]CacheMetadata, error) {
	var result []CacheMetadata
	if err := ms.db.Find(&result).Error; err != nil {
		return nil, err
	}
	return result, nil
}

// --- Clean Expired Data ---

// CleanExpiredData 删除超过保留天数的指数分钟线和K线历史数据
func (ms *MarketStore) CleanExpiredData(klineRetentionDays, minuteRetentionDays int) {
	minuteCutoff := time.Now().AddDate(0, 0, -minuteRetentionDays).Format("2006-01-02")

	result := ms.db.Where("trade_date < ?", minuteCutoff).Delete(&IndexMinute{})
	if result.Error != nil {
		ms.logger.Warn("clean expired index_minutes failed", "error", result.Error)
	} else {
		ms.logger.Info("cleaned expired index_minutes", "cutoff", minuteCutoff, "affected", result.RowsAffected)
	}

	klineCutoff := time.Now().AddDate(0, 0, -klineRetentionDays).Format("2006-01-02")

	result = ms.db.Where("date < ?", klineCutoff).Delete(&IndexKlineDaily{})
	if result.Error != nil {
		ms.logger.Warn("clean expired index_kline_daily failed", "error", result.Error)
	} else {
		ms.logger.Info("cleaned expired index_kline_daily", "cutoff", klineCutoff, "affected", result.RowsAffected)
	}
}

// --- Stock Ranking ---

// SaveStockRanking 更新插入股票排名数据，将排名项序列化为 JSON 存储
func (ms *MarketStore) SaveStockRanking(rankingType string, items []stockdomain.StockRankingItem, source string) error {
	jsonData, err := json.Marshal(items)
	if err != nil {
		return err
	}

	ranking := StockRanking{
		RankingType: rankingType,
		Data:        string(jsonData),
		DataSource:  source,
	}

	return ms.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "ranking_type"}},
		DoUpdates: clause.AssignmentColumns([]string{"data", "data_source"}),
	}).Create(&ranking).Error
}

// LoadStockRanking 根据排名类型加载股票排名数据，反序列化 JSON 并填充更新时间和数据来源
func (ms *MarketStore) LoadStockRanking(rankingType string) ([]stockdomain.StockRankingItem, string, error) {
	var ranking StockRanking
	if err := ms.db.Where("ranking_type = ?", rankingType).First(&ranking).Error; err != nil {
		return nil, "", err
	}

	var items []stockdomain.StockRankingItem
	if err := json.Unmarshal([]byte(ranking.Data), &items); err != nil {
		return nil, "", err
	}

	updatedAt := ranking.UpdatedAt.Format("2006-01-02 15:04:05")
	for i := range items {
		items[i].UpdateTime = updatedAt
		items[i].DataSource = ranking.DataSource
	}
	return items, ranking.DataSource, nil
}

// --- K-line Daily (MarketCacheStore) ---

// SaveKlineDaily 保存个股日K线数据
func (ms *MarketStore) SaveKlineDaily(code string, points []marketdomain.IndexKlinePoint) error {
	return ms.saveKline(&KlineDaily{}, code, points)
}

// GetKlineDaily 根据代码和日期范围查询日K线数据
func (ms *MarketStore) GetKlineDaily(code string, startDate, endDate string) ([]marketdomain.IndexKlinePoint, error) {
	return ms.getKline(&KlineDaily{}, code, startDate, endDate)
}

// GetLatestKlineDate 返回指定代码在日K线表中的最新日期
func (ms *MarketStore) GetLatestKlineDate(code string) (string, error) {
	return ms.getLatestKlineDate(&KlineDaily{}, code)
}

// GetKlineDailyCount 返回指定代码的日K线记录数
func (ms *MarketStore) GetKlineDailyCount(code string) (int, error) {
	return ms.getKlineCount(&KlineDaily{}, code)
}

// --- K-line Weekly ---

// SaveKlineWeekly 保存个股周K线数据
func (ms *MarketStore) SaveKlineWeekly(code string, points []marketdomain.IndexKlinePoint) error {
	return ms.saveKline(&KlineWeekly{}, code, points)
}

// GetKlineWeekly 根据代码和日期范围查询周K线数据
func (ms *MarketStore) GetKlineWeekly(code string, startDate, endDate string) ([]marketdomain.IndexKlinePoint, error) {
	return ms.getKline(&KlineWeekly{}, code, startDate, endDate)
}

// --- K-line Monthly ---

// SaveKlineMonthly 保存个股月K线数据
func (ms *MarketStore) SaveKlineMonthly(code string, points []marketdomain.IndexKlinePoint) error {
	return ms.saveKline(&KlineMonthly{}, code, points)
}

// GetKlineMonthly 根据代码和日期范围查询月K线数据
func (ms *MarketStore) GetKlineMonthly(code string, startDate, endDate string) ([]marketdomain.IndexKlinePoint, error) {
	return ms.getKline(&KlineMonthly{}, code, startDate, endDate)
}

// --- Financials ---

// SaveFinancials 更新插入个股财务数据
func (ms *MarketStore) SaveFinancials(code string, data []stockdomain.FinancialQuarter) error {
	if len(data) == 0 {
		return nil
	}

	models := make([]Financial, 0, len(data))
	for _, f := range data {
		models = append(models, Financial{
			Code:        code,
			ReportDate:  f.ReportDate,
			PE:          0,
			PB:          0,
			ROE:         f.ROE,
			EPS:         f.EPS,
			Revenue:     f.Revenue,
			NetProfit:   f.NetProfit,
			GrossMargin: f.GrossMargin,
			NetMargin:   f.NetMargin,
		})
	}

	return ms.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "code"}, {Name: "report_date"}},
		DoUpdates: clause.AssignmentColumns([]string{"pe", "pb", "roe", "eps", "revenue", "net_profit", "gross_margin", "net_margin"}),
	}).CreateInBatches(models, 500).Error
}

// GetFinancials 查询个股财务数据，按报告日期降序排列
func (ms *MarketStore) GetFinancials(code string) ([]stockdomain.FinancialQuarter, error) {
	var models []Financial
	if err := ms.db.Where("code = ?", code).Order("report_date DESC").Find(&models).Error; err != nil {
		return nil, err
	}

	result := make([]stockdomain.FinancialQuarter, 0, len(models))
	for _, m := range models {
		result = append(result, stockdomain.FinancialQuarter{
			ReportDate:  m.ReportDate,
			Revenue:     m.Revenue,
			NetProfit:   m.NetProfit,
			EPS:         m.EPS,
			GrossMargin: m.GrossMargin,
			NetMargin:   m.NetMargin,
			ROE:         m.ROE,
		})
	}
	return result, nil
}

// GetLatestFinancialDate 返回指定代码的最新财报日期
func (ms *MarketStore) GetLatestFinancialDate(code string) (string, error) {
	var date string
	err := ms.db.Model(&Financial{}).
		Select("MAX(report_date)").
		Where("code = ?", code).
		Row().Scan(&date)
	if err != nil {
		return "", err
	}
	return date, nil
}

// --- Fund List ---

// SaveFundList 更新插入基金列表数据
func (ms *MarketStore) SaveFundList(funds []funddomain.FundItem) error {
	if len(funds) == 0 {
		return nil
	}

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

	return ms.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "fund_code"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"fund_name", "fund_type", "pinyin_abbr", "pinyin_full",
			"company", "manager", "latest_nav", "cumulative_nav",
			"return1_m", "return3_m", "return6_m", "return1_y", "return3_y",
			"risk_level", "inception_date", "estimated_nav", "change_pct",
			"quote_date", "quote_source",
		}),
	}).CreateInBatches(models, 500).Error
}

// --- Stock List ---

// SaveStockList 更新插入股票列表数据
func (ms *MarketStore) SaveStockList(stocks []stockdomain.StockItem) error {
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

	return ms.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "stock_code"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"stock_name", "market", "industry", "list_date",
			"total_shares", "float_shares", "current_price", "change_pct",
			"volume", "amount", "turnover_rate", "pe_ratio", "pb_ratio",
			"total_mv", "pinyin", "pinyin_alt",
		}),
	}).CreateInBatches(models, 500).Error
}

// --- Generic K-line helpers ---

// saveKline 通用K线数据更新插入，按唯一索引 (code, date) 冲突时更新
func (ms *MarketStore) saveKline(model interface{}, code string, points []marketdomain.IndexKlinePoint) error {
	if len(points) == 0 {
		return nil
	}

	// Build slice of maps for generic GORM insertion
	records := make([]map[string]interface{}, 0, len(points))
	for _, p := range points {
		records = append(records, map[string]interface{}{
			"code":   code,
			"date":   p.Date,
			"open":   p.Open,
			"close":  p.Close,
			"high":   p.High,
			"low":    p.Low,
			"volume": p.Volume,
			"amount": p.Amount,
		})
	}

	tableName := ms.resolveTableName(model)

	return ms.db.Table(tableName).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "code"}, {Name: "date"}},
		DoUpdates: clause.AssignmentColumns([]string{"open", "close", "high", "low", "volume", "amount"}),
	}).CreateInBatches(records, 500).Error
}

// getKline 通用K线数据查询，根据模型类型、代码和日期范围获取K线数据
func (ms *MarketStore) getKline(model interface{}, code, startDate, endDate string) ([]marketdomain.IndexKlinePoint, error) {
	tableName := ms.resolveTableName(model)

	query := ms.db.Table(tableName).Where("code = ?", code)
	if startDate != "" {
		query = query.Where("date >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("date <= ?", endDate)
	}

	var results []struct {
		Date   string  `gorm:"column:date"`
		Open   float64 `gorm:"column:open"`
		Close  float64 `gorm:"column:close"`
		High   float64 `gorm:"column:high"`
		Low    float64 `gorm:"column:low"`
		Volume int64   `gorm:"column:volume"`
		Amount float64 `gorm:"column:amount"`
	}

	if err := query.Order("date ASC").Find(&results).Error; err != nil {
		return nil, err
	}

	points := make([]marketdomain.IndexKlinePoint, 0, len(results))
	for _, r := range results {
		points = append(points, marketdomain.IndexKlinePoint{
			Date:   r.Date,
			Open:   r.Open,
			Close:  r.Close,
			High:   r.High,
			Low:    r.Low,
			Volume: r.Volume,
			Amount: r.Amount,
		})
	}
	return points, nil
}

// getLatestKlineDate 返回指定代码在给定K线表中的最新日期
func (ms *MarketStore) getLatestKlineDate(model interface{}, code string) (string, error) {
	tableName := ms.resolveTableName(model)

	var date *string
	err := ms.db.Table(tableName).
		Select("MAX(date)").
		Where("code = ?", code).
		Row().Scan(&date)
	if err != nil {
		return "", err
	}
	if date == nil {
		return "", nil
	}
	return *date, nil
}

// getKlineCount 返回指定代码在给定K线表中的记录数
func (ms *MarketStore) getKlineCount(model interface{}, code string) (int, error) {
	tableName := ms.resolveTableName(model)

	var count int64
	err := ms.db.Table(tableName).Where("code = ?", code).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

// resolveTableName 通过 GORM Statement.Parse 可靠地解析模型结构体对应的表名，
// 避免 db.Model(model).Statement.Table 因语句未完全初始化而返回空字符串的问题。
func (ms *MarketStore) resolveTableName(model interface{}) string {
	stmt := &gorm.Statement{DB: ms.db}
	if err := stmt.Parse(model); err != nil {
		ms.logger.Warn("failed to resolve table name", "error", err)
		return ""
	}
	return stmt.Table
}
