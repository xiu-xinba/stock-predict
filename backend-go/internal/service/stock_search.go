package service

import (
	"sort"
	"strings"

	"stock-predict-go/internal/dto"
)

// stockMatchesKeyword checks whether any stock field contains the keyword.
// search_service.go must call this function instead of duplicating the logic;
// any field additions here must be reflected in stockSearchRelevance as well.
func stockMatchesKeyword(stock dto.StockItem, keyword string) bool {
	for _, value := range []string{
		stock.StockCode,
		stock.StockName,
		stock.Pinyin,
		stock.PinyinAlt,
		stock.Industry,
		stock.Market,
	} {
		if strings.Contains(strings.ToLower(value), keyword) {
			return true
		}
	}
	return false
}

func sortStockItems(items []dto.StockItem, sortBy, sortOrder, keyword string) {
	desc := strings.ToLower(sortOrder) != "asc"
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		cmp := 0
		switch sortBy {
		case "change_pct":
			cmp = compareFloat(a.ChangePct, b.ChangePct)
		case "current_price":
			cmp = compareFloat(a.CurrentPrice, b.CurrentPrice)
		case "volume":
			cmp = compareFloat(a.Volume, b.Volume)
		case "amount":
			cmp = compareFloat(a.Amount, b.Amount)
		case "pe_ratio":
			cmp = compareFloat(a.PERatio, b.PERatio)
		case "total_mv":
			cmp = compareFloat(a.TotalMV, b.TotalMV)
		default:
			if keyword != "" {
				as := stockSearchRelevance(a, keyword)
				bs := stockSearchRelevance(b, keyword)
				if as != bs {
					cmp = compareInt(as, bs)
				} else {
					cmp = strings.Compare(a.StockCode, b.StockCode)
				}
			} else {
				cmp = strings.Compare(a.StockCode, b.StockCode)
			}
		}
		if cmp == 0 {
			cmp = strings.Compare(a.StockCode, b.StockCode)
		}
		if desc {
			return cmp > 0
		}
		return cmp < 0
	})
}

// stockSearchRelevance returns a relevance score for sorting search results.
// Keep in sync with stockMatchesKeyword: any new searchable field must be scored here.
func stockSearchRelevance(stock dto.StockItem, keyword string) int {
	code := strings.ToLower(stock.StockCode)
	name := strings.ToLower(stock.StockName)
	pinyin := strings.ToLower(stock.Pinyin)
	pinyinAlt := strings.ToLower(stock.PinyinAlt)
	switch {
	case code == keyword || name == keyword:
		return 0
	case strings.HasPrefix(code, keyword):
		return 1
	case strings.HasPrefix(name, keyword):
		return 2
	case strings.HasPrefix(pinyin, keyword) || strings.HasPrefix(pinyinAlt, keyword):
		return 3
	case strings.Contains(code, keyword):
		return 5
	case strings.Contains(name, keyword):
		return 6
	case strings.Contains(pinyin, keyword) || strings.Contains(pinyinAlt, keyword):
		return 7
	default:
		return 9
	}
}
