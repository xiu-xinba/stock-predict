/** @module funds/store — 基金详情 Pinia store */
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { fetchFundDetail } from '@/features/funds/api/funds'
import { CancelError } from '@/shared/api/client'
import type { AppError } from '@/shared/types/errors'
import type { FundDetailData } from '@/features/funds/types'

/** useFundDetailStore - 基金详情 store */
export const useFundDetailStore = defineStore('fundDetail', () => {
  const detail = ref<FundDetailData | null>(null)
  const loading = ref(false)
  const error = ref<AppError | null>(null)
  let requestSeq = 0

  const APP_ERROR_TYPES: AppError['type'][] = [
    'network',
    'server',
    'business',
    'timeout',
    'unknown',
  ]

  function isAppError(e: unknown): e is AppError {
    return (
      typeof e === 'object' &&
      e !== null &&
      'message' in e &&
      'type' in e &&
      'retryable' in e &&
      APP_ERROR_TYPES.includes((e as AppError).type)
    )
  }

  /**
   * 获取基金详情
   * @param fundCode - 基金代码
   */
  async function fetchDetail(fundCode: string) {
    const seq = ++requestSeq
    loading.value = true
    error.value = null
    try {
      const res = await fetchFundDetail(fundCode)
      if (seq !== requestSeq) return
      if (res.code === 0 && res.data) {
        detail.value = res.data
      } else {
        error.value = {
          code: res.code,
          message: res.message || '获取基金详情失败',
          retryable: false,
          type: 'business',
        }
      }
    } catch (e: unknown) {
      if (seq !== requestSeq) return
      if (e instanceof CancelError) return
      if (isAppError(e)) {
        error.value = e
      } else {
        error.value = { code: 0, message: '网络异常，请稍后重试', retryable: true, type: 'network' }
      }
    } finally {
      if (seq === requestSeq) loading.value = false
    }
  }

  /** 重置 store 状态 */
  function reset() {
    detail.value = null
    loading.value = false
    error.value = null
  }

  return { detail, loading, error, fetchDetail, reset }
})
