package providers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	marketdomain "stock-predict-go/internal/domain/market"
	httpclient "stock-predict-go/internal/platform/httpclient"
)

// fetchUSIndexMinuteSina 通过新浪分时接口获取美股指数的分钟数据。
// 新浪接口返回 JSONP 格式，内容为分号分隔的字符串：
//
//	var minute=("09:30:00,0,0,51826.96;09:31:00,3649532,0,51808.29;...")
//
// 每段格式：HH:MM:SS,成交量,0,价格
func (c *IndexQuoteClient) fetchUSIndexMinuteSina(ctx context.Context, code string) []marketdomain.IndexMinutePoint {
	symbol, ok := usIndexSinaSymbol[code]
	if !ok {
		return nil
	}
	url := fmt.Sprintf("https://stock.finance.sina.com.cn/usstock/api/jsonp_v2.php/var%%20minute=/US_MinlineNService.getMinline?symbol=%s&day=1", symbol)
	if !isAllowedURL(url) {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Referer", "https://finance.sina.com.cn/")

	resp, err := c.resilient.Do(ctx, SourceSina, req)
	if err != nil {
		c.logger.Warn("sina US index minute HTTP request failed", "code", code, "error", err)
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.logger.Warn("sina US index minute HTTP status error", "code", code, "status", resp.StatusCode)
		return nil
	}
	payload, err := io.ReadAll(io.LimitReader(resp.Body, int64(MaxHTTPPayloadBytes)))
	if err != nil {
		return nil
	}

	// 提取括号内的数据部分：var minute=("09:30:00,...;16:00:00,...")
	text := string(payload)
	// 查找数据起始标记 =("
	startMarker := "=(\""
	markerIdx := strings.Index(text, startMarker)
	if markerIdx < 0 {
		// 兼容无等号格式: ("...")
		markerIdx = strings.Index(text, "(\"")
		if markerIdx < 0 {
			c.logger.Warn("sina US index minute: no data start marker found", "code", code)
			return nil
		}
		markerIdx++ // 跳过 '('
	} else {
		markerIdx += len(startMarker) // 跳过 '=("'
	}
	// 查找数据结束标记 ")
	endMarker := "\")"
	endIdx := strings.Index(text[markerIdx:], endMarker)
	if endIdx < 0 {
		// 兼容无引号格式: (...)
		endIdx = strings.Index(text[markerIdx:], ")")
	}
	if endIdx <= 0 {
		c.logger.Warn("sina US index minute: no data end marker found", "code", code)
		return nil
	}
	dataStr := text[markerIdx : markerIdx+endIdx]

	// 按分号分割各分钟数据点
	segments := strings.Split(dataStr, ";")
	points := make([]marketdomain.IndexMinutePoint, 0, len(segments))
	var prevCumVol int64
	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		fields := strings.Split(seg, ",")
		if len(fields) < 4 {
			continue
		}
		// 时间格式：HH:MM:SS → HH:MM
		timeStr := fields[0]
		if len(timeStr) > 5 {
			timeStr = timeStr[:5]
		}
		cumVol := int64(httpclient.ParseQuoteFloat(fields[1]))
		price := httpclient.ParseQuoteFloat(fields[3])
		if price <= 0 {
			continue
		}
		var vol int64
		if len(points) > 0 {
			vol = cumVol - prevCumVol
			if vol < 0 {
				vol = 0
			}
		} else {
			vol = cumVol
		}
		prevCumVol = cumVol

		var avgPrice float64
		if len(points) > 0 {
			avgPrice = (points[len(points)-1].AvgPrice*float64(len(points)) + price) / float64(len(points)+1)
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
	return normalizeIndexMinutePoints(points, MarketUS)
}

// fetchCNIndexQuotesSina 通过新浪 hq 接口获取 A 股指数实时行情。
// 返回格式示例：var hq_str_s_sh000001="上证指数,3368.07,28.24,0.85,4216728,46032700";
func (c *IndexQuoteClient) fetchCNIndexQuotesSina(ctx context.Context) []marketdomain.MarketIndex {
	url := "https://hq.sinajs.cn/list=s_sh000001,s_sz399001,s_sz399006"
	if !isAllowedURL(url) {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Referer", "https://finance.sina.com.cn/")

	resp, err := c.resilient.Do(ctx, SourceSina, req)
	if err != nil {
		c.logger.Warn("sina CN index quote request failed", "error", err)
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

	var indices []marketdomain.MarketIndex
	for _, line := range strings.Split(string(payload), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "var hq_str_") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) < 2 {
			continue
		}
		codePart := strings.TrimPrefix(parts[0], "var hq_str_")
		valPart := strings.Trim(strings.Trim(parts[1], "\";"), "\"")
		fields := strings.Split(valPart, ",")
		if len(fields) < 6 {
			continue
		}

		name := fields[0]
		price := httpclient.ParseQuoteFloat(fields[1])
		change := httpclient.ParseQuoteFloat(fields[2])
		changePct := httpclient.ParseQuoteFloat(fields[3])

		indices = append(indices, marketdomain.MarketIndex{
			Code:      codePart,
			Name:      name,
			Value:     price,
			Change:    change,
			ChangePct: changePct,
		})
	}
	return indices
}

// hkusIndexTencentKlineCode 港股/美股指数代码到腾讯K线接口参数的映射表。
var hkusIndexTencentKlineCode = map[string]string{
	"hsi":    "hkHSI",
	"hstech": "hkHSTECH",
	"dji":    "usDJI",
	"ixic":   "usIXIC",
	"spx":    "usINX",
}

// usIndexEastmoneySecid 美股指数代码到东方财富K线接口 secid 的映射表。
var usIndexEastmoneySecid = map[string]string{
	"dji":  "100.DJIA",
	"ixic": "100.NDX",
	"spx":  "100.SPX",
}
