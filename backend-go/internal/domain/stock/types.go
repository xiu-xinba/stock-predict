// Package stock 定义了股票相关的领域模型和数据结构。
package stock

import marketdomain "stock-predict-go/internal/domain/market"

// StockItem 表示一只股票的概要信息，用于列表展示和搜索结果。
type StockItem struct {
	StockCode    string  `json:"stock_code"`
	StockName    string  `json:"stock_name"`
	Market       string  `json:"market"`
	Industry     string  `json:"industry"`
	ListDate     string  `json:"list_date"`
	TotalShares  float64 `json:"total_shares"`
	FloatShares  float64 `json:"float_shares"`
	CurrentPrice float64 `json:"current_price"`
	ChangePct    float64 `json:"change_pct"`    // 涨跌幅，百分比
	Volume       float64 `json:"volume"`
	Amount       float64 `json:"amount"`
	TurnoverRate float64 `json:"turnover_rate"` // 换手率
	PERatio      float64 `json:"pe_ratio"`      // 市盈率（Price-to-Earnings）
	PBRatio      float64 `json:"pb_ratio"`      // 市净率（Price-to-Book）
	TotalMV      float64 `json:"total_mv"`      // 总市值
	Pinyin       string  `json:"pinyin"`        // 拼音首字母缩写
	PinyinAlt    string  `json:"pinyin_alt,omitempty"` // 备选拼音缩写
}

// StockSearchItem 表示股票搜索结果中的简要条目，用于联想搜索。
type StockSearchItem struct {
	StockCode string `json:"stock_code"`
	StockName string `json:"stock_name"`
	Market    string `json:"market"`
	Pinyin    string `json:"pinyin"` // 拼音首字母缩写
}

// StockSearchRequest 表示股票搜索与筛选的请求参数。
type StockSearchRequest struct {
	Keyword   string `form:"keyword"`
	Industry  string `form:"industry"`
	Market    string `form:"market"`
	SortBy    string `form:"sort_by"`
	SortOrder string `form:"sort_order"`
	Page      int    `form:"page"`
	Size      int    `form:"size"`
}

// StockSearchData 表示股票搜索的分页结果集。
type StockSearchData struct {
	Items []StockItem `json:"items"`
	Total int         `json:"total"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
}

// StockFilters 表示股票列表可用的筛选项，包含行业和市场分类。
type StockFilters struct {
	Industries []string `json:"industries"`
	Markets    []string `json:"markets"`
}

// StockBasicInfo 表示股票的基础信息，不包含行情数据。
type StockBasicInfo struct {
	StockCode   string  `json:"stock_code"`
	StockName   string  `json:"stock_name"`
	Market      string  `json:"market"`
	Industry    string  `json:"industry"`
	ListDate    string  `json:"list_date"`
	TotalShares float64 `json:"total_shares"`
	FloatShares float64 `json:"float_shares"`
}

// StockQuote 表示股票的实时行情报价。
type StockQuote struct {
	Price        float64 `json:"price"`
	Open         float64 `json:"open"`
	High         float64 `json:"high"`
	Low          float64 `json:"low"`
	PrevClose    float64 `json:"prev_close"`    // 昨收价
	Volume       float64 `json:"volume"`
	Amount       float64 `json:"amount"`
	TurnoverRate float64 `json:"turnover_rate"` // 换手率
	ChangePct    float64 `json:"change_pct"`    // 涨跌幅，百分比
	ChangeAmt    float64 `json:"change_amt"`    // 涨跌额
	BidPrice     float64 `json:"bid_price"`     // 买一价
	AskPrice     float64 `json:"ask_price"`     // 卖一价
	QuoteTime    string  `json:"quote_time"`
}

// KlinePoint 表示 K 线图中的一个数据点。
type KlinePoint struct {
	Date   string  `json:"date"`
	Open   float64 `json:"open"`
	Close  float64 `json:"close"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Volume float64 `json:"volume"`
	Amount float64 `json:"amount"`
}

// StockKlineData 表示股票的 K 线数据集合。
type StockKlineData struct {
	Period string       `json:"period"` // K 线周期，如 daily、weekly、monthly
	Klines []KlinePoint `json:"klines"`
}

// CapitalFlowPoint 表示单日资金流向数据。
type CapitalFlowPoint struct {
	Date          string  `json:"date"`
	MainInflow    float64 `json:"main_inflow"`    // 主力流入金额
	MainOutflow   float64 `json:"main_outflow"`   // 主力流出金额
	RetailInflow  float64 `json:"retail_inflow"`  // 散户流入金额
	RetailOutflow float64 `json:"retail_outflow"` // 散户流出金额
	NetInflow     float64 `json:"net_inflow"`     // 净流入金额
}

// StockCapitalFlow 表示股票的资金流向汇总数据。
type StockCapitalFlow struct {
	MainNetInflow   float64            `json:"main_net_inflow"`   // 主力净流入
	RetailNetInflow float64            `json:"retail_net_inflow"` // 散户净流入
	FlowHistory     []CapitalFlowPoint `json:"flow_history"`
}

// FinancialQuarter 表示单个季度的财务数据。
type FinancialQuarter struct {
	ReportDate  string  `json:"report_date"`
	Revenue     float64 `json:"revenue"`
	NetProfit   float64 `json:"net_profit"`
	EPS         float64 `json:"eps"`          // 每股收益（Earnings Per Share）
	GrossMargin float64 `json:"gross_margin"` // 毛利率
	NetMargin   float64 `json:"net_margin"`   // 净利率
	ROE         float64 `json:"roe"`          // 净资产收益率（Return on Equity）
}

// StockFinancials 表示股票的财务指标汇总。
type StockFinancials struct {
	PERatio     float64            `json:"pe_ratio"`      // 市盈率
	PBRatio     float64            `json:"pb_ratio"`      // 市净率
	ROE         float64            `json:"roe"`           // 净资产收益率
	Revenue     float64            `json:"revenue"`
	NetProfit   float64            `json:"net_profit"`
	EPS         float64            `json:"eps"`           // 每股收益
	GrossMargin float64            `json:"gross_margin"`  // 毛利率
	NetMargin   float64            `json:"net_margin"`    // 净利率
	Quarterly   []FinancialQuarter `json:"quarterly"`
}

// ShareholderItem 表示单个股东的信息。
type ShareholderItem struct {
	Name      string  `json:"name"`
	Ratio     float64 `json:"ratio"`      // 持股比例
	Change    float64 `json:"change"`     // 持股变动
	ShareType string  `json:"share_type"` // 股份类型
}

// StockShareholders 表示股票的股东信息汇总。
type StockShareholders struct {
	Top10            []ShareholderItem `json:"top10"`
	InstitutionCount int               `json:"institution_count"`  // 机构持仓数量
	InstitutionRatio float64           `json:"institution_ratio"`  // 机构持仓比例
}

// ResearchReport 表示单份研报评级。
type ResearchReport struct {
	Date        string  `json:"date"`         // 发布日期
	OrgName     string  `json:"org_name"`     // 机构名称
	Rating      string  `json:"rating"`       // 评级（买入/增持/中性/减持/卖出）
	TargetPrice float64 `json:"target_price"` // 目标价
	Researcher  string  `json:"researcher"`   // 研究员
}

// StockResearch 表示研报评级汇总。
type StockResearch struct {
	LatestRating string           `json:"latest_rating"` // 最新评级
	RatingCount  int              `json:"rating_count"`  // 评级数量
	Reports      []ResearchReport `json:"reports"`
}

// DividendRecord 表示单次分红送配记录。
type DividendRecord struct {
	Date     string  `json:"date"`      // 除权除息日
	Bonus    float64 `json:"bonus"`     // 每股送股
	Transfer float64 `json:"transfer"`  // 每股转增
	Dividend float64 `json:"dividend"`  // 每股分红（元）
	Progress string  `json:"progress"`  // 进度（实施/预案）
}

// StockDividends 表示分红送配汇总。
type StockDividends struct {
	TotalDividend float64          `json:"total_dividend"` // 累计分红
	Records       []DividendRecord `json:"records"`
}

// MarginData 表示单日融资融券数据。
type MarginData struct {
	Date          string  `json:"date"`           // 日期
	MarginBalance float64 `json:"margin_balance"` // 融资余额
	MarginBuy     float64 `json:"margin_buy"`     // 融资买入额
	ShortBalance  float64 `json:"short_balance"`  // 融券余额
	ShortVolume   float64 `json:"short_volume"`   // 融券余量
}

// StockMargin 表示融资融券汇总。
type StockMargin struct {
	LatestMarginBalance float64      `json:"latest_margin_balance"` // 最新融资余额
	History             []MarginData `json:"history"`
}

// ShareholderTrendPoint 表示股东人数变化数据点。
type ShareholderTrendPoint struct {
	Date       string  `json:"date"`        // 报告日期
	Count      float64 `json:"count"`       // 股东人数
	AvgHolding float64 `json:"avg_holding"` // 户均持股
	Change     float64 `json:"change"`      // 变化率
}

// StockShareholderTrend 表示股东人数变化汇总。
type StockShareholderTrend struct {
	LatestCount float64                 `json:"latest_count"` // 最新股东人数
	Trend       []ShareholderTrendPoint `json:"trend"`
}

// RestrictedRelease 表示单次限售解禁记录。
type RestrictedRelease struct {
	Date   string  `json:"date"`   // 解禁日期
	Volume float64 `json:"volume"` // 解禁数量
	Ratio  float64 `json:"ratio"`  // 占总股本比例
	Type   string  `json:"type"`   // 解禁类型
}

// StockRestricted 表示限售解禁汇总。
type StockRestricted struct {
	NextRelease *RestrictedRelease  `json:"next_release"` // 下次解禁
	History     []RestrictedRelease `json:"history"`
}

// StockDetailData 表示股票详情页的完整数据聚合。
type StockDetailData struct {
	Basic            StockBasicInfo                  `json:"basic"`
	Quote            StockQuote                      `json:"quote"`
	Kline            StockKlineData                  `json:"kline"`
	MinuteData       []marketdomain.IndexMinutePoint `json:"minute_data"`
	CapitalFlow      StockCapitalFlow                `json:"capital_flow"`
	Financials       StockFinancials                 `json:"financials"`
	Shareholders     StockShareholders               `json:"shareholders"`
	Research         StockResearch                   `json:"research"`
	Dividends        StockDividends                  `json:"dividends"`
	Margin           StockMargin                     `json:"margin"`
	ShareholderTrend StockShareholderTrend           `json:"shareholder_trend"`
	Restricted       StockRestricted                 `json:"restricted"`
}

// StockRankingItem 表示股票排行榜中的单条记录。
type StockRankingItem struct {
	Rank         int     `json:"rank"`
	StockCode    string  `json:"stock_code"`
	StockName    string  `json:"stock_name"`
	ChangePct    float64 `json:"change_pct"`     // 涨跌幅，百分比
	CurrentPrice float64 `json:"current_price"`
	Volume       float64 `json:"volume"`
	Amount       float64 `json:"amount"`
	UpdateTime   string  `json:"update_time,omitempty"`
	DataSource   string  `json:"data_source,omitempty"`
}

// StockQuoteRequest 表示批量查询股票行情的请求参数。
type StockQuoteRequest struct {
	Codes     []string `json:"codes" binding:"max=50,dive,len=6,numeric"` // 股票代码列表，最多 50 个
	Freshness string   `json:"freshness,omitempty"`                       // 数据新鲜度要求
}

// StockSyncResult 表示股票数据同步的结果统计。
type StockSyncResult struct {
	Total    int `json:"total"`    // 总记录数
	Imported int `json:"imported"` // 成功导入数
	Errors   int `json:"errors"`   // 错误数
}
