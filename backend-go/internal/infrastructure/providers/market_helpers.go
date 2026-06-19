package providers

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
	_ "time/tzdata"

	marketdomain "stock-predict-go/internal/domain/market"
	stockdomain "stock-predict-go/internal/domain/stock"
	httpclient "stock-predict-go/internal/platform/httpclient"
)

// validateMarketIndex 校验市场指数数据是否合法。
func validateMarketIndex(index marketdomain.MarketIndex) bool {
	if !isKnownIndexCode(index.Code) {
		return false
	}
	if strings.TrimSpace(index.Name) == "" || strings.TrimSpace(index.Market) == "" {
		return false
	}
	if !allFinite(index.Value, index.Change, index.ChangePct, index.High, index.Low, index.PrevClose, index.Volume) {
		return false
	}
	if index.Value <= 0 || index.High <= 0 || index.Low <= 0 || index.PrevClose <= 0 || index.High < index.Low || index.Volume < 0 {
		return false
	}
	expectedPct := (index.Value - index.PrevClose) / index.PrevClose * 100
	if math.Abs(expectedPct-index.ChangePct) > 0.05 {
		return false
	}
	return true
}

// normalizeMarketIndices 过滤并返回合法的市场指数数据。
func normalizeMarketIndices(indices []marketdomain.MarketIndex) []marketdomain.MarketIndex {
	normalized := make([]marketdomain.MarketIndex, 0, len(indices))
	for _, index := range indices {
		if validateMarketIndex(index) {
			normalized = append(normalized, index)
		}
	}
	return normalized
}

// convertUSTimeToBeijing 将美东交易时间无条件转换为北京时间（+12h）。
// 适用于腾讯/新浪等返回美东时间的数据源。东方财富 trends2 已返回北京时间，不应调用此函数。
func convertUSTimeToBeijing(timeStr string) string {
	if len(timeStr) != 5 || timeStr[2] != ':' {
		return timeStr
	}
	hour := int(timeStr[0]-'0')*10 + int(timeStr[1]-'0')
	minute := int(timeStr[3]-'0')*10 + int(timeStr[4]-'0')
	beijingHour := hour + 12
	if beijingHour >= 24 {
		beijingHour -= 24
	}
	return fmt.Sprintf("%02d:%02d", beijingHour, minute)
}

// normalizeIndexMinutePoints 去重、过滤并按时间排序指数分时数据。
func normalizeIndexMinutePoints(points []marketdomain.IndexMinutePoint, market ...Market) []marketdomain.IndexMinutePoint {
	isUS := len(market) > 0 && (market[0] == MarketUS || market[0] == MarketUSBeijing)
	convertTZ := len(market) > 0 && market[0] == MarketUS
	byTime := make(map[string]marketdomain.IndexMinutePoint, len(points))
	for _, point := range points {
		timeStr := point.Time
		if convertTZ {
			timeStr = convertUSTimeToBeijing(timeStr)
			point.Time = timeStr
		}
		if !isValidMarketMinute(timeStr, market...) {
			continue
		}
		if !allFinite(point.Price, point.AvgPrice) || point.Price <= 0 || point.Volume < 0 {
			continue
		}
		byTime[timeStr] = point
	}

	times := make([]string, 0, len(byTime))
	for timeValue := range byTime {
		times = append(times, timeValue)
	}
	if isUS {
		sort.SliceStable(times, func(i, j int) bool {
			mi := int(times[i][0]-'0')*600 + int(times[i][1]-'0')*60 + int(times[i][3]-'0')*10 + int(times[i][4]-'0')
			mj := int(times[j][0]-'0')*600 + int(times[j][1]-'0')*60 + int(times[j][3]-'0')*10 + int(times[j][4]-'0')
			if mi >= 21*60 && mj < 21*60 {
				return true
			}
			if mi < 21*60 && mj >= 21*60 {
				return false
			}
			return mi < mj
		})
	} else {
		sort.Strings(times)
	}

	normalized := make([]marketdomain.IndexMinutePoint, 0, len(times))
	for _, timeValue := range times {
		normalized = append(normalized, byTime[timeValue])
	}
	return normalized
}

// mergeIndexMinutePoints 合并两组分时数据，new 中同时间点的数据覆盖 old。
func mergeIndexMinutePoints(old, new []marketdomain.IndexMinutePoint) []marketdomain.IndexMinutePoint {
	if len(old) == 0 {
		return new
	}
	if len(new) == 0 {
		return old
	}
	byTime := make(map[string]marketdomain.IndexMinutePoint, len(old)+len(new))
	for _, p := range old {
		byTime[p.Time] = p
	}
	for _, p := range new {
		byTime[p.Time] = p
	}
	times := make([]string, 0, len(byTime))
	for t := range byTime {
		times = append(times, t)
	}
	sortMinuteTimes(times)
	result := make([]marketdomain.IndexMinutePoint, 0, len(times))
	for _, t := range times {
		result = append(result, byTime[t])
	}
	return result
}

// sortMinuteTimes 按交易时间排序分钟数据。当数据跨越午夜（如美股 21:30-04:00）时，
// 21:xx 排在 00:xx 之前；否则按字典序排列。
func sortMinuteTimes(times []string) {
	hasLateNight := false
	hasEarlyMorning := false
	for _, t := range times {
		if len(t) >= 2 {
			h := int(t[0]-'0')*10 + int(t[1]-'0')
			if h >= 21 {
				hasLateNight = true
			}
			if h <= 4 {
				hasEarlyMorning = true
			}
		}
	}
	if hasLateNight && hasEarlyMorning {
		sort.SliceStable(times, func(i, j int) bool {
			mi := minuteTotal(times[i])
			mj := minuteTotal(times[j])
			if mi >= 21*60 && mj < 21*60 {
				return true
			}
			if mi < 21*60 && mj >= 21*60 {
				return false
			}
			return mi < mj
		})
	} else {
		sort.Strings(times)
	}
}

// minuteTotal 将 "HH:MM" 转换为当天的分钟总数。
func minuteTotal(s string) int {
	if len(s) < 5 {
		return 0
	}
	return int(s[0]-'0')*600 + int(s[1]-'0')*60 + int(s[3]-'0')*10 + int(s[4]-'0')
}

// normalizeIndexKlinePoints 过滤并按日期排序指数 K 线数据。
func normalizeIndexKlinePoints(points []marketdomain.IndexKlinePoint) []marketdomain.IndexKlinePoint {
	normalized := make([]marketdomain.IndexKlinePoint, 0, len(points))
	for _, point := range points {
		if !isISODate(point.Date) {
			continue
		}
		if !allFinite(point.Open, point.Close, point.High, point.Low, point.Amount) {
			continue
		}
		if point.Open <= 0 || point.Close <= 0 || point.High <= 0 || point.Low <= 0 || point.High < point.Low || point.Volume < 0 || point.Amount < 0 {
			continue
		}
		normalized = append(normalized, point)
	}
	sort.SliceStable(normalized, func(i, j int) bool {
		return normalized[i].Date < normalized[j].Date
	})
	return normalized
}

// normalizeStockRankingItems 过滤、排序并截断股票排行数据。
func normalizeStockRankingItems(items []stockdomain.StockRankingItem, rankingType string, size int) []stockdomain.StockRankingItem {
	normalized := make([]stockdomain.StockRankingItem, 0, len(items))
	for _, item := range items {
		if !isSixDigitQuoteCode(item.StockCode) || strings.TrimSpace(item.StockName) == "" {
			continue
		}
		if !allFinite(item.ChangePct, item.CurrentPrice, item.Volume, item.Amount) {
			continue
		}
		if item.Volume < 0 || item.Amount < 0 {
			continue
		}
		normalized = append(normalized, item)
	}

	sort.SliceStable(normalized, func(i, j int) bool {
		if rankingType == "losers" {
			return normalized[i].ChangePct < normalized[j].ChangePct
		}
		if rankingType == "volume" {
			return normalized[i].Volume > normalized[j].Volume
		}
		return normalized[i].ChangePct > normalized[j].ChangePct
	})

	if size > 0 && len(normalized) > size {
		normalized = normalized[:size]
	}
	for i := range normalized {
		normalized[i].Rank = i + 1
	}
	return normalized
}

// allFinite 检查所有浮点数是否为有限值（非 NaN、非 Inf）。
func allFinite(values ...float64) bool {
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return false
		}
	}
	return true
}

// isKnownIndexCode 检查代码是否为已知指数代码。
func isKnownIndexCode(code string) bool {
	if isCNIndex(code) {
		return true
	}
	_, hk := hkIndexMeta[code]
	_, us := usIndexMeta[code]
	return hk || us
}

// isSixDigitQuoteCode 检查代码是否为6位纯数字行情代码。
func isSixDigitQuoteCode(code string) bool {
	return len(code) == 6 && httpclient.IsAllDigits(code)
}

// isISODate 检查字符串是否为 ISO 日期格式（YYYY-MM-DD）。
func isISODate(value string) bool {
	if len(value) != 10 || value[4] != '-' || value[7] != '-' {
		return false
	}
	return httpclient.IsAllDigits(value[:4]) && httpclient.IsAllDigits(value[5:7]) && httpclient.IsAllDigits(value[8:])
}

// isValidMarketMinute 检查时间字符串是否为有效的交易分钟（HH:MM）。
func isValidMarketMinute(value string, market ...Market) bool {
	if len(value) != 5 || value[2] != ':' || !httpclient.IsAllDigits(value[:2]) || !httpclient.IsAllDigits(value[3:]) {
		return false
	}
	hour := int(value[0]-'0')*10 + int(value[1]-'0')
	minute := int(value[3]-'0')*10 + int(value[4]-'0')
	if minute > 59 {
		return false
	}
	total := hour*60 + minute
	usMarket := len(market) > 0 && market[0] == MarketUS

	// A股: 9:30-11:30, 13:00-15:00
	if (total >= 9*60+30 && total <= 11*60+30) || (total >= 13*60 && total <= 15*60) {
		return true
	}
	// 港股: 9:30-12:00, 13:00-16:00
	if (total >= 9*60+30 && total <= 12*60) || (total >= 13*60 && total <= 16*60) {
		return true
	}
	// 美股（北京时间）: 21:30-23:59, 00:00-04:00
	if total >= 21*60+30 || total <= 4*60 {
		return true
	}
	// 美股（美东时间）: 9:30-16:00（新浪接口返回美东时间，normalize 前仍需通过）
	if !usMarket && total >= 9*60+30 && total <= 16*60 {
		return true
	}
	return false
}

// Market 表示市场类型（A股、港股、美股）。
type Market string

const (
	MarketCN        Market = "cn"    // A股市场
	MarketHK        Market = "hk"    // 港股市场
	MarketUS        Market = "us"    // 美股市场（美东时间，需时区转换）
	MarketUSBeijing Market = "us_bj" // 美股市场（北京时间，无需时区转换，如东方财富 trends2）
)

// marketLocations 各市场对应的时区。
var marketLocations = map[Market]*time.Location{
	MarketCN: mustLoadLocation("Asia/Shanghai"),
	MarketHK: mustLoadLocation("Asia/Hong_Kong"),
	MarketUS: mustLoadLocation("America/New_York"),
}

// mustLoadLocation 加载时区，失败时 panic。
func mustLoadLocation(name string) *time.Location {
	location, err := time.LoadLocation(name)
	if err != nil {
		panic("load market timezone " + name + ": " + err.Error())
	}
	return location
}

// IsMarketOpen 检查指定市场当前是否在交易时段。
func IsMarketOpen(market Market) bool {
	return IsMarketOpenAt(market, time.Now())
}

// IsMarketOpenAt 检查指定市场在给定时间是否在交易时段。
func IsMarketOpenAt(market Market, now time.Time) bool {
	location := marketLocations[market]
	if location == nil {
		return false
	}
	exchangeTime := now.In(location)
	if exchangeTime.Weekday() == time.Saturday || exchangeTime.Weekday() == time.Sunday {
		return false
	}
	hour, minute, _ := exchangeTime.Clock()
	t := hour*60 + minute

	switch market {
	case MarketCN:
		return (t >= 9*60+30 && t <= 11*60+30) || (t >= 13*60 && t <= 15*60)
	case MarketHK:
		return (t >= 9*60+30 && t <= 12*60) || (t >= 13*60 && t <= 16*60)
	case MarketUS:
		return t >= 9*60+30 && t <= 16*60
	default:
		return false
	}
}

// DetectMarket 根据股票代码推断所属市场类型。
func DetectMarket(code string) Market {
	if len(code) == 0 {
		return MarketCN
	}
	// US stocks: alphabetic codes like AAPL, MSFT
	if code[0] >= 'A' && code[0] <= 'Z' {
		return MarketUS
	}
	// HK stocks: 5-digit numeric codes like 00700, 09988
	if len(code) == 5 {
		allDigit := true
		for _, c := range code {
			if c < '0' || c > '9' {
				allDigit = false
				break
			}
		}
		if allDigit {
			return MarketHK
		}
	}
	// A stocks: 6-digit numeric codes
	return MarketCN
}
