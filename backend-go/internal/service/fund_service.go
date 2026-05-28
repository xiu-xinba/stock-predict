package service

import (
	"errors"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"stock-predict-go/internal/dto"
)

var (
	ErrInvalidRankingType = errors.New("invalid ranking type")
	ErrSyncSourceRequired = errors.New("fund sync source is required")
	ErrSyncUnsupported    = errors.New("fund repository does not support sync")
)

type FundRepository interface {
	ListFunds() []dto.FundItem
	FindFund(code string) (dto.FundItem, bool)
	CountFunds() int
}

type fundSyncRepository interface {
	SyncFundsFromCSV(path string) (int, error)
	CountFunds() int
}

type fundUniverseRepository interface {
	SyncFundsFromEastmoneySources(universeURL, metricsURL string) (int, error)
	CountFunds() int
}

type fundDataPathRepository interface {
	DataPath() string
}

type FundService struct {
	store FundRepository
}

func NewFundService(store FundRepository) *FundService {
	return &FundService{store: store}
}

func (s *FundService) Search(q dto.FundSearchRequest) dto.FundSearchData {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Size < 1 {
		q.Size = 20
	}
	if q.Size > 50 {
		q.Size = 50
	}

	keyword := strings.TrimSpace(strings.ToLower(q.Keyword))
	items := make([]dto.FundItem, 0)
	for _, fund := range s.store.ListFunds() {
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
	return dto.FundSearchData{
		Items: items[start:end],
		Total: total,
		Page:  q.Page,
		Size:  q.Size,
	}
}

func (s *FundService) Ranking(rankingType string, size int) ([]dto.FundRankingItem, error) {
	if rankingType != "gainers" && rankingType != "losers" {
		return nil, ErrInvalidRankingType
	}
	if size < 1 {
		size = 5
	}
	if size > 50 {
		size = 50
	}

	funds := s.store.ListFunds()
	items := make([]dto.FundRankingItem, 0, len(funds))
	for _, fund := range funds {
		if !hasTrustedQuote(fund) {
			continue
		}
		items = append(items, dto.FundRankingItem{
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

func (s *FundService) Filters() dto.FundFilters {
	types := map[string]bool{}
	companies := map[string]bool{}
	risks := map[string]bool{}
	for _, fund := range s.store.ListFunds() {
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
	return dto.FundFilters{
		Types:      keys(types),
		Companies:  keys(companies),
		RiskLevels: keys(risks),
	}
}

func (s *FundService) Count() int {
	return s.store.CountFunds()
}

func (s *FundService) SyncFromCSV(path string) (dto.FundSyncResult, error) {
	return s.SyncFromSources("", "", path)
}

func (s *FundService) SyncFromSources(universeURL, metricsURL, csvPath string) (dto.FundSyncResult, error) {
	universeURL = strings.TrimSpace(universeURL)
	metricsURL = strings.TrimSpace(metricsURL)
	csvPath = strings.TrimSpace(csvPath)
	if universeURL == "" && metricsURL == "" && csvPath == "" {
		return dto.FundSyncResult{}, ErrSyncSourceRequired
	}
	imported := 0
	sources := make([]string, 0, 3)
	if universeURL != "" || metricsURL != "" {
		syncer, ok := s.store.(fundUniverseRepository)
		if !ok {
			return dto.FundSyncResult{}, ErrSyncUnsupported
		}
		count, err := syncer.SyncFundsFromEastmoneySources(universeURL, metricsURL)
		if err != nil {
			return dto.FundSyncResult{}, err
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
			return dto.FundSyncResult{}, err
		}
		imported += result.Imported
		sources = append(sources, csvPath)
	}
	result := dto.FundSyncResult{
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

func hasTrustedQuote(fund dto.FundItem) bool {
	return fund.QuoteSource != "" && (fund.LatestNAV != 0 || fund.EstimatedNAV != 0)
}

func (s *FundService) syncCSVOnly(path string) (dto.FundSyncResult, error) {
	if strings.TrimSpace(path) == "" {
		return dto.FundSyncResult{}, ErrSyncSourceRequired
	}
	syncer, ok := s.store.(fundSyncRepository)
	if !ok {
		return dto.FundSyncResult{}, ErrSyncUnsupported
	}
	imported, err := syncer.SyncFundsFromCSV(path)
	if err != nil {
		return dto.FundSyncResult{}, err
	}
	result := dto.FundSyncResult{
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

func (s *FundService) Find(code string) (dto.FundItem, bool) {
	return s.store.FindFund(code)
}

func sortFunds(items []dto.FundItem, sortBy, sortOrder, keyword string) {
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

func searchRelevance(fund dto.FundItem, keyword string) int {
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

func fundMatchesKeyword(fund dto.FundItem, keyword string) bool {
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

func keys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}
