import request from './index'
import type { FundSearchResponse, FundFiltersResponse, PredictResponse } from '@/types'

export async function predictFund(fundCode: string): Promise<PredictResponse> {
  const { data } = await request.get<PredictResponse>(`/predict/${fundCode}`)
  return data
}

export async function searchFunds(
  keyword: string,
  page: number = 1,
  size: number = 20,
  filters?: {
    type?: string
    company?: string
    risk_level?: string
    manager?: string
    return_min?: number
    return_max?: number
    sort_by?: string
    sort_order?: string
  }
): Promise<FundSearchResponse> {
  const params: Record<string, string | number> = { keyword, page, size }
  if (filters) {
    if (filters.type) params.type = filters.type
    if (filters.company) params.company = filters.company
    if (filters.risk_level) params.risk_level = filters.risk_level
    if (filters.manager) params.manager = filters.manager
    if (filters.return_min !== undefined) params.return_min = filters.return_min
    if (filters.return_max !== undefined) params.return_max = filters.return_max
    if (filters.sort_by) params.sort_by = filters.sort_by
    if (filters.sort_order) params.sort_order = filters.sort_order
  }
  const { data } = await request.get<FundSearchResponse>('/funds/search', { params })
  return data
}

export async function fetchFundFilters(): Promise<FundFiltersResponse> {
  const { data } = await request.get<FundFiltersResponse>('/funds/filters')
  return data
}
