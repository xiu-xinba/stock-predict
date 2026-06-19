/** @module market/api — 行情模块 API 请求 */
import request from '@/shared/api/client'
import { API_ROUTES } from '@/shared/api/routes'
import type { ApiResponse } from '@/shared/api/types'
import type {
  MarketIndex,
  FundRankingItem,
  IndexKlinePoint,
  IndexMinutePoint,
  MarketSectorItem,
  NorthboundFlow,
  StockRankingItem,
  HSGTFlowDaily,
} from '../types'

/**
 * 获取市场指数列表
 * @param signal - 可选的 AbortSignal，用于取消请求
 * @returns 市场指数数据列表
 */
export async function fetchMarketIndices(
  signal?: AbortSignal,
): Promise<ApiResponse<MarketIndex[]>> {
  const { data } = await request.get<ApiResponse<MarketIndex[]>>(API_ROUTES.market.indices, {
    signal,
  })
  return data
}

/**
 * 获取基金排行数据
 * @param type - 排行类型，gainers 为涨幅榜，losers 为跌幅榜
 * @param size - 返回条数，默认 10
 * @param signal - 可选的 AbortSignal
 * @returns 基金排行列表
 */
export async function fetchFundRanking(
  type: 'gainers' | 'losers',
  size: number = 10,
  signal?: AbortSignal,
): Promise<ApiResponse<FundRankingItem[]>> {
  const { data } = await request.get<ApiResponse<FundRankingItem[]>>(
    API_ROUTES.market.ranking(type),
    { params: { size }, signal },
  )
  return data
}

/**
 * 获取股票排行数据
 * @param type - 排行类型
 * @param size - 返回条数，默认 10
 * @param signal - 可选的 AbortSignal
 * @returns 股票排行列表
 */
export async function fetchStockRanking(
  type: string,
  size: number = 10,
  signal?: AbortSignal,
): Promise<ApiResponse<StockRankingItem[]>> {
  const { data } = await request.get<ApiResponse<StockRankingItem[]>>(
    API_ROUTES.stock.ranking(type),
    { params: { size }, signal, skipDedup: true },
  )
  return data
}

/**
 * 获取指数 K 线数据
 * @param code - 指数代码
 * @param count - 返回条数，默认 120
 * @param signal - 可选的 AbortSignal
 * @returns K 线数据点列表
 */
export async function fetchIndexKline(
  code: string,
  count: number = 120,
  signal?: AbortSignal,
): Promise<ApiResponse<IndexKlinePoint[]>> {
  const { data } = await request.get<ApiResponse<IndexKlinePoint[]>>(
    API_ROUTES.market.indexKline(code),
    { params: { count }, signal, skipDedup: true },
  )
  return data
}

/**
 * 获取指数分时数据
 * @param code - 指数代码
 * @param signal - 可选的 AbortSignal
 * @returns 分时数据点列表
 */
export async function fetchIndexMinute(
  code: string,
  signal?: AbortSignal,
): Promise<ApiResponse<IndexMinutePoint[]>> {
  const { data } = await request.get<ApiResponse<IndexMinutePoint[]>>(
    API_ROUTES.market.indexMinute(code),
    { signal, skipDedup: true },
  )
  return data
}

/**
 * 获取行业板块排行数据
 * @param signal - 可选的 AbortSignal
 * @returns 行业板块列表
 */
export async function fetchSectorRanking(
  signal?: AbortSignal,
): Promise<ApiResponse<MarketSectorItem[]>> {
  const { data } = await request.get<ApiResponse<MarketSectorItem[]>>(API_ROUTES.market.sectors, {
    signal,
    skipDedup: true,
  })
  return data
}

/**
 * 获取北向资金流向数据
 * @param signal - 可选的 AbortSignal
 * @returns 北向资金数据
 */
export async function fetchNorthboundFlow(
  signal?: AbortSignal,
): Promise<ApiResponse<NorthboundFlow>> {
  const { data } = await request.get<ApiResponse<NorthboundFlow>>(API_ROUTES.market.northbound, {
    signal,
    skipDedup: true,
  })
  return data
}

/**
 * 获取沪深港通历史资金流向数据
 * @param days - 返回最近 N 个交易日的数据，默认 365
 * @param signal - 可选的 AbortSignal
 * @returns HSGT 每日资金流向列表
 */
export async function fetchHSGTHist(
  days: number = 365,
  signal?: AbortSignal,
): Promise<ApiResponse<HSGTFlowDaily[]>> {
  const { data } = await request.get<ApiResponse<HSGTFlowDaily[]>>(API_ROUTES.market.hsgtHist, {
    params: { days },
    signal,
    skipDedup: true,
  })
  return data
}

/** 数据源健康信息 */
export interface SourceHealthInfo {
  name: string
  status: string
  fail_count: number
  last_check: string
  last_error: string
}

/** 市场健康状态数据 */
export interface MarketHealthData {
  cache_stats: Record<string, unknown>
  sources: Record<string, SourceHealthInfo>
}

/**
 * 获取市场数据源健康状态
 * @returns 市场健康状态数据
 */
export async function getMarketHealth() {
  const { data } = await request.get<ApiResponse<MarketHealthData>>(API_ROUTES.market.health, {
    skipDedup: true,
  })
  return data
}
