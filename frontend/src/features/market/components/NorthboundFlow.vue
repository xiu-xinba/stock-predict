<template>
  <div class="northbound-flow card card-spotlight" @mousemove="handleSpotlight">
    <div class="flow-head">
      <h3 class="flow-title">北向资金</h3>
      <template v-if="!loading && !error && flow">
        <span :class="['live-dot', isIntradayUnavailable ? 'live-dot--muted' : '']"></span>
        <span class="flow-subtitle">{{
          isIntradayUnavailable ? '披露状态' : '沪深股通资金流向'
        }}</span>
      </template>
    </div>
    <template v-if="loading">
      <div class="flow-summary">
        <div class="flow-item">
          <span class="flow-label skeleton-pulse" style="width: 40px; height: 14px"></span>
          <span class="flow-value skeleton-pulse" style="width: 80px; height: 20px"></span>
        </div>
        <div class="flow-item flow-item--center">
          <span class="flow-label skeleton-pulse" style="width: 40px; height: 14px"></span>
          <span
            class="flow-value flow-value--total skeleton-pulse"
            style="width: 100px; height: 24px"
          ></span>
        </div>
        <div class="flow-item">
          <span class="flow-label skeleton-pulse" style="width: 40px; height: 14px"></span>
          <span class="flow-value skeleton-pulse" style="width: 80px; height: 20px"></span>
        </div>
      </div>
      <div class="flow-chart skeleton-pulse"></div>
    </template>
    <template v-else-if="error">
      <div class="flow-state error">{{ error }}</div>
    </template>
    <template v-else-if="flow">
      <div v-if="isIntradayUnavailable" class="flow-unavailable">
        <svg
          class="empty-icon"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.5"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="M3 3v18h18" />
          <path d="M7 15h3l2-6 3 8 2-5h4" />
        </svg>
        <span class="unavailable-title">实时分时暂不可用</span>
        <span class="unavailable-copy">{{ northboundNotice }}</span>
      </div>
      <template v-else>
        <div class="flow-summary">
          <div class="flow-item">
            <span class="flow-label">沪股通</span>
            <span :class="['flow-value', flow.sh_net_buy >= 0 ? 'up' : 'down']">
              {{ flow.sh_net_buy > 0 ? '+' : '' }}{{ formatVolume(flow.sh_net_buy) }}
            </span>
          </div>
          <div class="flow-item flow-item--center">
            <span class="flow-label">总净买入</span>
            <span
              :class="['flow-value flow-value--total', flow.total_net_buy >= 0 ? 'up' : 'down']"
            >
              {{ flow.total_net_buy > 0 ? '+' : '' }}{{ formatVolume(flow.total_net_buy) }}
            </span>
          </div>
          <div class="flow-item">
            <span class="flow-label">深股通</span>
            <span :class="['flow-value', flow.sz_net_buy >= 0 ? 'up' : 'down']">
              {{ flow.sz_net_buy > 0 ? '+' : '' }}{{ formatVolume(flow.sz_net_buy) }}
            </span>
          </div>
        </div>
        <div class="flow-divider"></div>
        <div v-if="chartData.length > 0" class="flow-chart-wrap">
          <div class="flow-chart-meta">
            <span>{{ usingSummaryFallback ? '日汇总回退' : '最新' }} {{ latestValues?.time }}</span>
            <span
              :class="['meta-value', latestValues && latestValues.total_flow >= 0 ? 'up' : 'down']"
            >
              总计 {{ latestValues && latestValues.total_flow > 0 ? '+' : ''
              }}{{ formatVolume(latestValues?.total_flow ?? 0) }}
            </span>
          </div>
          <div ref="chartRef" class="flow-chart"></div>
        </div>
        <div v-else class="flow-chart-empty">
          <svg
            class="empty-icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.5"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <path d="M3 3v18h18" />
            <path d="M7 15h3l2-6 3 8 2-5h4" />
          </svg>
          <span>实时分时暂不可用</span>
        </div>
      </template>
    </template>
    <div v-else class="empty-hint">
      <svg
        class="empty-icon"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="1.5"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        <path d="M3 3v18h18" />
        <path d="M7 16l4-8 4 4 4-6" />
      </svg>
      <span>暂无北向资金数据</span>
    </div>
  </div>
</template>

<script setup lang="ts">
/** 北向资金流向组件，展示沪深股通净买入及分时流向图 */
import { computed, ref, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import type { NorthboundFlow } from '@/features/market/types'
import { formatVolume, cssVar, colorWithAlpha } from '@/shared/utils/format'
import { useTheme } from '@/shared/composables/useTheme'
import {
  buildNorthboundChartData,
  getNorthboundLatestValues,
} from '@/features/market/utils/northboundFlowChart'
import type { EChartsType } from 'echarts/core'

defineOptions({ name: 'NorthboundFlow' })

import type echarts from '@/shared/charts/echarts'
type EChartsLib = typeof echarts

let echartsLoader: Promise<EChartsLib> | null = null

function loadECharts(): Promise<EChartsLib> {
  if (!echartsLoader) {
    echartsLoader = import('@/shared/charts/echarts').then((m) => m.default)
  }
  return echartsLoader
}

const props = defineProps<{
  flow: NorthboundFlow | null
  loading?: boolean
  error?: string | null
}>()

const chartRef = ref<HTMLElement>()
let chartInstance: EChartsType | null = null
let resizeObserver: ResizeObserver | null = null
let resizeRaf: number | null = null
let themeObserver: MutationObserver | null = null

const { isDark } = useTheme()
const chartData = computed(() => buildNorthboundChartData(props.flow))
const latestValues = computed(() => getNorthboundLatestValues(chartData.value))
const isIntradayUnavailable = computed(() => props.flow?.status === 'intraday_unavailable')
const northboundNotice = computed(
  () => props.flow?.notice || '官方不再公开北向实时分时，未接入授权数据源',
)
const usingSummaryFallback = computed(() => {
  const rawCount = props.flow?.timeline?.length ?? 0
  return rawCount > 0 && chartData.value.length === 2 && chartData.value[0]?.time === '09:30'
})

function handleSpotlight(e: MouseEvent) {
  const el = e.currentTarget as HTMLElement
  const rect = el.getBoundingClientRect()
  el.style.setProperty('--mouse-x', `${e.clientX - rect.left}px`)
  el.style.setProperty('--mouse-y', `${e.clientY - rect.top}px`)
}

async function renderChart() {
  if (!chartRef.value || !props.flow || chartData.value.length === 0) return
  const el = chartRef.value
  if (el.clientWidth === 0 || el.clientHeight === 0) return

  const echarts = await loadECharts()
  const timeline = chartData.value
  const times = timeline.map((p) => p.time)
  const shFlows = timeline.map((p) => p.sh_flow)
  const szFlows = timeline.map((p) => p.sz_flow)
  const totalFlows = timeline.map((p) => p.total_flow)
  const lastIndex = timeline.length - 1

  const shColor = cssVar('--color-chart-p1')
  const szColor = cssVar('--color-chart-p3')
  const totalColor = cssVar('--color-brand')
  const axisColor = cssVar('--color-chart-axis')
  const gridColor = cssVar('--color-chart-grid')
  const borderColor = cssVar('--color-border')
  const cardColor = cssVar('--color-bg-card')
  const textColor = cssVar('--color-text-primary')

  const toSeriesData = (values: number[]) =>
    values.map((value, index) => ({
      value,
      symbolSize: index === lastIndex ? 7 : 0,
      itemStyle: index === lastIndex ? { borderWidth: 2, borderColor: cardColor } : undefined,
    }))

  try {
    if (!chartInstance) {
      chartInstance = echarts.init(el)
    }
    chartInstance!.setOption(
      {
        tooltip: {
          trigger: 'axis',
          backgroundColor: cardColor,
          borderColor,
          padding: [10, 12],
          extraCssText: 'box-shadow: 0 12px 32px rgba(15, 23, 42, 0.14); border-radius: 8px;',
          textStyle: { color: textColor, fontSize: 12 },
          valueFormatter: (value: number) => formatVolume(value),
        },
        legend: {
          data: ['总净买入', '沪股通', '深股通'],
          bottom: 0,
          textStyle: { color: axisColor, fontSize: 11 },
          itemWidth: 16,
          itemHeight: 8,
          icon: 'roundRect',
        },
        grid: { top: 18, right: 34, bottom: 36, left: 58 },
        xAxis: {
          type: 'category',
          data: times,
          boundaryGap: false,
          axisLine: { lineStyle: { color: borderColor } },
          axisLabel: {
            color: axisColor,
            fontSize: 10,
            hideOverlap: true,
            interval: Math.max(0, Math.floor(times.length / 5) - 1),
          },
          axisTick: { show: false },
        },
        yAxis: {
          type: 'value',
          scale: true,
          axisLine: { show: false },
          axisTick: { show: false },
          splitLine: { lineStyle: { color: gridColor, type: 'dashed' } },
          axisLabel: {
            color: axisColor,
            fontSize: 10,
            formatter: (val: number) => formatVolume(val),
          },
        },
        series: [
          {
            name: '总净买入',
            type: 'line',
            data: toSeriesData(totalFlows),
            smooth: 0.35,
            showSymbol: true,
            symbol: 'circle',
            z: 4,
            lineStyle: {
              width: 3,
              color: totalColor,
              shadowBlur: 6,
              shadowColor: colorWithAlpha(totalColor, 0.16),
            },
            itemStyle: { color: totalColor },
            endLabel: {
              show: true,
              formatter: (params: { value: number }) => formatVolume(params.value),
              color: totalColor,
              fontSize: 11,
              fontWeight: 700,
              distance: 6,
            },
            areaStyle: {
              color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                { offset: 0, color: colorWithAlpha(totalColor, 0.18) },
                { offset: 1, color: colorWithAlpha(totalColor, 0.01) },
              ]),
            },
            markLine: {
              symbol: 'none',
              silent: true,
              label: {
                show: true,
                formatter: '零轴',
                color: axisColor,
                fontSize: 10,
                position: 'insideEndTop',
              },
              lineStyle: { color: colorWithAlpha(axisColor, 0.48), type: 'dashed', width: 1 },
              data: [{ yAxis: 0 }],
            },
          },
          {
            name: '沪股通',
            type: 'line',
            data: toSeriesData(shFlows),
            smooth: 0.3,
            showSymbol: true,
            symbol: 'circle',
            z: 3,
            lineStyle: { width: 2, color: shColor, opacity: 0.82 },
            itemStyle: { color: shColor },
            endLabel: {
              show: true,
              formatter: (params: { value: number }) => formatVolume(params.value),
              color: shColor,
              fontSize: 10,
              distance: 6,
            },
            areaStyle: {
              color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                { offset: 0, color: colorWithAlpha(shColor, 0.08) },
                { offset: 1, color: colorWithAlpha(shColor, 0.01) },
              ]),
            },
          },
          {
            name: '深股通',
            type: 'line',
            data: toSeriesData(szFlows),
            smooth: 0.3,
            showSymbol: true,
            symbol: 'circle',
            z: 2,
            lineStyle: { width: 2, color: szColor, opacity: 0.82 },
            itemStyle: { color: szColor },
            endLabel: {
              show: true,
              formatter: (params: { value: number }) => formatVolume(params.value),
              color: szColor,
              fontSize: 10,
              distance: 6,
            },
            areaStyle: {
              color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                { offset: 0, color: colorWithAlpha(szColor, 0.08) },
                { offset: 1, color: colorWithAlpha(szColor, 0.01) },
              ]),
            },
          },
        ],
        animation: true,
        animationDuration: 600,
      },
      true,
    )
  } catch {
    if (chartInstance) {
      chartInstance.dispose()
      chartInstance = null
    }
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
  () => [props.flow, isDark.value],
  () => {
    if (chartData.value.length) {
      nextTick(() => renderChart())
    }
  },
  { deep: true },
)

// Ensure chart renders when component mounts with data
watch(
  () => props.flow,
  () => {
    if (chartData.value.length && chartRef.value) {
      nextTick(() => renderChart())
    }
  },
  { immediate: true },
)

watch(chartRef, (el) => {
  resizeObserver?.disconnect()
  if (el) {
    resizeObserver = new ResizeObserver(handleResize)
    resizeObserver.observe(el)
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
  resizeRaf = null
  resizeObserver?.disconnect()
  themeObserver?.disconnect()
  chartInstance?.dispose()
  resizeObserver = null
  themeObserver = null
  chartInstance = null
})
</script>

<style scoped>
.northbound-flow {
  padding: var(--sp-3) var(--sp-4);
  border-radius: var(--radius-lg);
}

.flow-head {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  margin-bottom: var(--sp-2);
}

.flow-title {
  margin: 0;
  font-size: var(--fs-md);
  font-weight: var(--fw-bold);
  color: var(--color-text-primary);
  border-left: 3px solid var(--color-brand);
  padding-left: var(--sp-2);
}

.flow-subtitle {
  font-size: var(--fs-xs);
  color: var(--color-text-tertiary);
  letter-spacing: var(--ls-wide);
}

.live-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--color-up);
  box-shadow: 0 0 0 4px color-mix(in srgb, var(--color-up) 14%, transparent);
}

.live-dot--muted {
  background: var(--color-warning);
  box-shadow: 0 0 0 4px color-mix(in srgb, var(--color-warning) 14%, transparent);
}

.flow-summary {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  gap: var(--sp-2);
  margin-bottom: var(--sp-3);
  align-items: center;
}

.flow-item {
  display: flex;
  flex-direction: column;
  gap: var(--sp-0_5);
}

.flow-item--center {
  position: relative;
  align-items: center;
  text-align: center;
  padding: 0 var(--sp-2);
  border-left: 1px solid var(--color-border-light);
  border-right: 1px solid var(--color-border-light);
}

.flow-label {
  font-size: var(--fs-xs);
  color: var(--color-text-tertiary);
  text-transform: uppercase;
  letter-spacing: var(--ls-wider);
}

.flow-value {
  font-size: var(--fs-base);
  font-weight: var(--fw-semibold);
  font-family: var(--font-mono);
}

.flow-value.up {
  color: var(--color-up);
}

.flow-value.down {
  color: var(--color-down);
}

.flow-value.up::before {
  content: '▲ ';
  font-size: 0.65em;
}

.flow-value.down::before {
  content: '▼ ';
  font-size: 0.65em;
}

.flow-value--total {
  font-size: var(--fs-lg);
  font-weight: var(--fw-bold);
}

.flow-divider {
  height: 1px;
  background: var(--color-border);
  margin-bottom: var(--sp-3);
  opacity: 0.5;
}

.flow-chart {
  width: 100%;
  height: 220px;
}

.flow-chart-wrap {
  border-top: 1px solid var(--color-border);
  padding-top: var(--sp-2);
}

.flow-chart-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--sp-2);
  padding: 0 var(--sp-1) var(--sp-1);
  color: var(--color-text-tertiary);
  font-size: var(--fs-xs);
  font-family: var(--font-mono);
}

.meta-value {
  font-weight: var(--fw-semibold);
}

.meta-value.up {
  color: var(--color-up);
}

.meta-value.down {
  color: var(--color-down);
}

.flow-chart-empty {
  display: flex;
  min-height: 220px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--sp-2);
  border-top: 1px solid var(--color-border);
  padding-top: var(--sp-2);
  color: var(--color-text-tertiary);
  font-size: var(--fs-sm);
}

.flow-state {
  padding: var(--sp-4);
  color: var(--color-text-secondary);
  font-size: var(--fs-sm);
  text-align: center;
}

.flow-state.error {
  color: var(--color-warning);
}

.flow-unavailable {
  display: flex;
  min-height: 220px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--sp-2);
  border-top: 1px solid var(--color-border);
  padding: var(--sp-4) var(--sp-2) var(--sp-3);
  color: var(--color-text-tertiary);
  text-align: center;
}

.unavailable-title {
  color: var(--color-text-secondary);
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
}

.unavailable-copy {
  max-width: 320px;
  color: var(--color-text-tertiary);
  font-size: var(--fs-xs);
  line-height: 1.6;
}

.empty-hint {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--sp-2);
  padding: var(--sp-8) 0;
  color: var(--color-text-tertiary);
  font-size: var(--fs-sm);
}

.empty-icon {
  width: 40px;
  height: 40px;
  opacity: 0.35;
  color: var(--color-text-tertiary);
}

@media (max-width: 768px) {
  .flow-chart {
    height: 220px;
  }

  .flow-chart-empty {
    min-height: 220px;
  }

  .flow-chart-meta {
    align-items: flex-start;
    flex-direction: column;
    gap: var(--sp-0_5);
  }
}
</style>
