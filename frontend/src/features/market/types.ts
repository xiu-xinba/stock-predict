/** @module market/types — 行情模块类型定义 */
/** 市场指数数据 */
export interface MarketIndex {
  /** 指数代码 */
  code: string
  /** 指数名称 */
  name: string
  /** 所属市场（cn/hk/us） */
  market: string // cn, hk, us
  /** 当前点位 */
  value: number
  /** 涨跌点数 */
  change: number
  /** 涨跌幅百分比 */
  change_pct: number
  /** 最高点 */
  high: number
  /** 最低点 */
  low: number
  /** 昨收点数 */
  prev_close: number
  /** 开盘点数 */
  open: number
  /** 成交量 */
  volume: number
  /** 迷你走势数据 */
  mini_chart_data: number[]
  /** 数据更新时间 */
  update_time: string
  /** 数据来源标识 */
  data_source: string
  /** 是否已收盘 */
  is_closed: boolean
}

/** 基金排行项 */
export interface FundRankingItem {
  /** 排名 */
  rank: number
  /** 基金代码 */
  fund_code: string
  /** 基金名称 */
  fund_name: string
  /** 基金类型 */
  fund_type: string
  /** 涨跌幅 */
  change_pct: number
  /** 估算净值 */
  estimated_nav: number
  /** 行情日期 */
  quote_date?: string
  /** 行情来源 */
  quote_source?: string
}

/** 指数 K 线数据点 */
export interface IndexKlinePoint {
  /** 日期 */
  date: string
  /** 开盘价 */
  open: number
  /** 收盘价 */
  close: number
  /** 最高价 */
  high: number
  /** 最低价 */
  low: number
  /** 成交量 */
  volume: number
  /** 成交额 */
  amount: number
}

/** 指数分时数据点 */
export interface IndexMinutePoint {
  /** 时间 */
  time: string
  /** 价格 */
  price: number
  /** 均价 */
  avg_price: number
  /** 成交量 */
  volume: number
}

/** 行业板块项 */
export interface MarketSectorItem {
  /** 板块名称 */
  name: string
  /** 涨跌幅 */
  change_pct: number
  /** 上涨家数 */
  up_count: number
  /** 下跌家数 */
  down_count: number
  /** 领涨股 */
  lead_stock: string
}

/** 北向资金数据 */
export interface NorthboundFlow {
  /** 沪股通净买入 */
  sh_net_buy: number
  /** 深股通净买入 */
  sz_net_buy: number
  /** 总净买入 */
  total_net_buy: number
  /** 分时流向数据 */
  timeline: NorthboundPoint[]
  /** 数据状态：真实分时、日频回退或实时分时不可用 */
  status?: 'intraday' | 'daily_fallback' | 'intraday_unavailable'
  /** 数据来源标识 */
  data_source?: string
  /** 披露规则或授权说明 */
  notice?: string
}

/** 北向资金分时数据点 */
export interface NorthboundPoint {
  /** 时间 */
  time: string
  /** 沪股通流向 */
  sh_flow: number
  /** 深股通流向 */
  sz_flow: number
}

/** 股票排行项 */
export interface StockRankingItem {
  /** 排名 */
  rank: number
  /** 股票代码 */
  stock_code: string
  /** 股票名称 */
  stock_name: string
  /** 涨跌幅 */
  change_pct: number
  /** 当前价格 */
  current_price: number
  /** 成交量 */
  volume: number
  /** 成交额 */
  amount: number
  /** 数据来源 */
  data_source?: string
  /** 更新时间 */
  update_time?: string
}

/** 沪深港通每日资金流向数据 */
export interface HSGTFlowDaily {
  /** 日期 */
  date: string
  /** 沪股通净买入（万元） */
  north_sh_buy: number
  /** 深股通净买入（万元） */
  north_sz_buy: number
  /** 北向合计净买入（万元） */
  north_total_buy: number
  /** 北向合计成交额（万元） */
  north_total_amt: number
  /** 沪股通成交额（万元） */
  north_sh_amt: number
  /** 深股通成交额（万元） */
  north_sz_amt: number
  /** 港股通(沪)净买入（万元） */
  south_sh_buy: number
  /** 港股通(深)净买入（万元） */
  south_sz_buy: number
  /** 南向合计净买入（万元） */
  south_total_buy: number
  /** 数据来源 */
  source: string
}

/** HSGT 时间维度 */
export type HSGTTimeRange = 'daily' | 'weekly' | 'monthly'

/** HSGT 资金方向 */
export type HSGTDirection = 'north' | 'south'

/** HSGT 聚合数据点 */
export interface HSGTAggregatedPoint {
  /** 日期标签 */
  label: string
  /** 北向合计净买入 */
  north_total: number
  /** 沪股通净买入 */
  north_sh: number
  /** 深股通净买入 */
  north_sz: number
  /** 北向合计成交额 */
  north_total_amt: number
  /** 沪股通成交额 */
  north_sh_amt: number
  /** 深股通成交额 */
  north_sz_amt: number
  /** 南向合计净买入 */
  south_total: number
  /** 港股通沪净买入 */
  south_sh: number
  /** 港股通深净买入 */
  south_sz: number
}
