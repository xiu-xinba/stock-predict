package database

import (
	"testing"

	stockdomain "stock-predict-go/internal/domain/stock"
)

func TestStockStoreReplaceAndList(t *testing.T) {
	db := InitTestDB(t)
	s := NewStockStore(db)

	stocks := []stockdomain.StockItem{
		{StockCode: "600519", StockName: "贵州茅台"},
		{StockCode: "000858", StockName: "五粮液"},
	}
	if err := s.ReplaceStocks(stocks); err != nil {
		t.Fatalf("replace stocks: %v", err)
	}
	if s.CountStocks() != 2 {
		t.Fatalf("expected 2 stocks, got %d", s.CountStocks())
	}
}

func TestListStocksReturnsSorted(t *testing.T) {
	db := InitTestDB(t)
	s := NewStockStore(db)

	stocks := []stockdomain.StockItem{
		{StockCode: "600519", StockName: "贵州茅台"},
		{StockCode: "000858", StockName: "五粮液"},
		{StockCode: "300750", StockName: "宁德时代"},
	}
	if err := s.ReplaceStocks(stocks); err != nil {
		t.Fatalf("replace stocks: %v", err)
	}
	list := s.ListStocks()
	if len(list) != 3 {
		t.Fatalf("expected 3 stocks, got %d", len(list))
	}
	for i := 1; i < len(list); i++ {
		if list[i].StockCode < list[i-1].StockCode {
			t.Fatalf("stocks not sorted: %q before %q", list[i-1].StockCode, list[i].StockCode)
		}
	}
}

func TestFindStockExists(t *testing.T) {
	db := InitTestDB(t)
	s := NewStockStore(db)

	stocks := []stockdomain.StockItem{
		{StockCode: "600519", StockName: "贵州茅台"},
	}
	if err := s.ReplaceStocks(stocks); err != nil {
		t.Fatalf("replace stocks: %v", err)
	}
	stock, ok := s.FindStock("600519")
	if !ok {
		t.Fatalf("expected to find stock 600519")
	}
	if stock.StockName != "贵州茅台" {
		t.Fatalf("expected name '贵州茅台', got %q", stock.StockName)
	}
}

func TestFindStockNotExists(t *testing.T) {
	db := InitTestDB(t)
	s := NewStockStore(db)

	stocks := []stockdomain.StockItem{
		{StockCode: "600519", StockName: "贵州茅台"},
	}
	if err := s.ReplaceStocks(stocks); err != nil {
		t.Fatalf("replace stocks: %v", err)
	}
	_, ok := s.FindStock("999999")
	if ok {
		t.Fatalf("expected not to find stock 999999")
	}
}

func TestCountStocks(t *testing.T) {
	db := InitTestDB(t)
	s := NewStockStore(db)

	stocks := []stockdomain.StockItem{
		{StockCode: "600519", StockName: "贵州茅台"},
		{StockCode: "000858", StockName: "五粮液"},
	}
	if err := s.ReplaceStocks(stocks); err != nil {
		t.Fatalf("replace stocks: %v", err)
	}
	if s.CountStocks() != 2 {
		t.Fatalf("expected 2, got %d", s.CountStocks())
	}
}

func TestReplaceStocks(t *testing.T) {
	db := InitTestDB(t)
	s := NewStockStore(db)

	if err := s.ReplaceStocks([]stockdomain.StockItem{
		{StockCode: "600519", StockName: "贵州茅台"},
	}); err != nil {
		t.Fatalf("replace stocks: %v", err)
	}

	newStocks := []stockdomain.StockItem{
		{StockCode: "000858", StockName: "五粮液"},
		{StockCode: "300750", StockName: "宁德时代"},
	}
	if err := s.ReplaceStocks(newStocks); err != nil {
		t.Fatalf("replace stocks: %v", err)
	}
	if s.CountStocks() != 2 {
		t.Fatalf("expected 2 after replace, got %d", s.CountStocks())
	}
	if _, ok := s.FindStock("600519"); ok {
		t.Fatalf("expected old stock to be gone after replace")
	}
	if _, ok := s.FindStock("000858"); !ok {
		t.Fatalf("expected new stock 000858 to exist")
	}
}

func TestIsLoaded(t *testing.T) {
	db := InitTestDB(t)
	s := NewStockStore(db)

	if s.IsLoaded() {
		t.Fatalf("expected IsLoaded=false with no stocks")
	}
	s.ReplaceStocks([]stockdomain.StockItem{{StockCode: "600519", StockName: "贵州茅台"}})
	if !s.IsLoaded() {
		t.Fatalf("expected IsLoaded=true after adding stocks")
	}
}
