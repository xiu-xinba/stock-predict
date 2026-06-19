package stock

// Repository 定义了股票领域的数据访问接口。
type Repository interface {
	// ListStocks 返回所有已加载的股票列表。
	ListStocks() []StockItem
	// FindStock 根据股票代码查找股票，第二个返回值表示是否找到。
	FindStock(code string) (StockItem, bool)
	// CountStocks 返回已加载的股票总数。
	CountStocks() int
	// ReplaceStocks 用给定的股票列表替换当前存储的全部股票数据。
	ReplaceStocks(stocks []StockItem) error
	// IsLoaded 判断股票数据是否已加载到内存中。
	IsLoaded() bool
}
