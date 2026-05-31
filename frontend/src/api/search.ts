import request from './index'
import { API_ROUTES } from './routes'
import type { ApiResponse, FundSearchData, FundFilters, StockSearchData, StockFilters } from '@/types'

export interface UnifiedSearchResponse {
  query: string
  funds: FundSearchData
  stocks: StockSearchData
  suggestions?: string[]
}

export async function unifiedSearch(params: {
  q: string
  types?: string
  page?: number
  size?: number
}): Promise<ApiResponse<UnifiedSearchResponse>> {
  const { data } = await request.get<ApiResponse<UnifiedSearchResponse>>(API_ROUTES.search, { params })
  return data
}

export async function fetchFundFilters(): Promise<ApiResponse<FundFilters>> {
  const { data } = await request.get<ApiResponse<FundFilters>>(API_ROUTES.funds.filters)
  return data
}

export async function fetchStockFilters(): Promise<ApiResponse<StockFilters>> {
  const { data } = await request.get<ApiResponse<StockFilters>>(API_ROUTES.stock.filters)
  return data
}
