<template>
  <div :class="['market-dock', { 'has-popup': expandedMarket }]">
    <div
      v-for="mkt in markets"
      :key="mkt.key"
      :class="['mkt-dock-item', mkt.key, { expanded: expandedMarket === mkt.key }]"
    >
      <div class="mkt-dock-popup" :data-state="expandedMarket === mkt.key ? 'open' : 'closed'">
        <div class="popup-inner">
          <div class="popup-head">
            <span :class="['market-badge', mkt.key]">{{ mkt.label }}</span>
            <button class="popup-close" type="button" @click.stop="expandedMarket = null">
              <svg viewBox="0 0 24 24" width="16" height="16"><path fill="currentColor" d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/></svg>
            </button>
          </div>
          <div
            v-for="(idx, i) in mkt.indices"
            :key="idx.code"
            class="popup-idx"
            :style="{ '--i': i }"
          >
            <div class="popup-idx-top">
              <span class="popup-idx-name">{{ idx.name }}</span>
              <span class="popup-idx-value">{{ formatValue(idx.value) }}</span>
              <span :class="['pct-badge', idx.change_pct >= 0 ? 'up' : 'down']" style="margin-left: auto;">
                {{ idx.change_pct > 0 ? '+' : '' }}{{ idx.change_pct.toFixed(2) }}%
              </span>
            </div>
            <div class="popup-idx-meta">
              <span :class="['popup-idx-delta', idx.change_pct >= 0 ? 'up' : 'down']">
                {{ idx.change > 0 ? '+' : '' }}{{ idx.change.toFixed(2) }}
              </span>
              <span v-if="idx.high > 0" class="popup-idx-hl">高 {{ formatValue(idx.high) }}</span>
              <span v-if="idx.low > 0" class="popup-idx-hl">低 {{ formatValue(idx.low) }}</span>
            </div>
            <div class="popup-idx-chart" :ref="(el: any) => sparkline.setChartRef(idx.code, el)"></div>
          </div>
        </div>
      </div>

      <div class="mkt-dock-main" @click="toggleExpand(mkt.key)">
        <span :class="['market-badge', mkt.key]">{{ mkt.label }}</span>
        <template v-if="mkt.primary">
          <span class="mkt-dock-name">{{ mkt.primary.name }}</span>
          <span class="mkt-dock-value">{{ formatValue(mkt.primary.value) }}</span>
          <span :class="['pct-badge', mkt.primary.change_pct >= 0 ? 'up' : 'down']">
            {{ mkt.primary.change_pct > 0 ? '+' : '' }}{{ mkt.primary.change_pct.toFixed(2) }}%
          </span>
        </template>
        <span v-else class="mkt-dock-empty">--</span>
        <span :class="['mkt-dock-expand', { open: expandedMarket === mkt.key }]">
          <svg viewBox="0 0 24 24" width="14" height="14"><path fill="currentColor" d="M7 14l5-5 5 5z"/></svg>
        </span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import { useTheme } from '@/composables/useTheme'
import { useSparkline } from '@/composables/useSparkline'
import { formatValue } from '@/utils/format'
import type { MarketIndex } from '@/types'

defineOptions({ name: 'MarketDock' })

const props = defineProps<{
  cnIndices: MarketIndex[]
  hkIndices: MarketIndex[]
  usIndices: MarketIndex[]
}>()

const expandedMarket = ref<string | null>(null)
const { isDark } = useTheme()
const sparkline = useSparkline()

const markets = computed(() => [
  { key: 'cn', label: 'A股', indices: props.cnIndices, primary: props.cnIndices[0] ?? null },
  { key: 'hk', label: '港股', indices: props.hkIndices, primary: props.hkIndices[0] ?? null },
  { key: 'us', label: '美股', indices: props.usIndices, primary: props.usIndices[0] ?? null },
])

function toggleExpand(key: string) {
  expandedMarket.value = expandedMarket.value === key ? null : key
}

watch(expandedMarket, (newKey, oldKey) => {
  if (oldKey) {
    const oldMarket = markets.value.find(m => m.key === oldKey)
    if (oldMarket) sparkline.disposeCharts(oldMarket.indices)
  }
  if (newKey) {
    const newMarket = markets.value.find(m => m.key === newKey)
    if (newMarket) nextTick(() => sparkline.initCharts(newMarket.indices))
  }
})

watch(() => [props.cnIndices, props.hkIndices, props.usIndices], () => {
  if (!expandedMarket.value) return
  const market = markets.value.find(m => m.key === expandedMarket.value)
  if (market) nextTick(() => sparkline.initCharts(market.indices))
}, { deep: true })

watch(isDark, () => {
  sparkline.disposeAll()
  if (expandedMarket.value) {
    const market = markets.value.find(m => m.key === expandedMarket.value)
    if (market) nextTick(() => sparkline.initCharts(market.indices))
  }
})
</script>

<style scoped>
.market-dock {
  position: fixed;
  bottom: 96px;
  left: 50%;
  z-index: 60;
  display: flex;
  gap: var(--sp-0_5);
  padding: var(--sp-1);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-xl);
  background: var(--color-bg-topbar);
  backdrop-filter: blur(20px) saturate(180%);
  -webkit-backdrop-filter: blur(20px) saturate(180%);
  box-shadow: var(--shadow-lg);
  transform: translateX(-50%);
}

.market-dock.has-popup .mkt-dock-item:not(.expanded) {
  opacity: 0.5;
  transition: opacity var(--transition-normal);
}

.mkt-dock-item {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  min-width: 160px;
  position: relative;
  border: 1px solid transparent;
  border-radius: var(--radius-lg);
  transition: background-color var(--transition-fast), border-color var(--transition-spring), opacity var(--transition-normal);
  cursor: pointer;
}

.mkt-dock-item:hover {
  background: var(--color-bg-hover);
}

.mkt-dock-item.expanded {
  background: var(--color-bg-hover);
  border-color: var(--color-brand-muted);
}

.mkt-dock-item.expanded .mkt-dock-main {
  transform: scale(0.98);
  transition: transform var(--transition-spring);
}

.mkt-dock-popup {
  position: absolute;
  bottom: calc(100% + 8px);
  left: 0;
  right: 0;
  min-width: 240px;
  clip-path: inset(100% 0 0 0);
  transform: scaleY(0.92) translateY(6px);
  transform-origin: bottom center;
  opacity: 0;
  pointer-events: none;
  border: 1px solid transparent;
  border-radius: var(--radius-lg);
  background: var(--color-bg-card);
  box-shadow: none;
  transition:
    clip-path var(--transition-spring),
    transform var(--transition-spring),
    opacity var(--transition-fast),
    box-shadow var(--transition-normal),
    border-color var(--transition-normal);
}

.mkt-dock-popup[data-state="open"] {
  clip-path: inset(0 0 0 0);
  transform: scaleY(1) translateY(0);
  opacity: 1;
  pointer-events: auto;
  border-color: var(--color-border);
  box-shadow: var(--shadow-lg);
  transition:
    clip-path 0.52s cubic-bezier(0.32, 0.72, 0, 1),
    transform 0.52s cubic-bezier(0.32, 0.72, 0, 1),
    opacity var(--transition-normal),
    box-shadow var(--transition-normal),
    border-color var(--transition-normal);
}

.popup-inner {
  padding: var(--sp-3);
}

.popup-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--sp-2);
  padding-bottom: var(--sp-2);
  border-bottom: 1px solid var(--color-border-light);
}

.popup-close {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-disabled);
  cursor: pointer;
  transition: background-color var(--transition-fast), color var(--transition-fast);
}

.popup-close:hover {
  background: var(--color-bg-hover);
  color: var(--color-text-primary);
}

.popup-idx {
  padding: var(--sp-2) 0;
  border-bottom: 1px solid var(--color-border-light);
  opacity: 0;
  transform: translateY(8px);
  transition: opacity var(--transition-fast), transform var(--transition-fast);
  transition-delay: 0s;
}

.mkt-dock-popup[data-state="open"] .popup-idx {
  opacity: 1;
  transform: translateY(0);
  transition: opacity var(--transition-normal), transform var(--transition-spring);
  transition-delay: calc(var(--i, 0) * 60ms + 0.14s);
}

.popup-idx:last-child {
  border-bottom: none;
  padding-bottom: 0;
}

.popup-idx-top {
  display: flex;
  align-items: baseline;
  gap: var(--sp-2);
}

.popup-idx-name {
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
  min-width: 56px;
}

.popup-idx-value {
  color: var(--color-text-primary);
  font-size: var(--fs-sm);
  font-weight: var(--fw-bold);
}

.popup-idx-meta {
  display: flex;
  gap: var(--sp-3);
  margin-top: var(--sp-0_5);
  color: var(--color-text-disabled);
  font-size: var(--fs-xs);
}

.popup-idx-delta.up { color: var(--color-up); }
.popup-idx-delta.down { color: var(--color-down); }

.popup-idx-hl {
  color: var(--color-text-disabled);
}

.popup-idx-chart {
  width: 100%;
  height: 40px;
  margin-top: var(--sp-1);
}

.mkt-dock-main {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  padding: var(--sp-2) var(--sp-3);
  white-space: nowrap;
  transition: transform var(--transition-spring);
}

.mkt-dock-name {
  overflow: hidden;
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
  text-overflow: ellipsis;
  max-width: 56px;
}

.mkt-dock-value {
  color: var(--color-text-primary);
  font-size: var(--fs-sm);
  font-weight: var(--fw-bold);
}

.mkt-dock-empty {
  color: var(--color-text-disabled);
  font-size: var(--fs-sm);
}

.mkt-dock-expand {
  display: flex;
  align-items: center;
  justify-content: center;
  margin-left: auto;
  color: var(--color-text-disabled);
  transition: transform var(--transition-spring), color var(--transition-fast);
}

.mkt-dock-expand.open {
  transform: rotate(180deg);
  color: var(--color-brand);
}

@media (max-width: 768px) {
  .market-dock {
    right: var(--sp-3);
    left: var(--sp-3);
    transform: none;
    flex-direction: column;
    bottom: 92px;
  }

  .mkt-dock-item {
    min-width: 0;
  }

  .mkt-dock-name {
    max-width: 80px;
  }

  .mkt-dock-popup {
    min-width: 0;
    left: 0;
    right: 0;
  }
}

@media (prefers-reduced-motion: reduce) {
  .mkt-dock-popup,
  .mkt-dock-popup[data-state="open"],
  .popup-idx,
  .mkt-dock-popup[data-state="open"] .popup-idx,
  .mkt-dock-item.expanded .mkt-dock-main {
    transition-duration: 0.01ms !important;
    transition-delay: 0s !important;
  }
}

@media (prefers-reduced-transparency: reduce) {
  .market-dock {
    backdrop-filter: none;
    -webkit-backdrop-filter: none;
    background: var(--color-bg-card);
  }
}
</style>
