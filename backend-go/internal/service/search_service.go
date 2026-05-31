package service

import (
	"strings"

	"stock-predict-go/internal/dto"
	"stock-predict-go/internal/store"
)

type SearchService struct {
	fundRepo  store.FundRepository
	stockSvc  *StockService
	searchIdx *store.SearchIndex
}

func NewSearchService(fundRepo store.FundRepository, stockSvc *StockService, searchIdx *store.SearchIndex) *SearchService {
	return &SearchService{
		fundRepo:  fundRepo,
		stockSvc:  stockSvc,
		searchIdx: searchIdx,
	}
}

func (s *SearchService) Search(req dto.UnifiedSearchRequest) dto.UnifiedSearchResponse {
	req.Defaults()
	resp := dto.UnifiedSearchResponse{
		Query: req.Query,
	}

	if req.IncludeFunds() {
		resp.Funds = s.searchFunds(req)
	}

	if req.IncludeStocks() {
		resp.Stocks = s.searchStocks(req)
	}

	return resp
}

func (s *SearchService) searchFunds(req dto.UnifiedSearchRequest) dto.FundSearchData {
	result := dto.FundSearchData{
		Page: req.Page,
		Size: req.Size,
	}

	keyword := strings.TrimSpace(strings.ToLower(req.Query))
	if keyword == "" {
		return result
	}

	allFunds := s.fundRepo.ListFunds()

	matched := make(map[string]int)
	maxMatches := 5000

	for _, f := range allFunds {
		if searchFundMatchesKeyword(f, keyword) {
			matched[f.FundCode] = searchFundRelevance(f, keyword)
			if len(matched) >= maxMatches {
				break
			}
		}
	}

	if s.searchIdx != nil {
		ftsCodes, err := s.searchIdx.SearchFundsByCodeOrPinyin(req.Query, 200)
		if err == nil {
			for _, code := range ftsCodes {
				if _, exists := matched[code]; !exists {
					matched[code] = 10
				}
			}
		}
	}

	items := make([]dto.FundItem, 0, len(matched))
	for _, f := range allFunds {
		if _, ok := matched[f.FundCode]; ok {
			items = append(items, f)
		}
	}

	sortFundsByRelevance(items, matched)

	result.Total = len(items)

	start := (req.Page - 1) * req.Size
	if start >= len(items) {
		result.Items = []dto.FundItem{}
		return result
	}
	end := start + req.Size
	if end > len(items) {
		end = len(items)
	}
	result.Items = items[start:end]

	return result
}

func (s *SearchService) searchStocks(req dto.UnifiedSearchRequest) dto.StockSearchData {
	result := dto.StockSearchData{
		Page: req.Page,
		Size: req.Size,
	}

	keyword := strings.TrimSpace(strings.ToLower(req.Query))
	if keyword == "" {
		return result
	}

	allStocks := s.stockSvc.ListStocks()

	matched := make(map[string]int)
	maxMatches := 5000

	for _, st := range allStocks {
		if searchStockMatchesKeyword(st, keyword) {
			matched[st.StockCode] = searchStockRelevance(st, keyword)
			if len(matched) >= maxMatches {
				break
			}
		}
	}

	if s.searchIdx != nil {
		ftsCodes, err := s.searchIdx.SearchStocksByCodeOrPinyin(req.Query, 200)
		if err == nil {
			for _, code := range ftsCodes {
				if _, exists := matched[code]; !exists {
					matched[code] = 10
				}
			}
		}
	}

	items := make([]dto.StockItem, 0, len(matched))
	for _, st := range allStocks {
		if _, ok := matched[st.StockCode]; ok {
			items = append(items, st)
		}
	}

	sortStocksByRelevance(items, matched)

	result.Total = len(items)

	start := (req.Page - 1) * req.Size
	if start >= len(items) {
		result.Items = []dto.StockItem{}
		return result
	}
	end := start + req.Size
	if end > len(items) {
		end = len(items)
	}
	result.Items = items[start:end]

	return result
}

func searchFundMatchesKeyword(fund dto.FundItem, keyword string) bool {
	for _, value := range []string{
		fund.FundCode,
		fund.FundName,
		fund.PinyinAbbr,
		fund.PinyinFull,
		fund.Company,
		fund.Manager,
	} {
		if strings.Contains(strings.ToLower(value), keyword) {
			return true
		}
	}
	return false
}

func searchFundRelevance(fund dto.FundItem, keyword string) int {
	code := strings.ToLower(fund.FundCode)
	name := strings.ToLower(fund.FundName)
	pinyinAbbr := strings.ToLower(fund.PinyinAbbr)
	pinyinFull := strings.ToLower(fund.PinyinFull)
	switch {
	case code == keyword || name == keyword:
		return 0
	case strings.HasPrefix(code, keyword):
		return 1
	case strings.HasPrefix(name, keyword):
		return 2
	case strings.HasPrefix(pinyinAbbr, keyword):
		return 3
	case strings.HasPrefix(pinyinFull, keyword):
		return 4
	case strings.Contains(code, keyword):
		return 5
	case strings.Contains(name, keyword):
		return 6
	case strings.Contains(pinyinAbbr, keyword):
		return 7
	case strings.Contains(pinyinFull, keyword):
		return 8
	default:
		return 9
	}
}

func searchStockMatchesKeyword(stock dto.StockItem, keyword string) bool {
	for _, value := range []string{
		stock.StockCode,
		stock.StockName,
		stock.Pinyin,
		stock.PinyinAlt,
		stock.Industry,
		stock.Market,
	} {
		if strings.Contains(strings.ToLower(value), keyword) {
			return true
		}
	}
	return false
}

func searchStockRelevance(stock dto.StockItem, keyword string) int {
	code := strings.ToLower(stock.StockCode)
	name := strings.ToLower(stock.StockName)
	pinyin := strings.ToLower(stock.Pinyin)
	pinyinAlt := strings.ToLower(stock.PinyinAlt)
	switch {
	case code == keyword || name == keyword:
		return 0
	case strings.HasPrefix(code, keyword):
		return 1
	case strings.HasPrefix(name, keyword):
		return 2
	case strings.HasPrefix(pinyin, keyword) || strings.HasPrefix(pinyinAlt, keyword):
		return 3
	case strings.Contains(code, keyword):
		return 5
	case strings.Contains(name, keyword):
		return 6
	case strings.Contains(pinyin, keyword) || strings.Contains(pinyinAlt, keyword):
		return 7
	default:
		return 9
	}
}

func sortFundsByRelevance(items []dto.FundItem, scores map[string]int) {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0; j-- {
			si := scores[items[j].FundCode]
			sj := scores[items[j-1].FundCode]
			if si < sj || (si == sj && items[j].FundCode < items[j-1].FundCode) {
				items[j], items[j-1] = items[j-1], items[j]
			} else {
				break
			}
		}
	}
}

func sortStocksByRelevance(items []dto.StockItem, scores map[string]int) {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0; j-- {
			si := scores[items[j].StockCode]
			sj := scores[items[j-1].StockCode]
			if si < sj || (si == sj && items[j].StockCode < items[j-1].StockCode) {
				items[j], items[j-1] = items[j-1], items[j]
			} else {
				break
			}
		}
	}
}
