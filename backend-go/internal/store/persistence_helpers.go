package store

import (
	"strconv"
	"strings"

	"stock-predict-go/internal/dto"
	"stock-predict-go/internal/util"
)

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
	if len(value) == 6 && util.IsAllDigits(value) {
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
