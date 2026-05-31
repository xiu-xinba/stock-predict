package store

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"stock-predict-go/internal/dto"
)

var eastmoneyFundCodePattern = regexp.MustCompile(`\["([^"]*)","([^"]*)","([^"]*)","([^"]*)","([^"]*)"\]`)

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
		latestNAV := parseQuoteFloat(cols[4])
		funds = append(funds, dto.FundItem{
			FundCode:      code,
			FundName:      strings.TrimSpace(cols[1]),
			PinyinAbbr:    strings.ToUpper(strings.TrimSpace(cols[2])),
			QuoteDate:     strings.TrimSpace(cols[3]),
			LatestNAV:     latestNAV,
			CumulativeNAV: parseQuoteFloat(cols[5]),
			ChangePct:     parseQuoteFloat(cols[6]),
			Return1M:      parseQuoteFloat(cols[8]),
			Return3M:      parseQuoteFloat(cols[9]),
			Return6M:      parseQuoteFloat(cols[10]),
			Return1Y:      parseQuoteFloat(cols[11]),
			Return3Y:      parseQuoteFloat(cols[13]),
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
