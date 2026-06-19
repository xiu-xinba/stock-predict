<script setup lang="ts">
/** 股票财务指标组件，展示 PE/PB/ROE/EPS 等指标及季度营收利润图表 */
import { ref } from 'vue'
import { useECharts } from '@/shared/charts/useECharts'
import { useTheme } from '@/shared/composables/useTheme'
import { cssVar } from '@/shared/utils/format'
import { getBaseChartOption } from '@/shared/charts/echarts'
import type { StockFinancials } from '@/features/stocks/types'

defineOptions({ name: 'StockFinancials' })

const props = defineProps<{
  financials: StockFinancials
}>()

const { isDark } = useTheme()
const chartRef = ref<HTMLElement>()

function getChartOption() {
  const base = getBaseChartOption()
  const quarterly = props.financials.quarterly || []
  if (!quarterly.length) return {}

  const dates = quarterly.map((q) => q.report_date)
  const revenues = quarterly.map((q) => q.revenue)
  const netProfits = quarterly.map((q) => q.net_profit)

  return {
    ...base,
    legend: {
      data: ['营收', '净利润'],
      bottom: 0,
      textStyle: { color: cssVar('--color-chart-axis'), fontSize: 10 },
      itemWidth: 12,
      itemHeight: 8,
    },
    xAxis: {
      ...base.xAxis,
      type: 'category' as const,
      data: dates,
      axisLabel: {
        ...base.xAxis.axisLabel,
        formatter: (val: string) => val.slice(2, 7),
      },
    },
    yAxis: [
      {
        ...base.yAxis,
        type: 'value' as const,
      },
      {
        ...base.yAxis,
        type: 'value' as const,
        splitLine: { show: false },
      },
    ],
    series: [
      {
        name: '营收',
        type: 'bar',
        data: revenues,
        itemStyle: {
          color: cssVar('--color-brand'),
          opacity: 0.7,
        },
        barMaxWidth: 20,
      },
      {
        name: '净利润',
        type: 'bar',
        yAxisIndex: 1,
        data: netProfits,
        itemStyle: {
          color: cssVar('--color-up'),
          opacity: 0.7,
        },
        barMaxWidth: 20,
      },
    ],
    animation: true,
    animationDuration: 600,
  }
}

useECharts(chartRef, getChartOption, () => [props.financials.quarterly, isDark.value])

function getPEClass(pe: number | null): string {
  if (pe == null) return ''
  if (pe < 0) return 'val-negative'
  if (pe < 15) return 'val-low'
  if (pe > 40) return 'val-high'
  return ''
}

function getPELabel(pe: number | null): string {
  if (pe == null) return ''
  if (pe < 0) return '亏损'
  if (pe < 15) return '低估'
  if (pe > 40) return '高估'
  return ''
}

function getPBClass(pb: number | null): string {
  if (pb == null) return ''
  if (pb < 0) return 'val-negative'
  if (pb < 1) return 'val-low'
  if (pb > 5) return 'val-high'
  return ''
}

function getPBLabel(pb: number | null): string {
  if (pb == null) return ''
  if (pb < 0) return '亏损'
  if (pb < 1) return '破净'
  if (pb > 5) return '高估'
  return ''
}
</script>

<template>
  <section class="card card-tier-auxiliary fade-slide-up" style="--delay: 2">
    <div class="card-header">
      <div class="card-title-wrap">
        <span class="card-eyebrow">FINANCIALS</span>
        <h2 class="card-title">财务指标</h2>
      </div>
    </div>
    <div class="card-body">
      <div class="metrics-grid">
        <div class="kv-item metric-card">
          <span class="kv-label">PE</span>
          <span class="kv-value" :class="getPEClass(financials.pe_ratio)">{{
            financials.pe_ratio != null ? financials.pe_ratio.toFixed(2) : '--'
          }}</span>
          <span
            v-if="financials.pe_ratio != null"
            class="kv-indicator"
            :class="getPEClass(financials.pe_ratio)"
            >{{ getPELabel(financials.pe_ratio) }}</span
          >
        </div>
        <div class="kv-item metric-card">
          <span class="kv-label">PB</span>
          <span class="kv-value" :class="getPBClass(financials.pb_ratio)">{{
            financials.pb_ratio != null ? financials.pb_ratio.toFixed(2) : '--'
          }}</span>
          <span
            v-if="financials.pb_ratio != null"
            class="kv-indicator"
            :class="getPBClass(financials.pb_ratio)"
            >{{ getPBLabel(financials.pb_ratio) }}</span
          >
        </div>
        <div class="kv-item metric-card">
          <span class="kv-label">ROE</span>
          <span class="kv-value">{{
            financials.roe != null ? (financials.roe * 100).toFixed(2) + '%' : '--'
          }}</span>
        </div>
        <div class="kv-item metric-card">
          <span class="kv-label">EPS</span>
          <span class="kv-value">{{
            financials.eps != null ? financials.eps.toFixed(2) : '--'
          }}</span>
        </div>
        <div class="kv-item metric-card">
          <span class="kv-label">毛利率</span>
          <span class="kv-value">{{
            financials.gross_margin != null
              ? (financials.gross_margin * 100).toFixed(2) + '%'
              : '--'
          }}</span>
        </div>
        <div class="kv-item metric-card">
          <span class="kv-label">净利率</span>
          <span class="kv-value">{{
            financials.net_margin != null ? (financials.net_margin * 100).toFixed(2) + '%' : '--'
          }}</span>
        </div>
      </div>

      <div
        v-if="financials.quarterly && financials.quarterly.length"
        ref="chartRef"
        class="chart-wrap"
      />
      <div v-else class="empty-hint">暂无财务数据</div>
    </div>
  </section>
</template>

<style scoped>
.fade-slide-up {
  animation: fade-slide-up 0.5s var(--ease-out-expo) both;
  animation-delay: calc(var(--delay, 0) * 60ms);
}

.metrics-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--sp-2);
  margin-bottom: var(--sp-3);
}

.metric-card {
  padding: var(--sp-2_5);
  border-radius: var(--radius-md);
  background: var(--color-bg-elevated);
}

.metric-card .kv-value {
  font-size: var(--fs-base);
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-weight: var(--fw-semibold);
}

.kv-indicator {
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
  padding: 1px var(--sp-1);
  border-radius: var(--radius-sm);
  margin-top: 2px;
  letter-spacing: var(--ls-wide);
}

.kv-indicator.val-low {
  color: var(--color-up);
  background: var(--color-border-light);
}

.kv-indicator.val-high {
  color: var(--color-down);
  background: var(--color-border-light);
}

.kv-indicator.val-negative {
  color: var(--color-text-tertiary);
  background: var(--color-border-light);
}

.kv-value.val-low {
  color: var(--color-up);
}
.kv-value.val-high {
  color: var(--color-down);
}
.kv-value.val-negative {
  color: var(--color-text-tertiary);
}

.chart-wrap {
  width: 100%;
  height: 260px;
  padding: var(--sp-2) 0;
}

@media (max-width: 768px) {
  .metrics-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  .chart-wrap {
    height: 180px;
  }
}
</style>
