package service

import (
	"errors"
	"sort"
	"strings"

	"stock-predict-go/internal/dto"
)

var ErrInvalidRankingType = errors.New("invalid ranking type")

type FundRepository interface {
	ListFunds() []dto.FundItem
	FindFund(code string) (dto.FundItem, bool)
	CountFunds() int
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
		q.Size = 1
	}
	if q.Size > 50 {
		q.Size = 50
	}

	keyword := strings.TrimSpace(strings.ToLower(q.Keyword))
	items := make([]dto.FundItem, 0)
	for _, fund := range s.store.ListFunds() {
		if keyword != "" && !strings.Contains(strings.ToLower(fund.FundCode), keyword) && !strings.Contains(strings.ToLower(fund.FundName), keyword) {
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
		items = append(items, dto.FundRankingItem{
			FundCode:     fund.FundCode,
			FundName:     fund.FundName,
			FundType:     fund.FundType,
			ChangePct:    fund.ChangePct,
			EstimatedNAV: fund.EstimatedNAV,
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
				ai := strings.HasPrefix(strings.ToLower(a.FundCode), keyword) || strings.Contains(strings.ToLower(a.FundName), keyword)
				bi := strings.HasPrefix(strings.ToLower(b.FundCode), keyword) || strings.Contains(strings.ToLower(b.FundName), keyword)
				if ai != bi {
					if ai {
						cmp = -1
					} else {
						cmp = 1
					}
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

func keys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}
