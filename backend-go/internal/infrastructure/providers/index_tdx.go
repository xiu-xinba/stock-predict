package providers

import (
	"context"
	"strconv"
	"time"

	marketdomain "stock-predict-go/internal/domain/market"

	"gitee.com/quant1x/gotdx"
	"gitee.com/quant1x/gotdx/proto"
)

// fetchCNIndexMinuteTDX 通过通达信接口获取 A 股指数分时数据。
// 内部使用 recover 捕获 gotdx 可能产生的 panic，确保调用安全。
func (c *IndexQuoteClient) fetchCNIndexMinuteTDX(_ context.Context, code string) []marketdomain.IndexMinutePoint {
	var points []marketdomain.IndexMinutePoint
	func() {
		defer func() {
			if r := recover(); r != nil {
				c.logger.Warn("gotdx panicked", "method", "fetchCNIndexMinuteTDX", "code", code, "recover", r)
			}
		}()
		api := gotdx.GetTdxApi()
		if api == nil {
			return
		}
		symbol := tdxIndexSymbol(code)
		dateStr := time.Now().Format("20060102")
		today, _ := strconv.ParseUint(dateStr, 10, 32)
		reply, err := api.GetHistoryMinuteTimeData(symbol, uint32(today))
		if err != nil {
			c.logger.Warn("gotdx GetHistoryMinuteTimeData failed", "code", code, "error", err)
			return
		}
		if reply == nil || len(reply.List) == 0 {
			return
		}
		market := cnIndexMarkets[code]
		points = make([]marketdomain.IndexMinutePoint, 0, len(reply.List))
		var cumVol int64
		var cumAmount float64
		for i, m := range reply.List {
			minute := i + 1
			var timeStr string
			if minute <= 120 {
				timeStr = minuteToTime(9*60+30+minute, true)
			} else {
				timeStr = minuteToTime(13*60+(minute-120), false)
			}
			cumVol += int64(m.Vol)
			cumAmount += float64(m.Vol) * float64(m.Price)
			var avgPrice float64
			if cumVol > 0 {
				avgPrice = cumAmount / float64(cumVol)
			}
			points = append(points, marketdomain.IndexMinutePoint{
				Time:     timeStr,
				Price:    float64(m.Price),
				AvgPrice: avgPrice,
				Volume:   int64(m.Vol),
			})
			_ = market
		}
	}()
	return normalizeIndexMinutePoints(points, MarketCN)
}

// fetchCNIndexKlineTDX 通过通达信接口获取 A 股指数日K线数据。
// 内部使用 recover 捕获 gotdx 可能产生的 panic，确保调用安全。
func (c *IndexQuoteClient) fetchCNIndexKlineTDX(_ context.Context, code string, count int) []marketdomain.IndexKlinePoint {
	var points []marketdomain.IndexKlinePoint
	func() {
		defer func() {
			if r := recover(); r != nil {
				c.logger.Warn("gotdx panicked", "method", "fetchCNIndexKlineTDX", "code", code, "recover", r)
			}
		}()
		api := gotdx.GetTdxApi()
		if api == nil {
			return
		}
		symbol := tdxIndexSymbol(code)
		reply, err := api.GetIndexBars(symbol, proto.KLINE_TYPE_RI_K, 0, uint16(count))
		if err != nil {
			c.logger.Warn("gotdx GetIndexBars failed", "code", code, "error", err)
			return
		}
		if reply == nil || len(reply.List) == 0 {
			return
		}
		points = make([]marketdomain.IndexKlinePoint, 0, len(reply.List))
		for _, bar := range reply.List {
			points = append(points, marketdomain.IndexKlinePoint{
				Date:   bar.DateTime[:10],
				Open:   bar.Open,
				Close:  bar.Close,
				High:   bar.High,
				Low:    bar.Low,
				Volume: int64(bar.Vol),
				Amount: bar.Amount,
			})
		}
	}()
	return normalizeIndexKlinePoints(points)
}

// fixIsClosed 根据本地时间修正指数的 IsClosed 字段。
// A 股：周一至周五 9:30-15:00 CST
// 港股：周一至周五 9:30-16:00 HKT（与 CST 同时区）
// 美股：周一至周五 21:30-次日04:00 ET，转换为 CST 后为 21:30-次日04:00 或 09:30-16:00
func (c *IndexQuoteClient) fixIsClosed(indices []marketdomain.MarketIndex) {
	now := time.Now()
	weekday := now.Weekday()
	isWeekend := weekday == time.Saturday || weekday == time.Sunday
	hourMin := now.Hour()*60 + now.Minute()

	for i := range indices {
		market := indices[i].Market
		var isTradingTime bool

		switch market {
		case "sh", "sz", "cn":
			// A-share: 9:30-15:00 CST
			isTradingTime = !isWeekend && hourMin >= 9*60+30 && hourMin <= 15*60
		case "hk":
			// HK: 9:30-16:00 HKT (same timezone as CST)
			isTradingTime = !isWeekend && hourMin >= 9*60+30 && hourMin <= 16*60
		case "us":
			// US markets: 21:30-04:00 next day in CST
			// In CST: 21:30-24:00 same day OR 00:00-04:00 next day
			// Saturday 00:00-04:00 is still Friday night session
			// Sunday 21:30-24:00 is Monday's pre-market session start
			if weekday == time.Saturday && hourMin < 4*60 {
				// Saturday before 4:00 AM - Friday night session still open
				isTradingTime = true
			} else if weekday == time.Saturday || weekday == time.Sunday {
				// Saturday after 4:00 AM or Sunday before 21:30 - closed
				isTradingTime = false
			} else if weekday == time.Friday && hourMin >= 4*60 && hourMin < 21*60+30 {
				// Friday 4:00 AM - 21:30 - closed between sessions
				isTradingTime = false
			} else if weekday == time.Monday && hourMin < 4*60 {
				// Monday before 4:00 AM - Sunday night session still open
				isTradingTime = true
			} else {
				isTradingTime = hourMin >= 21*60+30 || hourMin < 4*60
			}
		default:
			continue
		}

		indices[i].IsClosed = !isTradingTime
	}
}
