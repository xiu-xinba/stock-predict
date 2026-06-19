<script setup lang="ts">
/** 股票资金流向组件，展示主力/散户净流入及累计净流入趋势图 */
import { ref, computed } from 'vue'
import { useECharts } from '@/shared/charts/useECharts'
import { useTheme } from '@/shared/composables/useTheme'
import { cssVar, formatVolume, getDirection } from '@/shared/utils/format'
import { getBaseChartOption } from '@/shared/charts/echarts'
import type { StockCapitalFlow } from '@/features/stocks/types'

defineOptions({ name: 'StockCapitalFlow' })

const props = defineProps<{
  capitalFlow: StockCapitalFlow
}>()

const { isDark } = useTheme()
const chartRef = ref<HTMLElement>()

const mainDir = computed(() => getDirection(props.capitalFlow.main_net_inflow))
const retailDir = computed(() => getDirection(props.capitalFlow.retail_net_inflow))

interface BarColorParam {
  value: number
}

function getChartOption() {
  const base = getBaseChartOption()
  const history = props.capitalFlow.flow_history || []
  if (!history.length) return {}

  const dates = history.map((h) => h.date)
  const mainNetInflows = history.map((h) => h.main_inflow - h.main_outflow)
  const retailNetInflows = history.map((h) => h.retail_inflow - h.retail_outflow)

  let cumulative = 0
  const cumulativeNet = history.map((h) => {
    cumulative += h.net_inflow
    return cumulative
  })

  const upColor = cssVar('--color-up')
  const downColor = cssVar('--color-down')

  return {
    ...base,
    legend: {
      data: ['主力净流入', '散户净流入', '累计净流入'],
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
        name: '主力净流入',
        type: 'bar',
        data: mainNetInflows,
        itemStyle: {
          color: (params: BarColorParam) => (params.value >= 0 ? upColor : downColor),
          opacity: 0.7,
        },
        barMaxWidth: 12,
      },
      {
        name: '散户净流入',
        type: 'bar',
        data: retailNetInflows,
        itemStyle: {
          color: (params: BarColorParam) => (params.value >= 0 ? upColor : downColor),
          opacity: 0.4,
        },
        barMaxWidth: 12,
      },
      {
        name: '累计净流入',
        type: 'line',
        yAxisIndex: 1,
        data: cumulativeNet,
        smooth: 0.3,
        showSymbol: false,
        lineStyle: { width: 2, color: cssVar('--color-brand') },
      },
    ],
    animation: true,
    animationDuration: 600,
  }
}

useECharts(chartRef, getChartOption, () => [props.capitalFlow.flow_history, isDark.value])
</script>

<template>
  <section class="card card-tier-secondary fade-slide-up" style="--delay: 1">
    <div class="card-header">
      <div class="card-title-wrap">
        <span class="card-eyebrow">CAPITAL FLOW</span>
        <h2 class="card-title">资金流向</h2>
      </div>
    </div>
    <div class="card-body">
      <div class="flow-overview">
        <div class="flow-item">
          <span class="flow-label">主力净流入</span>
          <span class="flow-value" :class="mainDir">
            <span class="flow-arrow">{{ capitalFlow.main_net_inflow >= 0 ? '▲' : '▼' }}</span>
            {{ formatVolume(capitalFlow.main_net_inflow) }}
          </span>
        </div>
        <div class="flow-item">
          <span class="flow-label">散户净流入</span>
          <span class="flow-value" :class="retailDir">
            <span class="flow-arrow">{{ capitalFlow.retail_net_inflow >= 0 ? '▲' : '▼' }}</span>
            {{ formatVolume(capitalFlow.retail_net_inflow) }}
          </span>
        </div>
      </div>

      <div ref="chartRef" class="chart-wrap" />
    </div>
  </section>
</template>

<style scoped>
.fade-slide-up {
  animation: fade-slide-up 0.5s var(--ease-out-expo) both;
  animation-delay: calc(var(--delay, 0) * 60ms);
}

.flow-overview {
  display: flex;
  gap: var(--sp-6);
  margin-bottom: var(--sp-3);
  padding: var(--sp-3);
  border-radius: var(--radius-md);
  background: var(--color-bg-elevated);
}

.flow-item {
  display: flex;
  flex-direction: column;
  gap: var(--sp-0_5);
}

.flow-label {
  font-size: var(--fs-xs);
  color: var(--color-text-tertiary);
}

.flow-value {
  font-size: var(--fs-2xl);
  font-weight: var(--fw-semibold);
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  display: flex;
  align-items: center;
  gap: 4px;
}

.flow-arrow {
  font-size: var(--fs-xs);
}

.flow-value.text-up {
  color: var(--color-up);
}
.flow-value.text-down {
  color: var(--color-down);
}
.flow-value.text-flat {
  color: var(--color-flat);
}

.chart-wrap {
  width: 100%;
  height: 260px;
}

@media (max-width: 768px) {
  .chart-wrap {
    height: 200px;
  }
}
</style>
