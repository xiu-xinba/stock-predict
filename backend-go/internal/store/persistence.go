package store

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"stock-predict-go/internal/dto"
)

var ErrNoSyncSource = errors.New("fund sync source is required")

type fundRecord struct {
	FundCode      string  `json:"fund_code"`
	FundName      string  `json:"fund_name"`
	FundType      string  `json:"fund_type"`
	PinyinAbbr    string  `json:"pinyin_abbr,omitempty"`
	PinyinFull    string  `json:"pinyin_full,omitempty"`
	Company       string  `json:"company,omitempty"`
	Manager       string  `json:"manager,omitempty"`
	LatestNAV     float64 `json:"latest_nav"`
	CumulativeNAV float64 `json:"cumulative_nav"`
	Return1M      float64 `json:"return_1m"`
	Return3M      float64 `json:"return_3m"`
	Return6M      float64 `json:"return_6m"`
	Return1Y      float64 `json:"return_1y"`
	Return3Y      float64 `json:"return_3y"`
	RiskLevel     string  `json:"risk_level,omitempty"`
	InceptionDate string  `json:"inception_date,omitempty"`
	EstimatedNAV  float64 `json:"estimated_nav"`
	ChangePct     float64 `json:"change_pct"`
	QuoteDate     string  `json:"quote_date,omitempty"`
	QuoteSource   string  `json:"quote_source,omitempty"`
}

var eastmoneyFundCodePattern = regexp.MustCompile(`\["([^"]*)","([^"]*)","([^"]*)","([^"]*)","([^"]*)"\]`)

func NewPersistentStore(path string) (*MemoryStore, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return NewMemoryStore(), nil
	}
	items, err := readFundsJSON(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		store := NewMemoryStore()
		store.path = path
		if err := store.saveLocked(); err != nil {
			return nil, err
		}
		return store, nil
	}
	store := NewMemoryStoreWithFunds(mergeSeedFunds(items))
	store.path = path
	return store, nil
}

func (s *MemoryStore) ReplaceFunds(funds []dto.FundItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	next := make(map[string]dto.FundItem, len(funds))
	for _, fund := range funds {
		fund.FundCode = normalizeFundCode(fund.FundCode)
		if fund.FundCode == "" {
			continue
		}
		next[fund.FundCode] = fund
	}
	if len(next) == 0 {
		return fmt.Errorf("no valid funds to store")
	}
	s.funds = next
	return s.saveLocked()
}

func (s *MemoryStore) MergeFunds(funds []dto.FundItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	valid := 0
	for _, fund := range funds {
		fund.FundCode = normalizeFundCode(fund.FundCode)
		if fund.FundCode == "" {
			continue
		}
		if existing, ok := s.funds[fund.FundCode]; ok {
			fund = mergeFund(existing, fund)
		}
		s.funds[fund.FundCode] = fund
		valid++
	}
	if valid == 0 {
		return fmt.Errorf("no valid funds to store")
	}
	return s.saveLocked()
}

func (s *MemoryStore) MergeFundUniverse(funds []dto.FundItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	valid := 0
	for _, fund := range funds {
		fund.FundCode = normalizeFundCode(fund.FundCode)
		if fund.FundCode == "" {
			continue
		}
		if existing, ok := s.funds[fund.FundCode]; ok {
			fund = mergeFund(existing, fund)
			if existing.QuoteSource == "" {
				fund = clearQuoteFields(fund)
			}
		}
		s.funds[fund.FundCode] = fund
		valid++
	}
	if valid == 0 {
		return fmt.Errorf("no valid funds to store")
	}
	return s.saveLocked()
}

func (s *MemoryStore) SyncFundsFromCSV(path string) (int, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return 0, ErrNoSyncSource
	}
	funds, err := ReadFundsCSV(path)
	if err != nil {
		return 0, err
	}
	if err := s.MergeFunds(funds); err != nil {
		return 0, err
	}
	return len(funds), nil
}

func (s *MemoryStore) SyncFundsFromEastmoneyURL(sourceURL string) (int, error) {
	return s.SyncFundsFromEastmoneySources(sourceURL, "")
}

func (s *MemoryStore) SyncFundsFromEastmoneySources(sourceURL, metricsURL string) (int, error) {
	sourceURL = strings.TrimSpace(sourceURL)
	metricsURL = strings.TrimSpace(metricsURL)
	if sourceURL == "" && metricsURL == "" {
		return 0, ErrNoSyncSource
	}
	imported := 0
	if sourceURL != "" {
		payload, err := fetchEastmoneyPayload(sourceURL, 10<<20)
		if err != nil {
			return imported, fmt.Errorf("fund universe request failed: %w", err)
		}
		funds, err := ReadEastmoneyFundCodeSearchJS(payload)
		if err != nil {
			return imported, err
		}
		if err := s.MergeFundUniverse(funds); err != nil {
			return imported, err
		}
		imported += len(funds)
	}
	if metricsURL != "" {
		payload, err := fetchEastmoneyPayload(metricsURL, 50<<20)
		if err != nil {
			return imported, fmt.Errorf("fund metrics request failed: %w", err)
		}
		funds, err := ReadEastmoneyFundRankHandlerJS(payload)
		if err != nil {
			return imported, err
		}
		if err := s.MergeFunds(funds); err != nil {
			return imported, err
		}
		imported += len(funds)
	}
	return imported, nil
}

func (s *MemoryStore) DataPath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.path
}

func (s *MemoryStore) saveLocked() error {
	if strings.TrimSpace(s.path) == "" {
		return nil
	}
	items := make([]dto.FundItem, 0, len(s.funds))
	for _, fund := range s.funds {
		items = append(items, fund)
	}
	return writeFundsJSON(s.path, items)
}

func ReadFundsCSV(path string) ([]dto.FundItem, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return nil, err
	}
	indexes := headerIndexes(header)
	var funds []dto.FundItem
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		fund := dto.FundItem{
			FundCode:      normalizeFundCode(csvValue(record, indexes, "fund_code", "code", "symbol", "基金代码")),
			FundName:      csvValue(record, indexes, "fund_name", "name", "基金名称"),
			FundType:      csvValue(record, indexes, "fund_type", "type", "基金类型"),
			PinyinAbbr:    strings.ToUpper(csvValue(record, indexes, "pinyin_abbr", "spell", "拼音缩写")),
			PinyinFull:    strings.ToUpper(csvValue(record, indexes, "pinyin_full", "full_spell", "拼音全称")),
			Company:       csvValue(record, indexes, "company", "fund_company", "基金公司"),
			Manager:       csvValue(record, indexes, "manager", "fund_manager", "基金经理"),
			RiskLevel:     csvValue(record, indexes, "risk_level", "risk", "风险等级"),
			InceptionDate: csvValue(record, indexes, "inception_date", "成立日期"),
			LatestNAV:     csvFloat(record, indexes, "latest_nav", "nav", "unit_nav", "close", "净值"),
			CumulativeNAV: csvFloat(record, indexes, "cumulative_nav", "acc_nav", "累计净值"),
			Return1M:      csvFloat(record, indexes, "return_1m", "ret_1m", "近1月"),
			Return3M:      csvFloat(record, indexes, "return_3m", "ret_3m", "近3月"),
			Return6M:      csvFloat(record, indexes, "return_6m", "ret_6m", "近6月"),
			Return1Y:      csvFloat(record, indexes, "return_1y", "ret_1y", "近1年"),
			Return3Y:      csvFloat(record, indexes, "return_3y", "ret_3y", "近3年"),
			EstimatedNAV:  csvFloat(record, indexes, "estimated_nav", "估算净值"),
			ChangePct:     csvFloat(record, indexes, "change_pct", "daily_change_pct", "涨跌幅"),
		}
		if hasQuoteFields(fund) {
			fund.QuoteSource = "csv"
		}
		if fund.FundCode == "" {
			continue
		}
		if fund.FundName == "" {
			fund.FundName = fund.FundCode
		}
		if fund.FundType == "" {
			fund.FundType = "未知"
		}
		funds = append(funds, fund)
	}
	if len(funds) == 0 {
		return nil, fmt.Errorf("no valid fund rows in %s", path)
	}
	return funds, nil
}

func ReadEastmoneyFundRankHandlerJS(payload []byte) ([]dto.FundItem, error) {
	text := strings.TrimPrefix(string(payload), "\ufeff")
	start := strings.Index(text, "datas:[")
	if start < 0 {
		return nil, fmt.Errorf("no fund rows found in eastmoney rank handler payload")
	}
	start += len("datas:[")
	end := strings.Index(text[start:], "],allRecords")
	if end < 0 {
		end = strings.Index(text[start:], "],pageIndex")
	}
	if end < 0 {
		return nil, fmt.Errorf("no fund rows found in eastmoney rank handler payload")
	}
	rawRows := text[start : start+end]
	if strings.TrimSpace(rawRows) == "" {
		return nil, fmt.Errorf("no fund rows found in eastmoney rank handler payload")
	}
	rowsReader := csv.NewReader(strings.NewReader(rawRows))
	rowsReader.FieldsPerRecord = -1
	rows, err := rowsReader.Read()
	if err != nil {
		return nil, err
	}
	funds := make([]dto.FundItem, 0, len(rows))
	for _, row := range rows {
		colsReader := csv.NewReader(strings.NewReader(row))
		colsReader.FieldsPerRecord = -1
		cols, err := colsReader.Read()
		if err != nil || len(cols) < 17 {
			continue
		}
		code := normalizeFundCode(cols[0])
		if code == "" {
			continue
		}
		latestNAV := parseEastmoneyFloat(cols[4])
		funds = append(funds, dto.FundItem{
			FundCode:      code,
			FundName:      strings.TrimSpace(cols[1]),
			PinyinAbbr:    strings.ToUpper(strings.TrimSpace(cols[2])),
			QuoteDate:     strings.TrimSpace(cols[3]),
			LatestNAV:     latestNAV,
			CumulativeNAV: parseEastmoneyFloat(cols[5]),
			ChangePct:     parseEastmoneyFloat(cols[6]),
			Return1M:      parseEastmoneyFloat(cols[8]),
			Return3M:      parseEastmoneyFloat(cols[9]),
			Return6M:      parseEastmoneyFloat(cols[10]),
			Return1Y:      parseEastmoneyFloat(cols[11]),
			Return3Y:      parseEastmoneyFloat(cols[13]),
			InceptionDate: strings.TrimSpace(cols[16]),
			EstimatedNAV:  latestNAV,
			QuoteSource:   "eastmoney_rank",
		})
	}
	if len(funds) == 0 {
		return nil, fmt.Errorf("no valid fund rows found in eastmoney rank handler payload")
	}
	return funds, nil
}

func ReadEastmoneyFundCodeSearchJS(payload []byte) ([]dto.FundItem, error) {
	text := strings.TrimPrefix(string(payload), "\ufeff")
	matches := eastmoneyFundCodePattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no fund rows found in eastmoney fund code search payload")
	}
	funds := make([]dto.FundItem, 0, len(matches))
	for _, match := range matches {
		if len(match) != 6 {
			continue
		}
		code := normalizeFundCode(match[1])
		if code == "" {
			continue
		}
		name := strings.TrimSpace(match[3])
		if name == "" {
			name = code
		}
		fundType := strings.TrimSpace(match[4])
		if fundType == "" {
			fundType = "未知"
		}
		funds = append(funds, dto.FundItem{
			FundCode:   code,
			FundName:   name,
			FundType:   fundType,
			PinyinAbbr: strings.ToUpper(strings.TrimSpace(match[2])),
			PinyinFull: strings.ToUpper(strings.TrimSpace(match[5])),
		})
	}
	if len(funds) == 0 {
		return nil, fmt.Errorf("no valid fund rows found in eastmoney fund code search payload")
	}
	return funds, nil
}

func mergeSeedFunds(funds []dto.FundItem) []dto.FundItem {
	merged := make(map[string]dto.FundItem, len(seedFunds())+len(funds))
	for _, fund := range seedFunds() {
		merged[fund.FundCode] = fund
	}
	for _, fund := range funds {
		if existing, ok := merged[fund.FundCode]; ok {
			fund = mergeFund(existing, fund)
		}
		merged[fund.FundCode] = fund
	}
	out := make([]dto.FundItem, 0, len(merged))
	for _, fund := range merged {
		out = append(out, fund)
	}
	return out
}

func mergeFund(existing, incoming dto.FundItem) dto.FundItem {
	if incoming.FundCode == "" {
		incoming.FundCode = existing.FundCode
	}
	if incoming.FundName == "" {
		incoming.FundName = existing.FundName
	}
	if incoming.FundType == "" || incoming.FundType == "未知" {
		incoming.FundType = existing.FundType
	}
	if incoming.PinyinAbbr == "" {
		incoming.PinyinAbbr = existing.PinyinAbbr
	}
	if incoming.PinyinFull == "" {
		incoming.PinyinFull = existing.PinyinFull
	}
	if incoming.Company == "" {
		incoming.Company = existing.Company
	}
	if incoming.Manager == "" {
		incoming.Manager = existing.Manager
	}
	if incoming.QuoteSource == "" {
		if incoming.LatestNAV == 0 {
			incoming.LatestNAV = existing.LatestNAV
		}
		if incoming.CumulativeNAV == 0 {
			incoming.CumulativeNAV = existing.CumulativeNAV
		}
		if incoming.Return1M == 0 {
			incoming.Return1M = existing.Return1M
		}
		if incoming.Return3M == 0 {
			incoming.Return3M = existing.Return3M
		}
		if incoming.Return6M == 0 {
			incoming.Return6M = existing.Return6M
		}
		if incoming.Return1Y == 0 {
			incoming.Return1Y = existing.Return1Y
		}
		if incoming.Return3Y == 0 {
			incoming.Return3Y = existing.Return3Y
		}
	}
	if incoming.RiskLevel == "" {
		incoming.RiskLevel = existing.RiskLevel
	}
	if incoming.InceptionDate == "" {
		incoming.InceptionDate = existing.InceptionDate
	}
	if incoming.QuoteSource == "" {
		if incoming.EstimatedNAV == 0 {
			incoming.EstimatedNAV = existing.EstimatedNAV
		}
		if incoming.ChangePct == 0 {
			incoming.ChangePct = existing.ChangePct
		}
		if incoming.QuoteDate == "" {
			incoming.QuoteDate = existing.QuoteDate
		}
		incoming.QuoteSource = existing.QuoteSource
	}
	return incoming
}

func readFundsJSON(path string) ([]dto.FundItem, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var records []fundRecord
	if err := json.NewDecoder(file).Decode(&records); err != nil {
		return nil, err
	}
	out := make([]dto.FundItem, 0, len(records))
	for _, record := range records {
		out = append(out, record.toFundItem())
	}
	return out, nil
}

func writeFundsJSON(path string, funds []dto.FundItem) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	records := make([]fundRecord, 0, len(funds))
	for _, fund := range funds {
		records = append(records, newFundRecord(fund))
	}
	payload, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	return os.WriteFile(path, payload, 0o644)
}

func newFundRecord(f dto.FundItem) fundRecord {
	return fundRecord{
		FundCode:      f.FundCode,
		FundName:      f.FundName,
		FundType:      f.FundType,
		PinyinAbbr:    f.PinyinAbbr,
		PinyinFull:    f.PinyinFull,
		Company:       f.Company,
		Manager:       f.Manager,
		LatestNAV:     f.LatestNAV,
		CumulativeNAV: f.CumulativeNAV,
		Return1M:      f.Return1M,
		Return3M:      f.Return3M,
		Return6M:      f.Return6M,
		Return1Y:      f.Return1Y,
		Return3Y:      f.Return3Y,
		RiskLevel:     f.RiskLevel,
		InceptionDate: f.InceptionDate,
		EstimatedNAV:  f.EstimatedNAV,
		ChangePct:     f.ChangePct,
		QuoteDate:     f.QuoteDate,
		QuoteSource:   f.QuoteSource,
	}
}

func (r fundRecord) toFundItem() dto.FundItem {
	return dto.FundItem{
		FundCode:      r.FundCode,
		FundName:      r.FundName,
		FundType:      r.FundType,
		PinyinAbbr:    r.PinyinAbbr,
		PinyinFull:    r.PinyinFull,
		Company:       r.Company,
		Manager:       r.Manager,
		LatestNAV:     r.LatestNAV,
		CumulativeNAV: r.CumulativeNAV,
		Return1M:      r.Return1M,
		Return3M:      r.Return3M,
		Return6M:      r.Return6M,
		Return1Y:      r.Return1Y,
		Return3Y:      r.Return3Y,
		RiskLevel:     r.RiskLevel,
		InceptionDate: r.InceptionDate,
		EstimatedNAV:  r.EstimatedNAV,
		ChangePct:     r.ChangePct,
		QuoteDate:     r.QuoteDate,
		QuoteSource:   r.QuoteSource,
	}
}

func fetchEastmoneyPayload(sourceURL string, maxBytes int64) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodGet, sourceURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://fund.eastmoney.com/")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s", resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxBytes))
}

func clearQuoteFields(f dto.FundItem) dto.FundItem {
	f.LatestNAV = 0
	f.CumulativeNAV = 0
	f.Return1M = 0
	f.Return3M = 0
	f.Return6M = 0
	f.Return1Y = 0
	f.Return3Y = 0
	f.EstimatedNAV = 0
	f.ChangePct = 0
	f.QuoteDate = ""
	f.QuoteSource = ""
	return f
}

func hasQuoteFields(f dto.FundItem) bool {
	return f.LatestNAV != 0 ||
		f.CumulativeNAV != 0 ||
		f.Return1M != 0 ||
		f.Return3M != 0 ||
		f.Return6M != 0 ||
		f.Return1Y != 0 ||
		f.Return3Y != 0 ||
		f.EstimatedNAV != 0 ||
		f.ChangePct != 0
}

func parseEastmoneyFloat(raw string) float64 {
	raw = strings.TrimSpace(strings.TrimSuffix(strings.ReplaceAll(raw, ",", ""), "%"))
	if raw == "" || raw == "--" || raw == "---" {
		return 0
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}
	return value
}

func headerIndexes(header []string) map[string]int {
	indexes := make(map[string]int, len(header))
	for idx, name := range header {
		indexes[normalizeHeader(name)] = idx
	}
	return indexes
}

func csvValue(record []string, indexes map[string]int, aliases ...string) string {
	for _, alias := range aliases {
		idx, ok := indexes[normalizeHeader(alias)]
		if ok && idx < len(record) {
			return strings.TrimSpace(record[idx])
		}
	}
	return ""
}

func csvFloat(record []string, indexes map[string]int, aliases ...string) float64 {
	raw := csvValue(record, indexes, aliases...)
	raw = strings.TrimSpace(strings.TrimSuffix(strings.ReplaceAll(raw, ",", ""), "%"))
	if raw == "" {
		return 0
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}
	return value
}

func normalizeHeader(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.TrimPrefix(value, "\ufeff")
	value = strings.ReplaceAll(value, " ", "_")
	value = strings.ReplaceAll(value, "-", "_")
	return value
}

func normalizeFundCode(value string) string {
	value = strings.TrimSpace(value)
	if len(value) == 6 && allDigits(value) {
		return value
	}
	var digits strings.Builder
	for _, r := range value {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	code := digits.String()
	if len(code) < 6 {
		return ""
	}
	return code[:6]
}

func allDigits(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
