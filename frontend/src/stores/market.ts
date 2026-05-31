import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { MarketIndex, FundRankingItem } from '@/types'
import { fetchMarketIndices, fetchFundRanking } from '@/api/market'
import { ElMessage } from 'element-plus'
import { CancelError } from '@/api/index'

export const useMarketStore = defineStore('market', () => {
  const indices = ref<MarketIndex[]>([])
  const topGainers = ref<FundRankingItem[]>([])
  const topLosers = ref<FundRankingItem[]>([])
  const loading = ref(false)
  const lastRefresh = ref<string | null>(null)
  const error = ref<string | null>(null)
  const refreshTimer = ref<number | null>(null)
  const lastFetchTime = ref<number>(0)
  let refCount = 0
  let abortController: AbortController | null = null
  let requestSeq = 0

  async function fetchMarketData(force = false) {
    if (loading.value && !force) return

    if (!force && lastFetchTime.value > 0 && Date.now() - lastFetchTime.value < 30000) {
      return
    }

    // Cancel previous in-flight request
    if (abortController) {
      abortController.abort()
    }
    abortController = new AbortController()
    const currentController = abortController
    const signal = abortController.signal
    const seq = ++requestSeq

    loading.value = true
    try {
      let hasError = false

      // Use Promise.allSettled to run all requests in parallel and update state atomically
      const [indicesResult, gainersResult, losersResult] = await Promise.allSettled([
        fetchMarketIndices(signal),
        fetchFundRanking('gainers', 5, signal),
        fetchFundRanking('losers', 5, signal),
      ])

      // Stale check: if a new request was fired while we were waiting, discard results
      if (signal.aborted || seq !== requestSeq) return

      // Process indices
      if (indicesResult.status === 'fulfilled') {
        indices.value = indicesResult.value.data ?? []
      } else {
        if (!(indicesResult.reason instanceof CancelError)) hasError = true
      }

      // Process gainers
      if (gainersResult.status === 'fulfilled') {
        topGainers.value = gainersResult.value.data ?? []
      } else {
        if (!(gainersResult.reason instanceof CancelError)) hasError = true
      }

      // Process losers
      if (losersResult.status === 'fulfilled') {
        topLosers.value = losersResult.value.data ?? []
      } else {
        if (!(losersResult.reason instanceof CancelError)) hasError = true
      }

      if (!hasError) {
        lastRefresh.value = new Date().toLocaleTimeString('zh-CN')
        lastFetchTime.value = Date.now()
        error.value = null
      } else if (indices.value.length > 0) {
        lastRefresh.value = new Date().toLocaleTimeString('zh-CN')
        error.value = '部分数据刷新失败'
        ElMessage.warning('部分行情数据刷新失败')
      } else {
        error.value = '行情数据加载失败'
        ElMessage.error('行情数据加载失败，请稍后重试')
      }
    } finally {
      if (seq === requestSeq) {
        loading.value = false
        if (abortController === currentController) {
          abortController = null
        }
      }
    }
  }

  function startRefresh(interval: number = 30000) {
    refCount++
    if (!refreshTimer.value) {
      fetchMarketData()
      refreshTimer.value = setInterval(() => {
        if (!loading.value) fetchMarketData()
      }, interval)
    }
  }

  function stopRefresh() {
    refCount = Math.max(0, refCount - 1)
    if (refCount === 0 && refreshTimer.value) {
      clearInterval(refreshTimer.value)
      refreshTimer.value = null
      if (abortController) {
        abortController.abort()
        abortController = null
      }
    }
  }

  return {
    indices,
    topGainers,
    topLosers,
    loading,
    lastRefresh,
    error,
    fetchMarketData,
    startRefresh,
    stopRefresh,
  }
})
