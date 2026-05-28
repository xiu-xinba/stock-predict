package dto

type APIResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

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

type MarketRankingPath struct {
	Type string `uri:"type"`
}

type MarketRankingQuery struct {
	Size int `form:"size"`
}

type PredictPath struct {
	FundCode string `uri:"fundCode"`
}

type Direction string

const (
	DirectionUp   Direction = "up"
	DirectionDown Direction = "down"
	DirectionFlat Direction = "flat"
)

type SignalStatus string

const (
	SignalStatusActionable    SignalStatus = "actionable"
	SignalStatusLowConfidence SignalStatus = "low_confidence"
	SignalStatusNoSignal      SignalStatus = "no_signal"
)

type ModelCoverageStatus string

const (
	ModelCoverageSupported        ModelCoverageStatus = "model_supported"
	ModelCoverageBaselineOnly     ModelCoverageStatus = "baseline_only"
	ModelCoverageUnsupportedFund  ModelCoverageStatus = "unsupported_fund"
	ModelCoverageModelUnavailable ModelCoverageStatus = "model_unavailable"
)

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

type MarketIndex struct {
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	Market        string    `json:"market"`
	Value         float64   `json:"value"`
	Change        float64   `json:"change"`
	ChangePct     float64   `json:"change_pct"`
	High          float64   `json:"high"`
	Low           float64   `json:"low"`
	PrevClose     float64   `json:"prev_close"`
	Volume        float64   `json:"volume"`
	MiniChartData []float64 `json:"mini_chart_data"`
	UpdateTime    string    `json:"update_time"`
	DataSource    string    `json:"data_source"`
}

type MarketSnapshot struct {
	ShIndex           float64 `json:"sh_index"`
	ShIndexChangePct  float64 `json:"sh_index_change_pct"`
	SzIndex           float64 `json:"sz_index"`
	SzIndexChangePct  float64 `json:"sz_index_change_pct"`
	CybIndex          float64 `json:"cyb_index"`
	CybIndexChangePct float64 `json:"cyb_index_change_pct"`
	UpdateTime        string  `json:"update_time"`
}

type ChangeRange struct {
	Low  float64 `json:"low"`
	High float64 `json:"high"`
}

type PredictionInterval struct {
	Low               float64  `json:"low"`
	High              float64  `json:"high"`
	Method            string   `json:"method,omitempty"`
	Level             *float64 `json:"level,omitempty"`
	EmpiricalCoverage *float64 `json:"empirical_coverage,omitempty"`
}

type ActionabilityGate struct {
	Actionable                bool     `json:"actionable"`
	Reason                    string   `json:"reason,omitempty"`
	MinHighConfidenceAccuracy *float64 `json:"min_high_confidence_accuracy,omitempty"`
	MinHighConfidenceCoverage *float64 `json:"min_high_confidence_coverage,omitempty"`
	HighConfidenceAccuracy    *float64 `json:"high_confidence_accuracy,omitempty"`
	HighConfidenceCoverage    *float64 `json:"high_confidence_coverage,omitempty"`
	MaxCalibrationECE         *float64 `json:"max_calibration_ece,omitempty"`
	CalibrationECE            *float64 `json:"calibration_ece,omitempty"`
}

type ReturnDecomposition struct {
	Enabled             bool     `json:"enabled"`
	Method              string   `json:"method"`
	Formula             string   `json:"formula"`
	IndexReturnPct      *float64 `json:"index_return_pct"`
	TrackingErrorPct    *float64 `json:"tracking_error_pct"`
	DirectFundReturnPct *float64 `json:"direct_fund_return_pct"`
	IndexReturnTarget   string   `json:"index_return_target,omitempty"`
	TrackingErrorTarget string   `json:"tracking_error_target,omitempty"`
}

type FactorItem struct {
	Name        string  `json:"name"`
	Importance  float64 `json:"importance"`
	Description string  `json:"description"`
}

type PredictionResult struct {
	Horizon             string               `json:"horizon"`
	TargetWindow        string               `json:"target_window"`
	ModelSource         string               `json:"model_source"`
	ModelCandidate      string               `json:"model_candidate,omitempty"`
	FeatureSet          string               `json:"feature_set,omitempty"`
	ModelAsOfTime       string               `json:"model_asof_time,omitempty"`
	ModelCoverageStatus ModelCoverageStatus  `json:"model_coverage_status"`
	ModelCoverageNote   string               `json:"model_coverage_note,omitempty"`
	Direction           Direction            `json:"direction"`
	DirectionConfidence float64              `json:"direction_confidence"`
	PredictedChangePct  float64              `json:"predicted_change_pct"`
	ChangeRange         ChangeRange          `json:"change_range"`
	PredictionInterval  *PredictionInterval  `json:"prediction_interval,omitempty"`
	ReturnDecomposition *ReturnDecomposition `json:"return_decomposition,omitempty"`
	ActionabilityGate   *ActionabilityGate   `json:"actionability_gate,omitempty"`
	TopFactors          []FactorItem         `json:"top_factors"`
	Reliability         string               `json:"reliability"`
	ReliabilityNote     string               `json:"reliability_note"`
	AccuracyTarget      float64              `json:"accuracy_target"`
	MeetsAccuracyTarget bool                 `json:"meets_accuracy_target"`
	SignalStatus        SignalStatus         `json:"signal_status"`
	IsActionable        bool                 `json:"is_actionable"`
	CalibrationNote     string               `json:"calibration_note"`
}

type PredictionDataQuality struct {
	HasRealtimeQuote           bool     `json:"has_realtime_quote"`
	HasMarketIndices           bool     `json:"has_market_indices"`
	HasHoldingsData            bool     `json:"has_holdings_data"`
	HasIntradayConstituentData bool     `json:"has_intraday_constituent_data"`
	HasEtfFlowData             bool     `json:"has_etf_flow_data"`
	CoverageScore              float64  `json:"coverage_score"`
	MissingSources             []string `json:"missing_sources"`
	Note                       string   `json:"note"`
}

type PredictionData struct {
	FundCode           string                `json:"fund_code"`
	FundName           string                `json:"fund_name"`
	Prediction         PredictionResult      `json:"prediction"`
	NextDayPrediction  PredictionResult      `json:"next_day_prediction"`
	WeeklyPrediction   PredictionResult      `json:"weekly_prediction"`
	IntradayPrediction PredictionResult      `json:"intraday_prediction"`
	DataQuality        PredictionDataQuality `json:"data_quality"`
	MarketSnapshot     MarketSnapshot        `json:"market_snapshot"`
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
