/** @module shared/api/routes - API 路由常量定义，集中管理所有后端接口路径 */

/** 后端 API 路由表，使用 as const 确保类型字面量推断 */
export const API_ROUTES = {
  health: '/health',
  admin: {
    restart: '/admin/restart',
  },
  search: '/search',
  funds: {
    search: '/funds/search',
    filters: '/funds/filters',
  },
  market: {
    indices: '/market/indices',
    ranking: (type: 'gainers' | 'losers') => `/market/ranking/${type}`,
    indexKline: (code: string) => `/market/index/${code}/kline`,
    indexMinute: (code: string) => `/market/index/${code}/minute`,
    sectors: '/market/sectors',
    northbound: '/market/northbound',
    hsgtHist: '/market/hsgt/hist',
    health: '/market/health',
  },
  fund: {
    detail: (fundCode: string) => `/fund/${fundCode}/detail`,
  },
  watchlist: {
    quotes: '/watchlist/quotes',
  },
  stock: {
    search: '/stocks/search',
    filters: '/stocks/filters',
    detail: (code: string) => `/stock/${code}/detail`,
    stockMinute: (code: string) => `/stock/${code}/minute`,
    kline: (code: string, period: string, fq: number) =>
      `/stock/${code}/kline?period=${period}&fq=${fq}`,
    quotes: '/stocks/quotes',
    ranking: (type: string) => `/market/stock-ranking/${type}`,
    sync: '/stocks/sync',
  },
} as const
