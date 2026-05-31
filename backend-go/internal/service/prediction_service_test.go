package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"stock-predict-go/internal/config"
	"stock-predict-go/internal/dto"
)

func TestPredictionServiceUsesConfiguredModelService(t *testing.T) {
	modelServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/predict/510300" {
			t.Fatalf("unexpected model service path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"fund_code": "510300",
			"fund_name": "沪深300ETF",
			"asof_time": "2026-05-27T15:00:00",
			"model": map[string]any{
				"candidate":   "extra_trees",
				"feature_set": "index_fund_daily_v1",
				"model_path":  "artifacts/public_mvp_index_fund_tournament_champion.joblib",
			},
			"prediction": map[string]any{
				"horizon":              "next_day",
				"target_window":        "下一个交易日",
				"direction":            "down",
				"direction_confidence": 0.71,
				"predicted_change_pct": -0.27,
				"change_range":         map[string]float64{"low": -0.9, "high": 0.2},
				"prediction_interval": map[string]any{
					"low":                -1.013459,
					"high":               2.338449,
					"method":             "empirical_residual_quantile",
					"level":              0.9,
					"empirical_coverage": 0.890411,
				},
				"return_decomposition": map[string]any{
					"enabled":                true,
					"method":                 "tracking_index_plus_error",
					"formula":                "fund_return = tracking_index_return + tracking_error",
					"index_return_pct":       -0.22,
					"tracking_error_pct":     -0.05,
					"direct_fund_return_pct": -0.26,
					"index_return_target":    "future_index_return_pct_next_day",
					"tracking_error_target":  "future_tracking_error_pct_next_day",
				},
				"signal_status":       "actionable",
				"is_actionable":       true,
				"reliability":         "model_mvp",
				"reliability_note":    "unit test model",
				"class_probabilities": map[string]float64{"down": 0.71, "flat": 0.2, "up": 0.09},
				"top_factors":         []map[string]any{{"name": "fear_score", "importance": 0.6, "description": "合成恐慌分数"}},
				"actionability_gate": map[string]any{
					"actionable":                   true,
					"reason":                       "passed",
					"min_high_confidence_accuracy": 0.5,
					"min_high_confidence_coverage": 0.05,
					"high_confidence_accuracy":     0.59375,
					"high_confidence_coverage":     0.438356,
					"max_calibration_ece":          0.12,
					"calibration_ece":              0.056112,
				},
			},
			"data_quality": map[string]any{
				"feature_count":        24,
				"has_panic_factor":     true,
				"has_futures_features": true,
				"note":                 "unit test quality",
			},
		})
	}))
	defer modelServer.Close()

	logger := slog.New(slog.NewTextHandler(&discardWriter{}, nil))
	service := NewPredictionService(
		fakeFundRepository{funds: []dto.FundItem{{FundCode: "510300", FundName: "沪深300ETF"}}},
		NewMarketService(logger),
		fakeStockFinder{},
		config.Config{ModelServiceURL: modelServer.URL, ReadTimeout: time.Second},
		logger,
	)

	got, err := service.PredictByFundCode(context.Background(), "510300")
	if err != nil {
		t.Fatalf("predict failed: %v", err)
	}
	if got.NextDayPrediction.Reliability != "model_mvp" {
		t.Fatalf("expected model prediction, got reliability %q", got.NextDayPrediction.Reliability)
	}
	if got.NextDayPrediction.ModelSource != "python_model_service" {
		t.Fatalf("expected python model source, got %q", got.NextDayPrediction.ModelSource)
	}
	if got.NextDayPrediction.ModelCandidate != "extra_trees" {
		t.Fatalf("expected model candidate extra_trees, got %q", got.NextDayPrediction.ModelCandidate)
	}
	if got.NextDayPrediction.FeatureSet != "index_fund_daily_v1" {
		t.Fatalf("expected feature set index_fund_daily_v1, got %q", got.NextDayPrediction.FeatureSet)
	}
	if got.NextDayPrediction.ModelAsOfTime != "2026-05-27T15:00:00" {
		t.Fatalf("expected model as-of time, got %q", got.NextDayPrediction.ModelAsOfTime)
	}
	if got.NextDayPrediction.ModelCoverageStatus != dto.ModelCoverageSupported {
		t.Fatalf("expected supported model coverage, got %q", got.NextDayPrediction.ModelCoverageStatus)
	}
	if got.NextDayPrediction.ModelCoverageNote == "" {
		t.Fatal("expected supported model coverage note")
	}
	if got.NextDayPrediction.Direction != dto.DirectionDown {
		t.Fatalf("expected model direction down, got %q", got.NextDayPrediction.Direction)
	}
	if got.NextDayPrediction.PredictedChangePct != -0.27 {
		t.Fatalf("expected model predicted pct -0.27, got %f", got.NextDayPrediction.PredictedChangePct)
	}
	if !got.NextDayPrediction.IsActionable {
		t.Fatal("expected actionable model signal")
	}
	if got.NextDayPrediction.SignalStatus != dto.SignalStatusActionable {
		t.Fatalf("expected actionable signal status, got %q", got.NextDayPrediction.SignalStatus)
	}
	if got.NextDayPrediction.ReturnDecomposition == nil || !got.NextDayPrediction.ReturnDecomposition.Enabled {
		t.Fatal("expected enabled return decomposition from model service")
	}
	if got.NextDayPrediction.ReturnDecomposition.IndexReturnPct == nil || *got.NextDayPrediction.ReturnDecomposition.IndexReturnPct != -0.22 {
		t.Fatalf("unexpected decomposition index return: %+v", got.NextDayPrediction.ReturnDecomposition.IndexReturnPct)
	}
	if got.NextDayPrediction.PredictionInterval == nil {
		t.Fatal("expected prediction interval from model service")
	}
	if got.NextDayPrediction.PredictionInterval.Method != "empirical_residual_quantile" {
		t.Fatalf("unexpected interval method: %q", got.NextDayPrediction.PredictionInterval.Method)
	}
	if got.NextDayPrediction.PredictionInterval.EmpiricalCoverage == nil || *got.NextDayPrediction.PredictionInterval.EmpiricalCoverage != 0.8904 {
		t.Fatalf("unexpected interval coverage: %+v", got.NextDayPrediction.PredictionInterval.EmpiricalCoverage)
	}
	if got.NextDayPrediction.ActionabilityGate == nil {
		t.Fatal("expected actionability gate from model service")
	}
	if got.NextDayPrediction.ActionabilityGate.Reason != "passed" {
		t.Fatalf("unexpected actionability reason: %q", got.NextDayPrediction.ActionabilityGate.Reason)
	}
	if got.NextDayPrediction.ActionabilityGate.HighConfidenceCoverage == nil || *got.NextDayPrediction.ActionabilityGate.HighConfidenceCoverage != 0.4384 {
		t.Fatalf("unexpected high-confidence coverage: %+v", got.NextDayPrediction.ActionabilityGate.HighConfidenceCoverage)
	}
	if got.DataQuality.CoverageScore <= 0.33 {
		t.Fatalf("expected model quality to improve coverage, got %f", got.DataQuality.CoverageScore)
	}
}

func TestPredictionServiceDerivesLowConfidenceSignalStatus(t *testing.T) {
	modelServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"fund_code": "510300",
			"prediction": map[string]any{
				"horizon":              "next_day",
				"direction":            "up",
				"direction_confidence": 0.52,
				"predicted_change_pct": 0.11,
				"change_range":         map[string]float64{"low": -0.1, "high": 0.3},
				"is_actionable":        false,
				"reliability":          "model_mvp",
			},
		})
	}))
	defer modelServer.Close()

	logger := slog.New(slog.NewTextHandler(&discardWriter{}, nil))
	service := NewPredictionService(
		fakeFundRepository{funds: []dto.FundItem{{FundCode: "510300", FundName: "沪深300ETF"}}},
		NewMarketService(logger),
		fakeStockFinder{},
		config.Config{ModelServiceURL: modelServer.URL, ReadTimeout: time.Second},
		logger,
	)

	got, err := service.PredictByFundCode(context.Background(), "510300")
	if err != nil {
		t.Fatalf("predict failed: %v", err)
	}
	if got.NextDayPrediction.SignalStatus != dto.SignalStatusLowConfidence {
		t.Fatalf("expected low confidence status, got %q", got.NextDayPrediction.SignalStatus)
	}
	if got.NextDayPrediction.IsActionable {
		t.Fatal("low confidence signal must not be actionable")
	}
}

func TestPredictionServiceFallsBackWhenModelServiceFails(t *testing.T) {
	modelServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer modelServer.Close()

	logger := slog.New(slog.NewTextHandler(&discardWriter{}, nil))
	service := NewPredictionService(
		fakeFundRepository{funds: []dto.FundItem{{FundCode: "510300", FundName: "沪深300ETF"}}},
		NewMarketService(logger),
		fakeStockFinder{},
		config.Config{ModelServiceURL: modelServer.URL, ReadTimeout: time.Second},
		logger,
	)

	got, err := service.PredictByFundCode(context.Background(), "510300")
	if err != nil {
		t.Fatalf("predict failed: %v", err)
	}
	if got.NextDayPrediction.Reliability != "baseline" {
		t.Fatalf("expected baseline fallback, got reliability %q", got.NextDayPrediction.Reliability)
	}
	if got.NextDayPrediction.ModelSource != "go_baseline" {
		t.Fatalf("expected Go baseline source, got %q", got.NextDayPrediction.ModelSource)
	}
	if got.NextDayPrediction.ModelCoverageStatus != dto.ModelCoverageModelUnavailable {
		t.Fatalf("expected unavailable model coverage, got %q", got.NextDayPrediction.ModelCoverageStatus)
	}
	if got.NextDayPrediction.ModelCoverageNote == "" {
		t.Fatal("expected fallback coverage note")
	}
}

func TestPredictionServiceMarksUnsupportedFundWhenModelHasNoSample(t *testing.T) {
	modelServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "No sample row found for fund_code=000001.",
		})
	}))
	defer modelServer.Close()

	logger := slog.New(slog.NewTextHandler(&discardWriter{}, nil))
	service := NewPredictionService(
		fakeFundRepository{funds: []dto.FundItem{{FundCode: "000001", FundName: "华夏成长混合"}}},
		NewMarketService(logger),
		fakeStockFinder{},
		config.Config{ModelServiceURL: modelServer.URL, ReadTimeout: time.Second},
		logger,
	)

	got, err := service.PredictByFundCode(context.Background(), "000001")
	if err != nil {
		t.Fatalf("predict failed: %v", err)
	}
	if got.NextDayPrediction.ModelSource != "go_baseline" {
		t.Fatalf("expected Go baseline fallback, got %q", got.NextDayPrediction.ModelSource)
	}
	if got.NextDayPrediction.ModelCoverageStatus != dto.ModelCoverageUnsupportedFund {
		t.Fatalf("expected unsupported-fund coverage, got %q", got.NextDayPrediction.ModelCoverageStatus)
	}
	if got.NextDayPrediction.ModelCoverageNote == "" {
		t.Fatal("expected unsupported-fund coverage note")
	}
}

func TestPredictionServiceMarksBaselineOnlyWhenNoModelServiceConfigured(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&discardWriter{}, nil))
	service := NewPredictionService(
		fakeFundRepository{funds: []dto.FundItem{{FundCode: "000001", FundName: "华夏成长混合"}}},
		NewMarketService(logger),
		fakeStockFinder{},
		config.Config{ReadTimeout: time.Second},
		logger,
	)

	got, err := service.PredictByFundCode(context.Background(), "000001")
	if err != nil {
		t.Fatalf("predict failed: %v", err)
	}
	if got.NextDayPrediction.ModelCoverageStatus != dto.ModelCoverageBaselineOnly {
		t.Fatalf("expected baseline-only coverage, got %q", got.NextDayPrediction.ModelCoverageStatus)
	}
	if got.NextDayPrediction.ModelCoverageNote == "" {
		t.Fatal("expected baseline-only coverage note")
	}
}

func TestPredictionServiceCanUseSeparateIntradayModelService(t *testing.T) {
	modelServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"fund_code": "510300",
			"prediction": map[string]any{
				"horizon":              "intraday_5m",
				"target_window":        "未来5分钟",
				"direction":            "up",
				"direction_confidence": 0.66,
				"predicted_change_pct": 0.08,
				"change_range":         map[string]float64{"low": 0.01, "high": 0.14},
				"signal_status":        "actionable",
				"is_actionable":        true,
				"reliability":          "intraday_model_mvp",
			},
			"data_quality": map[string]any{
				"feature_count":        20,
				"has_panic_factor":     true,
				"has_futures_features": false,
			},
		})
	}))
	defer modelServer.Close()

	logger := slog.New(slog.NewTextHandler(&discardWriter{}, nil))
	service := NewPredictionService(
		fakeFundRepository{funds: []dto.FundItem{{FundCode: "510300", FundName: "沪深300ETF"}}},
		NewMarketService(logger),
		fakeStockFinder{},
		config.Config{IntradayModelServiceURL: modelServer.URL, ReadTimeout: time.Second},
		logger,
	)

	got, err := service.PredictByFundCode(context.Background(), "510300")
	if err != nil {
		t.Fatalf("predict failed: %v", err)
	}
	if got.NextDayPrediction.Reliability != "baseline" {
		t.Fatalf("expected daily baseline to remain, got %q", got.NextDayPrediction.Reliability)
	}
	if got.IntradayPrediction.Reliability != "intraday_model_mvp" {
		t.Fatalf("expected intraday model prediction, got %q", got.IntradayPrediction.Reliability)
	}
	if got.IntradayPrediction.Horizon != "intraday_5m" {
		t.Fatalf("expected intraday horizon, got %q", got.IntradayPrediction.Horizon)
	}
}

func TestPredictionServiceCanUseSeparateWeeklyModelService(t *testing.T) {
	modelServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/predict/510300" {
			t.Fatalf("unexpected model service path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"fund_code": "510300",
			"prediction": map[string]any{
				"horizon":              "next_week",
				"target_window":        "未来一周",
				"direction":            "up",
				"direction_confidence": 0.68,
				"predicted_change_pct": 1.23,
				"change_range":         map[string]float64{"low": 0.4, "high": 2.0},
				"signal_status":        "actionable",
				"is_actionable":        true,
				"reliability":          "weekly_model_mvp",
			},
			"data_quality": map[string]any{
				"feature_count":        24,
				"has_panic_factor":     true,
				"has_futures_features": true,
			},
		})
	}))
	defer modelServer.Close()

	logger := slog.New(slog.NewTextHandler(&discardWriter{}, nil))
	service := NewPredictionService(
		fakeFundRepository{funds: []dto.FundItem{{FundCode: "510300", FundName: "沪深300ETF"}}},
		NewMarketService(logger),
		fakeStockFinder{},
		config.Config{WeeklyModelServiceURL: modelServer.URL, ReadTimeout: time.Second},
		logger,
	)

	got, err := service.PredictByFundCode(context.Background(), "510300")
	if err != nil {
		t.Fatalf("predict failed: %v", err)
	}
	if got.NextDayPrediction.Reliability != "baseline" {
		t.Fatalf("expected daily baseline to remain, got %q", got.NextDayPrediction.Reliability)
	}
	if got.WeeklyPrediction.Reliability != "weekly_model_mvp" {
		t.Fatalf("expected weekly model prediction, got %q", got.WeeklyPrediction.Reliability)
	}
	if got.WeeklyPrediction.Horizon != "next_week" {
		t.Fatalf("expected weekly horizon, got %q", got.WeeklyPrediction.Horizon)
	}
}

type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) {
	return len(p), nil
}
