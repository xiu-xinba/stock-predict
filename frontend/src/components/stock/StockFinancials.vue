<script setup lang="ts">
import { ref } from 'vue'
import { useECharts } from '@/composables/useECharts'
import { useTheme } from '@/composables/useTheme'
import { cssVar } from '@/utils/format'
import { getBaseChartOption } from '@/utils/echarts'
import CollapsibleCard from '@/components/CollapsibleCard.vue'
import type { StockFinancials } from '@/types'

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

  const dates = quarterly.map(q => q.report_date)
  const revenues = quarterly.map(q => q.revenue)
  const netProfits = quarterly.map(q => q.net_profit)

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
</script>

<template>
  <CollapsibleCard title="财务指标" body-max-height="600px">
    <div class="metrics-grid">
      <div class="kv-item">
        <span class="kv-label">PE</span>
        <span class="kv-value">{{ financials.pe_ratio != null ? financials.pe_ratio.toFixed(2) : '--' }}</span>
      </div>
      <div class="kv-item">
        <span class="kv-label">PB</span>
        <span class="kv-value">{{ financials.pb_ratio != null ? financials.pb_ratio.toFixed(2) : '--' }}</span>
      </div>
      <div class="kv-item">
        <span class="kv-label">ROE</span>
        <span class="kv-value">{{ financials.roe != null ? (financials.roe * 100).toFixed(2) + '%' : '--' }}</span>
      </div>
      <div class="kv-item">
        <span class="kv-label">EPS</span>
        <span class="kv-value">{{ financials.eps != null ? financials.eps.toFixed(2) : '--' }}</span>
      </div>
      <div class="kv-item">
        <span class="kv-label">毛利率</span>
        <span class="kv-value">{{ financials.gross_margin != null ? (financials.gross_margin * 100).toFixed(2) + '%' : '--' }}</span>
      </div>
      <div class="kv-item">
        <span class="kv-label">净利率</span>
        <span class="kv-value">{{ financials.net_margin != null ? (financials.net_margin * 100).toFixed(2) + '%' : '--' }}</span>
      </div>
    </div>

    <div v-if="financials.quarterly && financials.quarterly.length" ref="chartRef" class="chart-wrap" />
    <div v-else class="empty-hint">暂无财务数据</div>
  </CollapsibleCard>
</template>

<style scoped>
.metrics-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--sp-3);
  margin-bottom: var(--sp-4);
}

.chart-wrap {
  width: 100%;
  height: 240px;
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
