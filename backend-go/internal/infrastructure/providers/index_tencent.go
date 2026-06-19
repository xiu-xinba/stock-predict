package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	marketdomain "stock-predict-go/internal/domain/market"
	httpclient "stock-predict-go/internal/platform/httpclient"
)

// fetchCNIndexKlineTencent 通过腾讯日K线接口获取 A 股指数的K线数据。
func (c *IndexQuoteClient) fetchCNIndexKlineTencent(ctx context.Context, code string, count int) []marketdomain.IndexKlinePoint {
	market, ok := cnIndexMarkets[code]
	if !ok {
		return nil
	}
	url := fmt.Sprintf("https://web.ifzq.gtimg.cn/appstock/app/fqkline/get?param=%s%s,day,,,%d,qfq", market, code, count)
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
		c.logger.Warn("tencent kline request failed", "code", code, "error", err)
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
		c.logger.Warn("tencent kline parse failed", "code", code, "error", err)
		return nil
	}
	if raw.Code != 0 {
		return nil
	}

	key := market + code
	entry, ok := raw.Data[key]
	if !ok || len(entry.Day) == 0 {
		return nil
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

// fetchCNIndexMinuteTencent 通过腾讯分时接口获取 A 股指数的分钟数据。
func (c *IndexQuoteClient) fetchCNIndexMinuteTencent(ctx context.Context, code string) []marketdomain.IndexMinutePoint {
	market, ok := cnIndexMarkets[code]
	if !ok {
		return nil
	}
	return c.fetchTencentMinuteData(ctx, market+code, MarketCN)
}

// fetchTencentIndexRaw 通过腾讯批量行情接口获取原始指数行情数据。
// 传入腾讯格式的 symbol 列表，返回以 symbol 为键、字段数组为值的映射。
func (c *IndexQuoteClient) fetchTencentIndexRaw(ctx context.Context, symbols []string) map[string][]string {
	url := fmt.Sprintf("https://qt.gtimg.cn/q=%s", strings.Join(symbols, ","))
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
	return parseTencentIndexPayload(payload)
}

// parseTencentIndexPayload 解析腾讯行情接口返回的原始文本，提取各 symbol 的字段数组。
func parseTencentIndexPayload(payload []byte) map[string][]string {
	text := strings.TrimSpace(strings.TrimPrefix(string(payload), "\ufeff"))
	result := map[string][]string{}
	for statement := range strings.SplitSeq(text, ";") {
		start := strings.Index(statement, "\"")
		end := strings.LastIndex(statement, "\"")
		if start < 0 || end <= start {
			continue
		}
		content := statement[start+1 : end]
		fields := strings.Split(content, "~")
		if len(fields) < 35 {
			continue
		}
		symbolKey, _, _ := strings.Cut(statement, "=")
		symbolKey = strings.TrimSpace(symbolKey)
		symbolKey = strings.TrimPrefix(symbolKey, "v_")
		result[symbolKey] = fields
	}
	return result
}

// tencentIndexToMarketIndex 将腾讯行情字段数组转换为 MarketIndex 领域对象。
func tencentIndexToMarketIndex(code string, fields []string) marketdomain.MarketIndex {
	price := httpclient.ParseQuoteFloat(fields[3])
	prevClose := httpclient.ParseQuoteFloat(fields[4])
	open := httpclient.ParseQuoteFloat(fields[5])
	high := httpclient.ParseQuoteFloat(fields[33])
	low := httpclient.ParseQuoteFloat(fields[34])
	changePct := httpclient.ParseQuoteFloat(fields[32])
	change := httpclient.ParseQuoteFloat(fields[31])
	volume := httpclient.ParseQuoteFloat(fields[6])
	updateTime := strings.TrimSpace(fields[30])
	if updateTime == "" {
		updateTime = time.Now().Format("2006-01-02 15:04:05")
	}
	name := cnIndexNames[code]
	market := cnIndexMarkets[code]
	if name == "" && len(fields) > 1 {
		name = strings.TrimSpace(fields[1])
	}
	isClosed := false
	if len(fields) > 40 {
		status := strings.TrimSpace(fields[40])
		isClosed = status != "" && status != "0"
	}
	return marketdomain.MarketIndex{
		Code:       code,
		Name:       name,
		Market:     market,
		Value:      price,
		Change:     change,
		ChangePct:  changePct,
		High:       high,
		Low:        low,
		PrevClose:  prevClose,
		Open:       open,
		Volume:     volume,
		UpdateTime: updateTime,
		IsClosed:   isClosed,
	}
}

// tdxIndexSymbol 将 A 股指数代码转换为通达信格式的 symbol（如 "sh000001"）。
func tdxIndexSymbol(code string) string {
	prefix, ok := cnIndexMarkets[code]
	if !ok {
		return ""
	}
	return prefix + code
}

// isCNIndex 判断给定指数代码是否为 A 股指数。
func isCNIndex(code string) bool {
	return slices.Contains(cnIndexCodes, code)
}

// detectIndexMarket 根据指数代码判断所属市场（A 股、港股、美股）。
func detectIndexMarket(code string) Market {
	if isCNIndex(code) {
		return MarketCN
	}
	if _, ok := hkIndexMeta[code]; ok {
		return MarketHK
	}
	if _, ok := usIndexMeta[code]; ok {
		return MarketUS
	}
	// Fallback: use code prefix heuristics
	if len(code) > 0 && code[0] == '.' {
		return MarketUS
	}
	if len(code) == 6 {
		return MarketCN
	}
	return MarketCN
}

// minuteToTime 将总分钟数转换为 "HH:MM" 格式的时间字符串。
func minuteToTime(totalMinutes int, morning bool) string {
	h := totalMinutes / 60
	m := totalMinutes % 60
	_ = morning
	return fmt.Sprintf("%02d:%02d", h, m)
}

// usIndexSinaSymbol 美股指数代码到新浪 API symbol 格式的映射表。
var usIndexSinaSymbol = map[string]string{
	"dji":  ".dji",
	"ixic": ".ixic",
	"spx":  ".inx",
}

// fetchUSIndexMinuteSina fetches US index minute data via Sina API.
