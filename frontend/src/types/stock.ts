export interface StockItem {
  stock_code: string
  stock_name: string
  market: string
  industry: string
  list_date: string
  total_shares: number
  float_shares: number
  current_price: number
  change_pct: number
  volume: number
  amount: number
  turnover_rate: number
  pe_ratio: number
  pb_ratio: number
  total_mv: number
  pinyin: string
}

export interface StockSearchData {
  items: StockItem[]
  total: number
  page: number
  size: number
}

export interface StockFilters {
  industries: string[]
  markets: string[]
}

export interface StockBasicInfo {
  stock_code: string
  stock_name: string
  market: string
  industry: string
  list_date: string
  total_shares: number
  float_shares: number
}

export interface StockQuote {
  price: number
  open: number
  high: number
  low: number
  prev_close: number
  volume: number
  amount: number
  turnover_rate: number
  change_pct: number
  change_amt: number
  bid_price: number
  ask_price: number
  quote_time: string
  intradayData?: { time: string; price: number }[]
}

export interface KlinePoint {
  date: string
  open: number
  close: number
  high: number
  low: number
  volume: number
  amount: number
}

export interface StockKlineData {
  period: string
  klines: KlinePoint[]
}

export interface CapitalFlowPoint {
  date: string
  main_inflow: number
  main_outflow: number
  retail_inflow: number
  retail_outflow: number
  net_inflow: number
}

export interface StockCapitalFlow {
  main_net_inflow: number
  retail_net_inflow: number
  flow_history: CapitalFlowPoint[]
}

export interface FinancialQuarter {
  report_date: string
  revenue: number
  net_profit: number
  eps: number
  gross_margin: number
  net_margin: number
  roe: number
}

export interface StockFinancials {
  pe_ratio: number
  pb_ratio: number
  roe: number
  revenue: number
  net_profit: number
  eps: number
  gross_margin: number
  net_margin: number
  quarterly: FinancialQuarter[]
}

export interface ShareholderItem {
  name: string
  ratio: number
  change: number
  share_type: string
}

export interface StockShareholders {
  top10: ShareholderItem[]
  institution_count: number
  institution_ratio: number
}

export interface StockDetailData {
  basic: StockBasicInfo
  quote: StockQuote
  kline: StockKlineData
  capital_flow: StockCapitalFlow
  financials: StockFinancials
  shareholders: StockShareholders
}

export interface StockRankingItem {
  rank: number
  stock_code: string
  stock_name: string
  change_pct: number
  current_price: number
  volume: number
  amount: number
}
