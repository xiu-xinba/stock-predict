package providers

import (
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	funddomain "stock-predict-go/internal/domain/fund"
)

// fundListRepositoryWithError 支持带错误返回的基金列表接口。
type fundListRepositoryWithError interface {
	ListFundsWithError() ([]funddomain.FundItem, error)
}

// fundSyncRepository 支持从 CSV 同步基金的接口。
type fundSyncRepository interface {
	SyncFundsFromCSV(path string) (int, error)
	CountFunds() int
}

// fundUniverseRepository 支持从东方财富源同步基金的接口。
type fundUniverseRepository interface {
	SyncFundsFromEastmoneySources(universeURL, metricsURL string) (int, error)
	CountFunds() int
}

// fundDataPathRepository 支持获取数据存储路径的接口。
type fundDataPathRepository interface {
	DataPath() string
}

// FundService 基金服务，提供基金搜索、排行、筛选、同步等功能。
type FundService struct {
	store funddomain.Repository
}

// NewFundService 创建新的基金服务实例。
func NewFundService(store funddomain.Repository) *FundService {
	return &FundService{store: store}
}

// Search 根据关键词和筛选条件搜索基金，支持分页和排序。
func (s *FundService) Search(q funddomain.FundSearchRequest) (funddomain.FundSearchData, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Size < 1 {
		q.Size = DefaultSearchSize
	}
	if q.Size > MaxSearchSize {
		q.Size = MaxSearchSize
	}

	keyword := strings.TrimSpace(strings.ToLower(q.Keyword))
	items := make([]funddomain.FundItem, 0)
	funds, err := s.listFunds()
	if err != nil {
		return funddomain.FundSearchData{}, err
	}
	for _, fund := range funds {
		if keyword != "" && !fundMatchesKeyword(fund, keyword) {
			continue
		}
		if q.Type != "" && fund.FundType != q.Type {
			continue
		}
		if q.Company != "" && fund.Company != q.Company {
			continue
		}
		if q.RiskLevel != "" && fund.RiskLevel != q.RiskLevel {
			continue
		}
		if q.Manager != "" && !strings.Contains(fund.Manager, q.Manager) {
			continue
		}
		if q.ReturnMin != nil && fund.Return1Y < *q.ReturnMin {
			continue
		}
		if q.ReturnMax != nil && fund.Return1Y > *q.ReturnMax {
			continue
		}
		items = append(items, fund)
	}

	sortFunds(items, q.SortBy, q.SortOrder, keyword)
	total := len(items)
	start := (q.Page - 1) * q.Size
	if start > total {
		start = total
	}
	end := start + q.Size
	if end > total {
		end = total
	}
	return funddomain.FundSearchData{
		Items: items[start:end],
		Total: total,
		Page:  q.Page,
		Size:  q.Size,
	}, nil
}

// Ranking 获取基金排行榜，支持日涨幅、周涨幅、月涨幅、近3月涨幅排序。
func (s *FundService) Ranking(rankingType string, size int) ([]funddomain.FundRankingItem, error) {
	if rankingType != "gainers" && rankingType != "losers" {
		return nil, ErrInvalidRankingType
	}
	if size < 1 {
		size = DefaultFundRankingSize
	}
	if size > MaxRankingSize {
		size = MaxRankingSize
	}

	funds, err := s.listFunds()
	if err != nil {
		return nil, err
	}
	items := make([]funddomain.FundRankingItem, 0, len(funds))
	for _, fund := range funds {
		if !hasTrustedQuote(fund) {
			continue
		}
		items = append(items, funddomain.FundRankingItem{
			FundCode:     fund.FundCode,
			FundName:     fund.FundName,
			FundType:     fund.FundType,
			ChangePct:    fund.ChangePct,
			EstimatedNAV: fund.EstimatedNAV,
			QuoteDate:    fund.QuoteDate,
			QuoteSource:  fund.QuoteSource,
		})
	}
	SortRanking(items, rankingType)
	if len(items) > size {
		items = items[:size]
	}
	return items, nil
}

// Filters 返回当前基金列表中所有可用的筛选项（基金类型、基金公司等）。
func (s *FundService) Filters() (funddomain.FundFilters, error) {
	types := map[string]bool{}
	companies := map[string]bool{}
	risks := map[string]bool{}
	funds, err := s.listFunds()
	if err != nil {
		return funddomain.FundFilters{}, err
	}
	for _, fund := range funds {
		if fund.FundType != "" {
			types[fund.FundType] = true
		}
		if fund.Company != "" {
			companies[fund.Company] = true
		}
		if fund.RiskLevel != "" {
			risks[fund.RiskLevel] = true
		}
	}
	return funddomain.FundFilters{
		Types:      keys(types),
		Companies:  keys(companies),
		RiskLevels: keys(risks),
	}, nil
}

func (s *FundService) listFunds() ([]funddomain.FundItem, error) {
	if repository, ok := s.store.(fundListRepositoryWithError); ok {
		return repository.ListFundsWithError()
	}
	return s.store.ListFunds(), nil
}

// Count 返回基金总数。
func (s *FundService) Count() int {
	return s.store.CountFunds()
}

// SyncFromCSV 从 CSV 文件同步基金数据。
func (s *FundService) SyncFromCSV(path string) (funddomain.FundSyncResult, error) {
	return s.SyncFromSources("", "", path)
}

// SyncFromSources 从多个数据源同步基金数据，优先使用东方财富源，回退到 CSV。
func (s *FundService) SyncFromSources(universeURL, metricsURL, csvPath string) (funddomain.FundSyncResult, error) {
	universeURL = strings.TrimSpace(universeURL)
	metricsURL = strings.TrimSpace(metricsURL)
	csvPath = strings.TrimSpace(csvPath)
	if universeURL == "" && metricsURL == "" && csvPath == "" {
		return funddomain.FundSyncResult{}, ErrSyncSourceRequired
	}
	imported := 0
	sources := make([]string, 0, 3)
	if universeURL != "" || metricsURL != "" {
		syncer, ok := s.store.(fundUniverseRepository)
		if !ok {
			return funddomain.FundSyncResult{}, ErrSyncUnsupported
		}
		count, err := syncer.SyncFundsFromEastmoneySources(universeURL, metricsURL)
		if err != nil {
			return funddomain.FundSyncResult{}, err
		}
		imported += count
		if universeURL != "" {
			sources = append(sources, universeURL)
		}
		if metricsURL != "" {
			sources = append(sources, metricsURL)
		}
	}
	if csvPath != "" {
		result, err := s.syncCSVOnly(csvPath)
		if err != nil {
			return funddomain.FundSyncResult{}, err
		}
		imported += result.Imported
		sources = append(sources, csvPath)
	}
	result := funddomain.FundSyncResult{
		Source:    strings.Join(sources, ","),
		Imported:  imported,
		Total:     s.store.CountFunds(),
		UpdatedAt: time.Now().Format(time.RFC3339Nano),
	}
	if dataPath, ok := s.store.(fundDataPathRepository); ok {
		result.StoredPath = dataPath.DataPath()
	}
	return result, nil
}

func hasTrustedQuote(fund funddomain.FundItem) bool {
	return fund.QuoteSource != "" && (fund.LatestNAV != 0 || fund.EstimatedNAV != 0)
}

func (s *FundService) syncCSVOnly(path string) (funddomain.FundSyncResult, error) {
	if strings.TrimSpace(path) == "" {
		return funddomain.FundSyncResult{}, ErrSyncSourceRequired
	}
	syncer, ok := s.store.(fundSyncRepository)
	if !ok {
		return funddomain.FundSyncResult{}, ErrSyncUnsupported
	}
	imported, err := syncer.SyncFundsFromCSV(path)
	if err != nil {
		return funddomain.FundSyncResult{}, err
	}
	result := funddomain.FundSyncResult{
		Source:    path,
		Imported:  imported,
		Total:     syncer.CountFunds(),
		UpdatedAt: time.Now().Format(time.RFC3339Nano),
	}
	if dataPath, ok := s.store.(fundDataPathRepository); ok {
		result.StoredPath = dataPath.DataPath()
	}
	return result, nil
}

// Find 根据基金代码查找基金信息。
func (s *FundService) Find(code string) (funddomain.FundItem, bool) {
	return s.store.FindFund(code)
}

func sortFunds(items []funddomain.FundItem, sortBy, sortOrder, keyword string) {
	desc := strings.ToLower(sortOrder) != "asc"
	if sortBy == "" || sortBy == "relevance" {
		desc = false
	}
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		cmp := 0
		switch sortBy {
		case "name":
			cmp = strings.Compare(a.FundName, b.FundName)
		case "return_1m":
			cmp = compareFloat(a.Return1M, b.Return1M)
		case "return_3m":
			cmp = compareFloat(a.Return3M, b.Return3M)
		case "return_6m":
			cmp = compareFloat(a.Return6M, b.Return6M)
		case "return_1y":
			cmp = compareFloat(a.Return1Y, b.Return1Y)
		case "return_3y":
			cmp = compareFloat(a.Return3Y, b.Return3Y)
		case "latest_nav":
			cmp = compareFloat(a.LatestNAV, b.LatestNAV)
		case "inception_date":
			cmp = strings.Compare(a.InceptionDate, b.InceptionDate)
		case "change_pct":
			cmp = compareFloat(a.ChangePct, b.ChangePct)
		default:
			if keyword != "" {
				as := searchRelevance(a, keyword)
				bs := searchRelevance(b, keyword)
				if as != bs {
					cmp = compareInt(as, bs)
				} else if nameCmp := compareInt(utf8.RuneCountInString(a.FundName), utf8.RuneCountInString(b.FundName)); nameCmp != 0 {
					cmp = nameCmp
				} else {
					cmp = strings.Compare(a.FundCode, b.FundCode)
				}
			} else {
				cmp = strings.Compare(a.FundCode, b.FundCode)
			}
		}
		if cmp == 0 {
			cmp = strings.Compare(a.FundCode, b.FundCode)
		}
		if desc {
			return cmp > 0
		}
		return cmp < 0
	})
}

func compareFloat(a, b float64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func compareInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func searchRelevance(fund funddomain.FundItem, keyword string) int {
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

func fundMatchesKeyword(fund funddomain.FundItem, keyword string) bool {
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

// keys extracts sorted keys from a string set; also used by StockService.Filters().
func keys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}
