package store

import "stock-predict-go/internal/dto"

type FundRepository interface {
	LoadFunds() []dto.FundItem
	SaveFunds(funds []dto.FundItem) error
	GetFunds(codes []string) []dto.FundItem
	AddFund(fund dto.FundItem) error
	RemoveFund(code string) error
	IsFundInWatchlist(code string) bool
	FindFund(code string) (dto.FundItem, bool)
	CountFunds() int
	ListFunds() []dto.FundItem
	CoverageReport() *dto.CoverageReport
}
