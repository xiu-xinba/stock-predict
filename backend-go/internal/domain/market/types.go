// Package market 定义了市场行情相关的领域模型和数据结构。
package market

// MarketRankingPath 表示市场排行榜的路径参数。
type MarketRankingPath struct {
	Type string `uri:"type"` // 排行类型，如涨幅榜、跌幅榜等
}

// MarketRankingQuery 表示市场排行榜的查询参数。
type MarketRankingQuery struct {
	Size int `form:"size"` // 返回条数
}

// MarketIndex 表示市场指数的行情数据。
type MarketIndex struct {
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	Market        string    `json:"market"`
	Value         float64   `json:"value"`      // 指数点位
	Change        float64   `json:"change"`     // 涨跌点数
	ChangePct     float64   `json:"change_pct"` // 涨跌幅，百分比
	High          float64   `json:"high"`
	Low           float64   `json:"low"`
	PrevClose     float64   `json:"prev_close"` // 昨收点位
	Open          float64   `json:"open"`
	Volume        float64   `json:"volume"`
	MiniChartData []float64 `json:"mini_chart_data"` // 迷你走势图数据
	UpdateTime    string    `json:"update_time"`
	DataSource    string    `json:"data_source"` // 数据来源
	IsClosed      bool      `json:"is_closed"`   // 市场是否已收盘
}

// IndexKlinePoint 表示指数 K 线图中的一个数据点。
type IndexKlinePoint struct {
	Date   string  `json:"date"`
	Open   float64 `json:"open"`
	Close  float64 `json:"close"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Volume int64   `json:"volume"`
	Amount float64 `json:"amount"`
}

// IndexMinutePoint 表示指数分时图中的一个数据点。
type IndexMinutePoint struct {
	Time     string  `json:"time"`
	Price    float64 `json:"price"`     // 分时价格
	AvgPrice float64 `json:"avg_price"` // 均价
	Volume   int64   `json:"volume"`
}

// MarketSectorItem 表示市场板块行情数据。
type MarketSectorItem struct {
	Name      string  `json:"name"`
	ChangePct float64 `json:"change_pct"` // 板块涨跌幅，百分比
	UpCount   int     `json:"up_count"`   // 上涨股票数
	DownCount int     `json:"down_count"` // 下跌股票数
	LeadStock string  `json:"lead_stock"` // 领涨股票
}

// NorthboundFlow 表示北向资金流向数据。
type NorthboundFlow struct {
	SHNetBuy   float64           `json:"sh_net_buy"`    // 沪股通净买入
	SZNetBuy   float64           `json:"sz_net_buy"`    // 深股通净买入
	TotalBuy   float64           `json:"total_net_buy"` // 合计净买入
	Timeline   []NorthboundPoint `json:"timeline"`
	Status     string            `json:"status,omitempty"`      // intraday、daily_fallback、intraday_unavailable
	DataSource string            `json:"data_source,omitempty"` // 数据来源标识
	Notice     string            `json:"notice,omitempty"`      // 数据披露或授权说明
}

// NorthboundPoint 表示北向资金在某个时间点的流向数据。
type NorthboundPoint struct {
	Time   string  `json:"time"`
	SHFlow float64 `json:"sh_flow"` // 沪股通资金流向
	SZFlow float64 `json:"sz_flow"` // 深股通资金流向
}

const (
	NorthboundStatusIntraday            = "intraday"
	NorthboundStatusDailyFallback       = "daily_fallback"
	NorthboundStatusIntradayUnavailable = "intraday_unavailable"
)

const NorthboundIntradayUnavailableNotice = "官方不再公开北向实时分时，未接入授权数据源"

// NewNorthboundUnavailableFlow 返回一个可序列化的北向资金不可用状态对象。
func NewNorthboundUnavailableFlow() *NorthboundFlow {
	return &NorthboundFlow{
		Timeline:   []NorthboundPoint{},
		Status:     NorthboundStatusIntradayUnavailable,
		DataSource: "compliance",
		Notice:     NorthboundIntradayUnavailableNotice,
	}
}

// CacheStat 表示缓存数据的统计信息。
type CacheStat struct {
	Start string `json:"start"` // 缓存数据起始时间
	End   string `json:"end"`   // 缓存数据结束时间
	Count int    `json:"count"` // 缓存记录数
}
