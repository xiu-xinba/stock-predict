/** @module watchlist/api — 自选模块 API 请求 */
import request from '@/shared/api/client'
import { API_ROUTES } from '@/shared/api/routes'
import type { ApiResponse } from '@/shared/api/types'
import type { StockWatchlistQuote, WatchlistItem } from '../types'

/**
 * 批量获取基金自选行情
 * @param codes - 基金代码列表
 * @returns 基金自选行情列表
 */
export async function fetchWatchlistQuotes(codes: string[]): Promise<ApiResponse<WatchlistItem[]>> {
  if (!codes.length) {
    return { code: 0, message: 'success', data: [] }
  }
  const { data } = await request.post<ApiResponse<WatchlistItem[]>>(API_ROUTES.watchlist.quotes, {
    codes,
  })
  return data
}

/**
 * 批量获取股票自选行情
 * @param codes - 股票代码列表
 * @returns 股票代码到行情快照的映射
 */
export async function fetchStockWatchlistQuotes(
  codes: string[],
): Promise<ApiResponse<Record<string, StockWatchlistQuote>>> {
  const { data } = await request.post<ApiResponse<Record<string, StockWatchlistQuote>>>(
    API_ROUTES.stock.quotes,
    { codes, freshness: 'realtime' },
  )
  return data
}
