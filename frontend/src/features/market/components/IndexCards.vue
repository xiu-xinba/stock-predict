<template>
  <div class="index-cards">
    <div
      v-for="idx in indices"
      :key="idx.code"
      :class="['index-card', 'card', { selected: idx.code === selectedIndex }]"
      @click="emit('select', idx.code)"
    >
      <div class="index-card-head">
        <span class="index-card-name">{{ idx.name }}</span>
        <span :class="['market-badge', idx.market]">{{ marketLabel(idx.market) }}</span>
      </div>
      <div :class="['index-card-value', idx.change_pct >= 0 ? 'up' : 'down']">
        {{ formatValue(idx.value) }}
      </div>
      <div class="index-card-change">
        <span :class="['index-card-delta', idx.change_pct >= 0 ? 'up' : 'down']">
          {{ idx.change > 0 ? '+' : '' }}{{ idx.change.toFixed(2) }}
        </span>
        <span :class="['pct-badge', idx.change_pct >= 0 ? 'up' : 'down']">
          {{ idx.change_pct > 0 ? '+' : '' }}{{ idx.change_pct.toFixed(2) }}%
        </span>
      </div>
      <div class="index-card-sparkline">
        <IndexMinuteChart
          :code="idx.code"
          :minute-data="minuteData.get(idx.code) ?? []"
          :quote="idx"
          :height="40"
          :compact="true"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
/** 指数概览卡片条组件，水平滚动展示主要指数快速概览 */
import type { MarketIndex, IndexMinutePoint } from '@/features/market/types'
import { formatValue } from '@/shared/utils/format'
import IndexMinuteChart from './IndexMinuteChart.vue'

defineOptions({ name: 'IndexCards' })

defineProps<{
  indices: MarketIndex[]
  selectedIndex: string
  minuteData: Map<string, IndexMinutePoint[]>
}>()

const emit = defineEmits<{
  select: [code: string]
}>()

function marketLabel(market: string): string {
  switch (market) {
    case 'cn':
      return 'A股'
    case 'sh':
      return '上证'
    case 'sz':
      return '深证'
    case 'hk':
      return '港股'
    case 'us':
      return '美股'
    default:
      return market
  }
}
</script>

<style scoped>
.index-cards {
  display: flex;
  gap: var(--sp-3);
  overflow-x: auto;
  scroll-snap-type: x mandatory;
  -webkit-overflow-scrolling: touch;
  scrollbar-width: none;
}

.index-cards::-webkit-scrollbar {
  display: none;
}

.index-card {
  flex: 0 0 auto;
  width: 168px;
  padding: var(--sp-3);
  cursor: pointer;
  scroll-snap-align: start;
  transition:
    transform var(--transition-spring),
    border-color var(--transition-spring),
    background-color var(--transition-spring),
    box-shadow var(--transition-spring);
}

.index-card:hover {
  transform: translateY(-3px);
  border-color: var(--color-brand-muted);
  box-shadow:
    0 4px 16px color-mix(in srgb, var(--color-brand) 12%, transparent),
    var(--shadow-ambient);
}

.index-card:active {
  transform: translateY(-1px) scale(0.98);
}

.index-card.selected {
  border-left: 3px solid var(--color-brand);
  background: var(--color-brand-soft);
}

.index-card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--sp-1_5);
  margin-bottom: var(--sp-1_5);
}

.index-card-name {
  overflow: hidden;
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.index-card-value {
  font-family: var(--font-mono);
  font-size: var(--fs-lg);
  font-weight: var(--fw-bold);
  font-variant-numeric: tabular-nums;
  letter-spacing: var(--ls-tight);
  line-height: var(--lh-tight);
}

.index-card-value.up {
  color: var(--color-up);
}

.index-card-value.down {
  color: var(--color-down);
}

.index-card-change {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  margin-top: var(--sp-1);
  margin-bottom: var(--sp-2);
}

.index-card-delta {
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  font-variant-numeric: tabular-nums;
  letter-spacing: var(--ls-tight);
}

.index-card-delta.up {
  color: var(--color-up);
}

.index-card-delta.down {
  color: var(--color-down);
}

.index-card-sparkline {
  margin: 0 calc(var(--sp-1) * -1);
}

@media (min-width: 1024px) {
  .index-cards {
    overflow-x: visible;
    scroll-snap-type: none;
  }

  .index-card {
    flex: 1 1 0;
    min-width: 0;
    max-width: 220px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .index-card {
    transition-duration: 0.01ms !important;
  }
}
</style>
