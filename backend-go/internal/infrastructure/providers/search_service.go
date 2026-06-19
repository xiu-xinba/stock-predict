package providers

import (
	"strings"

	funddomain "stock-predict-go/internal/domain/fund"
	searchdomain "stock-predict-go/internal/domain/search"
	stockdomain "stock-predict-go/internal/domain/stock"
	database "stock-predict-go/internal/infrastructure/database"
)

// SearchService 统一搜索服务，支持基金和股票的联合搜索。
type SearchService struct {
	fundRepo  funddomain.Repository
	stockRepo stockdomain.Repository
	searchIdx *database.SearchStore
}

// NewSearchService 创建新的统一搜索服务实例。
func NewSearchService(fundRepo funddomain.Repository, stockRepo stockdomain.Repository, searchIdx *database.SearchStore) *SearchService {
	return &SearchService{
		fundRepo:  fundRepo,
		stockRepo: stockRepo,
		searchIdx: searchIdx,
	}
}

// Search 执行统一搜索，根据请求类型分别搜索基金和/或股票。
func (s *SearchService) Search(req searchdomain.UnifiedSearchRequest) (searchdomain.UnifiedSearchResponse, error) {
	req.Defaults()
	resp := searchdomain.UnifiedSearchResponse{
		Query: req.Query,
	}

	if req.IncludeFunds() {
		funds, err := s.searchFunds(req)
		if err != nil {
			return searchdomain.UnifiedSearchResponse{}, err
		}
		resp.Funds = funds
	}

	if req.IncludeStocks() {
		stocks, err := s.searchStocks(req)
		if err != nil {
			return searchdomain.UnifiedSearchResponse{}, err
		}
		resp.Stocks = stocks
	}

	return resp, nil
}

// searchFunds 搜索基金，优先使用全文搜索索引，回退到内存过滤。
func (s *SearchService) searchFunds(req searchdomain.UnifiedSearchRequest) (funddomain.FundSearchData, error) {
	result := funddomain.FundSearchData{
		Page: req.Page,
		Size: req.Size,
	}

	keyword := strings.TrimSpace(strings.ToLower(req.Query))
	if keyword == "" {
		return result, nil
	}

	allFunds, err := listFundsWithError(s.fundRepo)
	if err != nil {
		return funddomain.FundSearchData{}, err
	}

	matched := make(map[string]int)

	for _, f := range allFunds {
		if searchFundMatchesKeyword(f, keyword) {
			matched[f.FundCode] = searchFundRelevance(f, keyword)
			if len(matched) >= MaxSearchMatches {
				break
			}
		}
	}

	if s.searchIdx != nil {
		ftsCodes, err := s.searchIdx.SearchFundsByCodeOrPinyin(req.Query, FTSSearchLimit)
		if err != nil {
			return funddomain.FundSearchData{}, err
		}
		for _, code := range ftsCodes {
			if _, exists := matched[code]; !exists {
				matched[code] = 10
			}
		}
	}

	items := make([]funddomain.FundItem, 0, len(matched))
	for _, f := range allFunds {
		if _, ok := matched[f.FundCode]; ok {
			items = append(items, f)
		}
	}

	sortFundsByRelevance(items, matched)

	result.Total = len(items)

	start := (req.Page - 1) * req.Size
	if start >= len(items) {
		result.Items = []funddomain.FundItem{}
		return result, nil
	}
	end := start + req.Size
	if end > len(items) {
		end = len(items)
	}
	result.Items = items[start:end]

	return result, nil
}

// searchStocks 搜索股票，优先使用全文搜索索引，回退到内存过滤。
func (s *SearchService) searchStocks(req searchdomain.UnifiedSearchRequest) (stockdomain.StockSearchData, error) {
	result := stockdomain.StockSearchData{
		Page: req.Page,
		Size: req.Size,
	}

	keyword := strings.TrimSpace(strings.ToLower(req.Query))
	if keyword == "" {
		return result, nil
	}

	allStocks, err := listStocksWithError(s.stockRepo)
	if err != nil {
		return stockdomain.StockSearchData{}, err
	}

	matched := make(map[string]int)

	for _, st := range allStocks {
		if stockMatchesKeyword(st, keyword) {
			matched[st.StockCode] = stockSearchRelevance(st, keyword)
			if len(matched) >= MaxSearchMatches {
				break
			}
		}
	}

	if s.searchIdx != nil {
		ftsCodes, err := s.searchIdx.SearchStocksByCodeOrPinyin(req.Query, FTSSearchLimit)
		if err != nil {
			return stockdomain.StockSearchData{}, err
		}
		for _, code := range ftsCodes {
			if _, exists := matched[code]; !exists {
				matched[code] = 10
			}
		}
	}

	items := make([]stockdomain.StockItem, 0, len(matched))
	for _, st := range allStocks {
		if _, ok := matched[st.StockCode]; ok {
			items = append(items, st)
		}
	}

	sortStocksByRelevance(items, matched)

	result.Total = len(items)

	start := (req.Page - 1) * req.Size
	if start >= len(items) {
		result.Items = []stockdomain.StockItem{}
		return result, nil
	}
	end := start + req.Size
	if end > len(items) {
		end = len(items)
	}
	result.Items = items[start:end]

	return result, nil
}

func listFundsWithError(repository funddomain.Repository) ([]funddomain.FundItem, error) {
	type repositoryWithError interface {
		ListFundsWithError() ([]funddomain.FundItem, error)
	}
	if typed, ok := repository.(repositoryWithError); ok {
		return typed.ListFundsWithError()
	}
	return repository.ListFunds(), nil
}

func listStocksWithError(repository stockdomain.Repository) ([]stockdomain.StockItem, error) {
	type repositoryWithError interface {
		ListStocksWithError() ([]stockdomain.StockItem, error)
	}
	if typed, ok := repository.(repositoryWithError); ok {
		return typed.ListStocksWithError()
	}
	return repository.ListStocks(), nil
}

func searchFundMatchesKeyword(fund funddomain.FundItem, keyword string) bool {
	return fundMatchesKeyword(fund, keyword)
}

func searchFundRelevance(fund funddomain.FundItem, keyword string) int {
	return searchRelevance(fund, keyword)
}

func sortFundsByRelevance(items []funddomain.FundItem, scores map[string]int) {
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

func sortStocksByRelevance(items []stockdomain.StockItem, scores map[string]int) {
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
