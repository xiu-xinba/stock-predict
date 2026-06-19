// Package fund 定义了基金相关的领域模型和数据结构。
package fund

// Direction 表示基金涨跌方向类型。
type Direction string

const (
	DirectionUp   Direction = "up"   // 上涨
	DirectionDown Direction = "down" // 下跌
	DirectionFlat Direction = "flat" // 持平
)

// FundSearchRequest 表示基金搜索与筛选的请求参数。
type FundSearchRequest struct {
	Keyword   string   `form:"keyword"`
	Type      string   `form:"type"`
	Company   string   `form:"company"`
	RiskLevel string   `form:"risk_level"`
	Manager   string   `form:"manager"`
	ReturnMin *float64 `form:"return_min"` // 收益率下限
	ReturnMax *float64 `form:"return_max"` // 收益率上限
	SortBy    string   `form:"sort_by"`
	SortOrder string   `form:"sort_order"`
	Page      int      `form:"page"`
	Size      int      `form:"size"`
}

// FundItem 表示一只基金的概要信息，用于列表展示和搜索结果。
type FundItem struct {
	FundCode      string  `json:"fund_code"`
	FundName      string  `json:"fund_name"`
	FundType      string  `json:"fund_type"`
	PinyinAbbr    string  `json:"pinyin_abbr,omitempty"`    // 拼音首字母缩写
	PinyinFull    string  `json:"pinyin_full,omitempty"`    // 拼音全拼
	Company       string  `json:"company,omitempty"`        // 基金公司
	Manager       string  `json:"manager,omitempty"`        // 基金经理
	LatestNAV     float64 `json:"latest_nav"`               // 最新净值
	CumulativeNAV float64 `json:"cumulative_nav"`           // 累计净值
	Return1M      float64 `json:"return_1m"`                // 近 1 月收益率
	Return3M      float64 `json:"return_3m"`                // 近 3 月收益率
	Return6M      float64 `json:"return_6m"`                // 近 6 月收益率
	Return1Y      float64 `json:"return_1y"`                // 近 1 年收益率
	Return3Y      float64 `json:"return_3y"`                // 近 3 年收益率
	RiskLevel     string  `json:"risk_level,omitempty"`     // 风险等级
	InceptionDate string  `json:"inception_date,omitempty"` // 成立日期
	EstimatedNAV  float64 `json:"-"`                        // 估值净值，不序列化到 JSON
	ChangePct     float64 `json:"-"`                        // 估值涨跌幅，不序列化到 JSON
	QuoteDate     string  `json:"quote_date,omitempty"`     // 行情日期
	QuoteSource   string  `json:"quote_source,omitempty"`   // 行情数据来源
}

// FundSearchData 表示基金搜索的分页结果集。
type FundSearchData struct {
	Items []FundItem `json:"items"`
	Total int        `json:"total"`
	Page  int        `json:"page"`
	Size  int        `json:"size"`
}

// FundFilters 表示基金列表可用的筛选项，包含类型、公司和风险等级。
type FundFilters struct {
	Types      []string `json:"types"`
	Companies  []string `json:"companies"`
	RiskLevels []string `json:"risk_levels"`
}

// FundSyncResult 表示基金数据同步的结果统计。
type FundSyncResult struct {
	Source     string `json:"source"`
	StoredPath string `json:"stored_path,omitempty"` // 数据文件存储路径
	Imported   int    `json:"imported"`
	Total      int    `json:"total"`
	UpdatedAt  string `json:"updated_at"`
}

// FundRankingItem 表示基金排行榜中的单条记录。
type FundRankingItem struct {
	Rank         int     `json:"rank"`
	FundCode     string  `json:"fund_code"`
	FundName     string  `json:"fund_name"`
	FundType     string  `json:"fund_type"`
	ChangePct    float64 `json:"change_pct"`     // 涨跌幅，百分比
	EstimatedNAV float64 `json:"estimated_nav"`  // 估值净值
	QuoteDate    string  `json:"quote_date,omitempty"`
	QuoteSource  string  `json:"quote_source,omitempty"`
}

// WatchlistItem 表示自选基金列表中的单条记录。
type WatchlistItem struct {
	FundCode     string    `json:"fund_code"`
	FundName     string    `json:"fund_name"`
	FundType     string    `json:"fund_type"`
	EstimatedNAV float64   `json:"estimated_nav"` // 估值净值
	ChangePct    float64   `json:"change_pct"`    // 涨跌幅，百分比
	Direction    Direction `json:"direction"`     // 涨跌方向
	AddedAt      int64     `json:"added_at"`      // 添加时间戳
	QuoteDate    string    `json:"quote_date,omitempty"`
	QuoteSource  string    `json:"quote_source,omitempty"`
}

// WatchlistQuoteRequest 表示批量查询自选基金行情的请求参数。
type WatchlistQuoteRequest struct {
	Codes []string `json:"codes" binding:"max=50,dive,len=6,numeric"` // 基金代码列表，最多 50 个
}

// CoverageReport 表示基金数据的覆盖情况报告。
type CoverageReport struct {
	TotalFunds          int            `json:"total_funds"`            // 基金总数
	FundsWithQuote      int            `json:"funds_with_quote"`       // 有行情数据的基金数
	CountsByFundType    map[string]int `json:"counts_by_fund_type"`    // 按基金类型统计数量
	CountsByQuoteSource map[string]int `json:"counts_by_quote_source"` // 按行情来源统计数量
}

// FundDetailPath 表示基金详情页的路径参数。
type FundDetailPath struct {
	FundCode string `uri:"fundCode"` // 基金代码
}

// NAVPoint 表示基金净值历史中的一个数据点。
type NAVPoint struct {
	Date          string  `json:"date"`
	NAV           float64 `json:"nav"`            // 单位净值
	CumulativeNAV float64 `json:"cumulative_nav"` // 累计净值
	ChangePct     float64 `json:"change_pct"`     // 日涨跌幅，百分比
}

// FundPerformanceData 表示基金的业绩表现数据。
type FundPerformanceData struct {
	NAVHistory []NAVPoint `json:"nav_history"`
	Return1M   float64    `json:"return_1m"` // 近 1 月收益率
	Return3M   float64    `json:"return_3m"` // 近 3 月收益率
	Return6M   float64    `json:"return_6m"` // 近 6 月收益率
	Return1Y   float64    `json:"return_1y"` // 近 1 年收益率
	Return3Y   float64    `json:"return_3y"` // 近 3 年收益率
}

// FundManagerInfo 表示基金经理的基本信息。
type FundManagerInfo struct {
	Name        string `json:"name"`
	TenureDays  int    `json:"tenure_days"`  // 任职天数
	ManagedSize string `json:"managed_size"` // 管理规模
	FundCount   int    `json:"fund_count"`   // 在管基金数量
	Bio         string `json:"bio"`          // 简介
}

// HoldingItem 表示基金持仓中的单只证券。
type HoldingItem struct {
	Name  string  `json:"name"`
	Code  string  `json:"code"`
	Ratio float64 `json:"ratio"` // 持仓比例
}

// SectorItem 表示基金的行业配置项。
type SectorItem struct {
	Name  string  `json:"name"`
	Ratio float64 `json:"ratio"` // 配置比例
}

// FundPortfolioData 表示基金的投资组合数据。
type FundPortfolioData struct {
	TopHoldings      []HoldingItem `json:"top_holdings"`       // 十大重仓股
	SectorAllocation []SectorItem  `json:"sector_allocation"`  // 行业配置
}

// FundRiskMetrics 表示基金的风险指标。
type FundRiskMetrics struct {
	Volatility1Y float64 `json:"volatility_1y"`   // 近 1 年波动率
	MaxDrawdown  float64 `json:"max_drawdown_1y"` // 近 1 年最大回撤
	Sharpe1Y     float64 `json:"sharpe_1y"`       // 近 1 年夏普比率
	Beta1Y       float64 `json:"beta_1y"`         // 近 1 年 Beta 系数
}

// FundDetailData 表示基金详情页的完整数据聚合。
type FundDetailData struct {
	Basic       FundItem            `json:"basic"`
	Quote       FundItem            `json:"quote"`
	Performance FundPerformanceData `json:"performance"`
	Manager     FundManagerInfo     `json:"manager"`
	Portfolio   FundPortfolioData   `json:"portfolio"`
	Risk        FundRiskMetrics     `json:"risk"`
}
