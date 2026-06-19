/** @module funds/types — 基金模块类型定义 */
/** 基金搜索结果项 */
export interface FundItem {
  /** 基金代码，六位数字 */
  fund_code: string
  /** 基金名称 */
  fund_name: string
  /** 基金类型，如混合型、指数型、股票型等 */
  fund_type: string
  /** 基金公司 */
  company?: string
  /** 基金经理 */
  manager?: string
  /** 最新净值 */
  latest_nav?: number
  /** 累计净值 */
  cumulative_nav?: number
  /** 估算净值 */
  estimated_nav?: number
  /** 日涨跌幅百分比 */
  change_pct?: number
  /** 近一月收益率 */
  return_1m?: number
  /** 近三月收益率 */
  return_3m?: number
  /** 近六月收益率 */
  return_6m?: number
  /** 近一年收益率 */
  return_1y?: number
  /** 近三年收益率 */
  return_3y?: number
  /** 风险等级 */
  risk_level?: string
  /** 基金成立日期 */
  inception_date?: string
  /** 行情日期或估值时间 */
  quote_date?: string
  /** 行情数据来源 */
  quote_source?: string
}

/** 基金搜索结果分页数据 */
export interface FundSearchData {
  /** 基金列表 */
  items: FundItem[]
  /** 总数 */
  total: number
  /** 当前页码 */
  page: number
  /** 每页条数 */
  size: number
}

/** 基金筛选条件 */
export interface FundFilters {
  /** 基金类型列表 */
  types: string[]
  /** 基金公司列表 */
  companies: string[]
  /** 风险等级列表 */
  risk_levels: string[]
}

// --- 基金详情 ---

/** 净值历史数据点 */
export interface NAVPoint {
  /** 日期 */
  date: string
  /** 单位净值 */
  nav: number
  /** 累计净值 */
  cumulative_nav: number
  /** 涨跌幅 */
  change_pct: number
}

/** 基金业绩数据 */
export interface FundPerformanceData {
  /** 净值历史走势 */
  nav_history: NAVPoint[]
  /** 近一月收益率 */
  return_1m: number
  /** 近三月收益率 */
  return_3m: number
  /** 近六月收益率 */
  return_6m: number
  /** 近一年收益率 */
  return_1y: number
  /** 近三年收益率 */
  return_3y: number
}

/** 基金经理信息 */
export interface FundManagerInfo {
  /** 经理姓名 */
  name: string
  /** 任职天数 */
  tenure_days: number
  /** 管理规模 */
  managed_size: string
  /** 在管基金数量 */
  fund_count: number
  /** 个人简介 */
  bio: string
}

/** 持仓明细项 */
export interface HoldingItem {
  /** 持仓名称 */
  name: string
  /** 持仓代码 */
  code: string
  /** 持仓占比 */
  ratio: number
}

/** 行业配置项 */
export interface SectorItem {
  /** 行业名称 */
  name: string
  /** 配置占比 */
  ratio: number
}

/** 基金投资组合数据 */
export interface FundPortfolioData {
  /** 前十大持仓 */
  top_holdings: HoldingItem[]
  /** 行业配置分布 */
  sector_allocation: SectorItem[]
}

/** 基金风险指标 */
export interface FundRiskMetrics {
  /** 一年年化波动率 */
  volatility_1y: number
  /** 一年最大回撤 */
  max_drawdown_1y: number
  /** 一年夏普比率 */
  sharpe_1y: number
  /** 一年 Beta 系数 */
  beta_1y: number
}

/** 基金详情聚合数据 */
export interface FundDetailData {
  /** 基本信息 */
  basic: FundItem
  /** 行情数据 */
  quote: FundItem
  /** 业绩表现 */
  performance: FundPerformanceData
  /** 基金经理 */
  manager: FundManagerInfo
  /** 投资组合 */
  portfolio: FundPortfolioData
  /** 风险指标 */
  risk: FundRiskMetrics
}
