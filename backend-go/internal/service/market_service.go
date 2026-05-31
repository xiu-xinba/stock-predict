package service

import (
	"hash/fnv"
	"log/slog"
	"math"
	"sort"
	"time"

	"stock-predict-go/internal/dto"
	"stock-predict-go/internal/util"
)

type MarketService struct {
	logger *slog.Logger
}

type indexConfig struct {
	Code   string
	Name   string
	Market string
	Base   float64
	Vol    float64
}

var indexConfigs = []indexConfig{
	{Code: "000001", Name: "上证指数", Market: "cn", Base: 3500, Vol: 1.8},
	{Code: "399001", Name: "深证成指", Market: "cn", Base: 11000, Vol: 2.2},
	{Code: "399006", Name: "创业板指", Market: "cn", Base: 2200, Vol: 2.6},
	{Code: "HSI", Name: "恒生指数", Market: "hk", Base: 19000, Vol: 2.0},
	{Code: "HSTECH", Name: "恒生科技指数", Market: "hk", Base: 4200, Vol: 3.0},
	{Code: "DJI", Name: "道琼斯工业平均", Market: "us", Base: 39000, Vol: 1.5},
	{Code: "IXIC", Name: "纳斯达克综合", Market: "us", Base: 16500, Vol: 2.4},
	{Code: "SPX", Name: "标普500", Market: "us", Base: 5200, Vol: 1.8},
}

func NewMarketService(logger *slog.Logger) *MarketService {
	return &MarketService{logger: logger}
}

func (s *MarketService) Indices() []dto.MarketIndex {
	now := time.Now()
	items := make([]dto.MarketIndex, 0, len(indexConfigs))
	for _, cfg := range indexConfigs {
		changePct := deterministicChange(cfg.Code, cfg.Vol, now)
		prev := cfg.Base
		value := util.RoundVal(cfg.Base*(1+changePct/100), 2)
		change := util.RoundVal(value-prev, 2)
		items = append(items, dto.MarketIndex{
			Code:          cfg.Code,
			Name:          cfg.Name,
			Market:        cfg.Market,
			Value:         value,
			Change:        change,
			ChangePct:     changePct,
			High:          util.RoundVal(math.Max(value, prev)*(1+cfg.Vol/1000), 2),
			Low:           util.RoundVal(math.Min(value, prev)*(1-cfg.Vol/1000), 2),
			PrevClose:     prev,
			Volume:        util.RoundVal(100000000+float64(stableIndexHash(cfg.Code)%900000000), 0),
			MiniChartData: miniChart(prev, value, cfg.Code),
			UpdateTime:    now.Format(time.RFC3339Nano),
			DataSource:    "go_baseline",
		})
	}
	return items
}

func SortRanking(items []dto.FundRankingItem, rankingType string) {
	sort.SliceStable(items, func(i, j int) bool {
		if rankingType == "losers" {
			return items[i].ChangePct < items[j].ChangePct
		}
		return items[i].ChangePct > items[j].ChangePct
	})
	for i := range items {
		items[i].Rank = i + 1
	}
}

func deterministicChange(code string, vol float64, now time.Time) float64 {
	day := now.YearDay() + now.Year()*400
	raw := int(stableIndexHash(code+now.Location().String())%10000) + day*37
	centered := float64(raw%2000)/1000 - 1
	return util.RoundVal(centered*vol, 2)
}

func miniChart(prev, current float64, code string) []float64 {
	points := make([]float64, 60)
	hash := float64(stableIndexHash(code)%1000) / 1000
	for i := range points {
		progress := float64(i) / float64(len(points)-1)
		noise := math.Sin(progress*math.Pi*3+hash*math.Pi) * math.Abs(current-prev) * 0.18
		points[i] = util.RoundVal(prev+(current-prev)*progress+noise, 2)
	}
	return points
}

func stableIndexHash(value string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(value))
	return h.Sum32()
}
