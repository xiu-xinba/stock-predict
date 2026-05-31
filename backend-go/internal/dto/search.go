package dto

import (
	"strings"
)

type UnifiedSearchRequest struct {
	Query string `form:"q" binding:"required"`
	Types string `form:"types"`
	Page  int    `form:"page"`
	Size  int    `form:"size"`
}

func (r *UnifiedSearchRequest) Defaults() {
	if r.Page <= 0 {
		r.Page = 1
	}
	if r.Size <= 0 {
		r.Size = 10
	}
	if r.Size > 50 {
		r.Size = 50
	}
}

func (r *UnifiedSearchRequest) IncludeFunds() bool {
	return r.Types == "" || typesContains(r.Types, "fund")
}

func (r *UnifiedSearchRequest) IncludeStocks() bool {
	return r.Types == "" || typesContains(r.Types, "stock")
}

type UnifiedSearchResponse struct {
	Query       string          `json:"query"`
	Funds       FundSearchData  `json:"funds"`
	Stocks      StockSearchData `json:"stocks"`
	Suggestions []string        `json:"suggestions,omitempty"`
}

func typesContains(s, sub string) bool {
	for _, part := range strings.Split(s, ",") {
		if strings.TrimSpace(part) == sub {
			return true
		}
	}
	return false
}
