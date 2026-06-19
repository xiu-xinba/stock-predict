package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	stockdomain "stock-predict-go/internal/domain/stock"
	httpclient "stock-predict-go/internal/platform/httpclient"
)

const (
	eastmoneyStockSearchURL = "https://searchapi.eastmoney.com/api/suggest/get?input=%s&type=14" // 东方财富股票搜索 API
	eastmoneyStockListURL   = "https://push2his.eastmoney.com/api/qt/clist/get"                  // 东方财富股票列表 API
	eastmoneyDataCenterURL  = "https://datacenter-web.eastmoney.com/api/data/v1/get"              // 东方财富数据中心 API
)

// fetchAllStocksFromAPI 从东方财富 clist API 分页获取全部股票列表。
func (s *StockService) fetchAllStocksFromAPI(ctx context.Context) []stockdomain.StockItem {
	fsValues := []string{
		"m:0+t:6,m:0+t:80",
		"m:1+t:2,m:1+t:23",
		"m:0+t:81",
	}

	fields := "f2,f3,f5,f6,f8,f9,f12,f14,f20,f23,f100,f116,f117"

	var allItems []stockdomain.StockItem

	for _, fs := range fsValues {
		page := 1
		pageSize := 5000
		var fetchedTotal int
		for {
			url := fmt.Sprintf("%s?pn=%d&pz=%d&po=1&np=1&fltt=2&invt=2&fid=f12&fs=%s&fields=%s",
				eastmoneyStockListURL, page, pageSize, fs, fields)

			payload, err := s.eastmoney.GetWithLimit(ctx, url, "https://quote.eastmoney.com/", MaxSyncPayloadBytes)
			if err != nil {
				s.logger.Warn("stock list API failed", "fs", fs, "page", page, "error", err)
				break
			}

			var result struct {
				Data struct {
					Total int `json:"total"`
					Diff  []struct {
						F2   any    `json:"f2"`
						F3   any    `json:"f3"`
						F5   any    `json:"f5"`
						F6   any    `json:"f6"`
						F8   any    `json:"f8"`
						F9   any    `json:"f9"`
						F12  string `json:"f12"`
						F14  string `json:"f14"`
						F20  any    `json:"f20"`
						F23  any    `json:"f23"`
						F100 string `json:"f100"`
						F116 any    `json:"f116"`
						F117 any    `json:"f117"`
					} `json:"diff"`
				} `json:"data"`
			}

			if err := json.Unmarshal(payload, &result); err != nil {
				s.logger.Warn("stock list parse failed", "error", err)
				break
			}

			if len(result.Data.Diff) == 0 {
				break
			}

			for _, d := range result.Data.Diff {
				code := strings.TrimSpace(d.F12)
				if len(code) != 6 || !httpclient.IsAllDigits(code) {
					continue
				}
				market := stockMarketPrefix(code)
				if market == "" {
					continue
				}
				name := strings.TrimSpace(d.F14)
				if name == "" {
					continue
				}
				abbr := pinyinAbbr(name)
				abbrAll := pinyinAbbrAll(name)
				var alt string
				for _, a := range abbrAll {
					if a != abbr {
						alt = a
						break
					}
				}
				allItems = append(allItems, stockdomain.StockItem{
					StockCode:    code,
					StockName:    name,
					Market:       market,
					Industry:     strings.TrimSpace(d.F100),
					CurrentPrice: toNum(d.F2),
					ChangePct:    toNum(d.F3),
					Volume:       toNum(d.F5),
					Amount:       toNum(d.F6),
					TurnoverRate: toNum(d.F8),
					PERatio:      toNum(d.F9),
					PBRatio:      toNum(d.F23),
					TotalMV:      toNum(d.F20),
					TotalShares:  toNum(d.F116),
					FloatShares:  toNum(d.F117),
					Pinyin:       abbr,
					PinyinAlt:    alt,
				})
			}

			// Use total from API response to determine if more pages exist.
			// The API may cap the actual returned items per page (e.g. 100)
			// regardless of the requested pz value, so comparing len(diff) < pz
			// is unreliable. Instead, track cumulative fetched count against total.
			fetchedTotal += len(result.Data.Diff)
			if fetchedTotal >= result.Data.Total || len(result.Data.Diff) == 0 {
				break
			}
			page++

			time.Sleep(time.Duration(StockSyncPageDelay) * time.Millisecond)
		}
	}

	s.logger.Info("fetched stocks from API", "total", len(allItems))
	return allItems
}

// fetchStocksFromDataCenter 从东方财富数据中心 API 获取股票列表。
func (s *StockService) fetchStocksFromDataCenter(ctx context.Context) []stockdomain.StockItem {
	var allItems []stockdomain.StockItem

	for page := 1; page <= 100; page++ {
		url := fmt.Sprintf("%s?sortColumns=SECURITY_CODE&sortTypes=1&pageSize=500&pageNumber=%d&reportName=RPT_LICO_FN_CPD&columns=SECURITY_CODE,SECURITY_NAME_ABBR,SECUCODE,BOARD_NAME&source=WEB&client=WEB&filter=(ISNEW%%3D%%221%%22)",
			eastmoneyDataCenterURL, page)

		payload, err := s.eastmoney.GetWithLimit(ctx, url, "https://data.eastmoney.com/", MaxSyncPayloadBytes)
		if err != nil {
			s.logger.Warn("data center API failed", "page", page, "error", err)
			break
		}

		var result struct {
			Result struct {
				Pages int `json:"pages"`
				Data  []struct {
					SecurityCode string `json:"SECURITY_CODE"`
					SecurityName string `json:"SECURITY_NAME_ABBR"`
					Secucode     string `json:"SECUCODE"`
					BoardName    string `json:"BOARD_NAME"`
				} `json:"data"`
			} `json:"result"`
			Success bool `json:"success"`
		}

		if err := json.Unmarshal(payload, &result); err != nil {
			s.logger.Warn("data center parse failed", "error", err)
			break
		}

		if !result.Success || len(result.Result.Data) == 0 {
			break
		}

		for _, d := range result.Result.Data {
			code := strings.TrimSpace(d.SecurityCode)
			if len(code) != 6 || !httpclient.IsAllDigits(code) {
				continue
			}

			market := ""
			if strings.HasSuffix(d.Secucode, ".SZ") {
				market = "sz"
			} else if strings.HasSuffix(d.Secucode, ".SH") {
				market = "sh"
			} else if strings.HasSuffix(d.Secucode, ".BJ") {
				market = "bj"
			} else {
				market = stockMarketPrefix(code)
			}
			if market == "" {
				continue
			}

			name := strings.TrimSpace(d.SecurityName)
			if name == "" {
				continue
			}

			abbr := pinyinAbbr(name)
			abbrAll := pinyinAbbrAll(name)
			var alt string
			for _, a := range abbrAll {
				if a != abbr {
					alt = a
					break
				}
			}

			allItems = append(allItems, stockdomain.StockItem{
				StockCode: code,
				StockName: name,
				Market:    market,
				Industry:  strings.TrimSpace(d.BoardName),
				Pinyin:    abbr,
				PinyinAlt: alt,
			})
		}

		s.logger.Info("data center page fetched", "page", page, "total_pages", result.Result.Pages, "total_stocks", len(allItems))

		if page >= result.Result.Pages {
			break
		}

		select {
		case <-ctx.Done():
			s.logger.Warn("data center fetch interrupted", "error", ctx.Err())
			return allItems
		default:
		}
		time.Sleep(time.Duration(DataCenterPageDelay) * time.Millisecond)
	}

	s.logger.Info("fetched stocks from data center", "total", len(allItems))
	return allItems
}

// searchFromAPI 从东方财富搜索 API 获取股票搜索结果。
func (s *StockService) searchFromAPI(ctx context.Context, keyword string) []stockdomain.StockItem {
	url := fmt.Sprintf(eastmoneyStockSearchURL, url.QueryEscape(keyword))
	payload, err := s.eastmoney.GetWithLimit(ctx, url, "https://so.eastmoney.com/", 1<<20)
	if err != nil {
		s.logger.Warn("stock search API failed", "error", err)
		return nil
	}

	var result struct {
		QuotationCodeTable struct {
			Data []struct {
				Code         string `json:"Code"`
				Name         string `json:"Name"`
				Pinyin       string `json:"PingYin"`
				MarketNumber string `json:"MktNum"`
				Type         string `json:"SecurityTypeName"`
			} `json:"Data"`
		} `json:"QuotationCodeTable"`
	}
	if err := json.Unmarshal(payload, &result); err != nil {
		s.logger.Warn("stock search parse failed", "error", err)
		return nil
	}

	items := make([]stockdomain.StockItem, 0, len(result.QuotationCodeTable.Data))
	for _, d := range result.QuotationCodeTable.Data {
		code := strings.TrimSpace(d.Code)
		if len(code) != 6 || !httpclient.IsAllDigits(code) {
			continue
		}
		market := stockMarketPrefix(code)
		if market == "" {
			continue
		}
		items = append(items, stockdomain.StockItem{
			StockCode: code,
			StockName: strings.TrimSpace(d.Name),
			Market:    market,
			Pinyin:    strings.TrimSpace(d.Pinyin),
		})
	}
	return items
}

// fetchRankingFromTencent 从腾讯行情 API 获取股票排行数据。
func (s *StockService) fetchRankingFromTencent(ctx context.Context, rankingType string, size int) []stockdomain.StockRankingItem {
	if s.stockQuote == nil {
		return nil
	}

	allStocks := s.store.ListStocks()
	if len(allStocks) == 0 {
		return nil
	}

	codes := make([]string, 0, len(allStocks))
	for _, stock := range allStocks {
		if stockMarketPrefix(stock.StockCode) != "" {
			codes = append(codes, stock.StockCode)
		}
	}
	if len(codes) == 0 {
		return nil
	}

	quotes := s.stockQuote.FetchQuotes(ctx, codes)
	if len(quotes) == 0 {
		s.logger.Warn("tencent stock quote returned empty for ranking")
		return nil
	}

	items := make([]stockdomain.StockRankingItem, 0, len(quotes))
	for code, q := range quotes {
		if q.Price <= 0 {
			continue
		}
		items = append(items, stockdomain.StockRankingItem{
			StockCode:    code,
			StockName:    "", // will be filled from local stock list
			CurrentPrice: q.Price,
			ChangePct:    q.ChangePct,
			Volume:       q.Volume,
			Amount:       q.Amount,
		})
	}

	// Fill stock names from local list
	stockNameMap := make(map[string]string, len(allStocks))
	for _, stock := range allStocks {
		stockNameMap[stock.StockCode] = stock.StockName
	}
	for i := range items {
		if items[i].StockName == "" {
			items[i].StockName = stockNameMap[items[i].StockCode]
		}
	}

	return normalizeStockRankingItems(items, rankingType, size)
}

// fetchRankingFromAPI 从东方财富 API 获取股票排行数据。
func (s *StockService) fetchRankingFromAPI(ctx context.Context, rankingType string, size int) []stockdomain.StockRankingItem {
	sortField := "f3"
	sortOrder := "1"
	switch rankingType {
	case "losers":
		sortOrder = "0"
	case "volume":
		sortField = "f5"
	}

	url := fmt.Sprintf("%s?pn=1&pz=%d&po=%s&np=1&fltt=2&invt=2&fid=%s&fs=m:0+t:6,m:0+t:80,m:1+t:2,m:1+t:23&fields=f2,f3,f4,f5,f6,f12,f14", eastmoneyStockListURL, size, sortOrder, sortField)
	if !isAllowedURL(url) {
		return nil
	}
	payload, err := s.eastmoney.Get(ctx, url, "https://quote.eastmoney.com/")
	if err != nil {
		return nil
	}

	var result struct {
		Data struct {
			Diff []struct {
				F2  float64 `json:"f2"`
				F3  float64 `json:"f3"`
				F5  float64 `json:"f5"`
				F6  float64 `json:"f6"`
				F12 string  `json:"f12"`
				F14 string  `json:"f14"`
			} `json:"diff"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil
	}
	if len(result.Data.Diff) == 0 {
		return nil
	}

	items := make([]stockdomain.StockRankingItem, 0, len(result.Data.Diff))
	for i, d := range result.Data.Diff {
		items = append(items, stockdomain.StockRankingItem{
			Rank:         i + 1,
			StockCode:    d.F12,
			StockName:    d.F14,
			CurrentPrice: d.F2,
			ChangePct:    d.F3,
			Volume:       d.F5,
			Amount:       d.F6,
		})
	}
	return normalizeStockRankingItems(items, rankingType, size)
}

// localRanking 基于本地已加载的股票列表计算排行。
func (s *StockService) localRanking(rankingType string, size int) []stockdomain.StockRankingItem {
	allStocks := s.store.ListStocks()
	items := make([]stockdomain.StockRankingItem, 0, len(allStocks))
	for i, stock := range allStocks {
		items = append(items, stockdomain.StockRankingItem{
			Rank:         i + 1,
			StockCode:    stock.StockCode,
			StockName:    stock.StockName,
			CurrentPrice: stock.CurrentPrice,
			ChangePct:    stock.ChangePct,
			Volume:       stock.Volume,
			Amount:       stock.Amount,
		})
	}

	switch rankingType {
	case "gainers":
		sort.SliceStable(items, func(i, j int) bool {
			return items[i].ChangePct > items[j].ChangePct
		})
	case "losers":
		sort.SliceStable(items, func(i, j int) bool {
			return items[i].ChangePct < items[j].ChangePct
		})
	case "volume":
		sort.SliceStable(items, func(i, j int) bool {
			return items[i].Volume > items[j].Volume
		})
	}

	return normalizeStockRankingItems(items, rankingType, size)
}
