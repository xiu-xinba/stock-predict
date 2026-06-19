<script setup lang="ts">
/** 融资融券组件，展示最新融资余额及融资融券历史趋势图。
 * 使用 auxiliary tier + MARGIN eyebrow。
 */
import { ref } from 'vue'
import { useECharts } from '@/shared/charts/useECharts'
import { useTheme } from '@/shared/composables/useTheme'
import { cssVar, formatVolume } from '@/shared/utils/format'
import { getBaseChartOption } from '@/shared/charts/echarts'
import type { StockMargin } from '@/features/stocks/types'

defineOptions({ name: 'StockMargin' })

const props = defineProps<{
  margin: StockMargin
}>()

const { isDark } = useTheme()
const chartRef = ref<HTMLElement>()

function getChartOption() {
  const base = getBaseChartOption()
  const history = props.margin.history || []
  if (!history.length) return {}

  const dates = history.map((h) => h.date)
  const marginBalances = history.map((h) => h.margin_balance)
  const shortBalances = history.map((h) => h.short_balance)

  return {
    ...base,
    legend: {
      data: ['融资余额', '融券余额'],
      bottom: 0,
      textStyle: {
        color: cssVar('--color-chart-axis'),
        fontSize: Number(cssVar('--fs-xs').replace('px', '')),
      },
      itemWidth: 12,
      itemHeight: 8,
    },
    xAxis: {
      ...base.xAxis,
      type: 'category' as const,
      data: dates,
      axisLabel: {
        ...base.xAxis.axisLabel,
        formatter: (val: string) => val.slice(5),
      },
    },
    yAxis: {
      ...base.yAxis,
      type: 'value' as const,
      axisLabel: {
        ...base.yAxis.axisLabel,
        formatter: (val: number) => formatVolume(val),
      },
    },
    series: [
      {
        name: '融资余额',
        type: 'line',
        data: marginBalances,
        smooth: 0.3,
        showSymbol: false,
        lineStyle: { width: 2, color: cssVar('--color-brand') },
        areaStyle: {
          color: {
            type: 'linear',
            x: 0,
            y: 0,
            x2: 0,
            y2: 1,
            colorStops: [
              { offset: 0, color: cssVar('--color-brand') + '33' },
              { offset: 1, color: cssVar('--color-brand') + '00' },
            ],
          },
        },
      },
      {
        name: '融券余额',
        type: 'line',
        data: shortBalances,
        smooth: 0.3,
        showSymbol: false,
        lineStyle: { width: 1.5, color: cssVar('--color-down') },
      },
    ],
    animation: true,
    animationDuration: 600,
  }
}

useECharts(chartRef, getChartOption, () => [props.margin.history, isDark.value])
</script>

<template>
  <section class="card card-tier-auxiliary fade-slide-up" style="--delay: 5">
    <div class="card-header">
      <div class="card-title-wrap">
        <span class="card-eyebrow">MARGIN</span>
        <h2 class="card-title">融资融券</h2>
      </div>
    </div>
    <div class="card-body">
      <div class="margin-overview">
        <div class="overview-item">
          <span class="overview-label">最新融资余额</span>
          <span class="overview-value numeric">{{
            margin.latest_margin_balance != null ? formatVolume(margin.latest_margin_balance) : '--'
          }}</span>
        </div>
      </div>

      <div v-if="margin.history && margin.history.length" ref="chartRef" class="chart-wrap" />
      <div v-else class="empty-hint">暂无融资融券数据</div>
    </div>
  </section>
</template>

<style scoped>
.fade-slide-up {
  animation: fade-slide-up 0.5s var(--ease-out-expo) both;
  animation-delay: calc(var(--delay, 0) * 60ms);
}

.margin-overview {
  display: flex;
  gap: var(--sp-6);
  margin-bottom: var(--sp-3);
  padding: var(--sp-3);
  border-radius: var(--radius-md);
  background: var(--color-bg-elevated);
}

.overview-item {
  display: flex;
  flex-direction: column;
  gap: var(--sp-0_5);
}

.overview-label {
  font-size: var(--fs-xs);
  color: var(--color-text-tertiary);
  letter-spacing: var(--ls-wide);
}

.overview-value {
  font-size: var(--fs-lg);
  font-weight: var(--fw-semibold);
  color: var(--color-text-primary);
}

.overview-value.numeric {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
}

.chart-wrap {
  width: 100%;
  height: 240px;
}

@media (max-width: 768px) {
  .chart-wrap {
    height: 180px;
  }
}
</style>
