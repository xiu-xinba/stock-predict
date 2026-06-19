/** @module stocks/types — 股票模块类型定义 */
/** 股票搜索结果项 */
export interface StockItem {
  /** 股票代码 */
  stock_code: string
  /** 股票名称 */
  stock_name: string
  /** 所属市场（sh/sz/bj） */
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

/** 股票搜索结果分页数据 */
export interface StockSearchData {
  /** 股票列表 */
  items: StockItem[]
  /** 总数 */
  total: number
  /** 当前页码 */
  page: number
  /** 每页条数 */
  size: number
}

/** 股票筛选条件 */
export interface StockFilters {
  /** 行业列表 */
  industries: string[]
  /** 市场列表 */
  markets: string[]
}

/** 股票基本信息 */
export interface StockBasicInfo {
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
}

/** 分时数据点 */
export interface MinutePoint {
  /** 时间（HH:mm） */
  time: string
  /** 价格 */
  price: number
  /** 均价 */
  avg_price: number
  /** 成交量 */
  volume: number
}

/** 股票实时行情 */
export interface StockQuote {
  /** 当前价格 */
  price: number
  /** 开盘价 */
  open: number
  /** 最高价 */
  high: number
  /** 最低价 */
  low: number
  /** 昨收价 */
  prev_close: number
  /** 成交量 */
  volume: number
  /** 成交额 */
  amount: number
  /** 换手率 */
  turnover_rate: number
  /** 涨跌幅 */
  change_pct: number
  /** 涨跌额 */
  change_amt: number
  /** 买一价 */
  bid_price: number
  /** 卖一价 */
  ask_price: number
  /** 行情时间 */
  quote_time: string
  /** 分时数据 */
  intradayData?: MinutePoint[]
}

/** 行情数据新鲜度策略 */
export type StockQuoteFreshness = 'balanced' | 'realtime'

/** K 线数据点 */
export interface KlinePoint {
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

/** K 线数据集 */
export interface StockKlineData {
  /** 周期标识 */
  period: string
  /** K 线数据列表 */
  klines: KlinePoint[]
}

/** 资金流向数据点 */
export interface CapitalFlowPoint {
  /** 日期 */
  date: string
  /** 主力流入 */
  main_inflow: number
  /** 主力流出 */
  main_outflow: number
  /** 散户流入 */
  retail_inflow: number
  /** 散户流出 */
  retail_outflow: number
  /** 净流入 */
  net_inflow: number
}

/** 股票资金流向 */
export interface StockCapitalFlow {
  /** 主力净流入 */
  main_net_inflow: number
  /** 散户净流入 */
  retail_net_inflow: number
  /** 历史流向数据 */
  flow_history: CapitalFlowPoint[]
}

/** 财务季度数据 */
export interface FinancialQuarter {
  /** 报告期 */
  report_date: string
  /** 营业收入 */
  revenue: number
  /** 净利润 */
  net_profit: number
  /** 每股收益 */
  eps: number
  /** 毛利率 */
  gross_margin: number
  /** 净利率 */
  net_margin: number
  /** 净资产收益率 */
  roe: number
}

/** 股票财务指标 */
export interface StockFinancials {
  /** 市盈率 */
  pe_ratio: number
  /** 市净率 */
  pb_ratio: number
  /** 净资产收益率 */
  roe: number
  /** 营业收入 */
  revenue: number
  /** 净利润 */
  net_profit: number
  /** 每股收益 */
  eps: number
  /** 毛利率 */
  gross_margin: number
  /** 净利率 */
  net_margin: number
  /** 季度财务数据 */
  quarterly: FinancialQuarter[]
}

/** 股东明细项 */
export interface ShareholderItem {
  /** 股东名称 */
  name: string
  /** 持股比例 */
  ratio: number
  /** 增减变化 */
  change: number
  /** 股份类型 */
  share_type: string
}

/** 股东信息 */
export interface StockShareholders {
  /** 前十大股东 */
  top10: ShareholderItem[]
  /** 机构持仓数量 */
  institution_count: number
  /** 机构持仓比例 */
  institution_ratio: number
}

/** 股票详情聚合数据 */
export interface StockDetailData {
  /** 基本信息 */
  basic: StockBasicInfo
  /** 实时行情 */
  quote: StockQuote
  /** K 线数据 */
  kline: StockKlineData
  /** 分时数据 */
  minute_data: MinutePoint[]
  /** 资金流向 */
  capital_flow: StockCapitalFlow
  /** 财务指标 */
  financials: StockFinancials
  /** 股东信息 */
  shareholders: StockShareholders
  /** 研报评级 */
  research?: StockResearch
  /** 分红送配 */
  dividends?: StockDividends
  /** 融资融券 */
  margin?: StockMargin
  /** 股东人数变化 */
  shareholder_trend?: StockShareholderTrend
  /** 限售解禁 */
  restricted?: StockRestricted
}

/** 研报评级条目 */
export interface ResearchReport {
  /** 发布日期 */
  date: string
  /** 机构名称 */
  org_name: string
  /** 评级 */
  rating: string
  /** 目标价 */
  target_price: number
  /** 研究员 */
  researcher: string
}

/** 研报评级汇总 */
export interface StockResearch {
  /** 最新评级 */
  latest_rating: string
  /** 评级数量 */
  rating_count: number
  /** 研报列表 */
  reports: ResearchReport[]
}

/** 分红送配记录 */
export interface DividendRecord {
  /** 除权除息日 */
  date: string
  /** 每股送股 */
  bonus: number
  /** 每股转增 */
  transfer: number
  /** 每股分红（元） */
  dividend: number
  /** 进度 */
  progress: string
}

/** 分红送配汇总 */
export interface StockDividends {
  /** 累计分红 */
  total_dividend: number
  /** 分红记录 */
  records: DividendRecord[]
}

/** 融资融券数据点 */
export interface MarginData {
  /** 日期 */
  date: string
  /** 融资余额 */
  margin_balance: number
  /** 融资买入额 */
  margin_buy: number
  /** 融券余额 */
  short_balance: number
  /** 融券余量 */
  short_volume: number
}

/** 融资融券汇总 */
export interface StockMargin {
  /** 最新融资余额 */
  latest_margin_balance: number
  /** 历史数据 */
  history: MarginData[]
}

/** 股东人数变化数据点 */
export interface ShareholderTrendPoint {
  /** 报告日期 */
  date: string
  /** 股东人数 */
  count: number
  /** 户均持股 */
  avg_holding: number
  /** 变化率 */
  change: number
}

/** 股东人数变化汇总 */
export interface StockShareholderTrend {
  /** 最新股东人数 */
  latest_count: number
  /** 趋势数据 */
  trend: ShareholderTrendPoint[]
}

/** 限售解禁记录 */
export interface RestrictedRelease {
  /** 解禁日期 */
  date: string
  /** 解禁数量 */
  volume: number
  /** 占总股本比例 */
  ratio: number
  /** 解禁类型 */
  type: string
}

/** 限售解禁汇总 */
export interface StockRestricted {
  /** 下次解禁 */
  next_release: RestrictedRelease | null
  /** 历史解禁 */
  history: RestrictedRelease[]
}
