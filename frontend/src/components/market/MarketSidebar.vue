<template>
  <div
    :class="['mkt-panel', market, { 'expanded-state': expanded }]"
    @mouseenter="onInteraction"
    @mouseleave="scheduleAutoCollapse"
  >
    <div class="panel-body">
      <!-- 主要指数：可点击展开 -->
      <div
        v-if="primaryIdx"
        class="primary-idx"
        @click="toggleExpand"
        role="button"
        :aria-expanded="expanded"
        tabindex="0"
        @keydown.enter="toggleExpand"
        @keydown.space.prevent="toggleExpand"
      >
        <div class="primary-row-1">
          <div class="primary-label">
            <span :class="['market-badge', market]">{{ label }}</span>
            <span class="primary-name">{{ primaryIdx.name }}</span>
          </div>
          <span v-if="indices.length > 1" :class="['expand-hint', { expanded }]">
            <svg viewBox="0 0 24 24" width="16" height="16"><path fill="currentColor" d="M7 10l5 5 5-5z"/></svg>
          </span>
        </div>
        <div class="primary-row-2">
          <span class="primary-idx-value">{{ formatValue(primaryIdx.value) }}</span>
          <span :class="['primary-pct', primaryIdx.change_pct >= 0 ? 'up' : 'down']">
            {{ primaryIdx.change_pct > 0 ? '+' : '' }}{{ primaryIdx.change_pct.toFixed(2) }}%
          </span>
        </div>
        <div class="primary-row-3">
          <span :class="['primary-delta', primaryIdx.change_pct >= 0 ? 'up' : 'down']">
            {{ primaryIdx.change > 0 ? '+' : '' }}{{ primaryIdx.change.toFixed(2) }}
          </span>
          <span v-if="primaryIdx.high > 0" class="primary-hl">
            <span class="hl-label">高</span>{{ formatValue(primaryIdx.high) }}
          </span>
          <span v-if="primaryIdx.low > 0" class="primary-hl">
            <span class="hl-label">低</span>{{ formatValue(primaryIdx.low) }}
          </span>
        </div>
        <div class="primary-spark" :ref="el => setChartRef(primaryIdx.code, el)"></div>
      </div>
      <div v-else class="idx-empty">
        <svg viewBox="0 0 24 24" width="32" height="32" class="empty-icon"><path fill="currentColor" d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z" opacity="0.3"/><path fill="currentColor" d="M11 7h2v6h-2zm0 8h2v2h-2z"/></svg>
        <span>暂无数据</span>
      </div>

      <!-- 其他指数列表 -->
      <div class="others-wrap">
        <div class="others-body">
          <div v-for="(idx, i) in otherIndices" :key="idx.code" class="idx-item" :style="{ '--i': i }">
            <div class="idx-row-1">
              <span class="idx-name">{{ idx.name }}</span>
              <span :class="['idx-pct', idx.change_pct >= 0 ? 'up' : 'down']">
                {{ idx.change_pct > 0 ? '+' : '' }}{{ idx.change_pct.toFixed(2) }}%
              </span>
            </div>
            <div class="idx-row-2">
              <span class="idx-value">{{ formatValue(idx.value) }}</span>
              <span :class="['idx-delta', idx.change_pct >= 0 ? 'up' : 'down']">
                {{ idx.change > 0 ? '+' : '' }}{{ idx.change.toFixed(2) }}
              </span>
            </div>
            <div v-if="idx.high > 0 || idx.low > 0" class="idx-row-3">
              <span class="idx-hl"><span class="hl-label">高</span>{{ formatValue(idx.high) }}</span>
              <span class="idx-hl"><span class="hl-label">低</span>{{ formatValue(idx.low) }}</span>
            </div>
            <div class="idx-spark" :ref="el => setChartRef(idx.code, el)"></div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onUnmounted, type ComponentPublicInstance } from 'vue'
import echarts from '@/utils/echarts'
import { useTheme } from '@/composables/useTheme'
import { formatValue, colorWithAlpha, cssVar } from '@/utils/format'
import type { MarketIndex } from '@/types'

const props = withDefaults(defineProps<{
  label: string
  market: string
  indices: MarketIndex[]
  autoCollapseDelay?: number
}>(), {
  autoCollapseDelay: 8000,
})

const expanded = ref(false)
const { isDark } = useTheme()

const primaryIdx = computed(() => props.indices[0] ?? null)
const otherIndices = computed(() => props.indices.slice(1))

let collapseTimer: ReturnType<typeof setTimeout> | null = null

function toggleExpand() {
  if (props.indices.length <= 1) return
  expanded.value = !expanded.value
  if (expanded.value) {
    scheduleAutoCollapse()
  }
}

function onInteraction() {
  if (collapseTimer) {
    clearTimeout(collapseTimer)
    collapseTimer = null
  }
}

function scheduleAutoCollapse() {
  if (!expanded.value) return
  if (collapseTimer) clearTimeout(collapseTimer)
  collapseTimer = setTimeout(() => {
    expanded.value = false
  }, props.autoCollapseDelay)
}

// ECharts management
const chartInstances = new Map<string, echarts.ECharts>()
const resizeObservers = new Map<string, ResizeObserver>()
const resizeRafs = new Map<string, number>()

function setChartRef(code: string, el: Element | ComponentPublicInstance | null) {
  if (!el || !(el instanceof HTMLElement)) {
    const raf = resizeRafs.get(code)
    if (raf) cancelAnimationFrame(raf)
    resizeRafs.delete(code)
    resizeObservers.get(code)?.disconnect()
    resizeObservers.delete(code)
    chartInstances.get(code)?.dispose()
    chartInstances.delete(code)
    return
  }
  const idx = props.indices.find(i => i.code === code)
  if (!idx) return

  const existing = chartInstances.get(code)
  if (existing) existing.dispose()

  const isUp = idx.change_pct >= 0
  const lineColor = isUp ? cssVar('--color-up', '#b42318') : cssVar('--color-down', '#067647')

  try {
    const chart = echarts.init(el)
    chart.setOption({
      grid: { top: 2, right: 2, bottom: 2, left: 2 },
      xAxis: { show: false, type: 'category', data: idx.mini_chart_data.map((_: number, i: number) => i), boundaryGap: false },
      yAxis: { show: false, type: 'value', min: 'dataMin' },
      series: [{
        type: 'line',
        data: idx.mini_chart_data,
        smooth: 0.4,
        showSymbol: false,
        lineStyle: { width: 1.5, color: lineColor },
        areaStyle: { color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
          { offset: 0, color: colorWithAlpha(lineColor, 0.16) },
          { offset: 1, color: colorWithAlpha(lineColor, 0.02) },
        ]) },
      }],
      tooltip: { show: false },
      animation: true,
      animationDuration: 520,
      animationEasing: 'cubicOut',
    })
    chartInstances.set(code, chart)
    const ro = new ResizeObserver(() => {
      const prev = resizeRafs.get(code)
      if (prev) cancelAnimationFrame(prev)
      resizeRafs.set(code, requestAnimationFrame(() => {
        chart.resize()
        resizeRafs.delete(code)
      }))
    })
    ro.observe(el)
    resizeObservers.set(code, ro)
  } catch {
    chartInstances.get(code)?.dispose()
    chartInstances.delete(code)
  }
}

watch(expanded, (val) => {
  if (!val) return
  nextTick(() => {
    requestAnimationFrame(() => {
      for (const [, chart] of chartInstances) chart.resize()
    })
  })
})

watch(isDark, () => {
  nextTick(() => {
    for (const [code, chart] of chartInstances) {
      const idx = props.indices.find(i => i.code === code)
      if (!idx) continue
      const lineColor = idx.change_pct >= 0 ? cssVar('--color-up', '#b42318') : cssVar('--color-down', '#067647')
      chart.setOption({
        series: [{
          lineStyle: { color: lineColor },
          areaStyle: { color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: colorWithAlpha(lineColor, 0.16) },
            { offset: 1, color: colorWithAlpha(lineColor, 0.02) },
          ]) },
        }],
      })
    }
  })
})

onUnmounted(() => {
  if (collapseTimer) clearTimeout(collapseTimer)
  for (const [, raf] of resizeRafs) cancelAnimationFrame(raf)
  resizeRafs.clear()
  for (const [, ro] of resizeObservers) ro.disconnect()
  resizeObservers.clear()
  for (const [, chart] of chartInstances) chart.dispose()
  chartInstances.clear()
})
</script>

<style scoped>
.mkt-panel {
  position: relative;
  display: flex;
  overflow: hidden;
  border: 1px solid var(--color-border);
  border-top: 3px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-bg-card);
  transition: border-color var(--transition-fast), box-shadow var(--transition-normal), transform var(--transition-normal);
}

.mkt-panel:hover,
.mkt-panel.expanded-state {
  box-shadow: var(--shadow-md);
}

.mkt-panel.cn { border-top-color: var(--color-up); }
.mkt-panel.hk { border-top-color: var(--color-hk); }
.mkt-panel.us { border-top-color: var(--color-us); }

.panel-body {
  display: flex;
  flex: 1;
  min-width: 0;
  flex-direction: column;
}

.primary-idx {
  padding: var(--sp-4);
  border-bottom: 1px solid var(--color-border-light);
  cursor: pointer;
  transition: background-color var(--transition-fast), padding-bottom var(--transition-normal);
}

.primary-idx:hover {
  background: var(--color-bg-hover);
}

.mkt-panel.expanded-state .primary-idx {
  border-bottom-color: var(--color-border);
  padding-bottom: var(--sp-5);
}

.primary-row-1,
.idx-row-1 {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--sp-2);
  margin-bottom: var(--sp-2);
  line-height: var(--lh-snug);
}

.primary-label {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  min-width: 0;
}

.market-badge {
  flex-shrink: 0;
  padding: 1px 6px;
  border-radius: var(--radius-sm);
  font-size: var(--fs-2xs);
  font-weight: var(--fw-bold);
  line-height: var(--lh-snug);
}

.market-badge.cn { color: var(--color-up); background: var(--color-up-bg); }
.market-badge.hk { color: var(--color-hk); background: var(--color-hk-bg); }
.market-badge.us { color: var(--color-us); background: var(--color-us-bg); }

.primary-name {
  overflow: hidden;
  color: var(--color-text-primary);
  font-size: var(--fs-base);
  font-weight: var(--fw-semibold);
  line-height: var(--lh-snug);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.expand-hint {
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-disabled);
  transform: rotate(0deg);
  transition: color var(--transition-fast), transform 0.32s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.expand-hint.expanded {
  transform: rotate(180deg);
}

.primary-idx:hover .expand-hint {
  color: var(--color-brand);
}

.primary-row-2,
.idx-row-2 {
  display: flex;
  align-items: baseline;
  gap: var(--sp-2);
  margin-bottom: var(--sp-1);
  line-height: var(--lh-tight);
}

.primary-idx-value {
  color: var(--color-text-primary);
  font-size: var(--fs-2xl);
  font-weight: var(--fw-extrabold);
  line-height: var(--lh-tight);
  font-variant-numeric: tabular-nums;
}

.primary-pct,
.idx-pct {
  flex-shrink: 0;
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  font-size: var(--fs-sm);
  font-weight: var(--fw-bold);
  font-variant-numeric: tabular-nums;
}

.primary-pct {
  margin-left: auto;
}

.primary-pct.up,
.idx-pct.up {
  color: var(--color-up);
  background: var(--color-up-bg);
}

.primary-pct.down,
.idx-pct.down {
  color: var(--color-down);
  background: var(--color-down-bg);
}

.primary-row-3,
.idx-row-3 {
  display: flex;
  align-items: center;
  gap: var(--sp-3);
  margin-bottom: var(--sp-2);
  line-height: var(--lh-normal);
}

.primary-delta,
.idx-delta {
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  font-variant-numeric: tabular-nums;
}

.primary-delta.up,
.idx-delta.up {
  color: var(--color-up);
}

.primary-delta.down,
.idx-delta.down {
  color: var(--color-down);
}

.primary-hl,
.idx-hl {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  color: var(--color-text-disabled);
  font-size: var(--fs-xs);
  font-variant-numeric: tabular-nums;
}

.idx-hl {
  font-size: var(--fs-2xs);
}

.hl-label {
  color: var(--color-text-disabled);
  font-size: var(--fs-2xs);
  opacity: 0.75;
}

.primary-spark {
  width: 100%;
  height: 52px;
  pointer-events: none;
}

.others-wrap {
  display: grid;
  grid-template-rows: 0fr;
  opacity: 0;
  pointer-events: none;
  transform: translateY(-8px);
  transition:
    grid-template-rows 0.38s cubic-bezier(0.2, 0.8, 0.2, 1),
    opacity 0.24s ease,
    transform 0.38s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.others-body {
  min-height: 0;
  overflow: hidden;
}

.mkt-panel.expanded-state .others-wrap {
  grid-template-rows: 1fr;
  opacity: 1;
  pointer-events: auto;
  transform: translateY(0);
}

.idx-item {
  padding: var(--sp-3) var(--sp-4);
  border-bottom: 1px solid var(--color-border-light);
  opacity: 0;
  transform: translateY(-6px);
  transition:
    background-color var(--transition-fast),
    opacity 0.24s ease calc(var(--i, 0) * 45ms),
    transform 0.32s cubic-bezier(0.2, 0.8, 0.2, 1) calc(var(--i, 0) * 45ms);
}

.mkt-panel.expanded-state .idx-item {
  opacity: 1;
  transform: translateY(0);
}

.idx-item:hover {
  background: var(--color-bg-hover);
}

.idx-item:last-child {
  border-bottom: 0;
}

.idx-name {
  overflow: hidden;
  color: var(--color-text-secondary);
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.idx-value {
  color: var(--color-text-primary);
  font-size: var(--fs-lg);
  font-weight: var(--fw-extrabold);
  line-height: var(--lh-tight);
  font-variant-numeric: tabular-nums;
}

.idx-spark {
  width: 100%;
  height: 34px;
}

.idx-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--sp-2);
  min-height: 180px;
  padding: var(--sp-8) var(--sp-4);
  color: var(--color-text-secondary);
  font-size: var(--fs-sm);
}

.empty-icon {
  opacity: 0.4;
}
</style>
