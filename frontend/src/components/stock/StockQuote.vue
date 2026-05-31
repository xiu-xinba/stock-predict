<script setup lang="ts">
import { ref, computed } from 'vue'
import { useECharts } from '@/composables/useECharts'
import { useTheme } from '@/composables/useTheme'
import { cssVar, colorWithAlpha } from '@/utils/format'
import echarts, { getBaseChartOption } from '@/utils/echarts'
import CollapsibleCard from '@/components/CollapsibleCard.vue'
import type { StockQuote } from '@/types'

defineOptions({ name: 'StockQuote' })

const props = defineProps<{
  quote: StockQuote
}>()

const { isDark } = useTheme()
const chartRef = ref<HTMLElement>()
const hasIntradayData = computed(() => !!(props.quote.intradayData && props.quote.intradayData.length))

function getChartOption() {
  const base = getBaseChartOption()
  const lineColor = cssVar('--color-brand')
  const prevClose = props.quote.prev_close || 0

  return {
    ...base,
    grid: { top: 20, right: 16, bottom: 28, left: 56 },
    xAxis: {
      ...base.xAxis,
      type: 'category' as const,
      data: ['09:30', '10:00', '10:30', '11:00', '11:30', '13:00', '13:30', '14:00', '14:30', '15:00'],
      boundaryGap: false,
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
    series: [{
      type: 'line',
      data: [],
      smooth: 0.3,
      showSymbol: false,
      lineStyle: { width: 2, color: lineColor },
      areaStyle: {
        color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
          { offset: 0, color: colorWithAlpha(lineColor, 0.15) },
          { offset: 1, color: colorWithAlpha(lineColor, 0.01) },
        ]),
      },
      markLine: {
        silent: true,
        symbol: 'none',
        lineStyle: { color: cssVar('--color-flat'), type: 'dashed', width: 1 },
        data: [{ yAxis: prevClose, label: { show: false } }],
      },
    }],
    animation: true,
    animationDuration: 600,
  }
}

useECharts(chartRef, getChartOption, () => [props.quote, isDark.value])
</script>

<template>
  <CollapsibleCard title="分时走势" body-max-height="600px">
    <div v-if="hasIntradayData" class="chart-wrap" ref="chartRef" />
    <div v-else class="empty-hint">分时数据暂不可用</div>

    <div class="bid-ask-section">
      <div class="bid-ask-col">
        <h3 class="bid-ask-title">买盘</h3>
        <div class="bid-ask-row">
          <span class="bid-ask-label">买一</span>
          <span class="bid-ask-value text-up">{{ (quote.bid_price || 0).toFixed(2) }}</span>
        </div>
      </div>
      <div class="bid-ask-col">
        <h3 class="bid-ask-title">卖盘</h3>
        <div class="bid-ask-row">
          <span class="bid-ask-label">卖一</span>
          <span class="bid-ask-value text-down">{{ (quote.ask_price || 0).toFixed(2) }}</span>
        </div>
      </div>
    </div>
  </CollapsibleCard>
</template>

<style scoped>
.chart-wrap {
  width: 100%;
  height: 220px;
  margin-bottom: var(--sp-3);
}

.empty-hint {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 120px;
  margin-bottom: var(--sp-3);
  color: var(--color-text-tertiary);
  font-size: var(--fs-sm);
}

.bid-ask-section {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--sp-4);
}

.bid-ask-title {
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
  color: var(--color-text-secondary);
  margin: 0 0 var(--sp-2) 0;
}

.bid-ask-row {
  display: flex;
  justify-content: space-between;
  padding: var(--sp-1) 0;
}

.bid-ask-label {
  font-size: var(--fs-xs);
  color: var(--color-text-tertiary);
}

.bid-ask-value {
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
  font-family: var(--font-mono);
}

.bid-ask-value.text-up { color: var(--color-up); }
.bid-ask-value.text-down { color: var(--color-down); }

@media (max-width: 768px) {
  .chart-wrap {
    height: 180px;
  }
}
</style>
