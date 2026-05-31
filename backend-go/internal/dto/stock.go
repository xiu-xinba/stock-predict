package dto

type StockItem struct {
	StockCode    string  `json:"stock_code"`
	StockName    string  `json:"stock_name"`
	Market       string  `json:"market"`
	Industry     string  `json:"industry"`
	ListDate     string  `json:"list_date"`
	TotalShares  float64 `json:"total_shares"`
	FloatShares  float64 `json:"float_shares"`
	CurrentPrice float64 `json:"current_price"`
	ChangePct    float64 `json:"change_pct"`
	Volume       float64 `json:"volume"`
	Amount       float64 `json:"amount"`
	TurnoverRate float64 `json:"turnover_rate"`
	PERatio      float64 `json:"pe_ratio"`
	PBRatio      float64 `json:"pb_ratio"`
	TotalMV      float64 `json:"total_mv"`
	Pinyin       string  `json:"pinyin"`
	PinyinAlt    string  `json:"pinyin_alt,omitempty"`
}

type StockSearchRequest struct {
	Keyword   string `form:"keyword"`
	Industry  string `form:"industry"`
	Market    string `form:"market"`
	SortBy    string `form:"sort_by"`
	SortOrder string `form:"sort_order"`
	Page      int    `form:"page"`
	Size      int    `form:"size"`
}

type StockSearchData struct {
	Items []StockItem `json:"items"`
	Total int         `json:"total"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
}

type StockFilters struct {
	Industries []string `json:"industries"`
	Markets    []string `json:"markets"`
}

type StockBasicInfo struct {
	StockCode   string  `json:"stock_code"`
	StockName   string  `json:"stock_name"`
	Market      string  `json:"market"`
	Industry    string  `json:"industry"`
	ListDate    string  `json:"list_date"`
	TotalShares float64 `json:"total_shares"`
	FloatShares float64 `json:"float_shares"`
}

type StockQuote struct {
	Price        float64 `json:"price"`
	Open         float64 `json:"open"`
	High         float64 `json:"high"`
	Low          float64 `json:"low"`
	PrevClose    float64 `json:"prev_close"`
	Volume       float64 `json:"volume"`
	Amount       float64 `json:"amount"`
	TurnoverRate float64 `json:"turnover_rate"`
	ChangePct    float64 `json:"change_pct"`
	ChangeAmt    float64 `json:"change_amt"`
	BidPrice     float64 `json:"bid_price"`
	AskPrice     float64 `json:"ask_price"`
	QuoteTime    string  `json:"quote_time"`
}

type KlinePoint struct {
	Date   string  `json:"date"`
	Open   float64 `json:"open"`
	Close  float64 `json:"close"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Volume float64 `json:"volume"`
	Amount float64 `json:"amount"`
}

type StockKlineData struct {
	Period string        `json:"period"`
	Klines []KlinePoint  `json:"klines"`
}

type CapitalFlowPoint struct {
	Date          string  `json:"date"`
	MainInflow    float64 `json:"main_inflow"`
	MainOutflow   float64 `json:"main_outflow"`
	RetailInflow  float64 `json:"retail_inflow"`
	RetailOutflow float64 `json:"retail_outflow"`
	NetInflow     float64 `json:"net_inflow"`
}

type StockCapitalFlow struct {
	MainNetInflow   float64            `json:"main_net_inflow"`
	RetailNetInflow float64            `json:"retail_net_inflow"`
	FlowHistory     []CapitalFlowPoint `json:"flow_history"`
}

type FinancialQuarter struct {
	ReportDate  string  `json:"report_date"`
	Revenue     float64 `json:"revenue"`
	NetProfit   float64 `json:"net_profit"`
	EPS         float64 `json:"eps"`
	GrossMargin float64 `json:"gross_margin"`
	NetMargin   float64 `json:"net_margin"`
	ROE         float64 `json:"roe"`
}

type StockFinancials struct {
	PERatio     float64            `json:"pe_ratio"`
	PBRatio     float64            `json:"pb_ratio"`
	ROE         float64            `json:"roe"`
	Revenue     float64            `json:"revenue"`
	NetProfit   float64            `json:"net_profit"`
	EPS         float64            `json:"eps"`
	GrossMargin float64            `json:"gross_margin"`
	NetMargin   float64            `json:"net_margin"`
	Quarterly   []FinancialQuarter `json:"quarterly"`
}

type ShareholderItem struct {
	Name      string  `json:"name"`
	Ratio     float64 `json:"ratio"`
	Change    float64 `json:"change"`
	ShareType string  `json:"share_type"`
}

type StockShareholders struct {
	Top10            []ShareholderItem `json:"top10"`
	InstitutionCount int               `json:"institution_count"`
	InstitutionRatio float64           `json:"institution_ratio"`
}

type StockDetailData struct {
	Basic       StockBasicInfo    `json:"basic"`
	Quote       StockQuote        `json:"quote"`
	Kline       StockKlineData    `json:"kline"`
	CapitalFlow StockCapitalFlow  `json:"capital_flow"`
	Financials  StockFinancials   `json:"financials"`
	Shareholders StockShareholders `json:"shareholders"`
}

type StockRankingItem struct {
	Rank         int     `json:"rank"`
	StockCode    string  `json:"stock_code"`
	StockName    string  `json:"stock_name"`
	ChangePct    float64 `json:"change_pct"`
	CurrentPrice float64 `json:"current_price"`
	Volume       float64 `json:"volume"`
	Amount       float64 `json:"amount"`
}

type StockQuoteRequest struct {
	Codes []string `json:"codes"`
}

type StockSyncResult struct {
	Total    int `json:"total"`
	Imported int `json:"imported"`
	Errors   int `json:"errors"`
}
