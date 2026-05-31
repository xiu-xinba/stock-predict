package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"stock-predict-go/internal/dto"
	"stock-predict-go/internal/util"

	"golang.org/x/text/encoding/simplifiedchinese"
)

const (
	eastmoneyStockSearchURL = "https://searchapi.eastmoney.com/api/suggest/get?input=%s&type=14"
	eastmoneyStockListURL   = "https://push2.eastmoney.com/api/qt/clist/get"
	eastmoneyDataCenterURL  = "https://datacenter-web.eastmoney.com/api/data/v1/get"
)

func (s *StockService) fetchAllStocksFromAPI(ctx context.Context) []dto.StockItem {
	fsValues := []string{
		"m:0+t:6,m:0+t:80",
		"m:1+t:2,m:1+t:23",
		"m:0+t:81",
	}

	fields := "f2,f3,f5,f6,f8,f9,f12,f14,f20,f23,f100,f116,f117"

	var allItems []dto.StockItem

	for _, fs := range fsValues {
		page := 1
		pageSize := 5000
		for {
			url := fmt.Sprintf("%s?pn=%d&pz=%d&po=1&np=1&fltt=2&invt=2&fid=f12&fs=%s&fields=%s",
				eastmoneyStockListURL, page, pageSize, fs, fields)

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				s.logger.Warn("stock list request failed", "error", err)
				break
			}
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
			req.Header.Set("Referer", "https://quote.eastmoney.com/")

			resp, err := s.client.Do(req)
			if err != nil {
				s.logger.Warn("stock list API failed", "fs", fs, "page", page, "error", err)
				break
			}

			payload, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
			resp.Body.Close()
			if err != nil {
				s.logger.Warn("stock list read failed", "error", err)
				break
			}

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				s.logger.Warn("stock list API status", "status", resp.StatusCode)
				break
			}

			var result struct {
				Data struct {
					Total int `json:"total"`
					Diff  []struct {
						F2   interface{} `json:"f2"`
						F3   interface{} `json:"f3"`
						F5   interface{} `json:"f5"`
						F6   interface{} `json:"f6"`
						F8   interface{} `json:"f8"`
						F9   interface{} `json:"f9"`
						F12  string      `json:"f12"`
						F14  string      `json:"f14"`
						F20  interface{} `json:"f20"`
						F23  interface{} `json:"f23"`
						F100 string      `json:"f100"`
						F116 interface{} `json:"f116"`
						F117 interface{} `json:"f117"`
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
				if len(code) != 6 || !util.IsAllDigits(code) {
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
				allItems = append(allItems, dto.StockItem{
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

			if len(result.Data.Diff) < pageSize {
				break
			}
			page++

			time.Sleep(200 * time.Millisecond)
		}
	}

	s.logger.Info("fetched stocks from API", "total", len(allItems))
	return allItems
}

func (s *StockService) fetchStocksFromDataCenter(ctx context.Context) []dto.StockItem {
	var allItems []dto.StockItem

	for page := 1; page <= 100; page++ {
		url := fmt.Sprintf("%s?sortColumns=SECURITY_CODE&sortTypes=1&pageSize=500&pageNumber=%d&reportName=RPT_LICO_FN_CPD&columns=SECURITY_CODE,SECURITY_NAME_ABBR,SECUCODE,BOARD_NAME&source=WEB&client=WEB&filter=(ISNEW%%3D%%221%%22)",
			eastmoneyDataCenterURL, page)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			s.logger.Warn("data center request failed", "error", err)
			break
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
		req.Header.Set("Referer", "https://data.eastmoney.com/")

		resp, err := s.client.Do(req)
		if err != nil {
			s.logger.Warn("data center API failed", "page", page, "error", err)
			break
		}

		payload, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
		resp.Body.Close()
		if err != nil {
			s.logger.Warn("data center read failed", "error", err)
			break
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			s.logger.Warn("data center API status", "status", resp.StatusCode)
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
			if len(code) != 6 || !util.IsAllDigits(code) {
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

			allItems = append(allItems, dto.StockItem{
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
		time.Sleep(150 * time.Millisecond)
	}

	s.logger.Info("fetched stocks from data center", "total", len(allItems))
	return allItems
}

func (s *StockService) searchFromAPI(ctx context.Context, keyword string) []dto.StockItem {
	url := fmt.Sprintf(eastmoneyStockSearchURL, keyword)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		s.logger.Warn("stock search request failed", "error", err)
		return nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
	req.Header.Set("Referer", "https://so.eastmoney.com/")

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Warn("stock search API failed", "error", err)
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil
	}
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
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

	items := make([]dto.StockItem, 0, len(result.QuotationCodeTable.Data))
	for _, d := range result.QuotationCodeTable.Data {
		code := strings.TrimSpace(d.Code)
		if len(code) != 6 || !util.IsAllDigits(code) {
			continue
		}
		market := stockMarketPrefix(code)
		if market == "" {
			continue
		}
		items = append(items, dto.StockItem{
			StockCode: code,
			StockName: strings.TrimSpace(d.Name),
			Market:    market,
			Pinyin:    strings.TrimSpace(d.Pinyin),
		})
	}
	return items
}

func (s *StockService) fetchRankingFromAPI(ctx context.Context, rankingType string, size int) []dto.StockRankingItem {
	sortField := "f3"
	sortOrder := "0"
	if rankingType == "losers" {
		sortOrder = "1"
	} else if rankingType == "volume" {
		sortField = "f5"
	}

	url := fmt.Sprintf("%s?pn=1&pz=%d&po=%s&np=1&fltt=2&invt=2&fid=%s&fs=m:0+t:6,m:0+t:80,m:1+t:2,m:1+t:23&fields=f2,f3,f4,f5,f6,f12,f14", eastmoneyStockListURL, size, sortOrder, sortField)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
	req.Header.Set("Referer", "https://quote.eastmoney.com/")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil
	}
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
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

	items := make([]dto.StockRankingItem, 0, len(result.Data.Diff))
	for i, d := range result.Data.Diff {
		items = append(items, dto.StockRankingItem{
			Rank:         i + 1,
			StockCode:    d.F12,
			StockName:    d.F14,
			CurrentPrice: d.F2,
			ChangePct:    d.F3,
			Volume:       d.F5,
			Amount:       d.F6,
		})
	}
	return items
}

func (s *StockService) localRanking(rankingType string, size int) []dto.StockRankingItem {
	s.mu.RLock()
	items := make([]dto.StockRankingItem, 0, len(s.stocks))
	for i, stock := range s.stocks {
		items = append(items, dto.StockRankingItem{
			Rank:         i + 1,
			StockCode:    stock.StockCode,
			StockName:    stock.StockName,
			CurrentPrice: stock.CurrentPrice,
			ChangePct:    stock.ChangePct,
			Volume:       stock.Volume,
			Amount:       stock.Amount,
		})
	}
	s.mu.RUnlock()

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

	for i := range items {
		items[i].Rank = i + 1
	}
	if len(items) > size {
		items = items[:size]
	}
	return items
}

func toNum(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		if n < -1e15 || n > 1e15 {
			return 0
		}
		return n
	case string:
		if n == "" || n == "-" {
			return 0
		}
		f, err := strconv.ParseFloat(n, 64)
		if err != nil {
			return 0
		}
		if f < -1e15 || f > 1e15 {
			return 0
		}
		return f
	case json.Number:
		s := n.String()
		if s == "" || s == "-" {
			return 0
		}
		f, err := n.Float64()
		if err != nil {
			return 0
		}
		if f < -1e15 || f > 1e15 {
			return 0
		}
		return f
	default:
		return 0
	}
}

var polyphoneOverrides = map[rune][]string{
	0x884C: {"H", "X"},
	0x91CD: {"Z", "C"},
	0x957F: {"C", "Z"},
	0x4E50: {"L", "Y"},
	0x53C2: {"C", "S"},
	0x5355: {"D", "S"},
}

func pinyinAbbr(name string) string {
	var abbr strings.Builder
	for _, r := range name {
		if r >= 0x4e00 && r <= 0x9fff {
			if overrides, ok := polyphoneOverrides[r]; ok {
				abbr.WriteString(overrides[0])
				continue
			}
			initial := pinyinInitial(r)
			if initial != "" {
				abbr.WriteString(initial)
			}
		} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			abbr.WriteRune(r)
		}
	}
	return strings.ToLower(abbr.String())
}

func pinyinAbbrAll(name string) []string {
	type polyPos struct {
		idx  int
		alts []string
	}
	var polys []polyPos

	runes := []rune(name)
	for i, r := range runes {
		if r >= 0x4e00 && r <= 0x9fff {
			if overrides, ok := polyphoneOverrides[r]; ok {
				polys = append(polys, polyPos{idx: i, alts: overrides})
			}
		}
	}

	if len(polys) == 0 {
		return []string{pinyinAbbr(name)}
	}

	var baseRunes []rune
	for _, r := range runes {
		if r >= 0x4e00 && r <= 0x9fff {
			if overrides, ok := polyphoneOverrides[r]; ok {
				baseRunes = append(baseRunes, []rune(strings.ToLower(overrides[0]))...)
			} else {
				initial := pinyinInitial(r)
				if initial != "" {
					baseRunes = append(baseRunes, []rune(strings.ToLower(initial))...)
				}
			}
		} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			baseRunes = append(baseRunes, r)
		}
	}

	base := string(baseRunes)
	results := []string{base}

	for _, p := range polys {
		charIdx := 0
		for i := 0; i < p.idx; i++ {
			r := runes[i]
			if r >= 0x4e00 && r <= 0x9fff {
				if ov, ok := polyphoneOverrides[r]; ok {
					charIdx += len(ov[0])
				} else {
					init := pinyinInitial(r)
					if init != "" {
						charIdx++
					}
				}
			} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
				charIdx++
			}
		}

		for _, alt := range p.alts[1:] {
			newRunes := make([]rune, len(baseRunes))
			copy(newRunes, baseRunes)
			altLower := []rune(strings.ToLower(alt))
			if charIdx+len(altLower) <= len(newRunes) {
				for j, ar := range altLower {
					newRunes[charIdx+j] = ar
				}
				results = append(results, string(newRunes))
			}
		}
	}

	return results
}

var pinyinInitialTable = []struct {
	code    int
	initial string
}{
	{45217, "A"}, {45253, "B"}, {45761, "C"}, {46318, "D"},
	{46826, "E"}, {47010, "F"}, {47297, "G"}, {47614, "H"},
	{48119, "J"}, {49062, "K"}, {49324, "L"}, {49896, "M"},
	{50371, "N"}, {50614, "O"}, {50622, "P"}, {50906, "Q"},
	{51387, "R"}, {51446, "S"}, {52218, "T"}, {52698, "W"},
	{52980, "X"}, {53689, "Y"}, {54481, "Z"},
}

func pinyinInitial(r rune) string {
	if r < 0x4E00 || r > 0x9FFF {
		return ""
	}
	gbBytes, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(string(r)))
	if err != nil || len(gbBytes) != 2 {
		return ""
	}
	gbCode := int(gbBytes[0])<<8 | int(gbBytes[1])
	if gbCode < pinyinInitialTable[0].code || gbCode > 55289 {
		return ""
	}
	i := sort.Search(len(pinyinInitialTable), func(i int) bool {
		return pinyinInitialTable[i].code > gbCode
	})
	if i == 0 {
		return ""
	}
	return pinyinInitialTable[i-1].initial
}
