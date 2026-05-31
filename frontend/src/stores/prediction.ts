import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { PredictionData, StockPredictionData } from '@/types'
import { predictFund } from '@/api/predict'
import { predictStock } from '@/api/stock'
import { CancelError } from '@/api/index'

export const usePredictionStore = defineStore('prediction', () => {
  const fundCode = ref('')
  const fundName = ref('')
  const prediction = ref<PredictionData | null>(null)
  const loading = ref(false)
  const error = ref('')

  const stockPrediction = ref<StockPredictionData | null>(null)
  const stockLoading = ref(false)
  const stockError = ref('')

  let predictSeq = 0

  async function predict(code: string) {
    if (!/^\d{6}$/.test(code)) {
      error.value = '请输入6位基金代码'
      return
    }
    const seq = ++predictSeq
    loading.value = true
    error.value = ''
    fundCode.value = code
    try {
      const res = await predictFund(code)
      if (seq !== predictSeq) return
      if (res.code === 0 && res.data) {
        prediction.value = res.data
        fundName.value = res.data.fund_name
      } else {
        error.value = res.message || '预测失败'
      }
    } catch (e: unknown) {
      if (seq !== predictSeq) return
      if (e instanceof CancelError) return
      error.value = e instanceof Error ? e.message : '网络错误，请稍后重试'
    } finally {
      if (seq === predictSeq) {
        loading.value = false
      }
    }
  }

  let stockPredictSeq = 0

  async function predictStockAction(code: string) {
    if (!code) return
    const seq = ++stockPredictSeq
    stockLoading.value = true
    stockError.value = ''
    try {
      const res = await predictStock(code)
      if (seq !== stockPredictSeq) return
      if (res.code === 0 && res.data) {
        stockPrediction.value = res.data
      } else {
        stockError.value = res.message || '预测失败'
      }
    } catch (e: unknown) {
      if (seq !== stockPredictSeq) return
      if (e instanceof CancelError) return
      stockError.value = e instanceof Error ? e.message : '网络错误，请稍后重试'
    } finally {
      if (seq === stockPredictSeq) {
        stockLoading.value = false
      }
    }
  }

  return { fundCode, fundName, prediction, loading, error, predict, stockPrediction, stockLoading, stockError, predictStockAction }
})
