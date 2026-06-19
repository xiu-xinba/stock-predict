package database

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	funddomain "stock-predict-go/internal/domain/fund"
)

func TestFundStoreReplaceAndFind(t *testing.T) {
	db := InitTestDB(t)
	s := NewFundStore(db)

	if err := s.ReplaceFunds([]funddomain.FundItem{{
		FundCode:  "999999",
		FundName:  "测试基金",
		FundType:  "指数型",
		LatestNAV: 1.23,
		ChangePct: 0.45,
	}}); err != nil {
		t.Fatalf("replace funds: %v", err)
	}

	fund, ok := s.FindFund("999999")
	if !ok {
		t.Fatalf("expected fund to be found")
	}
	if fund.FundName != "测试基金" || fund.LatestNAV != 1.23 || fund.ChangePct != 0.45 {
		t.Fatalf("unexpected fund: %+v", fund)
	}
}

func TestFundStoreSeedFunds(t *testing.T) {
	db := InitTestDB(t)
	s := NewFundStore(db)

	// Seed funds should be inserted
	if err := SeedFunds(db); err != nil {
		t.Fatalf("seed funds: %v", err)
	}

	if _, ok := s.FindFund("000001"); !ok {
		t.Fatalf("expected seed fund 000001 to be found")
	}
}

func TestSyncFundsFromCSVMerges(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "funds.csv")
	csvContent := "fund_code,fund_name,fund_type,company,latest_nav,return_1y,change_pct\n" +
		"510300,沪深300ETF,指数型,华泰柏瑞基金,4.12,8.5,0.31\n" +
		"sh.159915,创业板ETF,指数型,易方达基金,2.56,12.2,-0.18\n"
	if err := os.WriteFile(csvPath, []byte(csvContent), 0o644); err != nil {
		t.Fatalf("write csv: %v", err)
	}

	db := InitTestDB(t)
	s := NewFundStore(db)

	imported, err := s.SyncFundsFromCSV(csvPath)
	if err != nil {
		t.Fatalf("sync csv: %v", err)
	}

	if imported != 2 {
		t.Fatalf("expected 2 imported funds, got imported=%d", imported)
	}
	fund, ok := s.FindFund("159915")
	if !ok {
		t.Fatalf("expected normalized fund code 159915")
	}
	if fund.FundName != "创业板ETF" || fund.Company != "易方达基金" {
		t.Fatalf("unexpected synced fund: %+v", fund)
	}
}

func TestReadEastmoneyFundCodeSearchJS(t *testing.T) {
	payload := "\ufeffvar r = [[\"000001\",\"HXCZHH\",\"华夏成长混合\",\"混合型-灵活\",\"HUAXIACHENGZHANGHUNHE\"],[\"110011\",\"YFDZXPHH\",\"易方达中小盘混合\",\"混合型-偏股\",\"YIFANGDAZHONGXIAOPANHUNHE\"]];"

	funds, err := ReadEastmoneyFundCodeSearchJS([]byte(payload))
	if err != nil {
		t.Fatalf("parse eastmoney fund list: %v", err)
	}

	if len(funds) != 2 {
		t.Fatalf("expected 2 funds, got %+v", funds)
	}
	if funds[0].FundCode != "000001" || funds[0].FundName != "华夏成长混合" || funds[0].PinyinAbbr != "HXCZHH" || funds[0].PinyinFull != "HUAXIACHENGZHANGHUNHE" {
		t.Fatalf("unexpected first fund: %+v", funds[0])
	}
}

func TestSyncFundsFromEastmoneyURLMergesWithSeeds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		_, _ = w.Write([]byte(`var r = [["003096","ZOYLJKHHA","中欧医疗健康混合A","混合型-偏股","ZHONGOUYILIAOJIANKANGHUNHEA"],["999998","CSJJ","测试基金","指数型","CESHIJIJIN"]];`))
	}))
	defer server.Close()

	db := InitTestDB(t)
	s := NewFundStore(db)

	// Seed first
	SeedFunds(db)

	imported, err := s.SyncFundsFromEastmoneyURL(server.URL)
	if err != nil {
		t.Fatalf("sync eastmoney url: %v", err)
	}

	if imported != 2 {
		t.Fatalf("expected imported=2, got %d", imported)
	}
	if _, ok := s.FindFund("999998"); !ok {
		t.Fatalf("expected remote fund to be present")
	}
	if _, ok := s.FindFund("000001"); !ok {
		t.Fatalf("expected seed fund 000001 to remain present")
	}
	fund, ok := s.FindFund("003096")
	if !ok {
		t.Fatalf("expected seed-backed remote fund 003096")
	}
	if fund.PinyinAbbr != "ZOYLJKHHA" {
		t.Fatalf("expected remote pinyin to be set, got %+v", fund)
	}
}

func TestReadEastmoneyFundRankHandlerJS(t *testing.T) {
	payload := `var rankData = {datas:["000001,华夏成长混合,HXCZHH,2026-05-27,1.333,3.906,-2.27,1.06,15.61,15.81,30.43,65.06,78.88,46.38,23.08,735.22,2001-12-18,1,65.6659,1.50%,0.15%,1,0.15%,1,-3.88"],allRecords:1,pageIndex:1,pageNum:1};`

	funds, err := ReadEastmoneyFundRankHandlerJS([]byte(payload))
	if err != nil {
		t.Fatalf("parse eastmoney rank handler: %v", err)
	}

	if len(funds) != 1 {
		t.Fatalf("expected one fund, got %+v", funds)
	}
	fund := funds[0]
	if fund.FundCode != "000001" || fund.FundName != "华夏成长混合" {
		t.Fatalf("unexpected fund identity: %+v", fund)
	}
	if fund.LatestNAV != 1.333 || fund.CumulativeNAV != 3.906 || fund.EstimatedNAV != 1.333 {
		t.Fatalf("unexpected nav fields: %+v", fund)
	}
	if fund.ChangePct != -2.27 || fund.Return1M != 15.61 || fund.Return1Y != 65.06 || fund.Return3Y != 46.38 {
		t.Fatalf("unexpected return fields: %+v", fund)
	}
	if fund.InceptionDate != "2001-12-18" || fund.QuoteDate != "2026-05-27" || fund.QuoteSource != "eastmoney_rank" {
		t.Fatalf("unexpected quote metadata: %+v", fund)
	}
}

func TestSyncFundsFromEastmoneySourcesMergesRankMetrics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		switch r.URL.Path {
		case "/codes.js":
			_, _ = w.Write([]byte(`var r = [["000001","HXCZHH","华夏成长混合","混合型-灵活","HUAXIACHENGZHANGHUNHE"]];`))
		case "/rank.js":
			_, _ = w.Write([]byte(`var rankData = {datas:["000001,华夏成长混合,HXCZHH,2026-05-27,1.333,3.906,-2.27,1.06,15.61,15.81,30.43,65.06,78.88,46.38,23.08,735.22,2001-12-18,1,65.6659,1.50%,0.15%,1,0.15%,1,-3.88"],allRecords:1,pageIndex:1,pageNum:1};`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	db := InitTestDB(t)
	s := NewFundStore(db)

	// Add initial fund with old quote data
	s.MergeFunds([]funddomain.FundItem{{
		FundCode:  "000001",
		FundName:  "华夏成长混合",
		FundType:  "混合型",
		ChangePct: 2.00,
	}})

	imported, err := s.SyncFundsFromEastmoneySources(server.URL+"/codes.js", server.URL+"/rank.js")
	if err != nil {
		t.Fatalf("sync eastmoney sources: %v", err)
	}
	if imported != 2 {
		t.Fatalf("expected two imported rows across sources, got %d", imported)
	}
	fund, ok := s.FindFund("000001")
	if !ok {
		t.Fatalf("expected fund 000001")
	}
	if fund.ChangePct != -2.27 || fund.LatestNAV != 1.333 || fund.QuoteSource != "eastmoney_rank" {
		t.Fatalf("expected rank metrics to replace old quote fields, got %+v", fund)
	}
	if fund.PinyinAbbr != "HXCZHH" || fund.FundType != "混合型-灵活" {
		t.Fatalf("expected universe metadata to be retained, got %+v", fund)
	}
}

func TestReadFundsCSVHandlesUTF8BOMHeader(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "funds.csv")
	csvContent := "\ufefffund_code,fund_name,fund_type\n510300,沪深300ETF,指数型\n"
	if err := os.WriteFile(csvPath, []byte(csvContent), 0o644); err != nil {
		t.Fatalf("write csv: %v", err)
	}

	funds, err := ReadFundsCSV(csvPath)
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}

	if len(funds) != 1 || funds[0].FundCode != "510300" {
		t.Fatalf("unexpected funds: %+v", funds)
	}
}
