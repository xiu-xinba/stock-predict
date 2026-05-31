import request from './index'
import { API_ROUTES } from './routes'
import type { ApiResponse, FundDetailData } from '@/types'

export async function fetchFundDetail(fundCode: string): Promise<ApiResponse<FundDetailData>> {
  const { data } = await request.get<ApiResponse<FundDetailData>>(API_ROUTES.fund.detail(fundCode))
  return data
}
