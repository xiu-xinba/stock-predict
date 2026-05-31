import { defineStore } from 'pinia'
import { ref } from 'vue'
import { fetchStockDetail } from '@/api/stock'
import type { StockDetailData, AppError } from '@/types'

export const useStockDetailStore = defineStore('stockDetail', () => {
  const detail = ref<StockDetailData | null>(null)
  const loading = ref(false)
  const error = ref<AppError | null>(null)
  let requestSeq = 0

  async function fetchDetail(stockCode: string) {
    const seq = ++requestSeq
    loading.value = true
    error.value = null
    try {
      const res = await fetchStockDetail(stockCode)
      if (seq !== requestSeq) return
      if (res.code === 0 && res.data) {
        detail.value = res.data
      } else {
        error.value = { code: res.code, message: res.message || '获取股票详情失败', retryable: false, type: 'business' }
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
