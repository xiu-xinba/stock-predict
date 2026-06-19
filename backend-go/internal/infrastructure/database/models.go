package database

import "time"

// Fund 基金信息表，存储基金基础数据及行情指标
type Fund struct {
	FundCode      string    `gorm:"primaryKey;size:6" json:"fund_code"`
	FundName      string    `gorm:"size:60;not null" json:"fund_name"`
	FundType      string    `gorm:"size:30;default:''" json:"fund_type"`       // 基金类型，如"混合型""股票型"
	PinyinAbbr    string    `gorm:"size:40;default:''" json:"pinyin_abbr"`     // 拼音缩写，用于搜索匹配
	PinyinFull    string    `gorm:"size:100;default:''" json:"pinyin_full"`    // 拼音全称，用于搜索匹配
	Company       string    `gorm:"size:60;default:''" json:"company"`         // 基金公司
	Manager       string    `gorm:"size:40;default:''" json:"manager"`         // 基金经理
	LatestNAV     float64   `gorm:"default:0" json:"latest_nav"`               // 最新净值（Net Asset Value）
	CumulativeNAV float64   `gorm:"default:0" json:"cumulative_nav"`           // 累计净值
	Return1M      float64   `gorm:"default:0" json:"return_1m"`                // 近1月收益率
	Return3M      float64   `gorm:"default:0" json:"return_3m"`                // 近3月收益率
	Return6M      float64   `gorm:"default:0" json:"return_6m"`                // 近6月收益率
	Return1Y      float64   `gorm:"default:0" json:"return_1y"`                // 近1年收益率
	Return3Y      float64   `gorm:"default:0" json:"return_3y"`                // 近3年收益率
	RiskLevel     string    `gorm:"size:10;default:''" json:"risk_level"`      // 风险等级，如"中高""高"
	InceptionDate string    `gorm:"size:20;default:''" json:"inception_date"`  // 成立日期
	EstimatedNAV  float64   `gorm:"default:0" json:"estimated_nav"`            // 估算净值
	ChangePct     float64   `gorm:"default:0" json:"change_pct"`               // 涨跌幅（百分比）
	QuoteDate     string    `gorm:"size:20;default:''" json:"quote_date"`      // 行情日期
	QuoteSource   string    `gorm:"size:30;default:''" json:"quote_source"`    // 行情数据来源，如"csv""eastmoney_rank"
	Industry      string    `gorm:"size:30;default:''" json:"industry"`        // 所属行业
	ListDate      string    `gorm:"size:20;default:''" json:"list_date"`       // 上市日期
	UpdatedAt     time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Fund) TableName() string { return "funds" }

// Stock 股票信息表，存储股票基础数据及行情指标
type Stock struct {
	StockCode    string  `gorm:"primaryKey;size:6" json:"stock_code"`
	StockName    string  `gorm:"size:60;not null" json:"stock_name"`
	Market       string  `gorm:"size:10;default:''" json:"market"`          // 所属市场，如"SH""SZ"
	Industry     string  `gorm:"size:40;default:''" json:"industry"`        // 所属行业
	ListDate     string  `gorm:"size:20;default:''" json:"list_date"`       // 上市日期
	TotalShares  float64 `gorm:"default:0" json:"total_shares"`             // 总股本
	FloatShares  float64 `gorm:"default:0" json:"float_shares"`             // 流通股本
	CurrentPrice float64 `gorm:"default:0" json:"current_price"`            // 当前价格
	ChangePct    float64 `gorm:"default:0" json:"change_pct"`               // 涨跌幅（百分比）
	Volume       float64 `gorm:"default:0" json:"volume"`                   // 成交量
	Amount       float64 `gorm:"default:0" json:"amount"`                   // 成交额
	TurnoverRate float64 `gorm:"default:0" json:"turnover_rate"`            // 换手率
	PERatio      float64 `gorm:"default:0" json:"pe_ratio"`                 // 市盈率（Price-to-Earnings）
	PBRatio      float64 `gorm:"default:0" json:"pb_ratio"`                 // 市净率（Price-to-Book）
	TotalMV      float64 `gorm:"default:0" json:"total_mv"`                 // 总市值（Market Value）
	Pinyin       string  `gorm:"size:60;default:''" json:"pinyin"`           // 拼音，用于搜索匹配
	PinyinAlt    string  `gorm:"size:60;default:''" json:"pinyin_alt"`      // 备选拼音
}

func (Stock) TableName() string { return "stocks" }

// IndexQuote 指数行情表，存储主要指数的实时/快照行情数据
type IndexQuote struct {
	Code       string    `gorm:"primaryKey;size:10" json:"code"`
	Name       string    `gorm:"size:40" json:"name"`
	Market     string    `gorm:"size:10" json:"market"`                     // 所属市场
	Value      float64   `gorm:"default:0" json:"value"`                    // 指数点位
	Change     float64   `gorm:"default:0" json:"change"`                   // 涨跌额
	ChangePct  float64   `gorm:"default:0" json:"change_pct"`               // 涨跌幅（百分比）
	Open       float64   `gorm:"default:0" json:"open"`                     // 开盘价
	High       float64   `gorm:"default:0" json:"high"`                     // 最高价
	Low        float64   `gorm:"default:0" json:"low"`                      // 最低价
	PrevClose  float64   `gorm:"default:0" json:"prev_close"`               // 昨收价
	Volume     int64     `gorm:"default:0" json:"volume"`                   // 成交量
	Amount     float64   `gorm:"default:0" json:"amount"`                   // 成交额
	UpdatedAt  time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	DataSource string    `gorm:"size:30" json:"data_source"`                // 数据来源
}

func (IndexQuote) TableName() string { return "index_quotes" }

// IndexMinute 指数分钟线表，存储指数日内分时数据
type IndexMinute struct {
	ID        uint    `gorm:"primaryKey;autoIncrement" json:"id"`
	Code      string  `gorm:"size:10;uniqueIndex:idx_code_date_time" json:"code"`
	TradeDate string  `gorm:"size:20;uniqueIndex:idx_code_date_time" json:"trade_date"` // 交易日期
	Time      string  `gorm:"size:10;uniqueIndex:idx_code_date_time" json:"time"`      // 分时时间点
	Price     float64 `gorm:"default:0" json:"price"`                                   // 成交价格
	AvgPrice  float64 `gorm:"default:0" json:"avg_price"`                               // 均价
	Volume    int64   `gorm:"default:0" json:"volume"`                                  // 成交量
}

func (IndexMinute) TableName() string { return "index_minutes" }

// IndexKlineDaily 指数日K线表，存储指数每日 OHLCV 数据
type IndexKlineDaily struct {
	ID     uint    `gorm:"primaryKey;autoIncrement" json:"id"`
	Code   string  `gorm:"size:10;uniqueIndex:idx_ikd_code_date" json:"code"`
	Date   string  `gorm:"size:20;uniqueIndex:idx_ikd_code_date" json:"date"`
	Open   float64 `gorm:"default:0" json:"open"`
	Close  float64 `gorm:"default:0" json:"close"`
	High   float64 `gorm:"default:0" json:"high"`
	Low    float64 `gorm:"default:0" json:"low"`
	Volume int64   `gorm:"default:0" json:"volume"`
	Amount float64 `gorm:"default:0" json:"amount"`
}

func (IndexKlineDaily) TableName() string { return "index_kline_daily" }

// KlineDaily 日K线表，存储个股每日 OHLCV 数据
type KlineDaily struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Code      string    `gorm:"size:10;uniqueIndex:idx_kd_code_date" json:"code"`
	Date      string    `gorm:"size:20;uniqueIndex:idx_kd_code_date" json:"date"`
	Open      float64   `gorm:"default:0" json:"open"`
	Close     float64   `gorm:"default:0" json:"close"`
	High      float64   `gorm:"default:0" json:"high"`
	Low       float64   `gorm:"default:0" json:"low"`
	Volume    int64     `gorm:"default:0" json:"volume"`
	Amount    float64   `gorm:"default:0" json:"amount"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (KlineDaily) TableName() string { return "kline_daily" }

// KlineWeekly 周K线表，存储个股每周 OHLCV 数据
type KlineWeekly struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Code      string    `gorm:"size:10;uniqueIndex:idx_kw_code_date" json:"code"`
	Date      string    `gorm:"size:20;uniqueIndex:idx_kw_code_date" json:"date"`
	Open      float64   `gorm:"default:0" json:"open"`
	Close     float64   `gorm:"default:0" json:"close"`
	High      float64   `gorm:"default:0" json:"high"`
	Low       float64   `gorm:"default:0" json:"low"`
	Volume    int64     `gorm:"default:0" json:"volume"`
	Amount    float64   `gorm:"default:0" json:"amount"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (KlineWeekly) TableName() string { return "kline_weekly" }

// KlineMonthly 月K线表，存储个股每月 OHLCV 数据
type KlineMonthly struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Code      string    `gorm:"size:10;uniqueIndex:idx_km_code_date" json:"code"`
	Date      string    `gorm:"size:20;uniqueIndex:idx_km_code_date" json:"date"`
	Open      float64   `gorm:"default:0" json:"open"`
	Close     float64   `gorm:"default:0" json:"close"`
	High      float64   `gorm:"default:0" json:"high"`
	Low       float64   `gorm:"default:0" json:"low"`
	Volume    int64     `gorm:"default:0" json:"volume"`
	Amount    float64   `gorm:"default:0" json:"amount"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (KlineMonthly) TableName() string { return "kline_monthly" }

// Financial 财务数据表，存储个股季度财务指标
type Financial struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Code        string    `gorm:"size:10;uniqueIndex:idx_code_report_date" json:"code"`
	ReportDate  string    `gorm:"size:20;uniqueIndex:idx_code_report_date" json:"report_date"` // 财报日期
	PE          float64   `gorm:"default:0" json:"pe"`                                         // 市盈率（Price-to-Earnings）
	PB          float64   `gorm:"default:0" json:"pb"`                                         // 市净率（Price-to-Book）
	ROE         float64   `gorm:"default:0" json:"roe"`                                        // 净资产收益率（Return on Equity）
	EPS         float64   `gorm:"default:0" json:"eps"`                                        // 每股收益（Earnings Per Share）
	Revenue     float64   `gorm:"default:0" json:"revenue"`                                    // 营业收入
	NetProfit   float64   `gorm:"default:0" json:"net_profit"`                                 // 净利润
	GrossMargin float64   `gorm:"default:0" json:"gross_margin"`                               // 毛利率
	NetMargin   float64   `gorm:"default:0" json:"net_margin"`                                 // 净利率
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Financial) TableName() string { return "financials" }

// CacheMetadata 缓存元数据表，记录各数据源的缓存状态信息
type CacheMetadata struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Code        string    `gorm:"size:10;uniqueIndex:idx_code_data_type" json:"code"`
	DataType    string    `gorm:"size:30;uniqueIndex:idx_code_data_type" json:"data_type"` // 数据类型，如"kline_daily""financial"
	Source      string    `gorm:"size:30" json:"source"`                                   // 数据来源
	StartDate   string    `gorm:"size:20" json:"start_date"`                               // 数据起始日期
	EndDate     string    `gorm:"size:20" json:"end_date"`                                 // 数据截止日期
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	RecordCount int       `gorm:"default:0" json:"record_count"`                           // 记录条数
}

func (CacheMetadata) TableName() string { return "cache_metadata" }

// StockRanking 股票排名表，按排名类型存储序列化后的排名数据
type StockRanking struct {
	RankingType string    `gorm:"primaryKey;size:20" json:"ranking_type"` // 排名类型，如"limit_up""limit_down"
	Data        string    `gorm:"type:text" json:"data"`                  // 排名数据（JSON 格式）
	DataSource  string    `gorm:"size:30" json:"data_source"`             // 数据来源
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (StockRanking) TableName() string { return "stock_ranking" }
