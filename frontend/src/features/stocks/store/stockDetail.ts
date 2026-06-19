/** @module stocks/store — 股票详情 Pinia store */
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { fetchStockDetail, fetchStockMinute, fetchStockQuotes } from '@/features/stocks/api/stocks'
import { CancelError } from '@/shared/api/client'
import type { AppError } from '@/shared/types/errors'
import type { StockDetailData, MinutePoint } from '@/features/stocks/types'

const STOCK_QUOTE_POLL_INTERVAL_MS = 3000
const STOCK_MINUTE_POLL_INTERVAL_MS = 30000

/** useStockDetailStore - 股票详情 store */
export const useStockDetailStore = defineStore('stockDetail', () => {
  const detail = ref<StockDetailData | null>(null)
  const loading = ref(false)
  const error = ref<AppError | null>(null)
  const minuteData = ref<MinutePoint[]>([])
  const minuteLoading = ref(false)
  let requestSeq = 0
  let minuteTimer: ReturnType<typeof setInterval> | null = null
  let quoteTimer: ReturnType<typeof setInterval> | null = null

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
   * 获取股票详情
   * @param stockCode - 股票代码
   */
  async function fetchDetail(stockCode: string) {
    const seq = ++requestSeq
    loading.value = true
    error.value = null
    try {
      const res = await fetchStockDetail(stockCode)
      if (seq !== requestSeq) return
      if (res.code === 0 && res.data) {
        detail.value = res.data
        // Use minute_data from detail response if available
        if (res.data.minute_data && res.data.minute_data.length > 0) {
          minuteData.value = res.data.minute_data
        }
        void refreshQuote(stockCode)
      } else {
        error.value = {
          code: res.code,
          message: res.message || '获取股票详情失败',
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

  /**
   * 刷新股票实时行情
   * @param code - 股票代码
   */
  async function refreshQuote(code: string) {
    try {
      const res = await fetchStockQuotes([code], 'realtime')
      const quote = res.data?.[code]
      if (res.code === 0 && quote && detail.value?.basic.stock_code === code) {
        detail.value.quote = quote
      }
    } catch (e: unknown) {
      if (e instanceof CancelError) return
    }
  }

  /**
   * 获取股票分时数据
   * @param code - 股票代码
   */
  async function fetchMinuteData(code: string) {
    minuteLoading.value = true
    try {
      const res = await fetchStockMinute(code)
      if (res.code === 0 && res.data) {
        minuteData.value = res.data
      } else {
        minuteData.value = []
      }
    } catch {
      minuteData.value = []
    } finally {
      minuteLoading.value = false
    }
  }

  /**
   * 启动分时和行情轮询
   * @param code - 股票代码
   */
  function startMinutePolling(code: string) {
    stopMinutePolling()
    // Don't fetch immediately — fetchDetail already provides minute_data
    minuteTimer = setInterval(() => {
      const now = new Date()
      const day = now.getDay()
      if (day === 0 || day === 6) return // Skip weekends
      fetchMinuteData(code)
    }, STOCK_MINUTE_POLL_INTERVAL_MS)
    quoteTimer = setInterval(() => {
      const now = new Date()
      const day = now.getDay()
      if (day === 0 || day === 6) return
      void refreshQuote(code)
    }, STOCK_QUOTE_POLL_INTERVAL_MS)
  }

  /** 停止分时和行情轮询 */
  function stopMinutePolling() {
    if (minuteTimer) {
      clearInterval(minuteTimer)
      minuteTimer = null
    }
    if (quoteTimer) {
      clearInterval(quoteTimer)
      quoteTimer = null
    }
  }

  /** 重置 store 状态 */
  function reset() {
    detail.value = null
    loading.value = false
    error.value = null
    minuteData.value = []
    minuteLoading.value = false
    stopMinutePolling()
  }

  return {
    detail,
    loading,
    error,
    minuteData,
    minuteLoading,
    fetchDetail,
    fetchMinuteData,
    startMinutePolling,
    stopMinutePolling,
    reset,
  }
})
