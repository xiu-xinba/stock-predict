export interface MarketIndex {
  code: string
  name: string
  market: string // cn, hk, us
  value: number
  change: number
  change_pct: number
  high: number
  low: number
  prev_close: number
  volume: number
  mini_chart_data: number[]
  update_time: string
  data_source: string
}

export interface FundRankingItem {
  rank: number
  fund_code: string
  fund_name: string
  fund_type: string
  change_pct: number
  estimated_nav: number
  quote_date?: string
  quote_source?: string
}
