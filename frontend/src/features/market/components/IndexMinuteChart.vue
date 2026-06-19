<template>
  <section :class="['index-minute-chart', { 'chart-compact': compact }]">
    <div
      v-if="!minuteData || minuteData.length < 2"
      class="chart-state"
      :style="compact ? { height: chartHeight + 'px' } : {}"
    >
      <span v-if="loading" class="skeleton-line" />
      <template v-else-if="!compact">
        <span>暂无分时数据</span>
      </template>
      <span v-else class="skeleton-wave" />
    </div>
    <div v-else ref="chartRef" class="chart-container" :style="{ height: chartHeight + 'px' }" />
  </section>
</template>

<script setup lang="ts">
/** 指数分时走势图组件，支持多指数切换和缩放 */
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { MarketIndex, IndexMinutePoint } from '@/features/market/types'
import { cssVar, colorWithAlpha } from '@/shared/utils/format'
import { useTheme } from '@/shared/composables/useTheme'
import type { EChartsType } from 'echarts/core'

defineOptions({ name: 'IndexMinuteChart' })

import type echarts from '@/shared/charts/echarts'
type EChartsLib = typeof echarts

let echartsLoader: Promise<EChartsLib> | null = null

function loadECharts(): Promise<EChartsLib> {
  if (!echartsLoader) {
    echartsLoader = import('@/shared/charts/echarts').then((m) => m.default)
  }
  return echartsLoader
}

/** Parse "HH:MM" to minutes from midnight */
function parseHM(hm: string): number {
  const [h, m] = hm.split(':').map(Number)
  return h * 60 + m
}

// ── Props ──
const props = withDefaults(
  defineProps<{
    code: string
    minuteData: IndexMinutePoint[]
    quote: MarketIndex
    height?: number
    loading?: boolean
    compact?: boolean
  }>(),
  {
    height: 300,
    loading: false,
    compact: false,
  },
)

const { isDark } = useTheme()

const chartRef = ref<HTMLElement>()
let chartInstance: EChartsType | null = null
let resizeObserver: ResizeObserver | null = null
let resizeRaf: number | null = null
let themeObserver: MutationObserver | null = null

const chartHeight = computed(() => {
  if (typeof window !== 'undefined' && window.innerWidth < 768) {
    return Math.round(props.height * 0.8)
  }
  return props.height
})

async function renderChart() {
  await nextTick()
  if (!chartRef.value || !props.minuteData || props.minuteData.length < 2) return
  const el = chartRef.value
  if (el.clientWidth === 0 || el.clientHeight === 0) {
    requestAnimationFrame(() => renderChart())
    return
  }

  const echartsLib = await loadECharts()
  const data = props.minuteData
  const prevClose = props.quote?.prev_close ?? 0
  const changePct = props.quote?.change_pct ?? 0
  if (prevClose <= 0) return

  // ── Build sequential x-axis data (sessions joined seamlessly) ──
  const prices = data.map((d) => d.price)
  const avgPrices = data.map((d) => d.avg_price)
  const volumes = data.map((d) => d.volume)

  // Find session boundary indices by detecting time gaps in the data
  const sessionBoundaryIndices: number[] = []
  for (let i = 1; i < data.length; i++) {
    const prevMin = parseHM(data[i - 1].time)
    const currMin = parseHM(data[i].time)
    // Gap > 2 minutes means a session boundary (lunch break / midnight)
    let diff = currMin - prevMin
    if (diff < 0) diff += 24 * 60 // handle midnight crossing
    if (diff > 2) {
      sessionBoundaryIndices.push(i)
    }
  }

  // ── Colors ──
  const lineColor =
    changePct >= 0 ? cssVar('--color-up', '#ef4444') : cssVar('--color-down', '#22c55e')
  const avgColor = cssVar('--color-chart-ma5', '#f59e0b')
  const prevCloseColor = cssVar('--color-text-disabled', '#9ca3af')
  const textColor = cssVar('--color-chart-axis', '#9ca3af')
  const gridColor = cssVar('--color-chart-grid', '#e5e7eb')
  const upColor = cssVar('--color-up', '#ef4444')
  const downColor = cssVar('--color-down', '#22c55e')

  // ── Y-axis range: symmetric around prev_close ──
  const maxDev =
    Math.max(Math.abs(Math.max(...prices) - prevClose), Math.abs(Math.min(...prices) - prevClose)) *
    1.1
  const yMin = prevClose - maxDev
  const yMax = prevClose + maxDev

  // ── X-axis: sequential indices ──
  const n = data.length
  const xCategories: string[] = []
  for (let i = 0; i < n; i++) {
    xCategories.push(String(i))
  }

  // Build axis label map: sequential index -> time label
  const axisLabelMap = new Map<number, string>()
  // First point
  axisLabelMap.set(0, data[0].time)
  // Session boundaries: show "endTime/startTime"
  for (const idx of sessionBoundaryIndices) {
    axisLabelMap.set(idx, `${data[idx - 1].time}/${data[idx].time}`)
  }
  // Last point
  axisLabelMap.set(n - 1, data[n - 1].time)

  // ── Volume bar colors ──
  const volumeData = volumes.map((v, i) => ({
    value: v,
    itemStyle: {
      color: prices[i] >= prevClose ? upColor : downColor,
      opacity: 0.6,
    },
  }))

  // ── Session boundary markLines on x-axis ──
  const sessionMarkLines = sessionBoundaryIndices.map((idx) => ({
    xAxis: idx,
    lineStyle: {
      color: prevCloseColor,
      type: 'dashed' as const,
      width: 1,
    },
    label: { show: false },
  }))

  try {
    if (!chartInstance) {
      chartInstance = echartsLib.init(el)
    }

    // ── Compact mode: price line + gradient only ──
    if (props.compact) {
      chartInstance.setOption(
        {
          animation: true,
          animationDuration: 500,
          tooltip: { show: false },
          grid: { top: 0, right: 0, bottom: 0, left: 0 },
          xAxis: {
            type: 'category',
            data: xCategories,
            boundaryGap: false,
            show: false,
          },
          yAxis: {
            type: 'value',
            min: 'dataMin',
            show: false,
          },
          series: [
            {
              type: 'line',
              data: prices,
              symbol: 'none',
              smooth: false,
              lineStyle: { width: 1.5, color: lineColor },
              areaStyle: {
                color: new echartsLib.graphic.LinearGradient(0, 0, 0, 1, [
                  { offset: 0, color: colorWithAlpha(lineColor, 0.25) },
                  { offset: 1, color: colorWithAlpha(lineColor, 0) },
                ]),
              },
            },
          ],
        },
        true,
      )
      return
    }

    // ── Full mode ──
    chartInstance.setOption(
      {
        animation: true,
        animationDuration: 500,
        tooltip: {
          trigger: 'axis',
          backgroundColor: cssVar('--color-bg-card'),
          borderColor: cssVar('--color-border'),
          textStyle: { color: cssVar('--color-text-primary'), fontSize: 12 },
          formatter: (params: unknown) => {
            const p = Array.isArray(params) ? params : [params]
            const priceItem = p.find((item: { seriesName?: string }) => item.seriesName === '价格')
            if (!priceItem) return ''
            const idx = (priceItem as { dataIndex?: number }).dataIndex
            if (idx == null) return ''
            const point = data[idx]
            if (!point) return ''
            const pct = (((point.price - prevClose) / prevClose) * 100).toFixed(2)
            const sign = point.price >= prevClose ? '+' : ''
            return `${point.time}<br/>价格: ${point.price.toFixed(2)} (${sign}${pct}%)<br/>均价: ${point.avg_price.toFixed(2)}<br/>量: ${point.volume}`
          },
        },
        axisPointer: { link: [{ xAxisIndex: 'all' }] },
        grid: [
          { left: 56, right: 56, top: 16, height: '60%' },
          { left: 56, right: 56, top: '76%', height: '16%' },
        ],
        xAxis: [
          {
            type: 'category',
            data: xCategories,
            gridIndex: 0,
            boundaryGap: false,
            axisLine: { lineStyle: { color: gridColor } },
            axisTick: { show: false },
            splitLine: { show: false },
            axisLabel: {
              color: textColor,
              fontSize: 10,
              interval: (val: string) => {
                const num = Number(val)
                return axisLabelMap.has(num)
              },
              formatter: (val: string) => {
                const num = Number(val)
                return axisLabelMap.get(num) ?? ''
              },
            },
          },
          {
            type: 'category',
            data: xCategories,
            gridIndex: 1,
            boundaryGap: false,
            axisLine: { lineStyle: { color: gridColor } },
            axisLabel: { show: false },
            axisTick: { show: false },
            splitLine: { show: false },
          },
        ],
        yAxis: [
          {
            type: 'value',
            gridIndex: 0,
            min: yMin,
            max: yMax,
            axisLine: { show: false },
            axisTick: { show: false },
            splitLine: { lineStyle: { color: gridColor, type: 'dashed' } },
            axisLabel: {
              color: textColor,
              fontSize: 10,
              formatter: (v: number) => {
                const pct = (((v - prevClose) / prevClose) * 100).toFixed(2)
                return `${v.toFixed(2)}\n${v >= prevClose ? '+' : ''}${pct}%`
              },
            },
          },
          {
            type: 'value',
            gridIndex: 1,
            axisLine: { show: false },
            axisTick: { show: false },
            axisLabel: { show: false },
            splitLine: { show: false },
          },
        ],
        series: [
          {
            name: '价格',
            type: 'line',
            xAxisIndex: 0,
            yAxisIndex: 0,
            data: prices,
            symbol: 'none',
            smooth: false,
            lineStyle: { width: 1.5, color: lineColor },
            areaStyle: {
              color: new echartsLib.graphic.LinearGradient(0, 0, 0, 1, [
                { offset: 0, color: colorWithAlpha(lineColor, 0.25) },
                { offset: 1, color: colorWithAlpha(lineColor, 0.02) },
              ]),
            },
            markLine: {
              silent: true,
              symbol: 'none',
              data: [
                {
                  yAxis: prevClose,
                  lineStyle: { color: prevCloseColor, type: 'dashed', width: 1 },
                  label: {
                    show: true,
                    position: 'insideEndTop',
                    formatter: `昨收 ${prevClose.toFixed(2)}`,
                    color: prevCloseColor,
                    fontSize: 10,
                  },
                },
                ...sessionMarkLines,
              ],
            },
          },
          {
            name: '均价',
            type: 'line',
            xAxisIndex: 0,
            yAxisIndex: 0,
            data: avgPrices,
            symbol: 'none',
            smooth: false,
            lineStyle: { width: 1, color: avgColor, type: 'dashed' },
          },
          {
            name: '成交量',
            type: 'bar',
            xAxisIndex: 1,
            yAxisIndex: 1,
            data: volumeData.map((v) => ({
              value: v.value,
              itemStyle: v.itemStyle,
            })),
            barWidth: '60%',
          },
        ],
      },
      true,
    )
  } catch {
    chartInstance?.dispose()
    chartInstance = null
  }
}

function handleResize() {
  if (resizeRaf) cancelAnimationFrame(resizeRaf)
  resizeRaf = requestAnimationFrame(() => {
    resizeRaf = null
    chartInstance?.resize()
  })
}

watch(
  () => [props.minuteData, props.quote, isDark.value],
  () => {
    renderChart()
  },
  { deep: true },
)

watch(chartRef, (el) => {
  resizeObserver?.disconnect()
  if (el) {
    resizeObserver = new ResizeObserver(handleResize)
    resizeObserver.observe(el)
    renderChart()
  } else {
    chartInstance?.dispose()
    chartInstance = null
  }
})

/* 监听暗色模式切换，重新渲染图表 */
onMounted(() => {
  themeObserver = new MutationObserver(() => {
    if (chartInstance) {
      renderChart()
    }
  })
  themeObserver.observe(document.documentElement, {
    attributes: true,
    attributeFilter: ['class'],
  })
})

onBeforeUnmount(() => {
  if (resizeRaf) cancelAnimationFrame(resizeRaf)
  resizeObserver?.disconnect()
  themeObserver?.disconnect()
  chartInstance?.dispose()
  resizeObserver = null
  themeObserver = null
  chartInstance = null
})
</script>

<style scoped>
.index-minute-chart {
  width: 100%;
}

.index-minute-chart:not(.chart-compact) {
  padding: var(--sp-3) var(--sp-4) 0;
}

.index-minute-chart.chart-compact {
  padding: 0;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
}

.chart-container {
  width: 100%;
}

.chart-state {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 120px;
  color: var(--color-text-secondary);
  font-size: var(--fs-sm);
  transition: opacity 0.3s ease;
}

.chart-compact .chart-state {
  min-height: 0;
  overflow: hidden;
  border-radius: 4px;
}

.skeleton-wave {
  display: block;
  width: 100%;
  height: 100%;
  background: linear-gradient(
    90deg,
    transparent 0%,
    color-mix(in srgb, var(--color-text-primary) 4%, transparent) 40%,
    color-mix(in srgb, var(--color-text-primary) 6%, transparent) 50%,
    color-mix(in srgb, var(--color-text-primary) 4%, transparent) 60%,
    transparent 100%
  );
  background-size: 200% 100%;
  animation: skeleton-wave 2s ease-in-out infinite;
  border-radius: 4px;
}

@keyframes skeleton-wave {
  0% {
    background-position: 200% 0;
  }
  100% {
    background-position: -200% 0;
  }
}

.skeleton-line {
  display: inline-block;
  width: 120px;
  height: 14px;
  border-radius: 4px;
  background: linear-gradient(
    90deg,
    var(--color-border-light) 25%,
    var(--color-bg-hover, var(--color-border-light)) 50%,
    var(--color-border-light) 75%
  );
  background-size: 200% 100%;
  animation: skeleton-shimmer 1.5s ease-in-out infinite;
}

@keyframes skeleton-shimmer {
  0% {
    background-position: 200% 0;
  }
  100% {
    background-position: -200% 0;
  }
}

@media (max-width: 768px) {
  .index-minute-chart:not(.chart-compact) .chart-state {
    min-height: 96px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .skeleton-wave,
  .skeleton-line {
    animation: none;
  }
}
</style>
