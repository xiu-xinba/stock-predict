<template>
  <div class="idx-card" :class="[direction, `mkt-${index.market}`]">
    <div class="card-inner">
      <div class="card-head">
        <span class="idx-name">{{ index.name }}</span>
        <span :class="['idx-pct', direction]">
          {{ index.change_pct > 0 ? '+' : '' }}{{ index.change_pct.toFixed(2) }}%
        </span>
      </div>
      <div class="card-body">
        <div class="body-left">
          <span class="idx-value">{{ formatValue(index.value) }}</span>
          <span :class="['idx-delta', direction]">
            {{ index.change > 0 ? '+' : '' }}{{ index.change.toFixed(2) }}
          </span>
        </div>
        <div class="body-right">
          <div class="sparkline" ref="chartRef"></div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import echarts from '@/utils/echarts'
import { useECharts } from '@/composables/useECharts'
import { useTheme } from '@/composables/useTheme'
import { formatValue, colorWithAlpha, cssVar } from '@/utils/format'
import type { MarketIndex } from '@/types'

const props = defineProps<{ index: MarketIndex }>()

const direction = computed(() => props.index.change_pct >= 0 ? 'up' : 'down')
const { isDark } = useTheme()

const chartRef = ref<HTMLElement>()
useECharts(
  chartRef,
  () => {
    const isUp = props.index.change_pct >= 0
    const color = isUp ? cssVar('--color-up', '#b42318') : cssVar('--color-down', '#067647')
    return {
      grid: { top: 2, right: 0, bottom: 0, left: 0 },
      xAxis: { show: false, type: 'category', data: props.index.mini_chart_data.map((_: number, i: number) => i), boundaryGap: false },
      yAxis: { show: false, type: 'value', min: 'dataMin' },
      series: [{
        type: 'line',
        data: props.index.mini_chart_data,
        smooth: 0.4,
        showSymbol: false,
        lineStyle: { width: 1.5, color },
        areaStyle: { color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
          { offset: 0, color: colorWithAlpha(color, 0.19) },
          { offset: 1, color: colorWithAlpha(color, 0.02) },
        ]) },
      }],
      tooltip: { show: false },
      animation: true,
      animationDuration: 600,
    }
  },
  () => [props.index.mini_chart_data, props.index.change_pct, isDark.value]
)
</script>

<style scoped>
.idx-card {
  display: flex;
  background: var(--color-bg-card);
  border-radius: var(--radius-lg);
  overflow: hidden;
  transition: background-color var(--transition-fast);
  cursor: default;
  border: 1px solid var(--color-border);
  border-top: 3px solid var(--color-border);
}
.idx-card:hover {
  background: var(--color-bg-hover);
}
.idx-card.up { border-top-color: var(--color-up); }
.idx-card.down { border-top-color: var(--color-down); }
.idx-card.mkt-hk { border-top-color: var(--color-hk); }
.idx-card.mkt-us { border-top-color: var(--color-us); }

/* 内容 */
.card-inner {
  flex: 1;
  padding: 12px 14px 10px;
  min-width: 0;
}
.card-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
}
.idx-name {
  font-size: var(--fs-sm);
  color: var(--color-text-secondary);
  font-weight: var(--fw-semibold);
  line-height: var(--lh-snug);
}
.idx-pct {
  font-size: var(--fs-sm);
  font-weight: var(--fw-bold);
  padding: 2px 8px;
  border-radius: 6px;
  font-variant-numeric: tabular-nums;
}
.idx-pct.up { color: var(--color-up); background: var(--color-up-bg); }
.idx-pct.down { color: var(--color-down); background: var(--color-down-bg); }

.card-body {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: var(--sp-3);
}
.body-left {
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.idx-value {
  font-size: var(--fs-2xl);
  font-weight: var(--fw-extrabold);
  color: var(--color-text-primary);
  line-height: var(--lh-tight);
  font-variant-numeric: tabular-nums;
}
.idx-delta {
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
  font-variant-numeric: tabular-nums;
}
.idx-delta.up { color: var(--color-up); }
.idx-delta.down { color: var(--color-down); }

.body-right {
  width: 90px;
  flex-shrink: 0;
}
.sparkline {
  height: 40px;
  width: 100%;
}
</style>
