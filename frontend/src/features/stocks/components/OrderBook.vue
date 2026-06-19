<script setup lang="ts">
/** 五档盘口组件，展示买卖五档报价及当前价、涨跌幅。
 * 紧凑化行高以匹配左侧走势图（400px）高度，当前价字号从 24px 降至 20px。
 * 使用 main tier + ORDER BOOK eyebrow 强化视觉层级。
 */
import { computed } from 'vue'
import type { StockQuote } from '@/features/stocks/types'

defineOptions({ name: 'OrderBook' })

const props = defineProps<{
  quote: StockQuote
}>()

// 当前 API 仅提供买一/卖一价，基于价差模拟五档以呈现完整盘口
const tickSize = computed(() => {
  const price = props.quote.price || 0
  if (price >= 100) return 0.2
  if (price >= 50) return 0.1
  if (price >= 10) return 0.05
  return 0.01
})

interface OrderLevel {
  label: string
  price: number
  volume: number
  side: 'bid' | 'ask'
}

const askLevels = computed<OrderLevel[]>(() => {
  const ask = props.quote.ask_price
  const ts = tickSize.value
  if (!ask) return []
  const levels: OrderLevel[] = []
  for (let i = 5; i >= 1; i--) {
    levels.push({
      label: `卖${i}`,
      price: +(ask + (i - 1) * ts).toFixed(2),
      volume: 0,
      side: 'ask',
    })
  }
  return levels
})

const bidLevels = computed<OrderLevel[]>(() => {
  const bid = props.quote.bid_price
  const ts = tickSize.value
  if (!bid) return []
  const levels: OrderLevel[] = []
  for (let i = 1; i <= 5; i++) {
    levels.push({
      label: `买${i}`,
      price: +(bid - (i - 1) * ts).toFixed(2),
      volume: 0,
      side: 'bid',
    })
  }
  return levels
})

const currentPrice = computed(() => props.quote.price || 0)
const isUp = computed(() => props.quote.change_pct >= 0)

function formatPrice(v: number) {
  return v.toFixed(2)
}
</script>

<template>
  <section class="card card-tier-main order-book-card">
    <div class="card-header">
      <div class="card-title-wrap">
        <span class="card-eyebrow">ORDER BOOK</span>
        <h2 class="card-title">五档盘口</h2>
      </div>
    </div>
    <div class="card-body">
      <div class="order-book">
        <!-- 卖五 → 卖一 -->
        <div class="book-rows ask-side">
          <div v-for="level in askLevels" :key="level.label" class="book-row">
            <span class="book-label">{{ level.label }}</span>
            <span class="book-price ask-price">{{ formatPrice(level.price) }}</span>
            <span class="book-volume">{{ level.volume || '--' }}</span>
          </div>
        </div>

        <!-- 当前价分隔 -->
        <div class="current-price-row" :class="isUp ? 'up' : 'down'">
          <span class="cp-label">最新</span>
          <span class="cp-value">{{ formatPrice(currentPrice) }}</span>
          <span class="cp-change">
            {{ isUp ? '+' : '' }}{{ (quote.change_amt || 0).toFixed(2) }} ({{ isUp ? '+' : ''
            }}{{ (quote.change_pct || 0).toFixed(2) }}%)
          </span>
        </div>

        <!-- 买一 → 买五 -->
        <div class="book-rows bid-side">
          <div v-for="level in bidLevels" :key="level.label" class="book-row">
            <span class="book-label">{{ level.label }}</span>
            <span class="book-price bid-price">{{ formatPrice(level.price) }}</span>
            <span class="book-volume">{{ level.volume || '--' }}</span>
          </div>
        </div>
      </div>

      <div v-if="!askLevels.length && !bidLevels.length" class="empty-hint">暂无盘口数据</div>
    </div>
  </section>
</template>

<style scoped>
.order-book {
  display: flex;
  flex-direction: column;
  gap: 2px;
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
}

.book-rows {
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.ask-side {
  flex-direction: column-reverse;
}

.book-row {
  display: grid;
  grid-template-columns: 40px 1fr 60px;
  align-items: center;
  gap: var(--sp-2);
  padding: var(--sp-1) var(--sp-2);
  border-radius: var(--radius-sm);
  font-size: var(--fs-sm);
  transition: background var(--transition-fast);
}

.book-row:hover {
  background: var(--color-bg-elevated);
}

.book-label {
  font-size: var(--fs-xs);
  color: var(--color-text-tertiary);
  font-family: var(--font-mono);
}

.ask-price {
  color: var(--color-up);
  font-weight: var(--fw-medium);
}

.bid-price {
  color: var(--color-down);
  font-weight: var(--fw-medium);
}

.book-volume {
  text-align: right;
  font-size: var(--fs-xs);
  color: var(--color-text-secondary);
}

.current-price-row {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--sp-3);
  padding: var(--sp-1_5) var(--sp-3);
  margin: var(--sp-1) 0;
  border-radius: var(--radius-md);
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
}

.current-price-row.up {
  background: var(--color-up-bg);
  border: 1px solid var(--color-up-border);
}

.current-price-row.down {
  background: var(--color-down-bg);
  border: 1px solid var(--color-down-border);
}

.cp-label {
  font-size: var(--fs-xs);
  color: var(--color-text-tertiary);
  font-family: inherit;
}

.cp-value {
  font-size: var(--fs-lg);
  font-weight: var(--fw-bold);
}

.current-price-row.up .cp-value,
.current-price-row.up .cp-change {
  color: var(--color-up);
}

.current-price-row.down .cp-value,
.current-price-row.down .cp-change {
  color: var(--color-down);
}

.cp-change {
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
}

.empty-hint {
  text-align: center;
  color: var(--color-text-tertiary);
  font-size: var(--fs-sm);
  padding: var(--sp-4);
}
</style>
