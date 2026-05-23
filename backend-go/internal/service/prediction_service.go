package service

import (
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
	store  FundRepository
	market *MarketService
	cfg    config.Config
	logger *slog.Logger
}

func NewPredictionService(store FundRepository, market *MarketService, cfg config.Config, logger *slog.Logger) *PredictionService {
	return &PredictionService{store: store, market: market, cfg: cfg, logger: logger}
}

func (s *PredictionService) ModelLoaded() bool {
	return s.cfg.ModelServiceURL != ""
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
	marketMean := s.market.AverageCNChange()
	nextPct := clamp(fund.Return1M*0.08+fund.Return3M*0.03+fund.Return1Y*0.01+marketMean*0.22+fund.ChangePct*0.10, -5, 5)
	nextConfidence := clamp(0.50+math.Min(math.Abs(nextPct)/3, 0.25), 0.35, 0.82)
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
	intraday := predictionResult("intraday_5m", "未来5分钟", intraPct, intraConfidence, factors[:4], "baseline_no_realtime", "Go 后端盘中代理信号；等待接入分钟级行情和模型服务")
	quality := dto.PredictionDataQuality{
		HasRealtimeQuote:           true,
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
		IntradayPrediction: intraday,
		DataQuality:        quality,
		MarketSnapshot:     s.market.Snapshot(),
	}
}

func (s *PredictionService) WatchlistQuotes(codes []string) []dto.WatchlistItem {
	now := time.Now().UnixMilli()
	items := make([]dto.WatchlistItem, 0, len(codes))
	for _, code := range codes {
		fund, ok := s.store.FindFund(code)
		if !ok {
			continue
		}
		items = append(items, dto.WatchlistItem{
			FundCode:     fund.FundCode,
			FundName:     fund.FundName,
			FundType:     fund.FundType,
			EstimatedNAV: fund.EstimatedNAV,
			ChangePct:    fund.ChangePct,
			Direction:    direction(fund.ChangePct, 0),
			AddedAt:      now,
		})
	}
	return items
}

func predictionResult(horizon, target string, expectedPct, confidence float64, factors []dto.FactorItem, reliability, note string) dto.PredictionResult {
	spread := 0.35 + (1-confidence)*1.1
	if horizon == "intraday_5m" {
		spread = 0.05 + (1-confidence)*0.35
	}
	return dto.PredictionResult{
		Horizon:             horizon,
		TargetWindow:        target,
		Direction:           direction(expectedPct, threshold(horizon)),
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
		IsActionable:        false,
		CalibrationNote:     "未通过滚动回测证明达到98%准确率，不能作为确定性交易信号。",
	}
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

func threshold(horizon string) float64 {
	if horizon == "intraday_5m" {
		return 0.01
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
