/** @module search/store — 搜索模块 Pinia store */
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { FundFilters, FundItem } from '@/features/funds'
import type { StockFilters, StockItem } from '@/features/stocks'
import { unifiedSearch, fetchFundFilters, fetchStockFilters } from '@/features/search/api/search'
import { CancelError } from '@/shared/api/client'

/** 搜索标签页类型 */
export type SearchTab = 'all' | 'funds' | 'stocks'

/** 搜索历史条目 */
export interface HistoryEntry {
  /** 搜索关键词 */
  keyword: string
  /** 搜索时间戳 */
  timestamp: number
  /** 搜索标签页 */
  type: SearchTab
}

const STORAGE_KEY = 'search_history_v2'
const MAX_HISTORY = 30

/** useSearchStore - 搜索 store */
export const useSearchStore = defineStore('search', () => {
  const query = ref('')
  const activeTab = ref<SearchTab>('all')
  const fundResults = ref<FundItem[]>([])
  const fundTotal = ref(0)
  const stockResults = ref<StockItem[]>([])
  const stockTotal = ref(0)
  const suggestions = ref<string[]>([])
  const page = ref(1)
  const size = ref(10)
  const loading = ref(false)
  const error = ref(false)
  const fundFilters = ref<FundFilters | null>(null)
  const stockFilters = ref<StockFilters | null>(null)
  const filtersLoading = ref(false)
  const history = ref<HistoryEntry[]>([])

  const hasResults = computed(() => fundResults.value.length > 0 || stockResults.value.length > 0)

  let searchSeq = 0

  /** 从 localStorage 加载搜索历史 */
  function loadHistory() {
    try {
      const raw = localStorage.getItem(STORAGE_KEY)
      if (raw) {
        const parsed = JSON.parse(raw)
        if (Array.isArray(parsed) && parsed.length > 0 && typeof parsed[0] === 'object') {
          history.value = parsed
        } else {
          const migrated = (parsed as string[]).map((k: string) => ({
            keyword: k,
            timestamp: Date.now(),
            type: 'all' as SearchTab,
          }))
          history.value = migrated
          persistHistory()
        }
      }
    } catch {
      // Ignore invalid or inaccessible persisted search history.
    }
  }

  /** 持久化搜索历史到 localStorage */
  function persistHistory() {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(history.value))
    } catch {
      // Search history is optional; private browsing may block localStorage.
    }
  }

  /**
   * 保存搜索关键词到历史
   * @param keyword - 搜索关键词
   * @param type - 搜索标签页，默认 'all'
   */
  function saveHistory(keyword: string, type: SearchTab = 'all') {
    const k = keyword.trim()
    if (!k) return
    const list = history.value.filter((h) => h.keyword !== k)
    list.unshift({ keyword: k, timestamp: Date.now(), type })
    if (list.length > MAX_HISTORY) list.length = MAX_HISTORY
    history.value = list
    persistHistory()
  }

  /**
   * 删除指定关键词的搜索历史
   * @param keyword - 要删除的关键词
   */
  function removeHistory(keyword: string) {
    history.value = history.value.filter((h) => h.keyword !== keyword)
    persistHistory()
  }

  /** 清空全部搜索历史 */
  function clearHistory() {
    history.value = []
    try {
      localStorage.removeItem(STORAGE_KEY)
    } catch {
      // Search history is optional; private browsing may block localStorage.
    }
  }

  /**
   * 执行搜索
   * @param keyword - 搜索关键词，默认使用当前 query
   * @param pageNum - 页码，不传则保持当前页
   * @param shouldSaveHistory - 是否保存搜索历史，默认 true；防抖实时搜索传 false
   */
  async function search(keyword?: string, pageNum?: number, shouldSaveHistory = true) {
    const q = (keyword ?? query.value).trim()
    if (!q) return

    if (pageNum !== undefined) page.value = pageNum
    query.value = q

    const seq = ++searchSeq
    loading.value = true
    error.value = false

    try {
      let types: string | undefined
      if (activeTab.value === 'funds') types = 'fund'
      else if (activeTab.value === 'stocks') types = 'stock'

      const res = await unifiedSearch({
        q,
        types,
        page: page.value,
        size: size.value,
      })

      if (seq !== searchSeq) return

      if (res.code === 0 && res.data) {
        fundResults.value = res.data.funds?.items ?? []
        fundTotal.value = res.data.funds?.total ?? 0
        stockResults.value = res.data.stocks?.items ?? []
        stockTotal.value = res.data.stocks?.total ?? 0
        suggestions.value = res.data.suggestions ?? []
        if (shouldSaveHistory && (pageNum === undefined || pageNum === 1)) {
          saveHistory(q, activeTab.value)
        }
      } else {
        error.value = true
      }
    } catch (e: unknown) {
      if (seq !== searchSeq) return
      if (e instanceof CancelError) return
      error.value = true
    } finally {
      if (seq === searchSeq) loading.value = false
    }
  }

  /** 加载基金和股票筛选条件 */
  async function loadFilters() {
    if (fundFilters.value && stockFilters.value) return
    filtersLoading.value = true
    try {
      const [fundRes, stockRes] = await Promise.allSettled([
        fetchFundFilters(),
        fetchStockFilters(),
      ])
      if (fundRes.status === 'fulfilled' && fundRes.value.code === 0 && fundRes.value.data) {
        fundFilters.value = fundRes.value.data
      }
      if (stockRes.status === 'fulfilled' && stockRes.value.code === 0 && stockRes.value.data) {
        stockFilters.value = stockRes.value.data
      }
    } catch {
      // Filter metadata is non-critical for search; keep overlay usable.
    } finally {
      filtersLoading.value = false
    }
  }

  /** 重置搜索状态 */
  function reset() {
    query.value = ''
    fundResults.value = []
    fundTotal.value = 0
    stockResults.value = []
    stockTotal.value = 0
    suggestions.value = []
    page.value = 1
    loading.value = false
    error.value = false
  }

  return {
    query,
    activeTab,
    fundResults,
    fundTotal,
    stockResults,
    stockTotal,
    suggestions,
    page,
    size,
    loading,
    error,
    hasResults,
    fundFilters,
    stockFilters,
    filtersLoading,
    history,
    search,
    loadFilters,
    loadHistory,
    saveHistory,
    removeHistory,
    clearHistory,
    reset,
  }
})
