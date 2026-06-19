package search

import "testing"

func TestUnifiedSearchRequestDefaultsPageAndSize(t *testing.T) {
	r := &UnifiedSearchRequest{}
	r.Defaults()
	if r.Page != 1 {
		t.Fatalf("expected default page 1, got %d", r.Page)
	}
	if r.Size != 10 {
		t.Fatalf("expected default size 10, got %d", r.Size)
	}
}

func TestUnifiedSearchRequestDefaultsSizeCap(t *testing.T) {
	r := &UnifiedSearchRequest{Page: 2, Size: 100}
	r.Defaults()
	if r.Size != 50 {
		t.Fatalf("expected size capped to 50, got %d", r.Size)
	}
	if r.Page != 2 {
		t.Fatalf("expected page to remain 2, got %d", r.Page)
	}
}

func TestIncludeFundsEmptyTypes(t *testing.T) {
	r := &UnifiedSearchRequest{Types: ""}
	if !r.IncludeFunds() {
		t.Fatalf("expected true when Types is empty")
	}
}

func TestIncludeFundsContainsFund(t *testing.T) {
	r := &UnifiedSearchRequest{Types: "fund"}
	if !r.IncludeFunds() {
		t.Fatalf("expected true when Types contains fund")
	}
}

func TestIncludeFundsNotContainsFund(t *testing.T) {
	r := &UnifiedSearchRequest{Types: "stock"}
	if r.IncludeFunds() {
		t.Fatalf("expected false when Types does not contain fund")
	}
}

func TestIncludeStocksEmptyTypes(t *testing.T) {
	r := &UnifiedSearchRequest{Types: ""}
	if !r.IncludeStocks() {
		t.Fatalf("expected true when Types is empty")
	}
}

func TestIncludeStocksContainsStock(t *testing.T) {
	r := &UnifiedSearchRequest{Types: "stock"}
	if !r.IncludeStocks() {
		t.Fatalf("expected true when Types contains stock")
	}
}

func TestIncludeStocksNotContainsStock(t *testing.T) {
	r := &UnifiedSearchRequest{Types: "fund"}
	if r.IncludeStocks() {
		t.Fatalf("expected false when Types does not contain stock")
	}
}
