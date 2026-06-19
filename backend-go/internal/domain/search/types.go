// Package search 定义了搜索相关的领域模型和数据结构。
package search

import (
	"strings"

	funddomain "stock-predict-go/internal/domain/fund"
	stockdomain "stock-predict-go/internal/domain/stock"
)

// UnifiedSearchRequest 表示统一搜索的请求参数，支持同时搜索基金和股票。
type UnifiedSearchRequest struct {
	Query string `form:"q" binding:"required"` // 搜索关键词
	Types string `form:"types"`                // 搜索类型过滤，逗号分隔，如 "fund,stock"
	Page  int    `form:"page"`
	Size  int    `form:"size"`
}

// Defaults 设置请求参数的默认值，确保 Page 和 Size 在合理范围内。
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

// IncludeFunds 判断搜索结果是否需要包含基金。
func (r *UnifiedSearchRequest) IncludeFunds() bool {
	return r.Types == "" || typesContains(r.Types, "fund")
}

// IncludeStocks 判断搜索结果是否需要包含股票。
func (r *UnifiedSearchRequest) IncludeStocks() bool {
	return r.Types == "" || typesContains(r.Types, "stock")
}

// UnifiedSearchResponse 表示统一搜索的响应结果，包含基金和股票的搜索数据。
type UnifiedSearchResponse struct {
	Query       string                      `json:"query"`
	Funds       funddomain.FundSearchData   `json:"funds"`
	Stocks      stockdomain.StockSearchData `json:"stocks"`
	Suggestions []string                    `json:"suggestions,omitempty"` // 搜索建议
}

// typesContains 判断逗号分隔的类型字符串中是否包含指定子类型。
func typesContains(s, sub string) bool {
	for _, part := range strings.Split(s, ",") {
		if strings.TrimSpace(part) == sub {
			return true
		}
	}
	return false
}
