package dto

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

type StockPredictionData struct {
	StockCode          string                `json:"stock_code"`
	StockName          string                `json:"stock_name"`
	NextDayPrediction  *PredictionResult     `json:"next_day_prediction,omitempty"`
	WeeklyPrediction   *PredictionResult     `json:"weekly_prediction,omitempty"`
	IntradayPrediction *PredictionResult     `json:"intraday_prediction,omitempty"`
	DataQuality        *PredictionDataQuality `json:"data_quality,omitempty"`
}
