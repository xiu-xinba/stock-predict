/** @module stocks/api — 股票模块 API 请求 */
import request from '@/shared/api/client'
import { API_ROUTES } from '@/shared/api/routes'
import type { ApiResponse } from '@/shared/api/types'
import type {
  StockSearchData,
  StockDetailData,
  StockQuote,
  StockKlineData,
  MinutePoint,
  StockQuoteFreshness,
} from '../types'

/**
 * 搜索股票列表
 * @param params - 搜索参数，包含关键词、行业、市场、排序、分页等
 * @returns 股票搜索结果分页数据
 */
export async function fetchStockList(params: {
  keyword?: string
  industry?: string
  market?: string
  sort_by?: string
  sort_order?: string
  page?: number
  size?: number
}): Promise<ApiResponse<StockSearchData>> {
  const { data } = await request.get<ApiResponse<StockSearchData>>(API_ROUTES.stock.search, {
    params,
  })
  return data
}

/**
 * 获取股票详情数据
 * @param stockCode - 股票代码，六位数字
 * @returns 股票详情聚合数据
 */
export async function fetchStockDetail(stockCode: string): Promise<ApiResponse<StockDetailData>> {
  const { data } = await request.get<ApiResponse<StockDetailData>>(
    API_ROUTES.stock.detail(stockCode),
  )
  return data
}

/**
 * 批量获取股票实时行情
 * @param codes - 股票代码列表
 * @param freshness - 行情新鲜度策略，默认 balanced
 * @returns 股票代码到行情数据的映射
 */
export async function fetchStockQuotes(
  codes: string[],
  freshness: StockQuoteFreshness = 'balanced',
): Promise<ApiResponse<Record<string, StockQuote>>> {
  const { data } = await request.post<ApiResponse<Record<string, StockQuote>>>(
    API_ROUTES.stock.quotes,
    { codes, freshness },
  )
  return data
}

/**
 * 获取股票分时数据
 * @param stockCode - 股票代码
 * @param signal - 可选的 AbortSignal，用于取消请求
 * @returns 分时数据点列表
 */
export async function fetchStockMinute(
  stockCode: string,
  signal?: AbortSignal,
): Promise<ApiResponse<MinutePoint[]>> {
  const { data } = await request.get<ApiResponse<MinutePoint[]>>(
    API_ROUTES.stock.stockMinute(stockCode),
    { signal, skipDedup: true },
  )
  return data
}

/**
 * 获取股票 K 线数据（支持复权类型和周期）
 * @param stockCode - 股票代码
 * @param period - 周期：daily/weekly/monthly
 * @param fq - 复权类型：0=不复权, 1=前复权, 2=后复权
 * @returns K 线数据
 */
export async function fetchStockKline(
  stockCode: string,
  period: string,
  fq: number,
): Promise<ApiResponse<StockKlineData>> {
  const { data } = await request.get<ApiResponse<StockKlineData>>(
    API_ROUTES.stock.kline(stockCode, period, fq),
  )
  return data
}
