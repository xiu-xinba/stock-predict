import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { PredictionData, FundItem, FundFilters } from '@/types'
import { predictFund, searchFunds, fetchFundFilters } from '@/api/predict'
import { CancelError } from '@/api/index'

export const usePredictionStore = defineStore('prediction', () => {
  const fundCode = ref('')
  const fundName = ref('')
  const prediction = ref<PredictionData | null>(null)
  const searchResults = ref<FundItem[]>([])
  const searchTotal = ref(0)
  const searchPage = ref(1)
  const searchSize = ref(20)
  const filters = ref<FundFilters>({ types: [], companies: [], risk_levels: [] })
  const loading = ref(false)
  const error = ref('')

  let searchSeq = 0
  let predictSeq = 0

  async function search(keyword: string, page: number = 1, filterParams?: {
    type?: string
    company?: string
    risk_level?: string
    sort_by?: string
    sort_order?: string
  }) {
    if (!keyword.trim() && !filterParams) {
      searchResults.value = []
      searchTotal.value = 0
      searchPage.value = 1
      return
    }
    const seq = ++searchSeq
    try {
      const res = await searchFunds(keyword, page, searchSize.value, filterParams)
      if (seq !== searchSeq) return
      searchResults.value = res.data?.items ?? []
      searchTotal.value = res.data?.total ?? 0
      searchPage.value = page
    } catch (e: unknown) {
      if (seq !== searchSeq) return
      if (e instanceof CancelError) return
      searchResults.value = []
      searchTotal.value = 0
      searchPage.value = 1
    }
  }

  async function loadFilters() {
    try {
      const res = await fetchFundFilters()
      if (res.data) {
        filters.value = res.data
      }
    } catch {
      // Silently fail - filters are optional
    }
  }

  async function predict(code: string) {
    if (!/^\d{6}$/.test(code)) {
      error.value = '请输入6位基金代码'
      return
    }
    const seq = ++predictSeq
    loading.value = true
    error.value = ''
    prediction.value = null
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

  return { fundCode, fundName, prediction, searchResults, searchTotal, searchPage, searchSize, filters, loading, error, search, loadFilters, predict }
})
