package dto

type MarketRankingPath struct {
	Type string `uri:"type"`
}

type MarketRankingQuery struct {
	Size int `form:"size"`
}

type MarketIndex struct {
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	Market        string    `json:"market"`
	Value         float64   `json:"value"`
	Change        float64   `json:"change"`
	ChangePct     float64   `json:"change_pct"`
	High          float64   `json:"high"`
	Low           float64   `json:"low"`
	PrevClose     float64   `json:"prev_close"`
	Volume        float64   `json:"volume"`
	MiniChartData []float64 `json:"mini_chart_data"`
	UpdateTime    string    `json:"update_time"`
	DataSource    string    `json:"data_source"`
}

type MarketSnapshot struct {
	ShIndex           float64 `json:"sh_index"`
	ShIndexChangePct  float64 `json:"sh_index_change_pct"`
	SzIndex           float64 `json:"sz_index"`
	SzIndexChangePct  float64 `json:"sz_index_change_pct"`
	CybIndex          float64 `json:"cyb_index"`
	CybIndexChangePct float64 `json:"cyb_index_change_pct"`
	UpdateTime        string  `json:"update_time"`
}
