package database

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	funddomain "stock-predict-go/internal/domain/fund"
	httpclient "stock-predict-go/internal/platform/httpclient"
)

// ErrNoSyncSource 表示基金同步源未配置时返回的错误
var ErrNoSyncSource = errors.New("fund sync source is required")

var eastmoneyFundCodePattern = regexp.MustCompile(`\["([^"]*)","([^"]*)","([^"]*)","([^"]*)","([^"]*)"\]`)

// ReadFundsCSV 从 CSV 文件读取基金数据并转换为 FundItem 列表。
// 支持多种列名别名（中英文），自动标准化基金代码和拼音字段。
func ReadFundsCSV(path string) ([]funddomain.FundItem, error) {
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
	var funds []funddomain.FundItem
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		fund := funddomain.FundItem{
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

// ReadEastmoneyFundCodeSearchJS 解析东方财富基金代码搜索接口返回的 JS 数据，
// 提取基金代码、名称、类型及拼音信息。
func ReadEastmoneyFundCodeSearchJS(payload []byte) ([]funddomain.FundItem, error) {
	text := strings.TrimPrefix(string(payload), "\ufeff")
	matches := eastmoneyFundCodePattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no fund rows found in eastmoney fund code search payload")
	}
	funds := make([]funddomain.FundItem, 0, len(matches))
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
		funds = append(funds, funddomain.FundItem{
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

// ReadEastmoneyFundRankHandlerJS 解析东方财富基金排名接口返回的 JS 数据，
// 提取基金代码、名称、净值、收益率等行情指标。
func ReadEastmoneyFundRankHandlerJS(payload []byte) ([]funddomain.FundItem, error) {
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
	funds := make([]funddomain.FundItem, 0, len(rows))
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
		latestNAV := httpclient.ParseQuoteFloat(cols[4])
		funds = append(funds, funddomain.FundItem{
			FundCode:      code,
			FundName:      strings.TrimSpace(cols[1]),
			PinyinAbbr:    strings.ToUpper(strings.TrimSpace(cols[2])),
			QuoteDate:     strings.TrimSpace(cols[3]),
			LatestNAV:     latestNAV,
			CumulativeNAV: httpclient.ParseQuoteFloat(cols[5]),
			ChangePct:     httpclient.ParseQuoteFloat(cols[6]),
			Return1M:      httpclient.ParseQuoteFloat(cols[8]),
			Return3M:      httpclient.ParseQuoteFloat(cols[9]),
			Return6M:      httpclient.ParseQuoteFloat(cols[10]),
			Return1Y:      httpclient.ParseQuoteFloat(cols[11]),
			Return3Y:      httpclient.ParseQuoteFloat(cols[13]),
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

func fetchEastmoneyPayload(sourceURL string, maxBytes int64) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, sourceURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://fund.eastmoney.com/")
	resilient := httpclient.NewResilientHTTPClient(httpclient.New(httpclient.Config{Timeout: 30 * time.Second}), []httpclient.SourcePolicy{{
		Source:    httpclient.SourceEastmoney,
		UserAgent: "Mozilla/5.0",
		Referer:   "https://fund.eastmoney.com/",
	}})
	resp, err := resilient.Do(req.Context(), httpclient.SourceEastmoney, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s", resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxBytes))
}

func normalizeFundCode(value string) string {
	value = strings.TrimSpace(value)
	if len(value) == 6 && httpclient.IsAllDigits(value) {
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

// mergeFund 将 incoming 基金数据合并到 existing 中，
// 优先保留 incoming 的非零值，缺失字段回退到 existing。
func mergeFund(existing, incoming funddomain.FundItem) funddomain.FundItem {
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

func clearQuoteFields(f funddomain.FundItem) funddomain.FundItem {
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

func hasQuoteFields(f funddomain.FundItem) bool {
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
