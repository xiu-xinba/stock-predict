import request from './index'
import { API_ROUTES } from './routes'
import type { PredictResponse } from '@/types'

export async function predictFund(fundCode: string): Promise<PredictResponse> {
  const { data } = await request.get<PredictResponse>(API_ROUTES.prediction.fund(fundCode))
  return data
}
