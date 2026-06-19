/** @module watchlist/types — 自选模块类型定义 */
/** 基金自选项 */
export interface WatchlistItem {
  /** 基金代码 */
  fund_code: string
  /** 基金名称 */
  fund_name: string
  /** 基金类型 */
  fund_type: string
  /** 估算净值 */
  estimated_nav: number
  /** 涨跌幅百分比 */
  change_pct: number
  /** 涨跌方向 */
  direction: 'up' | 'down' | 'flat'
  /** 添加时间戳 */
  added_at: number
  /** 行情日期 */
  quote_date?: string
  /** 行情来源 */
  quote_source?: string
}

/** 股票自选项 */
export interface StockWatchlistItem {
  /** 股票代码 */
  stock_code: string
  /** 股票名称 */
  stock_name: string
  /** 所属市场 */
  market: string
  /** 所属行业 */
  industry: string
  /** 上市日期 */
  list_date: string
  /** 总股本 */
  total_shares: number
  /** 流通股本 */
  float_shares: number
  /** 当前价格 */
  current_price: number
  /** 涨跌幅百分比 */
  change_pct: number
  /** 成交量 */
  volume: number
  /** 成交额 */
  amount: number
  /** 换手率 */
  turnover_rate: number
  /** 市盈率 */
  pe_ratio: number
  /** 市净率 */
  pb_ratio: number
  /** 总市值 */
  total_mv: number
  /** 拼音缩写 */
  pinyin: string
}

/** 股票自选行情快照 */
export interface StockWatchlistQuote {
  /** 当前价格 */
  price: number
  /** 涨跌幅百分比 */
  change_pct: number
}
