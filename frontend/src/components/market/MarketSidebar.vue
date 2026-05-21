<template>
  <div
    :class="['mkt-panel', market, { 'expanded-state': expanded }]"
    @mouseenter="onInteraction"
    @mouseleave="scheduleAutoCollapse"
  >
    <!-- 市场主题色轨道 -->
    <div :class="['panel-rail', market]"></div>

    <div class="panel-body">
      <!-- 主要指数：可点击展开 -->
      <div v-if="primaryIdx" class="primary-idx" @click="toggleExpand">
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
import { ref, computed, watch, onUnmounted, type ComponentPublicInstance } from 'vue'
import echarts from '@/utils/echarts'
import { formatValue, colorWithAlpha } from '@/utils/format'
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

  const style = getComputedStyle(el)
  const upColor = style.getPropertyValue('--color-up').trim() || '#cf2e2e'
  const downColor = style.getPropertyValue('--color-down').trim() || '#1a9956'
  const isUp = idx.change_pct >= 0
  const color = isUp ? upColor : downColor

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
        lineStyle: { width: 1.5, color },
        areaStyle: { color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
          { offset: 0, color: colorWithAlpha(color, 0.16) },
          { offset: 1, color: colorWithAlpha(color, 0.02) },
        ]) },
      }],
      tooltip: { show: false },
      animation: true,
      animationDuration: 600,
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

// 收起时 dispose 其他指数的图表（保留主要指数）
watch(expanded, (val) => {
  if (!val) {
    const primaryCode = primaryIdx.value?.code
    for (const [code, ro] of resizeObservers) {
      if (code !== primaryCode) ro.disconnect()
    }
    for (const [code, chart] of chartInstances) {
      if (code !== primaryCode) chart.dispose()
    }
    for (const code of [...chartInstances.keys()]) {
      if (code !== primaryCode) {
        chartInstances.delete(code)
        resizeObservers.delete(code)
      }
    }
  }
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
/* ============================================
   动画规范
   ============================================
   缓动函数：
   - ease-out:    cubic-bezier(0, 0, 0.2, 1)  展开（快出慢停）
   - ease-in:     cubic-bezier(0.4, 0, 1, 1)   收起（慢起快出）
   - standard:    cubic-bezier(0.4, 0, 0.2, 1)  通用双向
   - spring:      cubic-bezier(0.34, 1.56, 0.64, 1) 弹性效果

   时长规范：
   - micro:   150ms  微交互（hover色变）
   - fast:    200ms  快速反馈（按钮按压）
   - normal:  300ms  标准过渡（展开/收起）
   - smooth:  450ms  平滑动画（面板展开）
   ============================================ */

.mkt-panel {
  background: var(--color-bg-card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light);
  display: flex;
  overflow: hidden;
  box-shadow: var(--shadow-sm);
  transition: box-shadow 300ms cubic-bezier(0.4, 0, 0.2, 1),
              transform 300ms cubic-bezier(0.4, 0, 0.2, 1);
  position: relative;
}
.mkt-panel:hover {
  box-shadow: var(--shadow-lg);
  transform: translateY(-1px);
}

/* 左侧市场主题色轨道 */
.panel-rail {
  width: 4px;
  flex-shrink: 0;
  transition: width 150ms ease-out;
}
.mkt-panel:hover .panel-rail { width: 5px; }
.panel-rail.cn { background: linear-gradient(180deg, #cf2e2e 0%, #ef5350 100%); }
.panel-rail.hk { background: linear-gradient(180deg, #ff9900 0%, #ffbb44 100%); }
.panel-rail.us { background: linear-gradient(180deg, #4a7cf7 0%, #7da4ff 100%); }

html.dark .panel-rail.cn { background: linear-gradient(180deg, #f56c6c 0%, #ff9999 100%); }
html.dark .panel-rail.hk { background: linear-gradient(180deg, #ffaa33 0%, #ffcc66 100%); }
html.dark .panel-rail.us { background: linear-gradient(180deg, #6b9aff 0%, #9db8ff 100%); }

.panel-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}

/* 主要指数：可点击展开 */
.primary-idx {
  padding: var(--sp-4);
  cursor: pointer;
  transition: background 150ms ease-out;
  border-bottom: 1px solid var(--color-border-light);
}
.primary-idx:hover {
  background: var(--color-bg-hover);
}
.primary-idx:active {
  background: var(--color-border-light);
  transition-duration: 80ms;
}
.mkt-panel.expanded-state .primary-idx {
  border-bottom-color: var(--color-border);
}

.primary-row-1 { display: flex; justify-content: space-between; align-items: center; margin-bottom: 6px; line-height: var(--lh-snug); }
.primary-label { display: flex; align-items: center; gap: var(--sp-2); }

/* 市场标签徽章 */
.market-badge {
  font-size: var(--fs-2xs);
  font-weight: var(--fw-bold);
  padding: 1px 6px;
  border-radius: 4px;
  letter-spacing: var(--ls-wider);
  line-height: var(--lh-snug);
  flex-shrink: 0;
}
.market-badge.cn { color: var(--color-up); background: var(--color-up-bg); }
.market-badge.hk { color: var(--color-hk); background: var(--color-hk-bg); }
.market-badge.us { color: var(--color-us); background: var(--color-us-bg); }

.primary-name { font-size: var(--fs-base); font-weight: var(--fw-semibold); color: var(--color-text-primary); letter-spacing: var(--ls-wide); line-height: var(--lh-snug); }

/* 展开提示箭头 */
.expand-hint {
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-disabled);
  transition: transform 300ms cubic-bezier(0.34, 1.56, 0.64, 1),
              color 150ms ease-out;
  transform: rotate(0deg);
  margin-left: 4px;
}
.expand-hint:hover { color: var(--color-brand); }
.expand-hint.expanded { transform: rotate(180deg); }
.primary-idx:hover .expand-hint { color: var(--color-brand); }

.primary-row-2 { display: flex; align-items: baseline; gap: var(--sp-2); margin-bottom: 4px; line-height: var(--lh-tight); }
.primary-idx-value { font-size: var(--fs-2xl); font-weight: var(--fw-extrabold); color: var(--color-text-primary); letter-spacing: var(--ls-tighter); line-height: var(--lh-tight); font-variant-numeric: tabular-nums; }
.primary-pct { font-size: var(--fs-sm); font-weight: var(--fw-bold); padding: 2px 8px; border-radius: 6px; margin-left: auto; flex-shrink: 0; letter-spacing: var(--ls-wide); font-variant-numeric: tabular-nums; transition: transform 150ms ease-out; }
.primary-pct.up { color: var(--color-up); background: var(--color-up-bg); }
.primary-pct.down { color: var(--color-down); background: var(--color-down-bg); }
.primary-idx:active .primary-pct { transform: scale(0.95); }

.primary-row-3 { display: flex; align-items: center; gap: var(--sp-3); margin-bottom: 6px; line-height: var(--lh-normal); }
.primary-delta { font-size: var(--fs-xs); font-weight: var(--fw-semibold); font-variant-numeric: tabular-nums; }
.primary-delta.up { color: var(--color-up); }
.primary-delta.down { color: var(--color-down); }
.primary-hl { font-size: var(--fs-xs); color: var(--color-text-disabled); letter-spacing: var(--ls-wide); font-variant-numeric: tabular-nums; display: inline-flex; align-items: center; gap: 2px; }
.hl-label { font-size: var(--fs-2xs); color: var(--color-text-disabled); opacity: 0.7; }

.primary-spark { height: 48px; width: 100%; pointer-events: none; }

/* 其他指数列表：展开/收起动画 */
.others-wrap {
  overflow: hidden;
  flex: 0 0 auto;
  min-height: 0;
  max-height: 0;
  opacity: 0;
  transform: translateY(-6px);
  pointer-events: none;
  transition: max-height 450ms cubic-bezier(0, 0, 0.2, 1),
              opacity 300ms cubic-bezier(0, 0, 0.2, 1) 50ms,
              transform 350ms cubic-bezier(0, 0, 0.2, 1);
}
.mkt-panel.expanded-state .others-wrap {
  max-height: 600px;
  opacity: 1;
  transform: translateY(0);
  pointer-events: auto;
  transition: max-height 450ms cubic-bezier(0.4, 0, 1, 1),
              opacity 300ms cubic-bezier(0.4, 0, 1, 1),
              transform 350ms cubic-bezier(0.4, 0, 1, 1);
}

/* 其他指数项：交错入场动画 */
.idx-item {
  padding: var(--sp-3) var(--sp-4);
  border-bottom: 1px solid var(--color-border-light);
  transition: background 150ms ease-out,
              opacity 300ms cubic-bezier(0, 0, 0.2, 1) calc(60ms * var(--i, 0)),
              transform 300ms cubic-bezier(0, 0, 0.2, 1) calc(60ms * var(--i, 0));
}
.mkt-panel.expanded-state .idx-item {
  opacity: 1;
  transform: translateX(0);
}
.mkt-panel:not(.expanded-state) .idx-item {
  opacity: 0;
  transform: translateX(-8px);
}
.idx-item:hover { background: var(--color-bg-hover); }
.idx-item:last-child { border-bottom: none; }

.idx-row-1 { display: flex; justify-content: space-between; align-items: center; margin-bottom: 2px; line-height: var(--lh-snug); }
.idx-name { font-size: var(--fs-sm); font-weight: var(--fw-semibold); color: var(--color-text-secondary); letter-spacing: var(--ls-wide); }
.idx-pct { font-size: var(--fs-sm); font-weight: var(--fw-bold); padding: 1px 6px; border-radius: 4px; letter-spacing: var(--ls-wide); font-variant-numeric: tabular-nums; }
.idx-pct.up { color: var(--color-up); background: var(--color-up-bg); }
.idx-pct.down { color: var(--color-down); background: var(--color-down-bg); }

.idx-row-2 { display: flex; align-items: baseline; gap: 6px; margin-bottom: 2px; line-height: var(--lh-tight); }
.idx-value { font-size: var(--fs-lg); font-weight: var(--fw-extrabold); color: var(--color-text-primary); letter-spacing: var(--ls-tighter); line-height: var(--lh-tight); font-variant-numeric: tabular-nums; }
.idx-delta { font-size: var(--fs-xs); font-weight: var(--fw-medium); font-variant-numeric: tabular-nums; }
.idx-delta.up { color: var(--color-up); }
.idx-delta.down { color: var(--color-down); }

.idx-row-3 { display: flex; gap: var(--sp-3); margin-bottom: 4px; line-height: var(--lh-normal); }
.idx-hl { font-size: var(--fs-2xs); color: var(--color-text-disabled); letter-spacing: var(--ls-wide); font-variant-numeric: tabular-nums; display: inline-flex; align-items: center; gap: 2px; }

.idx-spark { height: 32px; width: 100%; }

/* 空状态 */
.idx-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: var(--color-text-secondary);
  padding: var(--sp-8) var(--sp-4);
  font-size: var(--fs-sm);
  gap: var(--sp-2);
}
.empty-icon {
  opacity: 0.3;
}
</style>
