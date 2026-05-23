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

func TestRankingUsesAllFundsBeforeLimiting(t *testing.T) {
	funds := make([]dto.FundItem, 60)
	for i := range funds {
		funds[i] = dto.FundItem{
			FundCode:  fmt.Sprintf("%06d", i),
			FundName:  fmt.Sprintf("Fund %02d", i),
			FundType:  "混合型",
			ChangePct: float64(i) - 30,
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
