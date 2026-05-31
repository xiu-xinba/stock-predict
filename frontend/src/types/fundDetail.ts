import type { FundItem } from './predict'

export interface NAVPoint {
  date: string
  nav: number
  cumulative_nav: number
  change_pct: number
}

export interface FundPerformanceData {
  nav_history: NAVPoint[]
  return_1m: number
  return_3m: number
  return_6m: number
  return_1y: number
  return_3y: number
}

export interface FundManagerInfo {
  name: string
  tenure_days: number
  managed_size: string
  fund_count: number
  bio: string
}

export interface HoldingItem {
  name: string
  code: string
  ratio: number
}

export interface SectorItem {
  name: string
  ratio: number
}

export interface FundPortfolioData {
  top_holdings: HoldingItem[]
  sector_allocation: SectorItem[]
}

export interface FundRiskMetrics {
  volatility_1y: number
  max_drawdown_1y: number
  sharpe_1y: number
  beta_1y: number
}

export interface FundDetailData {
  basic: FundItem
  quote: FundItem
  performance: FundPerformanceData
  manager: FundManagerInfo
  portfolio: FundPortfolioData
  risk: FundRiskMetrics
}
