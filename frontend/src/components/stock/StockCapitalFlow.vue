<script setup lang="ts">
import { ref, computed } from 'vue'
import { useECharts } from '@/composables/useECharts'
import { useTheme } from '@/composables/useTheme'
import { cssVar, formatVolume, getDirection } from '@/utils/format'
import { getBaseChartOption } from '@/utils/echarts'
import CollapsibleCard from '@/components/CollapsibleCard.vue'
import type { StockCapitalFlow } from '@/types'

defineOptions({ name: 'StockCapitalFlow' })

const props = defineProps<{
  capitalFlow: StockCapitalFlow
}>()

const { isDark } = useTheme()
const chartRef = ref<HTMLElement>()

const mainDir = computed(() => getDirection(props.capitalFlow.main_net_inflow))
const retailDir = computed(() => getDirection(props.capitalFlow.retail_net_inflow))

function getChartOption() {
  const base = getBaseChartOption()
  const history = props.capitalFlow.flow_history || []
  if (!history.length) return {}

  const dates = history.map(h => h.date)
  const mainNetInflows = history.map(h => h.main_inflow - h.main_outflow)
  const retailNetInflows = history.map(h => h.retail_inflow - h.retail_outflow)

  let cumulative = 0
  const cumulativeNet = history.map(h => {
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
      textStyle: { color: cssVar('--color-chart-axis'), fontSize: Number(cssVar('--fs-xs').replace('px','')) },
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
          color: (params: any) => params.value >= 0 ? upColor : downColor,
          opacity: 0.7,
        },
        barMaxWidth: 12,
      },
      {
        name: '散户净流入',
        type: 'bar',
        data: retailNetInflows,
        itemStyle: {
          color: (params: any) => params.value >= 0 ? upColor : downColor,
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
  <CollapsibleCard title="资金流向" body-max-height="600px">
    <div class="flow-overview">
      <div class="flow-item">
        <span class="flow-label">主力净流入</span>
        <span class="flow-value" :class="mainDir">
          {{ formatVolume(capitalFlow.main_net_inflow) }}
        </span>
      </div>
      <div class="flow-item">
        <span class="flow-label">散户净流入</span>
        <span class="flow-value" :class="retailDir">
          {{ formatVolume(capitalFlow.retail_net_inflow) }}
        </span>
      </div>
    </div>

    <div class="chart-wrap" ref="chartRef" />
  </CollapsibleCard>
</template>

<style scoped>
.flow-overview {
  display: flex;
  gap: var(--sp-6);
  margin-bottom: var(--sp-3);
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
  font-size: var(--fs-lg);
  font-weight: var(--fw-semibold);
  font-family: var(--font-mono);
}

.flow-value.text-up { color: var(--color-up); }
.flow-value.text-down { color: var(--color-down); }
.flow-value.text-flat { color: var(--color-flat); }

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
