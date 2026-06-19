package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	marketdomain "stock-predict-go/internal/domain/market"
	httpclient "stock-predict-go/internal/platform/httpclient"
)

// fetchHKUSIndexKlineTencent 通过腾讯日K线接口获取港股/美股指数的K线数据。
// 当腾讯返回的美股指数数据不足时，自动降级到东方财富接口。
func (c *IndexQuoteClient) fetchHKUSIndexKlineTencent(ctx context.Context, code string, count int) []marketdomain.IndexKlinePoint {
	tencentCode, ok := hkusIndexTencentKlineCode[code]
	if !ok {
		return nil
	}
	url := fmt.Sprintf("https://web.ifzq.gtimg.cn/appstock/app/fqkline/get?param=%s,day,,,%d,qfq", tencentCode, count)
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
		c.logger.Warn("tencent HK/US kline request failed", "code", code, "error", err)
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
			Day [][]string `json:"day"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		c.logger.Warn("tencent HK/US kline parse failed", "code", code, "error", err)
		return nil
	}
	if raw.Code != 0 {
		return nil
	}

	// Find the data entry (key may vary)
	var entry struct {
		Day [][]string `json:"day"`
	}
	found := false
	for _, v := range raw.Data {
		if len(v.Day) > 0 {
			entry = v
			found = true
			break
		}
	}
	if !found || len(entry.Day) == 0 {
		return nil
	}

	// Tencent returns only 1 day for US indices during trading hours; fallback to Eastmoney
	if usIndexEastmoneySecid[code] != "" && len(entry.Day) < count/2 {
		c.logger.Info("tencent HK/US kline insufficient, falling back to eastmoney", "code", code, "tencent_count", len(entry.Day), "requested", count)
		emPoints := c.fetchUSIndexKlineEastmoney(ctx, code, count)
		if len(emPoints) > len(entry.Day) {
			if c.health != nil {
				c.health.RecordSuccess("eastmoney")
			}
			return emPoints
		}
	}

	points := make([]marketdomain.IndexKlinePoint, 0, len(entry.Day))
	for _, d := range entry.Day {
		if len(d) < 6 {
			continue
		}
		points = append(points, marketdomain.IndexKlinePoint{
			Date:   d[0],
			Open:   httpclient.ParseQuoteFloat(d[1]),
			Close:  httpclient.ParseQuoteFloat(d[2]),
			High:   httpclient.ParseQuoteFloat(d[3]),
			Low:    httpclient.ParseQuoteFloat(d[4]),
			Volume: int64(httpclient.ParseQuoteFloat(d[5])),
			Amount: 0,
		})
	}
	return normalizeIndexKlinePoints(points)
}

// fetchUSIndexKlineEastmoney 通过东方财富K线接口获取美股指数的日K线数据。
func (c *IndexQuoteClient) fetchUSIndexKlineEastmoney(ctx context.Context, code string, count int) []marketdomain.IndexKlinePoint {
	secid, ok := usIndexEastmoneySecid[code]
	if !ok {
		return nil
	}
	url := fmt.Sprintf("https://push2his.eastmoney.com/api/qt/stock/kline/get?secid=%s&fields1=f1,f2,f3,f4,f5,f6&fields2=f51,f52,f53,f54,f55,f56,f57&klt=101&fqt=0&end=20500101&lmt=%d", secid, count)
	if !isAllowedURL(url) {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Referer", "https://quote.eastmoney.com/")

	resp, err := c.resilient.Do(ctx, SourceEastmoney, req)
	if err != nil {
		c.logger.Warn("eastmoney US index kline request failed", "code", code, "error", err)
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
		Data struct {
			Klines []string `json:"klines"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		c.logger.Warn("eastmoney US index kline parse failed", "code", code, "error", err)
		return nil
	}
	if len(raw.Data.Klines) == 0 {
		return nil
	}

	points := make([]marketdomain.IndexKlinePoint, 0, len(raw.Data.Klines))
	for _, line := range raw.Data.Klines {
		parts := strings.Split(line, ",")
		if len(parts) < 7 {
			continue
		}
		points = append(points, marketdomain.IndexKlinePoint{
			Date:   parts[0],
			Open:   httpclient.ParseQuoteFloat(parts[1]),
			Close:  httpclient.ParseQuoteFloat(parts[2]),
			High:   httpclient.ParseQuoteFloat(parts[3]),
			Low:    httpclient.ParseQuoteFloat(parts[4]),
			Volume: int64(httpclient.ParseQuoteFloat(parts[5])),
			Amount: 0,
		})
	}
	return normalizeIndexKlinePoints(points)
}

// fetchHKIndexMinuteTencent 通过腾讯分时接口获取港股指数的分钟数据。
func (c *IndexQuoteClient) fetchHKIndexMinuteTencent(ctx context.Context, code string) []marketdomain.IndexMinutePoint {
	tencentCode, ok := hkusIndexTencentKlineCode[code]
	if !ok {
		return nil
	}
	return c.fetchTencentMinuteData(ctx, tencentCode, MarketHK)
}
