import request from './index'
import { API_ROUTES } from './routes'
import type { ApiResponse, StockSearchData, StockDetailData, StockRankingItem, StockQuote } from '@/types'

export async function fetchStockList(params: {
  keyword?: string
  industry?: string
  market?: string
  sort_by?: string
  sort_order?: string
  page?: number
  size?: number
}): Promise<ApiResponse<StockSearchData>> {
  const { data } = await request.get<ApiResponse<StockSearchData>>(API_ROUTES.stock.search, { params })
  return data
}

export async function fetchStockDetail(stockCode: string): Promise<ApiResponse<StockDetailData>> {
  const { data } = await request.get<ApiResponse<StockDetailData>>(API_ROUTES.stock.detail(stockCode))
  return data
}

export async function fetchStockQuotes(codes: string[]): Promise<ApiResponse<Record<string, StockQuote>>> {
  const { data } = await request.post<ApiResponse<Record<string, StockQuote>>>(API_ROUTES.stock.quotes, { codes })
  return data
}

export async function fetchStockRanking(type: string): Promise<ApiResponse<StockRankingItem[]>> {
  const { data } = await request.get<ApiResponse<StockRankingItem[]>>(API_ROUTES.stock.ranking(type))
  return data
}
