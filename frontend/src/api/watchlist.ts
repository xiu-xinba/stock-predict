import request from './index'
import { API_ROUTES } from './routes'
import type { ApiResponse, WatchlistItem } from '@/types'

export async function fetchWatchlistQuotes(codes: string[]): Promise<ApiResponse<WatchlistItem[]>> {
  const { data } = await request.post<ApiResponse<WatchlistItem[]>>(API_ROUTES.watchlist.quotes, { codes })
  return data
}
