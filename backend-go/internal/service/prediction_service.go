package service

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"time"

	"stock-predict-go/internal/config"
	"stock-predict-go/internal/dto"
)

var (
	ErrInvalidFundCode = errors.New("invalid fund code")
	ErrFundNotFound    = errors.New("fund not found")
)

type PredictionService struct {
	store               FundRepository
	market              *MarketService
	cfg                 config.Config
	logger              *slog.Logger
	modelClient         *ModelClient
	weeklyModelClient   *ModelClient
	intradayModelClient *ModelClient
	quoteProvider       fundQuoteProvider
}

func NewPredictionService(store FundRepository, market *MarketService, cfg config.Config, logger *slog.Logger) *PredictionService {
	if logger == nil {
		logger = slog.Default()
	}
	modelClient, err := NewModelClient(cfg.ModelServiceURL, cfg.ReadTimeout, logger)
	if err != nil {
		logger.Warn("model service disabled", "error", err)
	}
	weeklyModelClient, err := NewModelClient(cfg.WeeklyModelServiceURL, cfg.ReadTimeout, logger)
	if err != nil {
		logger.Warn("weekly model service disabled", "error", err)
	}
	intradayModelClient, err := NewModelClient(cfg.IntradayModelServiceURL, cfg.ReadTimeout, logger)
	if err != nil {
		logger.Warn("intraday model service disabled", "error", err)
	}
	var quoteProvider fundQuoteProvider
	if cfg.FundRealtimeQuotesEnabled {
		quoteProvider = NewFundQuoteClient(cfg.ReadTimeout)
	}
	return &PredictionService{
		store:               store,
		market:              market,
		cfg:                 cfg,
		logger:              logger,
		modelClient:         modelClient,
		weeklyModelClient:   weeklyModelClient,
		intradayModelClient: intradayModelClient,
		quoteProvider:       quoteProvider,
	}
}

func (s *PredictionService) ModelLoaded() bool {
	return s.modelClient != nil || s.weeklyModelClient != nil || s.intradayModelClient != nil
}

func (s *PredictionService) PredictByFundCode(code string) (dto.PredictionData, error) {
	if len(code) != 6 || !allDigits(code) {
		return dto.PredictionData{}, ErrInvalidFundCode
	}
	fund, ok := s.store.FindFund(code)
	if !ok {
		return dto.PredictionData{}, ErrFundNotFound
	}
	return s.Predict(fund), nil
}

func (s *PredictionService) Predict(fund dto.FundItem) dto.PredictionData {
	data := s.baselinePrediction(fund)
	if s.modelClient != nil {
		if modelResult, err := s.modelClient.Predict(context.Background(), fund.FundCode); err != nil {
			s.logger.Warn("daily model prediction failed; keeping Go baseline", "fund_code", fund.FundCode, "error", err)
			data = withModelFallback(data, "next_day", err)
		} else {
			data = withModelPrediction(data, modelResult)
		}
	}
	if s.weeklyModelClient != nil {
		if modelResult, err := s.weeklyModelClient.Predict(context.Background(), fund.FundCode); err != nil {
			s.logger.Warn("weekly model prediction failed; keeping Go baseline", "fund_code", fund.FundCode, "error", err)
			data = withModelFallback(data, "next_week", err)
		} else {
			data = withModelPrediction(data, modelResult)
		}
	}
	if s.intradayModelClient != nil {
		if modelResult, err := s.intradayModelClient.Predict(context.Background(), fund.FundCode); err != nil {
			s.logger.Warn("intraday model prediction failed; keeping Go baseline", "fund_code", fund.FundCode, "error", err)
			data = withModelFallback(data, "intraday_5m", err)
		} else {
			data = withModelPrediction(data, modelResult)
		}
	}
	return data
}

func (s *PredictionService) baselinePrediction(fund dto.FundItem) dto.PredictionData {
	marketMean := s.market.AverageCNChange()
	nextPct := clamp(fund.Return1M*0.08+fund.Return3M*0.03+fund.Return1Y*0.01+marketMean*0.22+fund.ChangePct*0.10, -5, 5)
	nextConfidence := clamp(0.50+math.Min(math.Abs(nextPct)/3, 0.25), 0.35, 0.82)
	weeklyPct := clamp(fund.Return1M*0.18+fund.Return3M*0.08+fund.Return1Y*0.02+marketMean*0.35+fund.ChangePct*0.12, -12, 12)
	weeklyConfidence := clamp(0.48+math.Min(math.Abs(weeklyPct)/7, 0.24), 0.32, 0.80)
	intraPct := clamp(fund.ChangePct*0.10+marketMean*0.05, -0.8, 0.8)
	intraConfidence := clamp(0.42+math.Min(math.Abs(intraPct)/0.5, 0.22), 0.30, 0.76)

	factors := []dto.FactorItem{
		{Name: "momentum_5d", Importance: 0.24, Description: "短期净值/估值动量"},
		{Name: "market_beta", Importance: 0.21, Description: "市场指数联动"},
		{Name: "sector_momentum", Importance: 0.18, Description: "行业或主题趋势"},
		{Name: "flow_signal", Importance: 0.16, Description: "资金流代理信号"},
		{Name: "mean_reversion", Importance: 0.12, Description: "均值回归风险"},
	}

	next := predictionResult("next_day", "下一个交易日", nextPct, nextConfidence, factors, "baseline", "Go 后端基线预测；等待接入新训练模型")
	weekly := predictionResult("next_week", "未来一周", weeklyPct, weeklyConfidence, factors, "baseline", "Go 后端周频基线预测；等待接入周频冠军模型")
	intraday := predictionResult("intraday_5m", "未来5分钟", intraPct, intraConfidence, factors[:4], "baseline_no_realtime", "Go 后端盘中代理信号；等待接入分钟级行情和模型服务")
	quality := dto.PredictionDataQuality{
		HasRealtimeQuote:           fund.QuoteSource == "tencent_quote" || fund.QuoteSource == "eastmoney_fundgz",
		HasMarketIndices:           true,
		HasHoldingsData:            false,
		HasIntradayConstituentData: false,
		HasEtfFlowData:             false,
		CoverageScore:              0.33,
		MissingSources: []string{
			"基金最新持仓明细",
			"成分股分钟级成交量与涨跌幅",
			"场内ETF资金流与盘口数据",
			"按预测周期切分的历史回测标签",
		},
		Note: "Go 版已完成 API 替代骨架，预测仍为基线逻辑；接入新模型后再声明准确率。",
	}

	return dto.PredictionData{
		FundCode:           fund.FundCode,
		FundName:           fund.FundName,
		Prediction:         next,
		NextDayPrediction:  next,
		WeeklyPrediction:   weekly,
		IntradayPrediction: intraday,
		DataQuality:        quality,
		MarketSnapshot:     s.market.Snapshot(),
	}
}

func withModelPrediction(data dto.PredictionData, modelResult modelPredictionResponse) dto.PredictionData {
	direction := normalizeDirection(modelResult.Prediction.Direction)
	signalStatus := normalizeSignalStatus(modelResult.Prediction.SignalStatus, direction, modelResult.Prediction.IsActionable)
	next := dto.PredictionResult{
		Horizon:             stringOr(modelResult.Prediction.Horizon, "next_day"),
		TargetWindow:        stringOr(modelResult.Prediction.TargetWindow, "下一个交易日"),
		ModelSource:         "python_model_service",
		ModelCandidate:      modelResult.Model.Candidate,
		FeatureSet:          modelResult.Model.FeatureSet,
		ModelAsOfTime:       modelResult.AsOfTime,
		ModelCoverageStatus: dto.ModelCoverageSupported,
		ModelCoverageNote:   "当前周期使用 Python 模型服务和已处理样本推理。",
		Direction:           direction,
		DirectionConfidence: round(modelResult.Prediction.DirectionConfidence, 4),
		PredictedChangePct:  round(modelResult.Prediction.PredictedChangePct, 4),
		ChangeRange: dto.ChangeRange{
			Low:  round(modelResult.Prediction.ChangeRange.Low, 4),
			High: round(modelResult.Prediction.ChangeRange.High, 4),
		},
		PredictionInterval:  roundPredictionInterval(modelResult.Prediction.PredictionInterval),
		ReturnDecomposition: roundReturnDecomposition(modelResult.Prediction.ReturnDecomposition),
		ActionabilityGate:   roundActionabilityGate(modelResult.Prediction.ActionabilityGate),
		TopFactors:          modelFactors(modelResult.Prediction.TopFactors, data.NextDayPrediction.TopFactors),
		Reliability:         stringOr(modelResult.Prediction.Reliability, "model_service"),
		ReliabilityNote:     stringOr(modelResult.Prediction.ReliabilityNote, "模型服务预测；生产上线前仍需滚动回测与影子验证。"),
		AccuracyTarget:      0.90,
		MeetsAccuracyTarget: false,
		SignalStatus:        signalStatus,
		IsActionable:        signalStatus == dto.SignalStatusActionable,
		CalibrationNote:     "已接入训练模型服务，但尚未证明达到用户设定的日级/周级偏差目标，不能作为确定性交易信号。",
	}
	if isIntradayHorizon(next.Horizon) {
		data.IntradayPrediction = next
	} else if isWeeklyHorizon(next.Horizon) {
		data.WeeklyPrediction = next
	} else {
		data.Prediction = next
		data.NextDayPrediction = next
	}
	data.DataQuality = modelQuality(modelResult.DataQuality)
	return data
}

func withModelFallback(data dto.PredictionData, horizon string, err error) dto.PredictionData {
	status, note := modelFallbackCoverage(err)
	if isIntradayHorizon(horizon) {
		data.IntradayPrediction = applyModelCoverage(data.IntradayPrediction, status, note)
		return data
	}
	if isWeeklyHorizon(horizon) {
		data.WeeklyPrediction = applyModelCoverage(data.WeeklyPrediction, status, note)
		return data
	}
	data.NextDayPrediction = applyModelCoverage(data.NextDayPrediction, status, note)
	data.Prediction = data.NextDayPrediction
	return data
}

func modelFallbackCoverage(err error) (dto.ModelCoverageStatus, string) {
	if errors.Is(err, ErrModelUnsupportedFund) {
		return dto.ModelCoverageUnsupportedFund, "当前训练样本暂未覆盖该基金，已回退到 Go 基线。"
	}
	return dto.ModelCoverageModelUnavailable, "模型服务暂不可用，已回退到 Go 基线。"
}

func applyModelCoverage(result dto.PredictionResult, status dto.ModelCoverageStatus, note string) dto.PredictionResult {
	result.ModelCoverageStatus = status
	result.ModelCoverageNote = note
	if note != "" {
		result.ReliabilityNote = note + result.ReliabilityNote
	}
	return result
}

func modelFactors(factors []modelFactor, fallback []dto.FactorItem) []dto.FactorItem {
	if len(factors) == 0 {
		return fallback
	}
	out := make([]dto.FactorItem, 0, len(factors))
	for _, factor := range factors {
		out = append(out, dto.FactorItem{
			Name:        factor.Name,
			Importance:  round(factor.Importance, 4),
			Description: factor.Description,
		})
	}
	return out
}

func roundReturnDecomposition(raw *dto.ReturnDecomposition) *dto.ReturnDecomposition {
	if raw == nil {
		return nil
	}
	return &dto.ReturnDecomposition{
		Enabled:             raw.Enabled,
		Method:              raw.Method,
		Formula:             raw.Formula,
		IndexReturnPct:      roundFloatPointer(raw.IndexReturnPct),
		TrackingErrorPct:    roundFloatPointer(raw.TrackingErrorPct),
		DirectFundReturnPct: roundFloatPointer(raw.DirectFundReturnPct),
		IndexReturnTarget:   raw.IndexReturnTarget,
		TrackingErrorTarget: raw.TrackingErrorTarget,
	}
}

func roundPredictionInterval(raw *dto.PredictionInterval) *dto.PredictionInterval {
	if raw == nil {
		return nil
	}
	return &dto.PredictionInterval{
		Low:               round(raw.Low, 4),
		High:              round(raw.High, 4),
		Method:            raw.Method,
		Level:             roundFloatPointer(raw.Level),
		EmpiricalCoverage: roundFloatPointer(raw.EmpiricalCoverage),
	}
}

func roundActionabilityGate(raw *dto.ActionabilityGate) *dto.ActionabilityGate {
	if raw == nil {
		return nil
	}
	return &dto.ActionabilityGate{
		Actionable:                raw.Actionable,
		Reason:                    raw.Reason,
		MinHighConfidenceAccuracy: roundFloatPointer(raw.MinHighConfidenceAccuracy),
		MinHighConfidenceCoverage: roundFloatPointer(raw.MinHighConfidenceCoverage),
		HighConfidenceAccuracy:    roundFloatPointer(raw.HighConfidenceAccuracy),
		HighConfidenceCoverage:    roundFloatPointer(raw.HighConfidenceCoverage),
		MaxCalibrationECE:         roundFloatPointer(raw.MaxCalibrationECE),
		CalibrationECE:            roundFloatPointer(raw.CalibrationECE),
	}
}

func roundFloatPointer(value *float64) *float64 {
	if value == nil {
		return nil
	}
	rounded := round(*value, 4)
	return &rounded
}

func modelQuality(quality modelDataQuality) dto.PredictionDataQuality {
	coverage := 0.45
	missing := []string{
		"基金最新持仓明细",
		"成分股分钟级成交量与涨跌幅",
		"场内ETF资金流与盘口数据",
	}
	if quality.FeatureCount > 0 {
		coverage += 0.10
	}
	if quality.HasPanicFactor {
		coverage += 0.15
	} else {
		missing = append(missing, "恐慌/情绪因子")
	}
	if quality.HasFuturesFeatures {
		coverage += 0.15
	} else {
		missing = append(missing, "期货联动特征")
	}
	return dto.PredictionDataQuality{
		HasRealtimeQuote:           true,
		HasMarketIndices:           true,
		HasHoldingsData:            false,
		HasIntradayConstituentData: false,
		HasEtfFlowData:             false,
		CoverageScore:              round(clamp(coverage, 0.0, 0.85), 2),
		MissingSources:             missing,
		Note:                       stringOr(quality.Note, "模型服务已提供日级特征推理；分钟级数据仍待接入。"),
	}
}

func (s *PredictionService) WatchlistQuotes(codes []string) []dto.WatchlistItem {
	now := time.Now().UnixMilli()
	items := make([]dto.WatchlistItem, 0, len(codes))
	funds := make([]dto.FundItem, 0, len(codes))
	for _, code := range codes {
		fund, ok := s.store.FindFund(code)
		if !ok {
			continue
		}
		funds = append(funds, fund)
	}
	quotes := map[string]dto.FundItem{}
	if s.quoteProvider != nil && len(funds) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), s.cfg.ReadTimeout)
		defer cancel()
		quotes = s.quoteProvider.RefreshQuotes(ctx, funds)
	}
	for _, fund := range funds {
		if quote, ok := quotes[fund.FundCode]; ok {
			fund = mergeRealtimeQuote(fund, quote)
		}
		items = append(items, dto.WatchlistItem{
			FundCode:     fund.FundCode,
			FundName:     fund.FundName,
			FundType:     fund.FundType,
			EstimatedNAV: fund.EstimatedNAV,
			ChangePct:    fund.ChangePct,
			Direction:    direction(fund.ChangePct, 0),
			AddedAt:      now,
			QuoteDate:    fund.QuoteDate,
			QuoteSource:  fund.QuoteSource,
		})
	}
	return items
}

func mergeRealtimeQuote(fund, quote dto.FundItem) dto.FundItem {
	if quote.LatestNAV != 0 {
		fund.LatestNAV = quote.LatestNAV
	}
	if quote.EstimatedNAV != 0 {
		fund.EstimatedNAV = quote.EstimatedNAV
	}
	fund.ChangePct = quote.ChangePct
	if quote.QuoteDate != "" {
		fund.QuoteDate = quote.QuoteDate
	}
	if quote.QuoteSource != "" {
		fund.QuoteSource = quote.QuoteSource
	}
	return fund
}

func predictionResult(horizon, target string, expectedPct, confidence float64, factors []dto.FactorItem, reliability, note string) dto.PredictionResult {
	spread := 0.35 + (1-confidence)*1.1
	if horizon == "intraday_5m" {
		spread = 0.05 + (1-confidence)*0.35
	}
	direction := direction(expectedPct, threshold(horizon))
	return dto.PredictionResult{
		Horizon:             horizon,
		TargetWindow:        target,
		ModelSource:         "go_baseline",
		ModelCoverageStatus: dto.ModelCoverageBaselineOnly,
		ModelCoverageNote:   "当前预测周期未配置 Python 模型服务，使用 Go 基线。",
		Direction:           direction,
		DirectionConfidence: round(expectedPct*0+confidence, 4),
		PredictedChangePct:  round(expectedPct, 4),
		ChangeRange: dto.ChangeRange{
			Low:  round(expectedPct-spread, 4),
			High: round(expectedPct+spread, 4),
		},
		TopFactors:          factors,
		Reliability:         reliability,
		ReliabilityNote:     note,
		AccuracyTarget:      0.98,
		MeetsAccuracyTarget: false,
		SignalStatus:        normalizeSignalStatus("", direction, false),
		IsActionable:        false,
		CalibrationNote:     "未通过滚动回测证明达到98%准确率，不能作为确定性交易信号。",
	}
}

func normalizeSignalStatus(raw dto.SignalStatus, direction dto.Direction, actionable bool) dto.SignalStatus {
	switch raw {
	case dto.SignalStatusActionable, dto.SignalStatusLowConfidence, dto.SignalStatusNoSignal:
		return raw
	}
	if actionable {
		return dto.SignalStatusActionable
	}
	if direction == dto.DirectionFlat {
		return dto.SignalStatusNoSignal
	}
	return dto.SignalStatusLowConfidence
}

func direction(value, flatThreshold float64) dto.Direction {
	if value > flatThreshold {
		return dto.DirectionUp
	}
	if value < -flatThreshold {
		return dto.DirectionDown
	}
	return dto.DirectionFlat
}

func normalizeDirection(value dto.Direction) dto.Direction {
	switch value {
	case dto.DirectionUp, dto.DirectionDown, dto.DirectionFlat:
		return value
	default:
		return dto.DirectionFlat
	}
}

func stringOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func isIntradayHorizon(horizon string) bool {
	return horizon == "intraday_5m" || horizon == "intraday_3m"
}

func isWeeklyHorizon(horizon string) bool {
	return horizon == "next_week" || horizon == "weekly"
}

func threshold(horizon string) float64 {
	if isIntradayHorizon(horizon) {
		return 0.01
	}
	if isWeeklyHorizon(horizon) {
		return 0.15
	}
	return 0.05
}

func clamp(v, min, max float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return math.Min(math.Max(v, min), max)
}

func allDigits(value string) bool {
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}
