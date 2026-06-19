/** @module market/store — 市场行情 Pinia store */
import { defineStore } from 'pinia'
import { ref, reactive, computed, watch } from 'vue'
import type {
  MarketIndex,
  FundRankingItem,
  IndexKlinePoint,
  IndexMinutePoint,
  MarketSectorItem,
  NorthboundFlow,
  StockRankingItem,
  HSGTFlowDaily,
} from '@/features/market/types'
import {
  fetchMarketIndices,
  fetchFundRanking,
  fetchIndexKline,
  fetchIndexMinute,
  fetchSectorRanking,
  fetchNorthboundFlow,
  fetchStockRanking,
  fetchHSGTHist,
} from '@/features/market/api/market'
import { ElMessage } from 'element-plus'
import { CancelError } from '@/shared/api/client'
import dayjs from 'dayjs'
import { useSettingsStore } from '@/features/settings'
import { detectIndexMarket, isMarketInSession } from '@/features/market/utils/marketTime'

const MARKET_CLOSED_INTERVAL = 30 * 60 * 1000
const MINUTE_REFRESH_INTERVAL = 60000 // 分时数据每分钟刷新

/** 合并两组分时数据，newData 覆盖同时间点，oldData 保留不同时间点，按 time 排序。
 *  如果 newData 的第一个时间点早于 oldData 的第一个时间点，说明是新一天的数据，直接返回 newData。
 */
function mergeMinuteData(
  oldData: IndexMinutePoint[],
  newData: IndexMinutePoint[],
): IndexMinutePoint[] {
  if (oldData.length === 0) return newData
  if (newData.length === 0) return oldData
  // 如果新数据开头时间比旧数据早很多（跨日），直接用新数据
  if (newData[0].time < oldData[0].time) return newData
  const byTime = new Map<string, IndexMinutePoint>()
  for (const p of oldData) {
    byTime.set(p.time, p)
  }
  for (const p of newData) {
    byTime.set(p.time, p)
  }
  const times = sortMinuteTimes(Array.from(byTime.keys()))
  return times.map((t) => byTime.get(t)!)
}

/** 按交易时间排序分钟数据。当数据跨越午夜（如美股 21:30-04:00）时，21:xx 排在 00:xx 之前 */
function sortMinuteTimes(times: string[]): string[] {
  const hasLateNight = times.some((t) => {
    const h = parseInt(t.slice(0, 2), 10)
    return h >= 21
  })
  const hasEarlyMorning = times.some((t) => {
    const h = parseInt(t.slice(0, 2), 10)
    return h <= 4
  })
  if (hasLateNight && hasEarlyMorning) {
    return times.sort((a, b) => {
      const ma = parseInt(a.slice(0, 2), 10) * 60 + parseInt(a.slice(3, 5), 10)
      const mb = parseInt(b.slice(0, 2), 10) * 60 + parseInt(b.slice(3, 5), 10)
      if (ma >= 21 * 60 && mb < 21 * 60) return -1
      if (ma < 21 * 60 && mb >= 21 * 60) return 1
      return ma - mb
    })
  }
  return times.sort()
}

/** useMarketStore - 市场行情 store */
export const useMarketStore = defineStore('market', () => {
  const settings = useSettingsStore()
  const indices = ref<MarketIndex[]>([])
  const topGainers = ref<FundRankingItem[]>([])
  const topLosers = ref<FundRankingItem[]>([])
  const stockGainers = ref<StockRankingItem[]>([])
  const stockLosers = ref<StockRankingItem[]>([])
  const loading = ref(false)
  const stockRankingLoading = ref(false)
  const lastRefresh = ref<string | null>(null)
  const error = ref<string | null>(null)
  const stockRankingError = ref<string | null>(null)
  const refreshTimer = ref<ReturnType<typeof setInterval> | null>(null)
  const lastFetchTime = ref<number>(0)
  let refCount = 0
  let abortController: AbortController | null = null
  let stockRankingAbortController: AbortController | null = null
  let sectorAbortController: AbortController | null = null
  let northboundAbortController: AbortController | null = null
  let hsgtAbortController: AbortController | null = null
  let requestSeq = 0

  const indexKline = reactive(new Map<string, IndexKlinePoint[]>())
  const indexMinute = reactive(new Map<string, IndexMinutePoint[]>())
  const indexKlineLoading = reactive(new Map<string, boolean>())
  const indexKlineError = reactive(new Map<string, string | null>())
  const indexMinuteLoading = reactive(new Map<string, boolean>())
  const indexMinuteError = reactive(new Map<string, string | null>())
  const sectors = ref<MarketSectorItem[]>([])
  const northbound = ref<NorthboundFlow | null>(null)
  const sectorLoading = ref(false)
  const sectorError = ref<string | null>(null)
  const northboundLoading = ref(false)
  const northboundError = ref<string | null>(null)
  const hsgtData = ref<HSGTFlowDaily[] | null>(null)
  const hsgtLoading = ref(false)
  const hsgtError = ref<string | null>(null)

  // 市场开闭市判断：所有 A 股指数都收盘则认为休市
  const isMarketClosed = computed(() => {
    const cnIdx = indices.value.filter((i) => ['cn', 'sh', 'sz'].includes(i.market))
    if (cnIdx.length === 0) return false // 数据未加载时不降频
    return cnIdx.every((i) => i.is_closed)
  })

  /**
   * 获取市场行情数据（指数 + 基金排行）
   * @param skipThrottle - 是否跳过节流，强制请求
   */
  async function fetchMarketData(skipThrottle = false) {
    if (loading.value && !skipThrottle) return

    const throttleMs = isMarketClosed.value ? MARKET_CLOSED_INTERVAL : 30000
    if (!skipThrottle && lastFetchTime.value > 0 && Date.now() - lastFetchTime.value < throttleMs) {
      return
    }

    // Cancel previous in-flight request
    if (abortController && loading.value) {
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
        lastRefresh.value = dayjs().format('HH:mm:ss')
        lastFetchTime.value = Date.now()
        error.value = null
      } else if (indices.value.length > 0) {
        lastRefresh.value = dayjs().format('HH:mm:ss')
        error.value = '部分数据刷新失败'
        ElMessage.warning('部分行情数据刷新失败')
      } else {
        error.value = '行情数据加载失败'
        ElMessage.error('行情数据加载失败，请稍后重试')
      }

      // 刷新 kline 和 minute 数据（有错误时重试，无错误时按需刷新）
      refreshIndexChartData()
    } finally {
      if (seq === requestSeq) {
        loading.value = false
        if (abortController === currentController) {
          abortController = null
        }
      }
    }
  }

  // 刷新指数图表数据：kline 有错误时重试，minute 按需刷新
  function refreshIndexChartData() {
    // kline：有错误或无数据时重试
    if (indexKlineError.get('000001') || !indexKline.get('000001')?.length) {
      fetchIndexKlineData('000001', 120)
    }
    // 港股/美股 kline（迷你图回退用）
    for (const code of ['hsi', 'hstech', 'dji', 'ixic', 'spx']) {
      if (indexKlineError.get(code) || !indexKline.get(code)?.length) {
        fetchIndexKlineData(code, 120)
      }
    }
    // A股 minute
    const cnIdx = indices.value.filter((i) => ['cn', 'sh', 'sz'].includes(i.market))
    for (const idx of cnIdx) {
      if (indexMinuteError.get(idx.code) || !indexMinute.get(idx.code)?.length) {
        fetchIndexMinuteData(idx.code)
      }
    }
    // 港股 minute
    for (const code of ['hsi', 'hstech']) {
      if (indexMinuteError.get(code) || !indexMinute.get(code)?.length) {
        fetchIndexMinuteData(code)
      }
    }
    // 美股 minute
    for (const code of ['dji', 'ixic', 'spx']) {
      if (indexMinuteError.get(code) || !indexMinute.get(code)?.length) {
        fetchIndexMinuteData(code)
      }
    }
  }

  /**
   * 启动行情自动刷新
   * @param interval - 可选的自定义刷新间隔（毫秒）
   */
  function startRefresh(interval?: number) {
    refCount++
    if (!refreshTimer.value) {
      const actualInterval = interval ?? currentRefreshInterval()
      if (!loading.value) fetchMarketData()
      refreshTimer.value = setInterval(() => {
        if (!loading.value) fetchMarketData()
      }, actualInterval)
    }
  }

  function currentRefreshInterval(): number {
    return isMarketClosed.value ? MARKET_CLOSED_INTERVAL : settings.refreshIntervalSeconds * 1000
  }

  // 市场状态或用户设置变化时切换刷新频率，不改变订阅引用计数。
  watch([isMarketClosed, () => settings.refreshIntervalSeconds], () => {
    if (refreshTimer.value) {
      clearInterval(refreshTimer.value)
      refreshTimer.value = setInterval(() => {
        if (!loading.value) fetchMarketData()
      }, currentRefreshInterval())
    }
  })

  /** 停止行情自动刷新 */
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

  const klineAbortControllers: Map<string, AbortController> = new Map()

  /**
   * 获取指数 K 线数据
   * @param code - 指数代码
   * @param count - 返回条数，默认 120
   */
  async function fetchIndexKlineData(code: string, count: number = 120) {
    const prev = klineAbortControllers.get(code)
    if (prev && indexKlineLoading.get(code)) prev.abort()
    const controller = new AbortController()
    klineAbortControllers.set(code, controller)
    indexKlineLoading.set(code, true)
    indexKlineError.set(code, null)
    try {
      const res = await fetchIndexKline(code, count, controller.signal)
      if (res.code === 0 && res.data) {
        indexKline.set(code, res.data)
      } else {
        indexKlineError.set(code, '指数历史数据暂不可用')
      }
    } catch (e: unknown) {
      if (e instanceof CancelError) return
      indexKlineError.set(code, '指数历史数据暂不可用')
    } finally {
      indexKlineLoading.set(code, false)
    }
  }

  const minuteAbortControllers: Map<string, AbortController> = new Map()

  const usIndexSecid: Record<string, string> = {
    dji: '100.DJIA',
    ixic: '100.NDX',
    spx: '100.SPX',
  }

  // 从实时行情数据构建简化分时图（当所有API都无法获取分钟数据时的回退方案）
  function constructSimplifiedMinuteFromQuote(code: string): IndexMinutePoint[] {
    const quote = indices.value.find((i) => i.code === code)
    if (!quote) {
      console.warn(`[minute] no quote data for ${code}, indices count: ${indices.value.length}`)
      return []
    }

    const market = detectIndexMarket(code)
    // 使用value作为prev_close的回退（虽然不太准确，但至少能显示）
    const prevClose = quote.prev_close > 0 ? quote.prev_close : quote.value
    const open = quote.open > 0 ? quote.open : quote.value
    const current = quote.value
    const high = quote.high > 0 ? quote.high : Math.max(open, current)
    const low = quote.low > 0 ? quote.low : Math.min(open, current)

    if (prevClose <= 0 || current <= 0) {
      console.warn(
        `[minute] invalid quote data for ${code}: prev_close=${prevClose}, value=${current}`,
      )
      return []
    }

    // 美股交易时段 21:30-04:00
    const now = new Date()
    const currentHour = now.getHours()
    const currentMin = now.getMinutes()
    // 如果当前时间在00:00-04:00之间，属于前一天的晚间交易
    const isAfterMidnight = currentHour < 4

    // 构建关键时间点的分时数据
    const points: IndexMinutePoint[] = []

    if (market === 'us') {
      // 美股：21:30开盘 → 当前时间
      const openTime = '21:30'
      points.push({ time: openTime, price: open, avg_price: open, volume: 0 })

      // 如果有高低点，在中间插入
      if (high > Math.max(open, current)) {
        const midTime1 = isAfterMidnight ? '23:00' : '22:30'
        points.push({ time: midTime1, price: high, avg_price: (open + high) / 2, volume: 0 })
      }
      if (low < Math.min(open, current)) {
        const midTime2 = isAfterMidnight ? '00:30' : '23:30'
        points.push({ time: midTime2, price: low, avg_price: (open + low) / 2, volume: 0 })
      }

      // 当前时间点
      const curTimeStr = `${String(currentHour).padStart(2, '0')}:${String(currentMin).padStart(2, '0')}`
      points.push({ time: curTimeStr, price: current, avg_price: (open + current) / 2, volume: 0 })
    } else {
      // A股/港股：使用开盘价和当前价
      const openTime = market === 'hk' ? '09:30' : '09:30'
      points.push({ time: openTime, price: open, avg_price: open, volume: 0 })
      const curTimeStr = `${String(currentHour).padStart(2, '0')}:${String(currentMin).padStart(2, '0')}`
      points.push({ time: curTimeStr, price: current, avg_price: (open + current) / 2, volume: 0 })
    }

    console.log(
      `[minute] constructed simplified chart for ${code}: ${points.length} points, open=${open}, current=${current}`,
    )
    return points.length >= 2 ? points : []
  }

  /**
   * 获取指数分时数据
   * @param code - 指数代码
   */
  async function fetchIndexMinuteData(code: string) {
    const prev = minuteAbortControllers.get(code)
    if (prev && indexMinuteLoading.get(code)) prev.abort()
    const controller = new AbortController()
    minuteAbortControllers.set(code, controller)
    indexMinuteLoading.set(code, true)
    indexMinuteError.set(code, null)

    const isUS = !!usIndexSecid[code]

    try {
      // 所有指数走后端API（后端已有腾讯/新浪/东方财富数据源及缓存）
      const res = await fetchIndexMinute(code, controller.signal)
      if (res.code === 0 && res.data && res.data.length > 0) {
        // 增量合并：新数据覆盖同时间点，旧数据保留不同时间点
        const existing = indexMinute.get(code) ?? []
        if (existing.length > 0) {
          const merged = mergeMinuteData(existing, res.data)
          indexMinute.set(code, merged)
        } else {
          indexMinute.set(code, res.data)
        }
        return
      }

      // 后端返回空数据时，尝试从实时行情构建简化分时图
      if (isUS) {
        const simplified = constructSimplifiedMinuteFromQuote(code)
        if (simplified.length >= 2) {
          indexMinute.set(code, simplified)
          return
        }
      }

      indexMinuteError.set(code, '指数分时数据暂不可用')
    } catch (e: unknown) {
      if (e instanceof CancelError && !isUS) return
      // 后端请求失败时，美股指数尝试从实时行情构建简化分时图
      if (isUS) {
        const simplified = constructSimplifiedMinuteFromQuote(code)
        if (simplified.length >= 2) {
          indexMinute.set(code, simplified)
          return
        }
      }
      indexMinuteError.set(code, '指数分时数据暂不可用')
    } finally {
      indexMinuteLoading.set(code, false)
    }
  }

  /** 获取上证综指 K 线数据 */
  async function fetchShanghaiCompositeKline() {
    await fetchIndexKlineData('000001', 120)
  }

  /**
   * 获取股票排行数据（涨幅榜 + 跌幅榜）
   * @param size - 每个榜单返回条数，默认 5
   */
  async function fetchStockRankingData(size: number = 5) {
    if (stockRankingAbortController && stockRankingLoading.value) {
      stockRankingAbortController.abort()
    }
    stockRankingAbortController = new AbortController()
    const signal = stockRankingAbortController.signal
    stockRankingLoading.value = true
    stockRankingError.value = null
    try {
      const [gainersResult, losersResult] = await Promise.allSettled([
        fetchStockRanking('gainers', size, signal),
        fetchStockRanking('losers', size, signal),
      ])

      let hasError = false
      if (gainersResult.status === 'fulfilled' && gainersResult.value.data) {
        stockGainers.value = gainersResult.value.data
      } else {
        stockGainers.value = []
        if (gainersResult.status === 'rejected' && !(gainersResult.reason instanceof CancelError))
          hasError = true
      }
      if (losersResult.status === 'fulfilled' && losersResult.value.data) {
        stockLosers.value = losersResult.value.data
      } else {
        stockLosers.value = []
        if (losersResult.status === 'rejected' && !(losersResult.reason instanceof CancelError))
          hasError = true
      }
      if (hasError) {
        stockRankingError.value = '股票排行数据暂不可用'
      }
    } finally {
      if (stockRankingAbortController?.signal === signal) {
        stockRankingAbortController = null
      }
      stockRankingLoading.value = false
    }
  }

  /** 获取行业板块数据 */
  async function fetchSectorData() {
    if (sectorAbortController && sectorLoading.value) sectorAbortController.abort()
    sectorAbortController = new AbortController()
    const signal = sectorAbortController.signal
    sectorLoading.value = true
    sectorError.value = null
    try {
      const res = await fetchSectorRanking(signal)
      if (signal.aborted) return
      if (res.code === 0 && res.data) {
        sectors.value = res.data
      } else {
        sectorError.value = '板块数据暂不可用'
      }
    } catch (e: unknown) {
      if (e instanceof CancelError) return
      sectorError.value = '板块数据加载失败'
    } finally {
      sectorLoading.value = false
    }
  }

  /** 获取北向资金数据 */
  async function fetchNorthboundData() {
    if (northboundAbortController && northboundLoading.value) northboundAbortController.abort()
    northboundAbortController = new AbortController()
    const signal = northboundAbortController.signal
    northboundLoading.value = true
    northboundError.value = null
    try {
      const res = await fetchNorthboundFlow(signal)
      if (signal.aborted) return
      if (res.code === 0 && res.data) {
        northbound.value = res.data
      } else {
        northboundError.value = '北向资金数据暂不可用'
      }
    } catch (e: unknown) {
      if (e instanceof CancelError) return
      northboundError.value = '北向资金数据加载失败'
    } finally {
      northboundLoading.value = false
    }
  }

  /** 获取沪深港通历史资金流向数据 */
  async function fetchHSGTData() {
    if (hsgtAbortController && hsgtLoading.value) hsgtAbortController.abort()
    hsgtAbortController = new AbortController()
    const signal = hsgtAbortController.signal
    hsgtLoading.value = true
    hsgtError.value = null
    try {
      const res = await fetchHSGTHist(365, signal)
      if (signal.aborted) return
      if (res.code === 0 && res.data) {
        hsgtData.value = res.data
      } else {
        hsgtError.value = '沪深港通数据暂不可用'
      }
    } catch (e: unknown) {
      if (e instanceof CancelError) return
      hsgtError.value = '沪深港通数据加载失败'
    } finally {
      hsgtLoading.value = false
    }
  }

  // ── 分时数据独立刷新定时器 ──
  const minuteRefreshTimer = ref<ReturnType<typeof setInterval> | null>(null)
  let minuteRefreshRefCount = 0

  /** 启动分时数据独立刷新定时器 */
  function startMinuteRefresh() {
    minuteRefreshRefCount++
    if (!minuteRefreshTimer.value) {
      // 立即刷新一次已加载的分时数据
      refreshAllMinuteData()
      minuteRefreshTimer.value = setInterval(() => {
        refreshAllMinuteData()
      }, MINUTE_REFRESH_INTERVAL)
    }
  }

  /** 停止分时数据独立刷新定时器 */
  function stopMinuteRefresh() {
    minuteRefreshRefCount = Math.max(0, minuteRefreshRefCount - 1)
    if (minuteRefreshRefCount === 0 && minuteRefreshTimer.value) {
      clearInterval(minuteRefreshTimer.value)
      minuteRefreshTimer.value = null
    }
  }

  function refreshAllMinuteData() {
    // 刷新所有已加载的指数分时数据（仅开市中的市场）
    const codes = Array.from(indexMinute.keys())
    for (const code of codes) {
      const market = detectIndexMarket(code)
      if (isMarketInSession(market)) {
        fetchIndexMinuteData(code)
      }
    }
    // 同时确保所有指数的分时数据都已加载
    const allCodes = ['000001', '399001', '399006', 'hsi', 'hstech', 'dji', 'ixic', 'spx']
    for (const code of allCodes) {
      if (!indexMinute.get(code)?.length && !indexMinuteLoading.get(code)) {
        fetchIndexMinuteData(code)
      }
    }
  }

  return {
    indices,
    topGainers,
    topLosers,
    stockGainers,
    stockLosers,
    loading,
    stockRankingLoading,
    lastRefresh,
    error,
    stockRankingError,
    indexKline,
    indexMinute,
    indexKlineLoading,
    indexKlineError,
    indexMinuteLoading,
    indexMinuteError,
    sectors,
    northbound,
    sectorLoading,
    sectorError,
    northboundLoading,
    northboundError,
    hsgtData,
    hsgtLoading,
    hsgtError,
    isMarketClosed,
    fetchMarketData,
    fetchStockRankingData,
    fetchIndexKlineData,
    fetchIndexMinuteData,
    fetchShanghaiCompositeKline,
    fetchSectorData,
    fetchNorthboundData,
    fetchHSGTData,
    startRefresh,
    stopRefresh,
    startMinuteRefresh,
    stopMinuteRefresh,
  }
})
