<template>
  <div class="chart-section">
    <div class="chart-header">
      <span class="chart-title">📊 因子重要性分布</span>
    </div>
    <div class="chart-body">
      <div ref="chartRef" role="img" aria-label="预测因子重要性柱状图" style="width: 100%; height: 300px"></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import echarts from '@/utils/echarts'
import { useECharts } from '@/composables/useECharts'
import { usePredictionStore } from '@/stores/prediction'

const store = usePredictionStore()
const chartRef = ref<HTMLElement>()

useECharts(
  chartRef,
  () => {
    const pred = store.prediction?.prediction
    const factors = pred?.top_factors ?? []

    return {
      tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
      grid: { left: 100, right: 40, top: 20, bottom: 30 },
      xAxis: {
        type: 'value',
        max: factors.length > 0 ? Math.max(0.4, ...factors.map(f => f.importance)) * 1.1 : 0.4,
        axisLabel: { formatter: (v: number) => `${(v * 100).toFixed(0)}%` },
      },
      yAxis: {
        type: 'category',
        data: factors.map((f) => f.name).reverse(),
        axisLabel: { width: 80, overflow: 'truncate', fontSize: 12 },
      },
      series: [
        {
          type: 'bar',
          data: factors.map((f) => f.importance).reverse(),
          itemStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 1, 0, [
              { offset: 0, color: '#b3d8ff' },
              { offset: 1, color: '#3366ff' },
            ]),
            borderRadius: [0, 4, 4, 0],
          },
          barWidth: 16,
        },
      ],
    }
  },
  () => store.prediction?.prediction?.top_factors
)
</script>

<style scoped>
.chart-section {
  background: var(--color-bg-card);
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  overflow: hidden;
  margin-bottom: var(--sp-5);
}
.chart-header {
  padding: var(--sp-4) var(--sp-5);
  border-bottom: 1px solid var(--color-border);
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.chart-title {
  font-size: var(--fs-base);
  font-weight: 600;
}
.chart-body {
  padding: var(--sp-5);
}
</style>
