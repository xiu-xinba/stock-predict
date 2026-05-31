<script setup lang="ts">
import { ref, computed } from 'vue'
import { useECharts } from '@/composables/useECharts'
import { useTheme } from '@/composables/useTheme'
import { cssVar, colorWithAlpha, formatSignedPct, getDirection } from '@/utils/format'
import echarts, { getBaseChartOption } from '@/utils/echarts'
import CollapsibleCard from '@/components/CollapsibleCard.vue'
import type { FundPerformanceData } from '@/types'

defineOptions({ name: 'FundPerformance' })

const props = defineProps<{
  performance: FundPerformanceData
}>()

const { isDark } = useTheme()
const chartRef = ref<HTMLElement>()
type PeriodKey = '1m' | '3m' | '6m' | '1y' | '3y'

const period = ref<PeriodKey>('1y')

const periodLabels: Record<PeriodKey, string> = {
  '1m': '近1月',
  '3m': '近3月',
  '6m': '近6月',
  '1y': '近1年',
  '3y': '近3年',
}

const periodDays: Record<PeriodKey, number> = {
  '1m': 22,
  '3m': 66,
  '6m': 132,
  '1y': 252,
  '3y': 756,
}

const returnMap = computed(() => ({
  '1m': props.performance.return_1m,
  '3m': props.performance.return_3m,
  '6m': props.performance.return_6m,
  '1y': props.performance.return_1y,
  '3y': props.performance.return_3y,
}))

const filteredHistory = computed(() => {
  const h = props.performance.nav_history
  if (!h.length) return h
  const days = periodDays[period.value]
  const start = Math.max(0, h.length - days)
  return h.slice(start)
})

function getChartOption() {
  const base = getBaseChartOption()
  const data = filteredHistory.value
  if (!data.length) return {}

  const lineColor = cssVar('--color-brand')
  const dates = data.map(p => p.date)
  const navs = data.map(p => p.nav)

  return {
    ...base,
    grid: { top: 20, right: 16, bottom: 28, left: 56 },
    xAxis: {
      ...base.xAxis,
      type: 'category' as const,
      data: dates,
      boundaryGap: false,
      axisLabel: {
        ...base.xAxis.axisLabel,
        formatter: (val: string) => val.slice(5),
      },
    },
    yAxis: {
      ...base.yAxis,
      type: 'value' as const,
      scale: true,
      axisLabel: {
        ...base.yAxis.axisLabel,
        formatter: (val: number) => val.toFixed(2),
      },
    },
    tooltip: {
      ...base.tooltip,
      formatter: (params: any) => {
        const p = params[0]
        return `<div style="font-size:${cssVar('--fs-xs')};color:${cssVar('--color-chart-axis')}">${p.axisValue}</div>
                <div style="font-weight:600">净值: ${Number(p.value).toFixed(4)}</div>`
      },
    },
    series: [{
      type: 'line',
      data: navs,
      smooth: 0.3,
      showSymbol: false,
      lineStyle: { width: 2, color: lineColor },
      areaStyle: {
        color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
          { offset: 0, color: colorWithAlpha(lineColor, 0.15) },
          { offset: 1, color: colorWithAlpha(lineColor, 0.01) },
        ]),
      },
    }],
    animation: true,
    animationDuration: 600,
  }
}

useECharts(chartRef, getChartOption, () => [filteredHistory.value, isDark.value])
</script>

<template>
  <CollapsibleCard title="业绩表现" class="fund-performance card-container" body-max-height="600px">
    <div class="period-tabs">
      <button
        v-for="(_, key) in periodLabels"
        :key="key"
        class="period-tab"
        :class="{ active: period === key }"
        @click="period = key as any"
      >
        {{ periodLabels[key] }}
        <span class="period-return" :class="getDirection(returnMap[key])">
          {{ formatSignedPct(returnMap[key], 2) }}
        </span>
      </button>
    </div>

    <div class="chart-wrap" ref="chartRef" />

    <div class="return-grid">
      <div class="return-item" v-for="(_, key) in periodLabels" :key="key">
        <span class="return-label">{{ periodLabels[key] }}</span>
        <span class="return-value" :class="getDirection(returnMap[key])">
          {{ formatSignedPct(returnMap[key], 2) }}
        </span>
      </div>
    </div>
  </CollapsibleCard>
</template>

<style scoped>
.fund-performance {
  padding: var(--sp-4);
}

.period-tabs {
  display: flex;
  gap: var(--sp-1);
  margin-bottom: var(--sp-3);
  overflow-x: auto;
}

.period-tab {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--sp-0_5);
  padding: var(--sp-1) var(--sp-3);
  border-radius: var(--radius-md);
  background: transparent;
  border: 1px solid transparent;
  font-size: var(--fs-xs);
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
  white-space: nowrap;
}

.period-tab.active {
  background: var(--color-bg-elevated);
  border-color: var(--color-border);
  color: var(--color-text-primary);
  font-weight: var(--fw-medium);
}

.period-return {
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
  font-family: var(--font-mono);
}

.period-return.text-up { color: var(--color-up); }
.period-return.text-down { color: var(--color-down); }

.chart-wrap {
  width: 100%;
  height: 220px;
  margin-bottom: var(--sp-3);
}

.return-grid {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: var(--sp-2);
}

.return-item {
  text-align: center;
}

.return-label {
  display: block;
  font-size: var(--fs-xs);
  color: var(--color-text-disabled);
  margin-bottom: var(--sp-0_5);
}

.return-value {
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
  font-family: var(--font-mono);
}

.return-value.text-up { color: var(--color-up); }
.return-value.text-down { color: var(--color-down); }

@media (max-width: 768px) {
  .return-grid {
    grid-template-columns: repeat(3, 1fr);
  }
  .chart-wrap {
    height: 180px;
  }
}
</style>
