import { defineStore } from 'pinia'
import { ref } from 'vue'
import { fetchFundDetail } from '@/api/fundDetail'
import type { FundDetailData, AppError } from '@/types'

export const useFundDetailStore = defineStore('fundDetail', () => {
  const detail = ref<FundDetailData | null>(null)
  const loading = ref(false)
  const error = ref<AppError | null>(null)
  let requestSeq = 0

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
        error.value = { code: res.code, message: res.message || '获取基金详情失败', retryable: false, type: 'business' }
      }
    } catch (e: unknown) {
      if (seq !== requestSeq) return
      if (e instanceof Error && e.name === 'CanceledError') return
      if (typeof e === 'object' && e !== null && 'code' in e && 'message' in e && 'type' in e) {
        error.value = e as AppError
      } else {
        error.value = { code: 0, message: '网络异常，请稍后重试', retryable: true, type: 'network' }
      }
    } finally {
      if (seq === requestSeq) loading.value = false
    }
  }

  function reset() {
    detail.value = null
    loading.value = false
    error.value = null
  }

  return { detail, loading, error, fetchDetail, reset }
})
