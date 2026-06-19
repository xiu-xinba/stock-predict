package providers

import (
	"sort"
	"strings"

	stockdomain "stock-predict-go/internal/domain/stock"
)

// stockMatchesKeyword 检查股票的任意字段是否包含关键词。
// search_service.go 必须调用此函数而非重复逻辑；
// 此处新增的可搜索字段也必须同步到 stockSearchRelevance。
func stockMatchesKeyword(stock stockdomain.StockItem, keyword string) bool {
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

// sortStockItems 对股票列表排序，支持按涨跌幅、价格、成交量等字段排序，默认按搜索相关度排序。
func sortStockItems(items []stockdomain.StockItem, sortBy, sortOrder, keyword string) {
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

// stockSearchRelevance 返回搜索相关度评分，用于排序搜索结果。
// 与 stockMatchesKeyword 保持同步：新增可搜索字段必须在此处评分。
func stockSearchRelevance(stock stockdomain.StockItem, keyword string) int {
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
