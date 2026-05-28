package service

import (
	"fmt"
	"testing"

	"stock-predict-go/internal/dto"
)

type fakeFundRepository struct {
	funds []dto.FundItem
}

func (r fakeFundRepository) ListFunds() []dto.FundItem {
	return append([]dto.FundItem(nil), r.funds...)
}

func (r fakeFundRepository) FindFund(code string) (dto.FundItem, bool) {
	for _, fund := range r.funds {
		if fund.FundCode == code {
			return fund, true
		}
	}
	return dto.FundItem{}, false
}

func (r fakeFundRepository) CountFunds() int {
	return len(r.funds)
}

func TestSearchSupportsFrontendSortOptions(t *testing.T) {
	service := NewFundService(fakeFundRepository{funds: []dto.FundItem{
		{FundCode: "000001", FundName: "A", LatestNAV: 1.25, Return3Y: 12.5, InceptionDate: "2020-01-01"},
		{FundCode: "000002", FundName: "B", LatestNAV: 2.10, Return3Y: -3.2, InceptionDate: "2022-01-01"},
		{FundCode: "000003", FundName: "C", LatestNAV: 0.90, Return3Y: 30.1, InceptionDate: "2019-01-01"},
	}})

	tests := []struct {
		name      string
		sortBy    string
		sortOrder string
		wantFirst string
	}{
		{name: "return 3y desc", sortBy: "return_3y", sortOrder: "desc", wantFirst: "000003"},
		{name: "latest nav asc", sortBy: "latest_nav", sortOrder: "asc", wantFirst: "000003"},
		{name: "inception date desc", sortBy: "inception_date", sortOrder: "desc", wantFirst: "000002"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.Search(dto.FundSearchRequest{
				Page:      1,
				Size:      10,
				SortBy:    tt.sortBy,
				SortOrder: tt.sortOrder,
			})
			if len(got.Items) == 0 {
				t.Fatal("expected search results")
			}
			if got.Items[0].FundCode != tt.wantFirst {
				t.Fatalf("expected first fund %s, got %s", tt.wantFirst, got.Items[0].FundCode)
			}
		})
	}
}

func TestSearchDefaultsToFrontendPageSize(t *testing.T) {
	service := NewFundService(fakeFundRepository{funds: []dto.FundItem{
		{FundCode: "000001", FundName: "华夏成长混合"},
		{FundCode: "000011", FundName: "华夏大盘精选混合"},
	}})

	got := service.Search(dto.FundSearchRequest{Keyword: "华夏"})

	if got.Size != 20 {
		t.Fatalf("expected default search size 20, got %d", got.Size)
	}
	if len(got.Items) != 2 || got.Total != 2 {
		t.Fatalf("expected both matching funds, got total=%d items=%+v", got.Total, got.Items)
	}
}

func TestSearchRanksNamePrefixAboveNameContains(t *testing.T) {
	service := NewFundService(fakeFundRepository{funds: []dto.FundItem{
		{FundCode: "000051", FundName: "华夏沪深300ETF联接"},
		{FundCode: "510300", FundName: "沪深300ETF"},
	}})

	got := service.Search(dto.FundSearchRequest{Keyword: "沪深300", Size: 20})

	if len(got.Items) != 2 {
		t.Fatalf("expected two matching funds, got %+v", got.Items)
	}
	if got.Items[0].FundCode != "510300" {
		t.Fatalf("expected name-prefix match 510300 first, got %+v", got.Items)
	}
}

func TestSearchRanksShorterNameWhenRelevanceTies(t *testing.T) {
	service := NewFundService(fakeFundRepository{funds: []dto.FundItem{
		{FundCode: "159238", FundName: "沪深300增强ETF景顺"},
		{FundCode: "510300", FundName: "沪深300ETF"},
	}})

	got := service.Search(dto.FundSearchRequest{Keyword: "沪深300", Size: 20})

	if len(got.Items) != 2 {
		t.Fatalf("expected two matching funds, got %+v", got.Items)
	}
	if got.Items[0].FundCode != "510300" {
		t.Fatalf("expected shorter equally relevant name 510300 first, got %+v", got.Items)
	}
}

func TestSearchMatchesPinyinAliases(t *testing.T) {
	service := NewFundService(fakeFundRepository{funds: []dto.FundItem{
		{FundCode: "000001", FundName: "华夏成长混合", PinyinAbbr: "HXCZHH", PinyinFull: "HUAXIACHENGZHANGHUNHE"},
	}})

	got := service.Search(dto.FundSearchRequest{Keyword: "hxcz", Size: 20})

	if len(got.Items) != 1 || got.Items[0].FundCode != "000001" {
		t.Fatalf("expected pinyin search to find 000001, got %+v", got.Items)
	}
}

func TestRankingUsesAllFundsBeforeLimiting(t *testing.T) {
	funds := make([]dto.FundItem, 60)
	for i := range funds {
		funds[i] = dto.FundItem{
			FundCode:    fmt.Sprintf("%06d", i),
			FundName:    fmt.Sprintf("Fund %02d", i),
			FundType:    "混合型",
			LatestNAV:   1,
			ChangePct:   float64(i) - 30,
			QuoteSource: "test",
		}
	}
	service := NewFundService(fakeFundRepository{funds: funds})

	got, err := service.Ranking("losers", 3)
	if err != nil {
		t.Fatalf("ranking failed: %v", err)
	}
	want := []string{"000000", "000001", "000002"}
	if len(got) != len(want) {
		t.Fatalf("expected %d ranking items, got %d", len(want), len(got))
	}
	for i, code := range want {
		if got[i].FundCode != code {
			t.Fatalf("rank %d: expected %s, got %s", i+1, code, got[i].FundCode)
		}
		if got[i].Rank != i+1 {
			t.Fatalf("rank %d: expected rank field %d, got %d", i+1, i+1, got[i].Rank)
		}
	}
}

func TestRankingSkipsFundsWithoutQuoteSource(t *testing.T) {
	service := NewFundService(fakeFundRepository{funds: []dto.FundItem{
		{FundCode: "000001", FundName: "metadata only", FundType: "混合型", LatestNAV: 1.2, ChangePct: 2.49},
		{FundCode: "000002", FundName: "real quote", FundType: "混合型", LatestNAV: 1.3, ChangePct: -1.20, QuoteSource: "eastmoney_rank"},
	}})

	got, err := service.Ranking("losers", 10)
	if err != nil {
		t.Fatalf("ranking failed: %v", err)
	}

	if len(got) != 1 || got[0].FundCode != "000002" {
		t.Fatalf("expected ranking to exclude untrusted quote rows, got %+v", got)
	}
}
