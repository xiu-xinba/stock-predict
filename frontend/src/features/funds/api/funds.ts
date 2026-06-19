/** @module funds/api — 基金模块 API 请求 */
import request from '@/shared/api/client'
import { API_ROUTES } from '@/shared/api/routes'
import type { ApiResponse } from '@/shared/api/types'
import type { FundDetailData } from '../types'

/**
 * 获取基金详情数据
 * @param fundCode - 基金代码，六位数字
 * @returns 基金详情聚合数据
 */
export async function fetchFundDetail(fundCode: string): Promise<ApiResponse<FundDetailData>> {
  const { data } = await request.get<ApiResponse<FundDetailData>>(API_ROUTES.fund.detail(fundCode))
  return data
}
