<template>
  <div class="chart-section">
    <div class="chart-header">
      <span class="chart-title">因子重要性分布</span>
    </div>
    <div class="chart-body">
      <div ref="chartRef" role="img" aria-label="预测因子重要性柱状图" style="width: 100%; height: 300px"></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useECharts } from '@/composables/useECharts'
import { usePredictionStore } from '@/stores/prediction'
import { useTheme } from '@/composables/useTheme'
import { cssVar } from '@/utils/format'

const store = usePredictionStore()
const { isDark } = useTheme()
const chartRef = ref<HTMLElement>()

useECharts(
  chartRef,
  () => {
    const pred = store.prediction?.prediction
    const factors = pred?.top_factors ?? []
    const brand = cssVar('--color-brand', '#175cd3')
    const axis = cssVar('--color-chart-axis', '#98a2b3')
    const gridLine = cssVar('--color-chart-grid', '#f2f4f7')

    return {
      tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
      grid: { left: 100, right: 40, top: 20, bottom: 30 },
      xAxis: {
        type: 'value',
        max: factors.length > 0 ? Math.max(0.4, ...factors.map(f => f.importance)) * 1.1 : 0.4,
        axisLabel: { color: axis, formatter: (v: number) => `${(v * 100).toFixed(0)}%` },
        splitLine: { lineStyle: { color: gridLine } },
      },
      yAxis: {
        type: 'category',
        data: factors.map((f) => f.name).reverse(),
        axisLabel: { color: axis, width: 80, overflow: 'truncate', fontSize: 12 },
        axisLine: { lineStyle: { color: gridLine } },
        axisTick: { lineStyle: { color: gridLine } },
      },
      series: [
        {
          type: 'bar',
          data: factors.map((f) => f.importance).reverse(),
          itemStyle: {
            color: brand,
            borderRadius: [0, 4, 4, 0],
          },
          barWidth: 16,
        },
      ],
    }
  },
  () => [store.prediction?.prediction?.top_factors, isDark.value]
)
</script>

<style scoped>
.chart-section {
  background: var(--color-bg-card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border);
  overflow: hidden;
  margin-bottom: var(--sp-5);
}
.chart-header {
  min-height: 42px;
  padding: 0 var(--sp-4);
  border-bottom: 1px solid var(--color-border);
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.chart-title {
  font-size: var(--fs-sm);
  font-weight: var(--fw-bold);
}
.chart-body {
  padding: var(--sp-4);
}
</style>
