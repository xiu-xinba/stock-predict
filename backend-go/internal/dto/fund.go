package dto

type FundSearchRequest struct {
	Keyword   string   `form:"keyword"`
	Type      string   `form:"type"`
	Company   string   `form:"company"`
	RiskLevel string   `form:"risk_level"`
	Manager   string   `form:"manager"`
	ReturnMin *float64 `form:"return_min"`
	ReturnMax *float64 `form:"return_max"`
	SortBy    string   `form:"sort_by"`
	SortOrder string   `form:"sort_order"`
	Page      int      `form:"page"`
	Size      int      `form:"size"`
}

type FundItem struct {
	FundCode      string  `json:"fund_code"`
	FundName      string  `json:"fund_name"`
	FundType      string  `json:"fund_type"`
	PinyinAbbr    string  `json:"pinyin_abbr,omitempty"`
	PinyinFull    string  `json:"pinyin_full,omitempty"`
	Company       string  `json:"company,omitempty"`
	Manager       string  `json:"manager,omitempty"`
	LatestNAV     float64 `json:"latest_nav"`
	CumulativeNAV float64 `json:"cumulative_nav"`
	Return1M      float64 `json:"return_1m"`
	Return3M      float64 `json:"return_3m"`
	Return6M      float64 `json:"return_6m"`
	Return1Y      float64 `json:"return_1y"`
	Return3Y      float64 `json:"return_3y"`
	RiskLevel     string  `json:"risk_level,omitempty"`
	InceptionDate string  `json:"inception_date,omitempty"`
	EstimatedNAV  float64 `json:"-"`
	ChangePct     float64 `json:"-"`
	QuoteDate     string  `json:"quote_date,omitempty"`
	QuoteSource   string  `json:"quote_source,omitempty"`
}

type FundSearchData struct {
	Items []FundItem `json:"items"`
	Total int        `json:"total"`
	Page  int        `json:"page"`
	Size  int        `json:"size"`
}

type FundFilters struct {
	Types      []string `json:"types"`
	Companies  []string `json:"companies"`
	RiskLevels []string `json:"risk_levels"`
}

type FundSyncResult struct {
	Source     string `json:"source"`
	StoredPath string `json:"stored_path,omitempty"`
	Imported   int    `json:"imported"`
	Total      int    `json:"total"`
	UpdatedAt  string `json:"updated_at"`
}

type FundRankingItem struct {
	Rank         int     `json:"rank"`
	FundCode     string  `json:"fund_code"`
	FundName     string  `json:"fund_name"`
	FundType     string  `json:"fund_type"`
	ChangePct    float64 `json:"change_pct"`
	EstimatedNAV float64 `json:"estimated_nav"`
	QuoteDate    string  `json:"quote_date,omitempty"`
	QuoteSource  string  `json:"quote_source,omitempty"`
}

type WatchlistItem struct {
	FundCode     string    `json:"fund_code"`
	FundName     string    `json:"fund_name"`
	FundType     string    `json:"fund_type"`
	EstimatedNAV float64   `json:"estimated_nav"`
	ChangePct    float64   `json:"change_pct"`
	Direction    Direction `json:"direction"`
	AddedAt      int64     `json:"added_at"`
	QuoteDate    string    `json:"quote_date,omitempty"`
	QuoteSource  string    `json:"quote_source,omitempty"`
}

type WatchlistQuoteRequest struct {
	Codes []string `json:"codes" binding:"max=50,dive,len=6,numeric"`
}

type CoverageReport struct {
	TotalFunds          int            `json:"total_funds"`
	FundsWithQuote      int            `json:"funds_with_quote"`
	CountsByFundType    map[string]int `json:"counts_by_fund_type"`
	CountsByQuoteSource map[string]int `json:"counts_by_quote_source"`
}

type FundDetailPath struct {
	FundCode string `uri:"fundCode"`
}

type NAVPoint struct {
	Date          string  `json:"date"`
	NAV           float64 `json:"nav"`
	CumulativeNAV float64 `json:"cumulative_nav"`
	ChangePct     float64 `json:"change_pct"`
}

type FundPerformanceData struct {
	NAVHistory []NAVPoint `json:"nav_history"`
	Return1M   float64    `json:"return_1m"`
	Return3M   float64    `json:"return_3m"`
	Return6M   float64    `json:"return_6m"`
	Return1Y   float64    `json:"return_1y"`
	Return3Y   float64    `json:"return_3y"`
}

type FundManagerInfo struct {
	Name        string `json:"name"`
	TenureDays  int    `json:"tenure_days"`
	ManagedSize string `json:"managed_size"`
	FundCount   int    `json:"fund_count"`
	Bio         string `json:"bio"`
}

type HoldingItem struct {
	Name  string  `json:"name"`
	Code  string  `json:"code"`
	Ratio float64 `json:"ratio"`
}

type SectorItem struct {
	Name  string  `json:"name"`
	Ratio float64 `json:"ratio"`
}

type FundPortfolioData struct {
	TopHoldings      []HoldingItem `json:"top_holdings"`
	SectorAllocation []SectorItem  `json:"sector_allocation"`
}

type FundRiskMetrics struct {
	Volatility1Y float64 `json:"volatility_1y"`
	MaxDrawdown  float64 `json:"max_drawdown_1y"`
	Sharpe1Y     float64 `json:"sharpe_1y"`
	Beta1Y       float64 `json:"beta_1y"`
}

type FundDetailData struct {
	Basic       FundItem            `json:"basic"`
	Quote       FundItem            `json:"quote"`
	Performance FundPerformanceData `json:"performance"`
	Manager     FundManagerInfo     `json:"manager"`
	Portfolio   FundPortfolioData   `json:"portfolio"`
	Risk        FundRiskMetrics     `json:"risk"`
}
