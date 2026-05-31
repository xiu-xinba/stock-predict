package data

import (
	_ "embed"

	"stock-predict-go/internal/dto"
	"encoding/json"
)

//go:embed default_stocks.json
var defaultStocksJSON []byte

func LoadDefaultStocks() []dto.StockItem {
	var items []dto.StockItem
	if err := json.Unmarshal(defaultStocksJSON, &items); err != nil {
		return nil
	}
	return items
}
