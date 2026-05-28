package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"stock-predict-go/internal/dto"
)

const (
	tencentQuoteURL      = "https://qt.gtimg.cn/q=%s"
	eastmoneyFundGZURL   = "https://fundgz.1234567.com.cn/js/%s.js?rt=%d"
	maxFundGZConcurrency = 8
)

type fundQuoteProvider interface {
	RefreshQuotes(context.Context, []dto.FundItem) map[string]dto.FundItem
}

type FundQuoteClient struct {
	client *http.Client
	now    func() time.Time
}

func NewFundQuoteClient(timeout time.Duration) *FundQuoteClient {
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	return &FundQuoteClient{
		client: &http.Client{Timeout: timeout},
		now:    time.Now,
	}
}

func (c *FundQuoteClient) RefreshQuotes(ctx context.Context, funds []dto.FundItem) map[string]dto.FundItem {
	quotes := make(map[string]dto.FundItem, len(funds))
	listedSymbols := make([]string, 0, len(funds))
	for _, fund := range funds {
		if symbol, ok := listedFundSymbol(fund.FundCode); ok {
			listedSymbols = append(listedSymbols, symbol)
		}
	}
	if len(listedSymbols) > 0 {
		for code, quote := range c.fetchTencentQuotes(ctx, listedSymbols) {
			quotes[code] = quote
		}
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxFundGZConcurrency)
	for _, fund := range funds {
		fund := fund
		if _, ok := quotes[fund.FundCode]; ok {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}
			if quote, ok := c.fetchEastmoneyFundGZQuote(ctx, fund.FundCode); ok {
				mu.Lock()
				quotes[fund.FundCode] = quote
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return quotes
}

func (c *FundQuoteClient) fetchTencentQuotes(ctx context.Context, symbols []string) map[string]dto.FundItem {
	url := fmt.Sprintf(tencentQuoteURL, strings.Join(symbols, ","))
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
	return parseTencentFundQuotes(payload)
}

func (c *FundQuoteClient) fetchEastmoneyFundGZQuote(ctx context.Context, code string) (dto.FundItem, bool) {
	url := fmt.Sprintf(eastmoneyFundGZURL, code, c.now().UnixMilli())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return dto.FundItem{}, false
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://fund.eastmoney.com/"+code+".html")
	resp, err := c.client.Do(req)
	if err != nil {
		return dto.FundItem{}, false
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return dto.FundItem{}, false
	}
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return dto.FundItem{}, false
	}
	return parseEastmoneyFundGZQuote(payload)
}

func parseEastmoneyFundGZQuote(payload []byte) (dto.FundItem, bool) {
	text := strings.TrimSpace(strings.TrimPrefix(string(payload), "\ufeff"))
	start := strings.Index(text, "(")
	end := strings.LastIndex(text, ")")
	if start < 0 || end <= start {
		return dto.FundItem{}, false
	}
	var raw struct {
		FundCode string `json:"fundcode"`
		Name     string `json:"name"`
		NAVDate  string `json:"jzrq"`
		UnitNAV  string `json:"dwjz"`
		Estimate string `json:"gsz"`
		Change   string `json:"gszzl"`
		Time     string `json:"gztime"`
	}
	if err := json.Unmarshal([]byte(text[start+1:end]), &raw); err != nil {
		return dto.FundItem{}, false
	}
	code := strings.TrimSpace(raw.FundCode)
	if len(code) != 6 || !allDigits(code) {
		return dto.FundItem{}, false
	}
	latestNAV := parseQuoteFloat(raw.UnitNAV)
	estimatedNAV := parseQuoteFloat(raw.Estimate)
	if estimatedNAV == 0 {
		estimatedNAV = latestNAV
	}
	if estimatedNAV == 0 {
		return dto.FundItem{}, false
	}
	quoteTime := strings.TrimSpace(raw.Time)
	if quoteTime == "" {
		quoteTime = strings.TrimSpace(raw.NAVDate)
	}
	return dto.FundItem{
		FundCode:     code,
		FundName:     strings.TrimSpace(raw.Name),
		LatestNAV:    latestNAV,
		EstimatedNAV: estimatedNAV,
		ChangePct:    parseQuoteFloat(raw.Change),
		QuoteDate:    quoteTime,
		QuoteSource:  "eastmoney_fundgz",
	}, true
}

func parseTencentFundQuotes(payload []byte) map[string]dto.FundItem {
	text := strings.TrimSpace(strings.TrimPrefix(string(payload), "\ufeff"))
	quotes := map[string]dto.FundItem{}
	for _, statement := range strings.Split(text, ";") {
		start := strings.Index(statement, "\"")
		end := strings.LastIndex(statement, "\"")
		if start < 0 || end <= start {
			continue
		}
		fields := strings.Split(statement[start+1:end], "~")
		if len(fields) <= 32 {
			continue
		}
		code := strings.TrimSpace(fields[2])
		if len(code) != 6 || !allDigits(code) {
			continue
		}
		price := parseQuoteFloat(fields[3])
		if price == 0 {
			continue
		}
		quotes[code] = dto.FundItem{
			FundCode:     code,
			FundName:     strings.TrimSpace(fields[1]),
			LatestNAV:    price,
			EstimatedNAV: price,
			ChangePct:    parseQuoteFloat(fields[32]),
			QuoteDate:    strings.TrimSpace(fields[30]),
			QuoteSource:  "tencent_quote",
		}
	}
	return quotes
}

func listedFundSymbol(code string) (string, bool) {
	switch {
	case len(code) == 6 && allDigits(code) && strings.HasPrefix(code, "5"):
		return "sh" + code, true
	case len(code) == 6 && allDigits(code) && strings.HasPrefix(code, "1"):
		return "sz" + code, true
	default:
		return "", false
	}
}

func parseQuoteFloat(raw string) float64 {
	raw = strings.TrimSpace(strings.TrimSuffix(strings.ReplaceAll(raw, ",", ""), "%"))
	if raw == "" || raw == "--" || raw == "---" {
		return 0
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}
	return value
}
