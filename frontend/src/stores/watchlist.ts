import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'
import type { WatchlistItem } from '@/types'
import { fetchWatchlistQuotes } from '@/api/watchlist'
import { CancelError } from '@/api/index'
import { ElMessage } from 'element-plus'

const STORAGE_KEY = 'fund-watchlist'
const MAX_WATCHLIST_ITEMS = 50
type AddItemResult = 'added' | 'duplicate' | 'limit'

function loadFromStorage(): WatchlistItem[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    const parsed = raw ? JSON.parse(raw) : []
    return Array.isArray(parsed) ? parsed.slice(0, MAX_WATCHLIST_ITEMS) : []
  } catch {
    return []
  }
}

function saveToStorage(items: WatchlistItem[]) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(items))
  } catch {
    // 隐私模式、存储配额已满或用户禁用本地存储时静默忽略
  }
}

export const useWatchlistStore = defineStore('watchlist', () => {
  const items = ref<WatchlistItem[]>(loadFromStorage())
  const loading = ref(false)
  const lastRefresh = ref<string | null>(null)
  const sortBy = ref<'change_pct' | 'estimated_nav' | 'fund_name' | 'added_at'>('added_at')
  const sortOrder = ref<'asc' | 'desc'>('desc')

  let saveTimer: ReturnType<typeof setTimeout> | null = null
  let refreshSeq = 0
  watch(items, (newItems) => {
    if (saveTimer) clearTimeout(saveTimer)
    saveTimer = setTimeout(() => saveToStorage(newItems), 300)
  }, { deep: true })

  const sortedItems = computed(() => {
    const sorted = [...items.value]
    sorted.sort((a, b) => {
      let cmp = 0
      switch (sortBy.value) {
        case 'change_pct':
          cmp = a.change_pct - b.change_pct
          break
        case 'estimated_nav':
          cmp = a.estimated_nav - b.estimated_nav
          break
        case 'fund_name':
          cmp = a.fund_name.localeCompare(b.fund_name, 'zh-CN')
          break
        case 'added_at':
          cmp = a.added_at - b.added_at
          break
      }
      return sortOrder.value === 'asc' ? cmp : -cmp
    })
    return sorted
  })

  const directionCounts = computed(() => {
    let up = 0, down = 0, flat = 0
    for (const i of items.value) {
      if (i.direction === 'up') up++
      else if (i.direction === 'down') down++
      else flat++
    }
    return { up, down, flat }
  })

  function addItem(fund: { fund_code: string; fund_name: string; fund_type: string }): AddItemResult {
    if (items.value.some((i) => i.fund_code === fund.fund_code)) return 'duplicate'
    if (items.value.length >= MAX_WATCHLIST_ITEMS) return 'limit'
    const item: WatchlistItem = {
      fund_code: fund.fund_code,
      fund_name: fund.fund_name,
      fund_type: fund.fund_type,
      estimated_nav: 0,
      change_pct: 0,
      direction: 'flat',
      added_at: Date.now(),
      quote_date: '',
      quote_source: '',
    }
    items.value.push(item)
    // Immediately fetch real-time data for the newly added fund
    refreshQuotes()
    return 'added'
  }

  function removeItem(fundCode: string) {
    const idx = items.value.findIndex((i) => i.fund_code === fundCode)
    if (idx !== -1) {
      items.value.splice(idx, 1)
      return true
    }
    return false
  }

  function isInWatchlist(fundCode: string) {
    return items.value.some((i) => i.fund_code === fundCode)
  }

  function setSort(field: typeof sortBy.value) {
    if (sortBy.value === field) {
      sortOrder.value = sortOrder.value === 'asc' ? 'desc' : 'asc'
    } else {
      sortBy.value = field
      sortOrder.value = 'desc'
    }
  }

  async function refreshQuotes() {
    if (items.value.length === 0) return
    // Use sequence number to prevent stale results instead of blocking
    const seq = ++refreshSeq
    const codes = items.value.map((i) => i.fund_code)
    try {
      loading.value = true
      const res = await fetchWatchlistQuotes(codes)
      if (seq !== refreshSeq) return // Stale result, discard
      if (res.code === 0 && res.data) {
        const quoteMap = new Map(res.data.map((q) => [q.fund_code, q]))
        for (const item of items.value) {
          const quote = quoteMap.get(item.fund_code)
          if (quote) {
            item.estimated_nav = quote.estimated_nav
            item.change_pct = quote.change_pct
            item.direction = quote.direction
            item.quote_date = quote.quote_date
            item.quote_source = quote.quote_source
          }
        }
        lastRefresh.value = new Date().toLocaleTimeString('zh-CN')
      }
    } catch (e: unknown) {
      if (e instanceof CancelError) {
        // Cancelled by a newer request — loading will be managed by the newer request
        return
      }
      ElMessage.error('自选基金数据刷新失败，请稍后重试')
    } finally {
      // Only reset loading if this is the latest request
      // If a newer request superseded this one, it owns the loading state
      if (seq === refreshSeq) {
        loading.value = false
      }
    }
  }

  return {
    items,
    sortedItems,
    directionCounts,
    loading,
    lastRefresh,
    sortBy,
    sortOrder,
    addItem,
    removeItem,
    isInWatchlist,
    setSort,
    refreshQuotes,
  }
})
