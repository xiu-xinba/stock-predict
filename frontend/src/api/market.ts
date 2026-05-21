import request from './index'
import type { ApiResponse, MarketIndex, FundRankingItem } from '@/types'

export async function fetchMarketIndices(signal?: AbortSignal): Promise<ApiResponse<MarketIndex[]>> {
  const { data } = await request.get<ApiResponse<MarketIndex[]>>('/market/indices', { signal })
  return data
}

export async function fetchFundRanking(type: 'gainers' | 'losers', size: number = 10, signal?: AbortSignal): Promise<ApiResponse<FundRankingItem[]>> {
  const { data } = await request.get<ApiResponse<FundRankingItem[]>>(`/market/ranking/${type}`, { params: { size }, signal })
  return data
}
