package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"stock-predict-go/internal/dto"
	"stock-predict-go/internal/util"
)

type StockQuoteClient struct {
	client *http.Client
}

func NewStockQuoteClient(timeout time.Duration) *StockQuoteClient {
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	return &StockQuoteClient{
		client: &http.Client{Timeout: timeout},
	}
}

func (c *StockQuoteClient) FetchQuotes(ctx context.Context, codes []string) map[string]dto.StockQuote {
	quotes := make(map[string]dto.StockQuote, len(codes))
	if len(codes) == 0 {
		return quotes
	}

	symbols := make([]string, 0, len(codes))
	for _, code := range codes {
		market := stockMarketPrefix(code)
		if market != "" {
			symbols = append(symbols, market+code)
		}
	}
	if len(symbols) == 0 {
		return quotes
	}

	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i := 0; i < len(symbols); i += 30 {
		end := i + 30
		if end > len(symbols) {
			end = len(symbols)
		}
		batch := symbols[i:end]
		wg.Add(1)
		go func(batch []string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			result := c.fetchTencentStockQuotes(ctx, batch)
			if result != nil {
				mu.Lock()
				for k, v := range result {
					quotes[k] = v
				}
				mu.Unlock()
			}
		}(batch)
	}
	wg.Wait()
	return quotes
}

func (c *StockQuoteClient) fetchTencentStockQuotes(ctx context.Context, symbols []string) map[string]dto.StockQuote {
	url := fmt.Sprintf("https://qt.gtimg.cn/q=%s", strings.Join(symbols, ","))

	if !isAllowedURL(url) {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://gu.qq.com/")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil
	}
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil
	}
	return parseTencentStockQuotes(payload)
}

func parseTencentStockQuotes(payload []byte) map[string]dto.StockQuote {
	text := strings.TrimSpace(strings.TrimPrefix(string(payload), "\ufeff"))
	quotes := map[string]dto.StockQuote{}
	const minFields = 39
	for _, statement := range strings.Split(text, ";") {
		start := strings.Index(statement, "\"")
		end := strings.LastIndex(statement, "\"")
		if start < 0 || end <= start {
			continue
		}
		fields := strings.Split(statement[start+1:end], "~")
		if len(fields) < minFields {
			continue
		}
		code := strings.TrimSpace(fields[2])
		if len(code) != 6 || !util.IsAllDigits(code) {
			continue
		}
		name := strings.TrimSpace(fields[1])
		if name == "" {
			continue
		}
		priceStr := strings.TrimSpace(fields[3])
		if priceStr == "" {
			continue
		}
		price := parseQuoteFloat(priceStr)
		if price == 0 {
			continue
		}
		changePctStr := strings.TrimSpace(fields[32])
		if changePctStr == "" {
			continue
		}
		quotes[code] = dto.StockQuote{
			Price:        price,
			Open:         parseQuoteFloat(fields[5]),
			High:         parseQuoteFloat(fields[33]),
			Low:          parseQuoteFloat(fields[34]),
			PrevClose:    parseQuoteFloat(fields[4]),
			Volume:       parseQuoteFloat(fields[6]),
			Amount:       parseQuoteFloat(fields[37]),
			TurnoverRate: parseQuoteFloat(fields[38]),
			ChangePct:    parseQuoteFloat(changePctStr),
			ChangeAmt:    parseQuoteFloat(fields[31]),
			BidPrice:     parseQuoteFloat(fields[9]),
			AskPrice:     parseQuoteFloat(fields[18]),
			QuoteTime:    strings.TrimSpace(fields[30]),
		}
	}
	return quotes
}

func stockMarketPrefix(code string) string {
	if len(code) != 6 || !util.IsAllDigits(code) {
		return ""
	}
	switch {
	case strings.HasPrefix(code, "6"), strings.HasPrefix(code, "9"):
		return "sh"
	case strings.HasPrefix(code, "0"), strings.HasPrefix(code, "3"):
		return "sz"
	case strings.HasPrefix(code, "8"), strings.HasPrefix(code, "4"):
		return "bj"
	default:
		return ""
	}
}
