package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	marketdomain "stock-predict-go/internal/domain/market"
	httpclient "stock-predict-go/internal/platform/httpclient"
)

// FetchStockMinute 获取个股分时数据。根据市场类型自动选择对应的腾讯接口参数。
func (c *IndexQuoteClient) FetchStockMinute(ctx context.Context, code string) []marketdomain.IndexMinutePoint {
	cacheKey := "stock_minute:" + code
	if cached, ok := c.minuteCache.Get(cacheKey); ok {
		if val, ok2 := cached.([]marketdomain.IndexMinutePoint); ok2 {
			return val
		}
	}

	market := DetectMarket(code)
	var result []marketdomain.IndexMinutePoint

	switch market {
	case MarketCN:
		prefix := stockMarketPrefix(code)
		if prefix != "" {
			result = c.fetchTencentMinuteData(ctx, prefix+code, MarketCN)
		}
	case MarketHK:
		result = c.fetchTencentMinuteData(ctx, "hk"+code, MarketHK)
	case MarketUS:
		result = c.fetchTencentMinuteData(ctx, "us"+code, MarketUS)
	}

	if len(result) > 0 {
		c.minuteCache.Set(cacheKey, result)
	}
	return result
}

// fetchTencentMinuteData 通过腾讯分时接口获取分钟级行情数据。
// symbol 格式示例：A 股 "sh600519"、港股 "hk00700"、美股 "usAAPL"。
func (c *IndexQuoteClient) fetchTencentMinuteData(ctx context.Context, symbol string, market Market) []marketdomain.IndexMinutePoint {
	url := fmt.Sprintf("https://web.ifzq.gtimg.cn/appstock/app/minute/query?code=%s", symbol)
	if !isAllowedURL(url) {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Referer", "https://gu.qq.com/")

	resp, err := c.resilient.Do(ctx, SourceTencent, req)
	if err != nil {
		c.logger.Warn("tencent minute request failed", "symbol", symbol, "error", err)
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil
	}
	payload, err := io.ReadAll(io.LimitReader(resp.Body, int64(MaxHTTPPayloadBytes)))
	if err != nil {
		return nil
	}

	var raw struct {
		Code int `json:"code"`
		Data map[string]struct {
			Data struct {
				Data []string `json:"data"`
			} `json:"data"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		c.logger.Warn("tencent minute parse failed", "symbol", symbol, "error", err)
		return nil
	}
	if raw.Code != 0 {
		return nil
	}

	// Find data entries (key varies by market)
	var rawLines []string
	for _, v := range raw.Data {
		if len(v.Data.Data) > 0 {
			rawLines = v.Data.Data
			break
		}
	}
	if len(rawLines) == 0 {
		return nil
	}

	points := make([]marketdomain.IndexMinutePoint, 0, len(rawLines))
	var prevCumVol int64
	for _, line := range rawLines {
		parts := strings.Split(line, " ")
		if len(parts) < 3 {
			continue
		}
		timeRaw := parts[0]
		if len(timeRaw) < 4 {
			continue
		}
		timeStr := timeRaw[:2] + ":" + timeRaw[2:]
		price := httpclient.ParseQuoteFloat(parts[1])
		cumVol := int64(httpclient.ParseQuoteFloat(parts[2]))
		var cumAmount float64
		if len(parts) >= 4 {
			cumAmount = httpclient.ParseQuoteFloat(parts[3])
		}
		var vol int64
		if len(points) > 0 {
			vol = cumVol - prevCumVol
		} else {
			vol = cumVol
		}
		prevCumVol = cumVol
		var avgPrice float64
		if cumVol > 0 && cumAmount > 0 {
			avgPrice = cumAmount / float64(cumVol) / 100 // cumVol is in "手"(lots of 100 shares), cumAmount is in 元
		} else if len(points) > 0 {
			avgPrice = points[len(points)-1].AvgPrice
		} else {
			avgPrice = price
		}
		points = append(points, marketdomain.IndexMinutePoint{
			Time:     timeStr,
			Price:    price,
			AvgPrice: avgPrice,
			Volume:   vol,
		})
	}
	return normalizeIndexMinutePoints(points, market)
}

// fetchUSIndexMinuteTencent 通过腾讯分时接口获取美股指数分钟数据。
func (c *IndexQuoteClient) fetchUSIndexMinuteTencent(ctx context.Context, code string) []marketdomain.IndexMinutePoint {
	tencentCode, ok := hkusIndexTencentKlineCode[code]
	if !ok {
		return nil
	}
	return c.fetchTencentMinuteData(ctx, tencentCode, MarketUS)
}

// fetchUSIndexMinuteEastmoney 通过东方财富 trends2 接口获取美股指数分钟数据。
func (c *IndexQuoteClient) fetchUSIndexMinuteEastmoney(ctx context.Context, code string) []marketdomain.IndexMinutePoint {
	secid, ok := usIndexEastmoneySecid[code]
	if !ok {
		return nil
	}
	url := fmt.Sprintf("https://push2.eastmoney.com/api/qt/stock/trends2/get?secid=%s&fields1=f1,f2,f3,f4,f5,f6,f7,f8,f9,f10,f11,f12,f13&fields2=f51,f52,f53,f54,f55,f56,f57,f58&iscr=0&ndays=1", secid)
	if !isAllowedURL(url) {
		return nil
	}

	payload, err := c.fetchViaGoHTTP(ctx, url)
	if err != nil || len(payload) == 0 {
		c.logger.Warn("Go HTTP client failed for eastmoney trends2", "code", code, "error", err)
		return nil
	}

	var raw struct {
		Rc   int `json:"rc"`
		Data struct {
			PreClose    float64  `json:"preClose"`
			TrendsTotal int      `json:"trendsTotal"`
			Trends      []string `json:"trends"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		c.logger.Warn("eastmoney US index trends2 parse failed", "code", code, "error", err)
		return nil
	}
	if raw.Rc != 0 || len(raw.Data.Trends) == 0 {
		c.logger.Warn("eastmoney US index trends2 empty", "code", code, "rc", raw.Rc, "trends", len(raw.Data.Trends))
		return nil
	}

	// trends 格式: "2026-06-05 21:30,51610.02,51610.02,51610.02,51610.02,0,0.00,51610.020"
	// 字段: 日期时间,价格,均价,最高,最低,成交量,成交额,均价2
	points := make([]marketdomain.IndexMinutePoint, 0, len(raw.Data.Trends))
	for _, line := range raw.Data.Trends {
		parts := strings.Split(line, ",")
		if len(parts) < 6 {
			continue
		}
		// 提取时间部分 "HH:MM"
		dateTime := parts[0]
		timeStr := dateTime
		if idx := strings.Index(dateTime, " "); idx >= 0 {
			timeStr = dateTime[idx+1:]
		}
		vol := int64(httpclient.ParseQuoteFloat(parts[5]))
		// 东方财富 trends2 返回全天数据，非交易时段 volume=0，过滤掉
		if vol == 0 {
			continue
		}
		price := httpclient.ParseQuoteFloat(parts[1])
		avgPrice := httpclient.ParseQuoteFloat(parts[2])
		points = append(points, marketdomain.IndexMinutePoint{
			Time:     timeStr,
			Price:    price,
			AvgPrice: avgPrice,
			Volume:   vol,
		})
	}
	return normalizeIndexMinutePoints(points, MarketUSBeijing) // 东方财富 trends2 已返回北京时间，无需时区转换
}

// fetchViaGoHTTP 使用 Go 标准 HTTP 客户端（带重试）请求指定 URL 并返回响应体。
// 最多重试 2 次，重试间隔 500 毫秒。
func (c *IndexQuoteClient) fetchViaGoHTTP(ctx context.Context, url string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(500 * time.Millisecond):
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Referer", "https://quote.eastmoney.com/")
		req.Header.Set("Connection", "close")

		resp, err := c.resilient.Do(ctx, SourceEastmoney, req)
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("bad status %d", resp.StatusCode)
			resp.Body.Close()
			continue
		}
		payload, err := io.ReadAll(io.LimitReader(resp.Body, int64(MaxHTTPPayloadBytes)))
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		return payload, nil
	}
	return nil, lastErr
}

// fetchUSIndexMinuteTencentKline 通过腾讯1分钟K线接口获取美股指数分钟数据。
// 当腾讯分时接口不支持美股指数时，此接口可作为备选数据源。
func (c *IndexQuoteClient) fetchUSIndexMinuteTencentKline(ctx context.Context, code string) []marketdomain.IndexMinutePoint {
	tencentCode, ok := hkusIndexTencentKlineCode[code]
	if !ok {
		return nil
	}
	// 腾讯1分钟K线API: param=code,m1,,240,qfq
	url := fmt.Sprintf("https://web.ifzq.gtimg.cn/appstock/app/fqkline/get?param=%s,m1,,240,qfq", tencentCode)
	if !isAllowedURL(url) {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Referer", "https://gu.qq.com/")

	resp, err := c.resilient.Do(ctx, SourceTencent, req)
	if err != nil {
		c.logger.Warn("tencent US index minute kline request failed", "code", code, "error", err)
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil
	}
	payload, err := io.ReadAll(io.LimitReader(resp.Body, int64(MaxHTTPPayloadBytes)))
	if err != nil {
		return nil
	}

	var raw struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		c.logger.Warn("tencent US index minute kline parse failed", "code", code, "error", err)
		return nil
	}
	if raw.Code != 0 {
		return nil
	}
	// 处理 data 为空数组的情况（美股指数 m1 不支持时返回 data:[]）
	if len(raw.Data) == 0 || string(raw.Data) == "[]" {
		return nil
	}

	var dataMap map[string]struct {
		M1 [][]string `json:"m1"`
	}
	if err := json.Unmarshal(raw.Data, &dataMap); err != nil {
		c.logger.Warn("tencent US index minute kline data parse failed", "code", code, "error", err)
		return nil
	}

	var entries [][]string
	for _, v := range dataMap {
		if len(v.M1) > 0 {
			entries = v.M1
			break
		}
	}
	if len(entries) == 0 {
		return nil
	}

	points := make([]marketdomain.IndexMinutePoint, 0, len(entries))
	var prevClose float64
	for _, entry := range entries {
		if len(entry) < 6 {
			continue
		}
		dateTime := entry[0]
		timeStr := dateTime
		if idx := strings.Index(dateTime, " "); idx >= 0 {
			timeStr = dateTime[idx+1:]
		}
		closePrice := httpclient.ParseQuoteFloat(entry[2])
		vol := int64(httpclient.ParseQuoteFloat(entry[5]))
		var avgPrice float64
		if prevClose > 0 {
			avgPrice = (prevClose + closePrice) / 2
		} else {
			avgPrice = closePrice
		}
		prevClose = closePrice
		points = append(points, marketdomain.IndexMinutePoint{
			Time:     timeStr,
			Price:    closePrice,
			AvgPrice: avgPrice,
			Volume:   vol,
		})
	}
	return normalizeIndexMinutePoints(points, MarketUS)
}
