/** @module watchlist/store/stockWatchlist — 股票自选 Pinia store */
import { defineStore } from 'pinia'
import { ref, watch } from 'vue'
import { CancelError } from '@/shared/api/client'
import { fetchStockWatchlistQuotes } from '../api/watchlist'
import type { StockWatchlistItem, StockWatchlistQuote } from '../types'

const STOCK_STORAGE_KEY = 'stock-watchlist'
const MAX_WATCHLIST_ITEMS = 50
const STOCK_QUOTE_REFRESH_INTERVAL_MS = 3000
type AddItemResult = 'added' | 'duplicate' | 'limit'

function loadStockFromStorage(): StockWatchlistItem[] {
  try {
    const raw = localStorage.getItem(STOCK_STORAGE_KEY)
    const parsed = raw ? JSON.parse(raw) : []
    return Array.isArray(parsed) ? parsed.slice(0, MAX_WATCHLIST_ITEMS) : []
  } catch {
    return []
  }
}

function saveStockToStorage(items: StockWatchlistItem[]) {
  try {
    localStorage.setItem(STOCK_STORAGE_KEY, JSON.stringify(items))
  } catch {
    // Watchlist persistence is optional; keep in-memory state usable.
  }
}

/** useStockWatchlistStore - 股票自选 store */
export const useStockWatchlistStore = defineStore('stockWatchlist', () => {
  const stockItems = ref<StockWatchlistItem[]>(loadStockFromStorage())
  const loading = ref(false)
  const error = ref<string | null>(null)
  const lastRefresh = ref<string | null>(null)
  const lastStockQuoteRefresh = ref<number>(0)

  let stockSaveTimer: ReturnType<typeof setTimeout> | null = null

  watch(
    stockItems,
    (newItems) => {
      if (stockSaveTimer) clearTimeout(stockSaveTimer)
      stockSaveTimer = setTimeout(() => saveStockToStorage(newItems), 300)
    },
    { deep: true },
  )

  /**
   * 添加股票到自选
   * @param stock - 股票基本信息
   * @returns 添加结果：'added' | 'duplicate' | 'limit'
   */
  function addStockItem(stock: {
    stock_code: string
    stock_name: string
    industry: string
    market: string
  }): AddItemResult {
    if (stockItems.value.some((i) => i.stock_code === stock.stock_code)) return 'duplicate'
    if (stockItems.value.length >= MAX_WATCHLIST_ITEMS) return 'limit'
    const item: StockWatchlistItem = {
      stock_code: stock.stock_code,
      stock_name: stock.stock_name,
      market: stock.market || '',
      industry: stock.industry || '',
      list_date: '',
      total_shares: 0,
      float_shares: 0,
      current_price: 0,
      change_pct: 0,
      volume: 0,
      amount: 0,
      turnover_rate: 0,
      pe_ratio: 0,
      pb_ratio: 0,
      total_mv: 0,
      pinyin: '',
    }
    stockItems.value.push(item)
    return 'added'
  }

  /**
   * 从自选移除股票
   * @param stockCode - 股票代码
   * @returns 是否成功移除
   */
  function removeStockItem(stockCode: string) {
    const idx = stockItems.value.findIndex((i) => i.stock_code === stockCode)
    if (idx !== -1) {
      stockItems.value.splice(idx, 1)
      return true
    }
    return false
  }

  /**
   * 判断股票是否在自选中
   * @param stockCode - 股票代码
   * @returns 是否在自选中
   */
  function isInStockWatchlist(stockCode: string) {
    return stockItems.value.some((i) => i.stock_code === stockCode)
  }

  /** 刷新全部股票自选行情 */
  async function refreshStockQuotes() {
    if (stockItems.value.length === 0) return
    if (
      lastStockQuoteRefresh.value > 0 &&
      Date.now() - lastStockQuoteRefresh.value < STOCK_QUOTE_REFRESH_INTERVAL_MS
    )
      return
    const codes = stockItems.value.map((i) => i.stock_code)
    try {
      loading.value = true
      error.value = null
      const res = await fetchStockWatchlistQuotes(codes)
      if (res.code === 0 && res.data) {
        const quoteMap = new Map<string, StockWatchlistQuote>(Object.entries(res.data))
        for (const item of stockItems.value) {
          const quote = quoteMap.get(item.stock_code)
          if (quote) {
            item.current_price = quote.price ?? 0
            item.change_pct = quote.change_pct ?? 0
          }
        }
        lastRefresh.value = new Date().toLocaleTimeString('zh-CN')
        lastStockQuoteRefresh.value = Date.now()
      }
    } catch (e: unknown) {
      if (e instanceof CancelError) return
      error.value = '股票数据刷新失败，请稍后重试'
    } finally {
      loading.value = false
    }
  }

  return {
    stockItems,
    loading,
    error,
    lastRefresh,
    lastStockQuoteRefresh,
    addStockItem,
    removeStockItem,
    isInStockWatchlist,
    refreshStockQuotes,
  }
})
