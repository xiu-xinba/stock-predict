// Package seed 提供股票和基金的种子数据，用于数据库初始化。
package seed

import (
	_ "embed"

	"encoding/json"
	stockdomain "stock-predict-go/internal/domain/stock"
)

//go:embed default_stocks.json
var defaultStocksJSON []byte

// LoadDefaultStocks 从嵌入的 default_stocks.json 加载默认股票种子数据
func LoadDefaultStocks() []stockdomain.StockItem {
	var items []stockdomain.StockItem
	if err := json.Unmarshal(defaultStocksJSON, &items); err != nil {
		return nil
	}
	return items
}
