/** @module search/api — 搜索模块 API 请求 */
import request from '@/shared/api/client'
import { API_ROUTES } from '@/shared/api/routes'
import type { ApiResponse } from '@/shared/api/types'
import type { FundFilters, FundSearchData } from '@/features/funds'
import type { StockFilters, StockSearchData } from '@/features/stocks'

/** 统一搜索响应数据 */
export interface UnifiedSearchResponse {
  /** 搜索关键词 */
  query: string
  /** 基金搜索结果 */
  funds: FundSearchData
  /** 股票搜索结果 */
  stocks: StockSearchData
  /** 搜索建议列表 */
  suggestions?: string[]
}

/**
 * 统一搜索基金和股票
 * @param params - 搜索参数，包含关键词、类型、分页等
 * @returns 统一搜索响应数据
 */
export async function unifiedSearch(params: {
  q: string
  types?: string
  page?: number
  size?: number
}): Promise<ApiResponse<UnifiedSearchResponse>> {
  const { data } = await request.get<ApiResponse<UnifiedSearchResponse>>(API_ROUTES.search, {
    params,
  })
  return data
}

/**
 * 获取基金筛选条件
 * @returns 基金筛选条件数据
 */
export async function fetchFundFilters(): Promise<ApiResponse<FundFilters>> {
  const { data } = await request.get<ApiResponse<FundFilters>>(API_ROUTES.funds.filters)
  return data
}

/**
 * 获取股票筛选条件
 * @returns 股票筛选条件数据
 */
export async function fetchStockFilters(): Promise<ApiResponse<StockFilters>> {
  const { data } = await request.get<ApiResponse<StockFilters>>(API_ROUTES.stock.filters)
  return data
}
