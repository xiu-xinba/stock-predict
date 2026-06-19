<script setup lang="ts">
/** 基金业绩表现卡片，包含净值走势图和多周期收益率展示 */
import { ref, computed } from 'vue'
import { useECharts } from '@/shared/charts/useECharts'
import { useTheme } from '@/shared/composables/useTheme'
import { cssVar, colorWithAlpha, formatSignedPct, getDirection } from '@/shared/utils/format'
import echarts, { getBaseChartOption } from '@/shared/charts/echarts'
import type { FundPerformanceData } from '@/features/funds/types'

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

const periodEntries = Object.entries(periodLabels) as [PeriodKey, string][]

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

interface TooltipParam {
  axisValue?: string
  value?: unknown
}

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
  const dates = data.map((p) => p.date)
  const navs = data.map((p) => p.nav)

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
      formatter: (params: TooltipParam[]) => {
        const p = params[0]
        return `<div style="font-size:${cssVar('--fs-xs')};color:${cssVar('--color-chart-axis')}">${p.axisValue}</div>
                <div style="font-weight:600">净值: ${Number(p.value).toFixed(4)}</div>`
      },
    },
    series: [
      {
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
      },
    ],
    animation: true,
    animationDuration: 600,
  }
}

useECharts(chartRef, getChartOption, () => [filteredHistory.value, isDark.value])
</script>

<template>
  <section class="card card-tier-main fund-performance card-container fade-slide-up" style="--delay: 1">
    <div class="card-header">
      <div class="card-title-wrap">
        <h2 class="card-title">
          业绩表现
        </h2>
      </div>
    </div>
    <div class="card-body">
      <div class="period-tabs">
        <button
          v-for="[key, label] in periodEntries"
          :key="key"
          type="button"
          class="period-tab"
          :class="{ active: period === key }"
          @click="period = key"
        >
          {{ label }}
          <span class="period-return" :class="getDirection(returnMap[key])">
            {{ formatSignedPct(returnMap[key], 2) }}
          </span>
        </button>
      </div>

      <div ref="chartRef" class="chart-wrap" />

      <div class="return-grid">
        <div
          v-for="[key, label] in periodEntries"
          :key="key"
          class="return-item"
          :class="getDirection(returnMap[key])"
        >
          <span class="return-label">{{ label }}</span>
          <span class="return-value">
            <span class="return-dot" />
            {{ formatSignedPct(returnMap[key], 2) }}
          </span>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.fade-slide-up {
  animation: fade-slide-up 0.5s var(--ease-out-expo) both;
  animation-delay: calc(var(--delay, 0) * 60ms);
}

.fund-performance {
  padding: var(--sp-4);
}

.period-tabs {
  display: flex;
  gap: var(--sp-1);
  margin-bottom: var(--sp-3);
  overflow-x: auto;
  background: var(--color-bg-elevated);
  border-radius: var(--radius-full);
  padding: 3px;
}

.period-tab {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--sp-0_5);
  padding: var(--sp-1) var(--sp-3);
  border-radius: var(--radius-full);
  background: transparent;
  border: none;
  font-size: var(--fs-xs);
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
  white-space: nowrap;
  flex: 1;
}

.period-tab.active {
  background: var(--color-bg-card);
  box-shadow: 0 1px 3px color-mix(in srgb, var(--color-text-primary) 8%, transparent);
  color: var(--color-text-primary);
  font-weight: var(--fw-medium);
}

.period-tab:active {
  transform: scale(0.95);
}

.period-return {
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
  font-family: var(--font-mono);
}

.period-return.text-up {
  color: var(--color-up);
}
.period-return.text-down {
  color: var(--color-down);
}

.chart-wrap {
  width: 100%;
  height: 280px;
  margin-bottom: var(--sp-3);
  border-radius: var(--radius-md);
  background: linear-gradient(180deg, var(--color-bg-elevated) 0%, transparent 40%);
}

.return-grid {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: var(--sp-2);
}

.return-item {
  text-align: center;
  padding: var(--sp-2);
  border-radius: var(--radius-md);
  background: var(--color-bg-elevated);
  transition: transform var(--transition-fast);
}

.return-item:hover {
  transform: translateY(-1px);
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
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
}

.return-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
}

.return-item.text-up .return-value {
  color: var(--color-up);
}
.return-item.text-down .return-value {
  color: var(--color-down);
}
.return-item.text-up .return-dot {
  background: var(--color-up);
}
.return-item.text-down .return-dot {
  background: var(--color-down);
}
.return-item.text-flat .return-dot {
  background: var(--color-flat);
}

@media (max-width: 768px) {
  .return-grid {
    grid-template-columns: repeat(3, 1fr);
  }
  .chart-wrap {
    height: 180px;
  }
}
</style>
