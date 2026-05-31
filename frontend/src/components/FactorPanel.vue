<template>
  <div class="panel factors-panel card">
    <div class="panel-header">
      <span class="panel-mark"></span>
      <span>关键预测因子</span>
    </div>
    <div class="panel-body">
      <div v-for="(f, i) in factors" :key="f.name" class="factor-row">
        <span :class="['factor-rank', { top: i === 0 }]">{{ i + 1 }}</span>
        <div class="factor-info">
          <span class="factor-name">{{ f.name }}</span>
          <span class="factor-desc">{{ f.description }}</span>
        </div>
        <div class="factor-bar-wrap">
          <div class="factor-bar" :style="{ width: (f.importance * 100) + '%' }"></div>
        </div>
        <span class="factor-pct">{{ (f.importance * 100).toFixed(0) }}%</span>
      </div>
    </div>
  </div>

  <div class="panel chart-panel card">
    <div class="panel-header">
      <span class="panel-mark"></span>
      <span>因子重要性分布</span>
    </div>
    <div class="panel-body chart-body">
      <div ref="chartRef" role="img" aria-label="预测因子重要性柱状图"></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useECharts } from '@/composables/useECharts'
import { useTheme } from '@/composables/useTheme'
import { cssVar } from '@/utils/format'
import type { FactorItem } from '@/types/predict'

defineOptions({ name: 'FactorPanel' })

const props = defineProps<{
  factors: FactorItem[]
}>()

const { isDark } = useTheme()
const chartRef = ref<HTMLElement>()

useECharts(
  chartRef,
  () => {
    const factors = props.factors ?? []
    const brand = cssVar('--color-brand')
    const axis = cssVar('--color-chart-axis')
    const gridLine = cssVar('--color-chart-grid')
    return {
      tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
      grid: { left: 90, right: 24, top: 10, bottom: 20 },
      xAxis: {
        type: 'value',
        max: factors.length > 0 ? Math.max(0.4, ...factors.map(f => f.importance)) * 1.1 : 0.4,
        axisLabel: { color: axis, formatter: (v: number) => `${(v * 100).toFixed(0)}%`, fontSize: 11 },
        splitLine: { lineStyle: { color: gridLine } },
      },
      yAxis: {
        type: 'category',
        data: factors.map(f => f.name).reverse(),
        axisLabel: { color: axis, width: 72, overflow: 'truncate', fontSize: 11 },
        axisLine: { lineStyle: { color: gridLine } },
        axisTick: { lineStyle: { color: gridLine } },
      },
      series: [{
        type: 'bar',
        data: factors.map(f => f.importance).reverse(),
        itemStyle: {
          color: brand,
          borderRadius: [0, Number(cssVar('--radius-sm').replace('px','')), Number(cssVar('--radius-sm').replace('px','')), 0],
        },
        barWidth: 14,
      }],
    }
  },
  () => [props.factors, isDark.value]
)
</script>

<style scoped>
.panel {
  overflow: hidden;
}

.panel-header {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  min-height: 42px;
  padding: 0 var(--sp-4);
  border-bottom: 1px solid var(--color-border-light);
  color: var(--color-text-primary);
  font-size: var(--fs-sm);
  font-weight: var(--fw-bold);
}

.panel-mark {
  width: 8px;
  height: 8px;
  border-radius: var(--radius-sm);
  background: var(--color-brand);
}

.panel-body {
  padding: var(--sp-3) var(--sp-4);
}

.chart-body > div {
  width: 100%;
  height: 260px;
}

.factor-row {
  display: grid;
  grid-template-columns: 24px minmax(0, 1fr) 72px 36px;
  align-items: center;
  gap: var(--sp-2);
  min-height: 48px;
  border-bottom: 1px solid var(--color-border-light);
}

.factor-row:last-child {
  border-bottom: 0;
}

.factor-rank {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: var(--radius-sm);
  background: var(--color-bg-hover);
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
  font-weight: var(--fw-bold);
}

.factor-rank.top {
  color: var(--color-brand);
  background: var(--color-brand-soft);
}

.factor-info {
  min-width: 0;
}

.factor-name {
  display: block;
  overflow: hidden;
  color: var(--color-text-primary);
  font-size: var(--fs-xs);
  font-weight: var(--fw-bold);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.factor-desc {
  display: block;
  overflow: hidden;
  margin-top: 1px;
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.factor-bar-wrap {
  width: 72px;
  height: 5px;
  overflow: hidden;
  border-radius: var(--radius-sm);
  background: var(--color-bg-hover);
}

.factor-bar {
  height: 100%;
  border-radius: var(--radius-sm);
  background: var(--color-brand);
  transition: width 0.5s var(--ease-out-quart);
}

.factor-pct {
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
  text-align: right;
}

@media (max-width: 768px) {
  .factor-row {
    grid-template-columns: 24px minmax(0, 1fr) 36px;
  }

  .factor-bar-wrap {
    display: none;
  }
}
</style>
