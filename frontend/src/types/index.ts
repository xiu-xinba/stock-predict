export interface AppError {
  code: number
  message: string
  retryable: boolean
  type: 'network' | 'server' | 'business' | 'timeout' | 'unknown'
}

export * from './predict'
export * from './watchlist'
export * from './market'
export type { FundDetailData, NAVPoint, FundPerformanceData, FundManagerInfo, HoldingItem, SectorItem, FundPortfolioData, FundRiskMetrics } from './fundDetail'
export * from './stock'
