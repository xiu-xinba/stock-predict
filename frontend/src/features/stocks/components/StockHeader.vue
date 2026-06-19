<script setup lang="ts">
/** 股票头部信息卡片，展示股票名称、价格、涨跌幅、自选操作及快速指标条。
 * 快速指标条精简为 5 个核心指标（总市值、市盈率、市净率、振幅、委比），
 * 流通市值移入 CompanyInfo 组件避免重复。
 * 主价格使用 useAnimatedNumber 实现数字滚动，刷新时根据涨跌方向闪烁背景色。
 */
import { computed, ref, watch } from 'vue'
import { useWatchlistStore } from '@/features/watchlist'
import { formatSignedPct, formatVolume } from '@/shared/utils/format'
import { useAnimatedNumber } from '@/shared/composables/useAnimatedNumber'
import AssetHeader from '@/shared/components/AssetHeader.vue'
import type { StockBasicInfo, StockQuote } from '@/features/stocks/types'

defineOptions({ name: 'StockHeader' })

const props = defineProps<{
  basic: StockBasicInfo
  quote: StockQuote
  financials?: {
    pe_ratio?: number | null
    pb_ratio?: number | null
  } | null
}>()

const watchlistStore = useWatchlistStore()

const isInWatchlist = computed(() => watchlistStore.isInStockWatchlist(props.basic.stock_code))

// 快速指标：振幅
const amplitude = computed(() => {
  const { high, low, prev_close } = props.quote
  if (!prev_close) return null
  return (((high - low) / prev_close) * 100).toFixed(2)
})

// 快速指标：总市值
const totalMarketCap = computed(() => {
  const shares = props.basic.total_shares
  if (!shares || !props.quote.price) return null
  return formatVolume(shares * props.quote.price)
})

// 快速指标：委比（基于买一/卖一价近似）
const weiBi = computed(() => {
  const { bid_price, ask_price } = props.quote
  if (!bid_price || !ask_price) return null
  const diff = bid_price - ask_price
  const sum = bid_price + ask_price
  if (sum === 0) return null
  return ((diff / sum) * 100).toFixed(2)
})

const infoItems = computed(() => {
  const items: Array<{ label: string; value: string }> = []
  items.push({ label: '开盘', value: (props.quote.open || 0).toFixed(2) })
  items.push({ label: '最高', value: (props.quote.high || 0).toFixed(2) })
  items.push({ label: '最低', value: (props.quote.low || 0).toFixed(2) })
  items.push({ label: '昨收', value: (props.quote.prev_close || 0).toFixed(2) })
  items.push({ label: '成交量', value: formatVolume(props.quote.volume) })
  items.push({ label: '成交额', value: formatVolume(props.quote.amount) })
  items.push({
    label: '换手率',
    value: props.quote.turnover_rate != null ? props.quote.turnover_rate.toFixed(2) + '%' : '--',
  })
  if (props.quote.quote_time) items.push({ label: '数据时间', value: props.quote.quote_time })
  return items
})

// 快速指标条：精简为 5 个核心指标（流通市值移入 CompanyInfo）
const quickStats = computed(() => {
  const stats: Array<{ label: string; value: string; dir?: 'up' | 'down' | 'flat' }> = []
  if (totalMarketCap.value) stats.push({ label: '总市值', value: totalMarketCap.value })
  if (props.financials?.pe_ratio != null) {
    stats.push({ label: '市盈率', value: props.financials.pe_ratio.toFixed(2) })
  }
  if (props.financials?.pb_ratio != null) {
    stats.push({ label: '市净率', value: props.financials.pb_ratio.toFixed(2) })
  }
  if (amplitude.value) stats.push({ label: '振幅', value: amplitude.value + '%' })
  if (weiBi.value) {
    const v = parseFloat(weiBi.value)
    stats.push({
      label: '委比',
      value: (v >= 0 ? '+' : '') + weiBi.value + '%',
      dir: v > 0 ? 'up' : v < 0 ? 'down' : 'flat',
    })
  }
  return stats
})

const badges = computed(() => {
  const result: Array<{ text: string; type: 'primary' | 'secondary' }> = []
  result.push({ text: props.basic.market, type: 'primary' })
  if (props.basic.industry) result.push({ text: props.basic.industry, type: 'secondary' })
  return result
})

// === 数字滚动 + 涨跌闪烁 ===
const priceRef = computed(() => props.quote.price || 0)
const animatedPrice = useAnimatedNumber(priceRef, 400)
const displayPrice = computed(() => animatedPrice.value.toFixed(2))

// 涨跌闪烁：监听价格变化方向
const flashClass = ref('')
let flashTimer: ReturnType<typeof setTimeout> | null = null
watch(
  () => props.quote.price,
  (newPrice, oldPrice) => {
    if (newPrice == null || oldPrice == null || newPrice === oldPrice) return
    if (flashTimer) clearTimeout(flashTimer)
    flashClass.value = newPrice > oldPrice ? 'price-flash-up' : 'price-flash-down'
    flashTimer = setTimeout(() => {
      flashClass.value = ''
    }, 300)
  },
)

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
  <div class="stock-header-wrap">
    <AssetHeader
      :name="basic.stock_name"
      :code="basic.stock_code"
      :price="displayPrice"
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
    <div v-if="quickStats.length" class="quick-stats-bar fade-slide-up" style="--delay: 1">
      <div v-for="(stat, idx) in quickStats" :key="idx" class="stat-chip" :class="flashClass">
        <span class="stat-label">{{ stat.label }}</span>
        <span class="stat-value" :class="stat.dir ? `text-${stat.dir}` : ''">{{ stat.value }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.fade-slide-up {
  animation: fade-slide-up 0.5s var(--ease-out-expo) both;
  animation-delay: calc(var(--delay, 0) * 60ms);
}

.quick-stats-bar {
  display: flex;
  flex-wrap: wrap;
  gap: var(--sp-1_5);
  padding: var(--sp-3) var(--sp-4);
  border-radius: var(--radius-lg);
  background: var(--color-bg-card);
  border: 1px solid var(--color-border);
  margin-top: var(--sp-3);
  box-shadow: var(--shadow-sm);
}

.stat-chip {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  padding: var(--sp-2) var(--sp-3);
  border-radius: var(--radius-md);
  background: var(--color-bg-elevated);
  min-width: 76px;
  flex: 1;
  transition:
    transform var(--transition-fast),
    background var(--transition-fast);
}

.stat-chip:hover {
  transform: translateY(-1px);
  background: var(--color-bg-hover);
}

.stat-label {
  font-size: var(--fs-xs);
  color: var(--color-text-tertiary);
  letter-spacing: var(--ls-wide);
  white-space: nowrap;
}

.stat-value {
  font-size: var(--fs-base);
  font-weight: var(--fw-semibold);
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  color: var(--color-text-primary);
  white-space: nowrap;
}

.text-up {
  color: var(--color-up);
}
.text-down {
  color: var(--color-down);
}
.text-flat {
  color: var(--color-flat);
}

@media (max-width: 768px) {
  .stat-chip {
    min-width: 64px;
    padding: var(--sp-1) var(--sp-2);
  }
  .stat-value {
    font-size: var(--fs-sm);
  }
}
</style>
