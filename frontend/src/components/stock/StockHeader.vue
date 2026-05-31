<script setup lang="ts">
import { computed } from 'vue'
import { useWatchlistStore } from '@/stores/watchlist'
import { formatSignedPct, formatVolume } from '@/utils/format'
import AssetHeader from '@/components/common/AssetHeader.vue'
import type { StockBasicInfo, StockQuote } from '@/types'

defineOptions({ name: 'StockHeader' })

const props = defineProps<{
  basic: StockBasicInfo
  quote: StockQuote
}>()

const watchlistStore = useWatchlistStore()

const isInWatchlist = computed(() =>
  watchlistStore.isInStockWatchlist(props.basic.stock_code)
)

const infoItems = computed(() => {
  const items: Array<{ label: string; value: string }> = []
  items.push({ label: '开盘', value: (props.quote.open || 0).toFixed(2) })
  items.push({ label: '最高', value: (props.quote.high || 0).toFixed(2) })
  items.push({ label: '最低', value: (props.quote.low || 0).toFixed(2) })
  items.push({ label: '昨收', value: (props.quote.prev_close || 0).toFixed(2) })
  items.push({ label: '成交量', value: formatVolume(props.quote.volume) })
  items.push({ label: '成交额', value: formatVolume(props.quote.amount) })
  items.push({ label: '换手率', value: props.quote.turnover_rate != null ? props.quote.turnover_rate.toFixed(2) + '%' : '--' })
  if (props.quote.quote_time) items.push({ label: '数据时间', value: props.quote.quote_time })
  return items
})

const badges = computed(() => {
  const result: Array<{ text: string; type: 'primary' | 'secondary' }> = []
  result.push({ text: props.basic.market, type: 'primary' })
  if (props.basic.industry) result.push({ text: props.basic.industry, type: 'secondary' })
  return result
})

function toggleWatchlist() {
  if (isInWatchlist.value) {
    watchlistStore.removeStockItem(props.basic.stock_code)
  } else {
    watchlistStore.addStockItem({
      stock_code: props.basic.stock_code,
      stock_name: props.basic.stock_name,
      industry: props.basic.industry || '',
      market: props.basic.market,
    })
  }
}
</script>

<template>
  <AssetHeader
    :name="basic.stock_name"
    :code="basic.stock_code"
    :price="(quote.price || 0).toFixed(2)"
    :change="formatSignedPct(quote.change_pct, 2)"
    :change-percent="quote.change_pct"
    :is-up="quote.change_pct > 0"
    :info-items="infoItems"
    :is-in-watchlist="isInWatchlist"
    :watchlist-loading="false"
    :grid-columns="4"
    :badges="badges"
    live-dot-title="实时行情"
    @toggle-watchlist="toggleWatchlist"
  />
</template>
