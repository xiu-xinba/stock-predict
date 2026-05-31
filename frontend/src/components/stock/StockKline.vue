<script setup lang="ts">
import { ref, computed } from 'vue'
import { useECharts } from '@/composables/useECharts'
import { useTheme } from '@/composables/useTheme'
import { cssVar, colorWithAlpha } from '@/utils/format'
import { getBaseChartOption } from '@/utils/echarts'
import CollapsibleCard from '@/components/CollapsibleCard.vue'
import type { StockKlineData } from '@/types'

defineOptions({ name: 'StockKline' })

const props = defineProps<{
  kline: StockKlineData
}>()

const { isDark } = useTheme()
const chartRef = ref<HTMLElement>()
type PeriodKey = 'daily' | 'weekly' | 'monthly'

const period = ref<PeriodKey>('daily')

const periodLabels: Record<PeriodKey, string> = {
  daily: '日K',
  weekly: '周K',
  monthly: '月K',
}

const filteredKlines = computed(() => {
  const klines = props.kline.klines || []
  if (period.value === 'daily') return klines

  const groups = new Map<string, typeof klines>()
  for (const k of klines) {
    let key: string
    if (period.value === 'weekly') {
      const d = new Date(k.date)
      const jan1 = new Date(d.getFullYear(), 0, 1)
      const weekNum = Math.ceil(((d.getTime() - jan1.getTime()) / 86400000 + jan1.getDay() + 1) / 7)
      key = `${d.getFullYear()}-W${String(weekNum).padStart(2, '0')}`
    } else {
      key = k.date.slice(0, 7)
    }
    if (!groups.has(key)) groups.set(key, [])
    groups.get(key)!.push(k)
  }

  const result = []
  for (const [, chunk] of groups) {
    if (chunk.length === 0) continue
    result.push({
      date: chunk[0].date,
      open: chunk[0].open,
      close: chunk[chunk.length - 1].close,
      high: Math.max(...chunk.map(k => k.high)),
      low: Math.min(...chunk.map(k => k.low)),
      volume: chunk.reduce((s, k) => s + k.volume, 0),
      amount: chunk.reduce((s, k) => s + k.amount, 0),
    })
  }
  return result
})

function calculateMA(data: { close: number }[], period: number): (number | null)[] {
  return data.map((_, i) => {
    if (i < period - 1) return null
    let sum = 0
    for (let j = 0; j < period; j++) {
      sum += data[i - j].close
    }
    return sum / period
  })
}

function getChartOption() {
  const base = getBaseChartOption()
  const data = filteredKlines.value
  if (!data.length) return {}

  const dates = data.map(k => k.date)
  const ohlc = data.map(k => [k.open, k.close, k.low, k.high])
  const volumes = data.map(k => k.volume)
  const ma5 = calculateMA(data, 5)
  const ma10 = calculateMA(data, 10)
  const ma20 = calculateMA(data, 20)
  const upColor = cssVar('--color-up')
  const downColor = cssVar('--color-down')

  return {
    ...base,
    legend: {
      data: ['K线', 'MA5', 'MA10', 'MA20'],
      bottom: 0,
      textStyle: { color: cssVar('--color-chart-axis'), fontSize: Number(cssVar('--fs-xs').replace('px','')) },
      itemWidth: 12,
      itemHeight: 8,
    },
    grid: [
      { top: 20, right: 16, bottom: 100, left: 56 },
      { top: '68%', right: 16, bottom: 48, left: 56 },
    ],
    dataZoom: [
      { type: 'inside', xAxisIndex: [0, 1], start: 50, end: 100 },
      { type: 'slider', xAxisIndex: [0, 1], bottom: 8, height: 16, start: 50, end: 100, borderColor: 'transparent', fillerColor: colorWithAlpha(cssVar('--color-brand'), 0.15), handleStyle: { color: cssVar('--color-brand') }, textStyle: { color: cssVar('--color-chart-axis'), fontSize: Number(cssVar('--fs-xs').replace('px','')) } },
    ],
    xAxis: [
      {
        ...base.xAxis,
        type: 'category' as const,
        data: dates,
        boundaryGap: true,
        axisLabel: { show: false },
        gridIndex: 0,
      },
      {
        ...base.xAxis,
        type: 'category' as const,
        data: dates,
        boundaryGap: true,
        axisLabel: {
          ...base.xAxis.axisLabel,
          formatter: (val: string) => val.slice(5),
        },
        gridIndex: 1,
      },
    ],
    yAxis: [
      {
        ...base.yAxis,
        type: 'value' as const,
        scale: true,
        axisLabel: {
          ...base.yAxis.axisLabel,
          formatter: (val: number) => val.toFixed(2),
        },
        gridIndex: 0,
      },
      {
        ...base.yAxis,
        type: 'value' as const,
        scale: true,
        splitLine: { show: false },
        gridIndex: 1,
      },
    ],
    tooltip: {
      ...base.tooltip,
      axisPointer: { type: 'cross' },
      formatter: (params: any) => {
        if (!Array.isArray(params)) return ''
        const date = params[0]?.axisValue ?? ''
        let html = `<div style="font-size:${cssVar('--fs-sm')};margin-bottom:4px">${date}</div>`
        for (const p of params) {
          const color = p.color
          const name = p.seriesName
          const val = p.value
          if (name === 'K线' && Array.isArray(val)) {
            html += `<div style="display:flex;align-items:center;gap:4px;font-size:${cssVar('--fs-sm')}"><span style="display:inline-block;width:8px;height:8px;border-radius:50%;background:${color}"></span>${name} 开:${val[1]} 收:${val[2]} 低:${val[3]} 高:${val[4]}</div>`
          } else if (val !== null && val !== undefined && val !== '-') {
            html += `<div style="display:flex;align-items:center;gap:4px;font-size:${cssVar('--fs-sm')}"><span style="display:inline-block;width:8px;height:8px;border-radius:50%;background:${color}"></span>${name}: ${typeof val === 'number' ? val.toFixed(2) : val}</div>`
          }
        }
        return html
      },
    },
    series: [
      {
        name: 'K线',
        type: 'candlestick',
        data: ohlc,
        itemStyle: {
          color: upColor,
          color0: downColor,
          borderColor: upColor,
          borderColor0: downColor,
        },
        xAxisIndex: 0,
        yAxisIndex: 0,
      },
      {
        name: 'MA5',
        type: 'line',
        data: ma5,
        smooth: true,
        showSymbol: false,
        lineStyle: { width: 1, color: cssVar('--color-chart-ma5') },
        xAxisIndex: 0,
        yAxisIndex: 0,
      },
      {
        name: 'MA10',
        type: 'line',
        data: ma10,
        smooth: true,
        showSymbol: false,
        lineStyle: { width: 1, color: cssVar('--color-chart-ma10') },
        xAxisIndex: 0,
        yAxisIndex: 0,
      },
      {
        name: 'MA20',
        type: 'line',
        data: ma20,
        smooth: true,
        showSymbol: false,
        lineStyle: { width: 1, color: cssVar('--color-chart-ma20') },
        xAxisIndex: 0,
        yAxisIndex: 0,
      },
      {
        type: 'bar',
        data: volumes,
        itemStyle: {
          color: (params: any) => {
            const idx = params.dataIndex
            return data[idx].close >= data[idx].open ? upColor : downColor
          },
          opacity: 0.6,
        },
        xAxisIndex: 1,
        yAxisIndex: 1,
      },
    ],
    animation: true,
    animationDuration: 600,
  }
}

useECharts(chartRef, getChartOption, () => [filteredKlines.value, isDark.value])
</script>

<template>
  <CollapsibleCard title="K线走势" body-max-height="600px">
    <template #header-extra>
      <div class="period-tabs">
        <button
          v-for="(_, key) in periodLabels"
          :key="key"
          class="period-tab"
          :class="{ active: period === key }"
          @click.stop="period = key"
        >
          {{ periodLabels[key] }}
        </button>
      </div>
    </template>

    <div class="chart-wrap" ref="chartRef" />
  </CollapsibleCard>
</template>

<style scoped>
.period-tabs {
  display: flex;
  gap: var(--sp-1);
}

.period-tab {
  padding: var(--sp-1) var(--sp-3);
  border-radius: var(--radius-md);
  background: transparent;
  border: 1px solid transparent;
  font-size: var(--fs-xs);
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
  white-space: nowrap;
}

.period-tab.active {
  background: var(--color-bg-elevated);
  border-color: var(--color-border);
  color: var(--color-text-primary);
  font-weight: var(--fw-medium);
}

.chart-wrap {
  width: 100%;
  height: 360px;
}

@media (max-width: 768px) {
  .chart-wrap {
    height: 280px;
  }
}
</style>
