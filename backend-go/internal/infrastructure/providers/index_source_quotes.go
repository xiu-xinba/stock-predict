package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	marketdomain "stock-predict-go/internal/domain/market"
	httpclient "stock-predict-go/internal/platform/httpclient"

	"gitee.com/quant1x/gotdx"
	"gitee.com/quant1x/gotdx/quotes"
)

// fetchCNIndexQuotesTDX 通过通达信接口获取 A 股指数实时行情快照。
// 内部使用 recover 捕获 gotdx 可能产生的 panic，确保调用安全。
func (c *IndexQuoteClient) fetchCNIndexQuotesTDX(_ context.Context) []marketdomain.MarketIndex {
	var result []marketdomain.MarketIndex
	func() {
		defer func() {
			if r := recover(); r != nil {
				c.logger.Warn("gotdx panicked", "method", "fetchCNIndexQuotesTDX", "recover", r)
			}
		}()
		api := gotdx.GetTdxApi()
		if api == nil {
			return
		}
		for _, code := range cnIndexCodes {
			symbol := tdxIndexSymbol(code)
			snapshots, err := api.GetSnapshot([]string{symbol})
			if err != nil {
				c.logger.Warn("gotdx GetSnapshot failed", "code", code, "error", err)
				continue
			}
			if len(snapshots) == 0 {
				continue
			}
			s := snapshots[0]
			prevClose := s.LastClose
			change := s.Price - prevClose
			var changePct float64
			if prevClose > 0 {
				changePct = change / prevClose * 100
			}
			isClosed := s.ExchangeState != quotes.EXCHANGE_STATE_NORMAL
			result = append(result, marketdomain.MarketIndex{
				Code:       code,
				Name:       cnIndexNames[code],
				Market:     cnIndexMarkets[code],
				Value:      s.Price,
				Change:     change,
				ChangePct:  changePct,
				High:       s.High,
				Low:        s.Low,
				PrevClose:  prevClose,
				Volume:     float64(s.Vol),
				UpdateTime: time.Now().Format("2006-01-02 15:04:05"),
				DataSource: "tdx",
				IsClosed:   isClosed,
			})
		}
	}()
	return normalizeMarketIndices(result)
}

// fetchCNIndexQuotesTencent 通过腾讯批量行情接口获取 A 股指数实时行情。
func (c *IndexQuoteClient) fetchCNIndexQuotesTencent(ctx context.Context) []marketdomain.MarketIndex {
	symbols := make([]string, 0, len(cnIndexCodes))
	for _, code := range cnIndexCodes {
		symbols = append(symbols, tencentIndexSymbols[code])
	}
	raw := c.fetchTencentIndexRaw(ctx, symbols)
	if len(raw) == 0 {
		return nil
	}

	result := make([]marketdomain.MarketIndex, 0, len(cnIndexCodes))
	for _, code := range cnIndexCodes {
		sym := tencentIndexSymbols[code]
		fields, ok := raw[sym]
		if !ok {
			continue
		}
		idx := tencentIndexToMarketIndex(code, fields)
		idx.DataSource = "tencent"
		result = append(result, idx)
	}
	return normalizeMarketIndices(result)
}

// fetchCNIndexQuotesEastmoney 通过东方财富日K线接口获取 A 股指数实时行情。
// 利用最近两日K线数据推算当前价格、涨跌幅等信息。
func (c *IndexQuoteClient) fetchCNIndexQuotesEastmoney(ctx context.Context) []marketdomain.MarketIndex {
	var result []marketdomain.MarketIndex
	for _, code := range cnIndexCodes {
		var secid string
		switch code {
		case "000001":
			secid = "1.000001"
		case "399001":
			secid = "0.399001"
		case "399006":
			secid = "0.399006"
		default:
			continue
		}
		url := fmt.Sprintf("https://push2his.eastmoney.com/api/qt/stock/kline/get?secid=%s&fields1=f1,f2,f3,f4,f5,f6&fields2=f51,f52,f53,f54,f55,f56,f57,f58,f59,f60,f61&klt=101&fqt=1&end=20500101&lmt=2", secid)
		if !isAllowedURL(url) {
			continue
		}

		body, err := c.fetchEastmoneyURL(ctx, url)
		if err != nil {
			c.logger.Warn("eastmoney index quote request failed", "code", code, "error", err)
			continue
		}

		var raw struct {
			Data struct {
				Klines []string `json:"klines"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &raw); err != nil {
			c.logger.Warn("eastmoney index quote parse failed", "code", code, "error", err)
			continue
		}
		if len(raw.Data.Klines) == 0 {
			c.logger.Warn("eastmoney index quote empty klines", "code", code)
			continue
		}

		// K-line data format: "date,open,close,high,low,volume,amount,amplitude,changePct,changeAmount,turnoverRate"
		// Use the last point for current data, second-to-last for prevClose
		lastLine := raw.Data.Klines[len(raw.Data.Klines)-1]
		parts := strings.Split(lastLine, ",")
		if len(parts) < 11 {
			c.logger.Warn("eastmoney index quote invalid kline format", "code", code, "parts", len(parts))
			continue
		}

		value := httpclient.ParseQuoteFloat(parts[2])        // close
		high := httpclient.ParseQuoteFloat(parts[3])         // high
		low := httpclient.ParseQuoteFloat(parts[4])          // low
		volume := httpclient.ParseQuoteFloat(parts[5])       // volume
		changePct := httpclient.ParseQuoteFloat(parts[8])    // changePct (already in percentage)
		changeAmount := httpclient.ParseQuoteFloat(parts[9]) // changeAmount (absolute value)

		var prevClose float64
		if len(raw.Data.Klines) >= 2 {
			prevLine := raw.Data.Klines[len(raw.Data.Klines)-2]
			prevParts := strings.Split(prevLine, ",")
			if len(prevParts) >= 3 {
				prevClose = httpclient.ParseQuoteFloat(prevParts[2])
			}
		}
		if prevClose == 0 {
			prevClose = value - changeAmount
		}

		change := changeAmount
		if prevClose > 0 && change == 0 && changePct == 0 {
			change = value - prevClose
			changePct = change / prevClose * 100
		}

		result = append(result, marketdomain.MarketIndex{
			Code:       code,
			Name:       cnIndexNames[code],
			Market:     cnIndexMarkets[code],
			Value:      value,
			Change:     change,
			ChangePct:  changePct,
			High:       high,
			Low:        low,
			PrevClose:  prevClose,
			Volume:     volume,
			UpdateTime: time.Now().Format("2006-01-02 15:04:05"),
			DataSource: "eastmoney",
		})
	}
	return normalizeMarketIndices(result)
}

// fetchEastmoneyURL 通过共享的弹性 HTTP 客户端请求东方财富 URL。
// 使用标准证书验证，超时时间为 10 秒。
func (c *IndexQuoteClient) fetchEastmoneyURL(ctx context.Context, url string) ([]byte, error) {
	fetchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return c.fetchViaGoHTTP(fetchCtx, url)
}

// fetchHKIndexQuotesTencent 通过腾讯批量行情接口获取港股指数实时行情。
func (c *IndexQuoteClient) fetchHKIndexQuotesTencent(ctx context.Context) []marketdomain.MarketIndex {
	symbols := make([]string, 0, len(hkIndexMeta))
	for code := range hkIndexMeta {
		symbols = append(symbols, tencentIndexSymbols[code])
	}
	raw := c.fetchTencentIndexRaw(ctx, symbols)
	if len(raw) == 0 {
		return nil
	}

	result := make([]marketdomain.MarketIndex, 0, len(hkIndexMeta))
	for code, meta := range hkIndexMeta {
		sym := tencentIndexSymbols[code]
		fields, ok := raw[sym]
		if !ok {
			continue
		}
		idx := tencentIndexToMarketIndex(code, fields)
		idx.Name = meta.name
		idx.Market = meta.market
		idx.DataSource = "tencent"
		result = append(result, idx)
	}
	return normalizeMarketIndices(result)
}

// fetchUSIndexQuotesTencent 通过腾讯批量行情接口获取美股指数实时行情。
func (c *IndexQuoteClient) fetchUSIndexQuotesTencent(ctx context.Context) []marketdomain.MarketIndex {
	symbols := make([]string, 0, len(usIndexMeta))
	for code := range usIndexMeta {
		symbols = append(symbols, tencentIndexSymbols[code])
	}
	raw := c.fetchTencentIndexRaw(ctx, symbols)
	if len(raw) == 0 {
		return nil
	}

	result := make([]marketdomain.MarketIndex, 0, len(usIndexMeta))
	for code, meta := range usIndexMeta {
		sym := tencentIndexSymbols[code]
		fields, ok := raw[sym]
		if !ok {
			continue
		}
		idx := tencentIndexToMarketIndex(code, fields)
		idx.Name = meta.name
		idx.Market = meta.market
		idx.DataSource = "tencent"
		result = append(result, idx)
	}
	return normalizeMarketIndices(result)
}
