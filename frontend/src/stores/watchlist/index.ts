import { computed } from 'vue'
import { defineStore } from 'pinia'
import { useFundWatchlistStore } from './fundWatchlist'
import { useStockWatchlistStore } from './stockWatchlist'

export { useFundWatchlistStore } from './fundWatchlist'
export { useStockWatchlistStore } from './stockWatchlist'

export const useWatchlistStore = defineStore('watchlist', () => {
  const fundStore = useFundWatchlistStore()
  const stockStore = useStockWatchlistStore()

  const loading = computed(() => fundStore.loading || stockStore.loading)
  const error = computed(() => fundStore.error || stockStore.error)
  const lastRefresh = computed(() => fundStore.lastRefresh || stockStore.lastRefresh)

  return {
    items: fundStore.items,
    stockItems: stockStore.stockItems,
    sortedItems: fundStore.sortedItems,
    directionCounts: fundStore.directionCounts,
    loading,
    error,
    lastRefresh,
    sortBy: fundStore.sortBy,
    sortOrder: fundStore.sortOrder,
    addItem: fundStore.addItem,
    removeItem: fundStore.removeItem,
    isInWatchlist: fundStore.isInWatchlist,
    addStockItem: stockStore.addStockItem,
    removeStockItem: stockStore.removeStockItem,
    isInStockWatchlist: stockStore.isInStockWatchlist,
    setSort: fundStore.setSort,
    refreshQuotes: fundStore.refreshQuotes,
    refreshStockQuotes: stockStore.refreshStockQuotes,
  }
})
