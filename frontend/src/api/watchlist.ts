import request from './index'
import type { ApiResponse, WatchlistItem } from '@/types'

export async function fetchWatchlistQuotes(codes: string[]): Promise<ApiResponse<WatchlistItem[]>> {
  const { data } = await request.post<ApiResponse<WatchlistItem[]>>('/watchlist/quotes', { codes })
  return data
}
