package providers

import (
	"math"
	"testing"
	"time"

	marketdomain "stock-predict-go/internal/domain/market"
	stockdomain "stock-predict-go/internal/domain/stock"
)

func TestIsMarketOpenAtUsesExchangeTimeZones(t *testing.T) {
	if !IsMarketOpenAt(MarketUS, time.Date(2026, time.January, 12, 15, 0, 0, 0, time.UTC)) {
		t.Fatal("expected US market open at 10:00 EST")
	}
	if !IsMarketOpenAt(MarketUS, time.Date(2026, time.June, 10, 14, 0, 0, 0, time.UTC)) {
		t.Fatal("expected US market open at 10:00 EDT")
	}
	if IsMarketOpenAt(MarketUS, time.Date(2026, time.June, 13, 14, 0, 0, 0, time.UTC)) {
		t.Fatal("expected US market closed on Saturday")
	}
}

func TestValidateMarketIndexAcceptsConsistentCNQuote(t *testing.T) {
	index := marketdomain.MarketIndex{
		Code:      "000001",
		Name:      "SSE Composite",
		Market:    "sh",
		Value:     3100,
		Change:    31,
		ChangePct: 1.01,
		High:      3120,
		Low:       3060,
		PrevClose: 3069,
		Volume:    100000,
	}

	if !validateMarketIndex(index) {
		t.Fatalf("expected valid index quote to pass validation")
	}
}

func TestValidateMarketIndexRejectsBadPercentageAndNonfiniteValues(t *testing.T) {
	badPct := marketdomain.MarketIndex{
		Code:      "000001",
		Name:      "SSE Composite",
		Market:    "sh",
		Value:     3100,
		Change:    31,
		ChangePct: 9.99,
		High:      3120,
		Low:       3060,
		PrevClose: 3069,
	}
	if validateMarketIndex(badPct) {
		t.Fatalf("expected inconsistent change percentage to be rejected")
	}

	nonfinite := badPct
	nonfinite.ChangePct = 1.01
	nonfinite.Value = math.NaN()
	if validateMarketIndex(nonfinite) {
		t.Fatalf("expected nonfinite quote value to be rejected")
	}
}

func TestNormalizeIndexMinutePointsFiltersSortsAndDeduplicates(t *testing.T) {
	points := []marketdomain.IndexMinutePoint{
		{Time: "13:01", Price: 3103, AvgPrice: 3101, Volume: 20},
		{Time: "08:59", Price: 3090, AvgPrice: 3090, Volume: 10},
		{Time: "09:31", Price: 3100, AvgPrice: 3100, Volume: 15},
		{Time: "09:31", Price: 3101, AvgPrice: 3101, Volume: 18},
		{Time: "11:31", Price: 3102, AvgPrice: 3101, Volume: 1},
		{Time: "13:02", Price: math.Inf(1), AvgPrice: 3101, Volume: 3},
	}

	got := normalizeIndexMinutePoints(points)

	// 11:31 is valid for HK market (9:30-12:00), so now 3 points pass
	if len(got) != 3 {
		t.Fatalf("expected 3 valid minute points, got %d: %+v", len(got), got)
	}
	if got[0].Time != "09:31" || got[0].Price != 3101 {
		t.Fatalf("expected latest duplicate 09:31 point first, got %+v", got[0])
	}
	if got[1].Time != "11:31" || got[1].Price != 3102 {
		t.Fatalf("expected 11:31 point second, got %+v", got[1])
	}
	if got[2].Time != "13:01" || got[2].Price != 3103 {
		t.Fatalf("expected sorted afternoon point third, got %+v", got[2])
	}
}

func TestNormalizeIndexKlinePointsFiltersInvalidRows(t *testing.T) {
	points := []marketdomain.IndexKlinePoint{
		{Date: "2026-06-01", Open: 3080, Close: 3100, High: 3110, Low: 3070, Volume: 100, Amount: 200},
		{Date: "bad", Open: 3080, Close: 3100, High: 3110, Low: 3070, Volume: 100, Amount: 200},
		{Date: "2026-05-30", Open: 3080, Close: 3100, High: 3060, Low: 3070, Volume: 100, Amount: 200},
	}

	got := normalizeIndexKlinePoints(points)

	if len(got) != 1 {
		t.Fatalf("expected only one valid K-line point, got %d: %+v", len(got), got)
	}
	if got[0].Date != "2026-06-01" {
		t.Fatalf("unexpected K-line point retained: %+v", got[0])
	}
}

func TestNormalizeStockRankingItemsFiltersSortsLimitsAndRanks(t *testing.T) {
	items := []stockdomain.StockRankingItem{
		{Rank: 9, StockCode: "600002", StockName: "B", ChangePct: 3.2, CurrentPrice: 10},
		{Rank: 8, StockCode: "bad", StockName: "Bad", ChangePct: 9.9, CurrentPrice: 10},
		{Rank: 7, StockCode: "600001", StockName: "A", ChangePct: 5.1, CurrentPrice: 11},
		{Rank: 6, StockCode: "600003", StockName: "C", ChangePct: math.NaN(), CurrentPrice: 12},
	}

	got := normalizeStockRankingItems(items, "gainers", 2)

	if len(got) != 2 {
		t.Fatalf("expected 2 ranking items, got %d: %+v", len(got), got)
	}
	if got[0].Rank != 1 || got[0].StockCode != "600001" {
		t.Fatalf("expected top gainer first with reassigned rank, got %+v", got[0])
	}
	if got[1].Rank != 2 || got[1].StockCode != "600002" {
		t.Fatalf("expected second gainer with reassigned rank, got %+v", got[1])
	}
}

func TestMergeIndexMinutePointsCombinesOldAndNew(t *testing.T) {
	old := []marketdomain.IndexMinutePoint{
		{Time: "09:31", Price: 3100, AvgPrice: 3100, Volume: 10},
		{Time: "09:32", Price: 3101, AvgPrice: 3100.5, Volume: 15},
		{Time: "09:33", Price: 3102, AvgPrice: 3101, Volume: 20},
	}
	new := []marketdomain.IndexMinutePoint{
		{Time: "09:32", Price: 3101.5, AvgPrice: 3100.75, Volume: 18},
		{Time: "09:34", Price: 3103, AvgPrice: 3101.5, Volume: 25},
	}

	got := mergeIndexMinutePoints(old, new)

	if len(got) != 4 {
		t.Fatalf("expected 4 merged points, got %d: %+v", len(got), got)
	}
	if got[1].Time != "09:32" || got[1].Price != 3101.5 {
		t.Fatalf("expected new data to override old at 09:32, got %+v", got[1])
	}
	if got[0].Time != "09:31" || got[0].Price != 3100 {
		t.Fatalf("expected old data preserved at 09:31, got %+v", got[0])
	}
	if got[3].Time != "09:34" {
		t.Fatalf("expected last point at 09:34, got %+v", got[3])
	}
}

func TestMergeIndexMinutePointsEmptyOld(t *testing.T) {
	new := []marketdomain.IndexMinutePoint{
		{Time: "09:31", Price: 3100, AvgPrice: 3100, Volume: 10},
	}
	got := mergeIndexMinutePoints(nil, new)
	if len(got) != 1 {
		t.Fatalf("expected 1 point, got %d", len(got))
	}
}

func TestMergeIndexMinutePointsEmptyNew(t *testing.T) {
	old := []marketdomain.IndexMinutePoint{
		{Time: "09:31", Price: 3100, AvgPrice: 3100, Volume: 10},
	}
	got := mergeIndexMinutePoints(old, nil)
	if len(got) != 1 {
		t.Fatalf("expected 1 point, got %d", len(got))
	}
}

func TestMergeIndexMinutePointsBothEmpty(t *testing.T) {
	got := mergeIndexMinutePoints(nil, nil)
	if len(got) != 0 {
		t.Fatalf("expected 0 points, got %d", len(got))
	}
}

func TestMergeIndexMinutePointsCrossMidnightSort(t *testing.T) {
	old := []marketdomain.IndexMinutePoint{
		{Time: "21:30", Price: 51610, AvgPrice: 51610, Volume: 100},
		{Time: "22:00", Price: 51620, AvgPrice: 51615, Volume: 200},
	}
	new := []marketdomain.IndexMinutePoint{
		{Time: "22:00", Price: 51625, AvgPrice: 51618, Volume: 250},
		{Time: "01:00", Price: 51700, AvgPrice: 51650, Volume: 300},
		{Time: "03:00", Price: 51650, AvgPrice: 51640, Volume: 150},
	}
	got := mergeIndexMinutePoints(old, new)
	if len(got) != 4 {
		t.Fatalf("expected 4 merged points, got %d: %+v", len(got), got)
	}
	// 跨午夜排序：21:30, 22:00, 01:00, 03:00
	if got[0].Time != "21:30" {
		t.Fatalf("expected first point at 21:30, got %s", got[0].Time)
	}
	if got[1].Time != "22:00" {
		t.Fatalf("expected second point at 22:00, got %s", got[1].Time)
	}
	if got[2].Time != "01:00" {
		t.Fatalf("expected third point at 01:00, got %s", got[2].Time)
	}
	if got[3].Time != "03:00" {
		t.Fatalf("expected fourth point at 03:00, got %s", got[3].Time)
	}
	// new data overrides old at 22:00
	if got[1].Price != 51625 {
		t.Fatalf("expected new data to override old at 22:00, got %f", got[1].Price)
	}
}

func TestNormalizeIndexMinutePointsConvertsUSEasternToBeijing(t *testing.T) {
	points := []marketdomain.IndexMinutePoint{
		{Time: "09:30", Price: 51610, AvgPrice: 51610, Volume: 100},
		{Time: "10:00", Price: 51620, AvgPrice: 51615, Volume: 200},
		{Time: "16:00", Price: 51700, AvgPrice: 51650, Volume: 300},
	}
	got := normalizeIndexMinutePoints(points, MarketUS)
	if len(got) != 3 {
		t.Fatalf("expected 3 points, got %d", len(got))
	}
	if got[0].Time != "21:30" {
		t.Fatalf("expected 09:30 converted to 21:30, got %s", got[0].Time)
	}
	if got[1].Time != "22:00" {
		t.Fatalf("expected 10:00 converted to 22:00, got %s", got[1].Time)
	}
	if got[2].Time != "04:00" {
		t.Fatalf("expected 16:00 converted to 04:00, got %s", got[2].Time)
	}
}

func TestNormalizeIndexMinutePointsKeepsBeijingTime(t *testing.T) {
	points := []marketdomain.IndexMinutePoint{
		{Time: "21:30", Price: 51610, AvgPrice: 51610, Volume: 100},
		{Time: "23:00", Price: 51620, AvgPrice: 51615, Volume: 200},
		{Time: "01:00", Price: 51700, AvgPrice: 51650, Volume: 300},
	}
	got := normalizeIndexMinutePoints(points, MarketUSBeijing)
	if len(got) != 3 {
		t.Fatalf("expected 3 points, got %d", len(got))
	}
	if got[0].Time != "21:30" {
		t.Fatalf("expected 21:30 unchanged, got %s", got[0].Time)
	}
	if got[1].Time != "23:00" {
		t.Fatalf("expected 23:00 unchanged, got %s", got[1].Time)
	}
	if got[2].Time != "01:00" {
		t.Fatalf("expected 01:00 unchanged, got %s", got[2].Time)
	}
}

func TestNormalizeIndexMinutePointsCNMarketNoConversion(t *testing.T) {
	points := []marketdomain.IndexMinutePoint{
		{Time: "09:31", Price: 3100, AvgPrice: 3100, Volume: 10},
	}
	got := normalizeIndexMinutePoints(points, MarketCN)
	if len(got) != 1 {
		t.Fatalf("expected 1 point, got %d", len(got))
	}
	if got[0].Time != "09:31" {
		t.Fatalf("expected 09:31 unchanged for CN market, got %s", got[0].Time)
	}
}

func TestConvertUSTimeToBeijing(t *testing.T) {
	tests := []struct{ in, want string }{
		{"09:30", "21:30"},
		{"10:00", "22:00"},
		{"12:00", "00:00"},
		{"15:59", "03:59"},
		{"16:00", "04:00"},
	}
	for _, tt := range tests {
		got := convertUSTimeToBeijing(tt.in)
		if got != tt.want {
			t.Errorf("convertUSTimeToBeijing(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestSortIndicesByOrder(t *testing.T) {
	indices := []marketdomain.MarketIndex{
		{Code: "spx", Name: "标普500"},
		{Code: "000001", Name: "上证指数"},
		{Code: "dji", Name: "道琼斯"},
		{Code: "hsi", Name: "恒生指数"},
		{Code: "399006", Name: "创业板指"},
		{Code: "ixic", Name: "纳斯达克"},
		{Code: "hstech", Name: "恒生科技"},
		{Code: "399001", Name: "深证成指"},
	}
	sortIndicesByOrder(indices)
	if len(indices) != 8 {
		t.Fatalf("expected 8 indices, got %d", len(indices))
	}
	expected := []string{"000001", "399001", "399006", "hsi", "hstech", "dji", "ixic", "spx"}
	for i, idx := range indices {
		if idx.Code != expected[i] {
			t.Errorf("position %d: got %s, want %s", i, idx.Code, expected[i])
		}
	}
}
