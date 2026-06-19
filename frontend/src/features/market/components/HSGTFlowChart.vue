<template>
  <div class="hsgt-flow card card-spotlight" @mousemove="handleSpotlight">
    <!-- Header: Title + Toolbar -->
    <div class="flow-head">
      <div class="flow-title-group">
        <span class="flow-eyebrow">CAPITAL FLOW</span>
        <h3 class="flow-title">沪深港通资金</h3>
      </div>
      <div v-if="!loading && !error && data" class="flow-toolbar">
        <div class="toolbar-group">
          <button
            :class="['toolbar-btn dir-btn', { active: direction === 'north' }]"
            type="button"
            aria-label="北向资金"
            @click="direction = 'north'"
          >
            <span class="dir-dot north"></span>北向
          </button>
          <button
            :class="['toolbar-btn dir-btn', { active: direction === 'south' }]"
            type="button"
            aria-label="南向资金"
            @click="direction = 'south'"
          >
            <span class="dir-dot south"></span>南向
          </button>
        </div>
        <div class="toolbar-divider"></div>
        <div class="toolbar-group">
          <button
            v-for="opt in timeRangeOptions"
            :key="opt.value"
            :class="['toolbar-btn', { active: timeRange === opt.value }]"
            type="button"
            :aria-label="opt.label"
            @click="timeRange = opt.value"
          >
            {{ opt.label }}
          </button>
        </div>
      </div>
    </div>

    <!-- Loading -->
    <template v-if="loading">
      <div class="flow-summary">
        <div class="summary-item" v-for="i in 3" :key="i">
          <span class="summary-label skeleton-pulse" style="width: 48px; height: 12px"></span>
          <span class="summary-value skeleton-pulse" style="width: 72px; height: 22px"></span>
        </div>
      </div>
      <div class="flow-chart skeleton-pulse"></div>
    </template>

    <!-- Error -->
    <template v-else-if="error">
      <div class="flow-state error">{{ error }}</div>
    </template>

    <!-- Data -->
    <template v-else-if="data && data.length > 0">
      <!-- Summary Row -->
      <div class="flow-summary">
        <div
          :class="[
            'summary-card',
            { 'summary-primary': selectedIndex === null },
            { 'summary-selected': selectedIndex !== null },
          ]"
        >
          <span class="summary-label"
            >{{ direction === 'north' ? '北向合计' : '南向合计'
            }}<span v-if="dirSummary?.metric" class="summary-metric"
              >（{{ dirSummary.metric }}）</span
            ></span
          >
          <span :class="['summary-value', summaryColorClass]">
            {{ dirSummary ? formatFlowValue(animatedTotal) : '--' }}
          </span>
        </div>
        <div class="summary-card">
          <span class="summary-label">{{ direction === 'north' ? '沪股通' : '港股通沪' }}</span>
          <span :class="['summary-value', summaryColorClass]">
            {{ dirSummary ? formatFlowValue(animatedSh) : '--' }}
          </span>
        </div>
        <div class="summary-card">
          <span class="summary-label">{{ direction === 'north' ? '深股通' : '港股通深' }}</span>
          <span :class="['summary-value', summaryColorClass]">
            {{ dirSummary ? formatFlowValue(animatedSz) : '--' }}
          </span>
        </div>
        <div class="summary-date">
          {{ dirSummary?.date ?? '' }}
        </div>
      </div>

      <!-- Chart -->
      <div class="flow-chart-wrap">
        <div ref="chartRef" class="flow-chart"></div>
      </div>
    </template>

    <!-- Empty -->
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
      <span>暂无沪深港通数据</span>
    </div>
  </div>
</template>

<script setup lang="ts">
/** 沪深港通资金流向组件，支持北向/南向切换，展示日/周/月维度走势图 */
import { computed, ref, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import type { HSGTFlowDaily, HSGTTimeRange, HSGTDirection } from '@/features/market/types'
import { cssVar, colorWithAlpha } from '@/shared/utils/format'
import { useTheme } from '@/shared/composables/useTheme'
import {
  aggregateHSGTData,
  getLatestDirectionSummary,
} from '@/features/market/utils/hsgtFlowAggregator'
import type { EChartsType } from 'echarts/core'

defineOptions({ name: 'HSGTFlowChart' })

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
  data: HSGTFlowDaily[] | null
  loading?: boolean
  error?: string | null
}>()

const direction = ref<HSGTDirection>('north')
const timeRange = ref<HSGTTimeRange>('daily')
const chartRef = ref<HTMLElement>()
let chartInstance: EChartsType | null = null
let resizeObserver: ResizeObserver | null = null
let resizeRaf: number | null = null
let themeObserver: MutationObserver | null = null

/** 点击柱子时选中的聚合数据索引，null 表示显示最新数据 */
const selectedIndex = ref<number | null>(null)

/** 动画数值：当 dirSummary 变化时，数字平滑过渡 */
const animatedTotal = ref(0)
const animatedSh = ref(0)
const animatedSz = ref(0)
let animFrameTotal: number | null = null
let animFrameSh: number | null = null
let animFrameSz: number | null = null
const ANIM_DURATION = 400 // ms

function animateValue(
  from: number,
  to: number,
  setter: (v: number) => void,
  cancelRef: { value: number | null },
) {
  const start = performance.now()
  const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  if (prefersReducedMotion || from === to) {
    setter(to)
    return
  }
  function tick(now: number) {
    const t = Math.min((now - start) / ANIM_DURATION, 1)
    // easeOutCubic
    const ease = 1 - Math.pow(1 - t, 3)
    setter(from + (to - from) * ease)
    if (t < 1) {
      cancelRef.value = requestAnimationFrame(tick)
    }
  }
  if (cancelRef.value) cancelAnimationFrame(cancelRef.value)
  cancelRef.value = requestAnimationFrame(tick)
}

const { isDark } = useTheme()

const timeRangeOptions: { label: string; value: HSGTTimeRange }[] = [
  { label: '日', value: 'daily' },
  { label: '周', value: 'weekly' },
  { label: '月', value: 'monthly' },
]

const aggregatedData = computed(() =>
  props.data ? aggregateHSGTData(props.data, timeRange.value) : [],
)

const dirSummary = computed(() => {
  if (!props.data) return null
  // 点击了柱子时显示选中日期的数据
  if (selectedIndex.value !== null && aggregatedData.value.length > selectedIndex.value) {
    const pt = aggregatedData.value[selectedIndex.value]
    const isNorth = direction.value === 'north'
    return {
      total: isNorth ? pt.north_total_amt : pt.south_total,
      sh: isNorth ? pt.north_sh_amt : pt.south_sh,
      sz: isNorth ? pt.north_sz_amt : pt.south_sz,
      date: pt.label,
      metric: isNorth ? '成交额' : '',
    }
  }
  // 默认显示最新数据
  return getLatestDirectionSummary(props.data, direction.value)
})

/** 北向成交额始终为正用品牌色，南向净买额有正负用涨跌色 */
const summaryColorClass = computed(() => {
  if (direction.value === 'north') return 'neutral'
  return dirSummary.value && dirSummary.value.total >= 0 ? 'up' : 'down'
})

/** 监听 dirSummary 变化，触发数字过渡动画 */
watch(
  () => dirSummary.value,
  (cur) => {
    if (!cur) return
    animateValue(
      animatedTotal.value,
      cur.total,
      (v) => {
        animatedTotal.value = v
      },
      { value: animFrameTotal },
    )
    animateValue(
      animatedSh.value,
      cur.sh,
      (v) => {
        animatedSh.value = v
      },
      { value: animFrameSh },
    )
    animateValue(
      animatedSz.value,
      cur.sz,
      (v) => {
        animatedSz.value = v
      },
      { value: animFrameSz },
    )
  },
  { immediate: true },
)

function handleSpotlight(e: MouseEvent) {
  const el = e.currentTarget as HTMLElement
  const rect = el.getBoundingClientRect()
  el.style.setProperty('--mouse-x', `${e.clientX - rect.left}px`)
  el.style.setProperty('--mouse-y', `${e.clientY - rect.top}px`)
}

/** 格式化万元数值（后端单位为万元） */
function formatFlowValue(val: number): string {
  if (val == null || isNaN(val)) return '--'
  // 后端返回万元，转为亿元显示
  const yiYuan = val / 10000
  const abs = Math.abs(yiYuan)
  const sign = yiYuan > 0 ? '+' : ''
  if (abs >= 100) return sign + yiYuan.toFixed(1) + '亿'
  if (abs >= 1) return sign + yiYuan.toFixed(2) + '亿'
  if (abs >= 0.01) return sign + yiYuan.toFixed(3) + '亿'
  return sign + yiYuan.toFixed(4) + '亿'
}

async function renderChart() {
  if (!chartRef.value || !props.data || aggregatedData.value.length === 0) return
  const el = chartRef.value
  if (el.clientWidth === 0 || el.clientHeight === 0) return

  const echarts = await loadECharts()
  const aggData = aggregatedData.value
  const labels = aggData.map((p) => p.label)
  const isNorth = direction.value === 'north'

  // 根据方向选择数据（北向用成交额，南向用净买额）
  const shData = isNorth ? aggData.map((p) => p.north_sh_amt) : aggData.map((p) => p.south_sh)
  const szData = isNorth ? aggData.map((p) => p.north_sz_amt) : aggData.map((p) => p.south_sz)

  const brandColor = cssVar('--color-brand')
  const p1Color = cssVar('--color-chart-p1')
  const p3Color = cssVar('--color-chart-p3')
  const axisColor = cssVar('--color-chart-axis')
  const gridColor = cssVar('--color-chart-grid')
  const textColor = cssVar('--color-text-primary')
  const surfaceColor = cssVar('--color-bg-elevated')

  const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches

  const shName = isNorth ? '沪股通成交额' : '港股通沪净买额'
  const szName = isNorth ? '深股通成交额' : '港股通深净买额'
  const totalName = isNorth ? '合计成交额' : '合计净买额'

  // 柱状图颜色：北向用品牌色系，南向用涨跌色
  const upColor = cssVar('--color-up')
  const downColor = cssVar('--color-down')

  // 构建带逐项样式的数据数组，确保图例与柱子颜色一致
  const shDataStyled = shData.map((v) => {
    return {
      value: v,
      itemStyle: {
        color: isNorth ? p1Color : v >= 0 ? upColor : downColor,
        borderRadius: [0, 0, 0, 0], // 底层系列始终无圆角
      },
    }
  })
  const szDataStyled = szData.map((v, i) => {
    const totalVal = shData[i] + v
    // 顶层系列：正值时顶部圆角，负值时底部圆角
    const radius = totalVal >= 0 ? [3, 3, 0, 0] : [0, 0, 3, 3]
    return {
      value: v,
      itemStyle: {
        color: isNorth
          ? p3Color
          : v >= 0
            ? colorWithAlpha(upColor, 0.7)
            : colorWithAlpha(downColor, 0.7),
        borderRadius: radius,
      },
    }
  })

  try {
    if (!chartInstance) {
      chartInstance = echarts.init(el)
      // 点击柱子时更新 summary + 视觉选中
      chartInstance.on('click', (params: any) => {
        if (params.componentType === 'series') {
          const idx = params.dataIndex as number
          const newIdx = selectedIndex.value === idx ? null : idx
          selectedIndex.value = newIdx
          // 通过 dispatchAction 实现柱子选中高亮
          if (newIdx !== null) {
            chartInstance!.dispatchAction({ type: 'downplay', seriesIndex: 0 })
            chartInstance!.dispatchAction({ type: 'downplay', seriesIndex: 1 })
            chartInstance!.dispatchAction({ type: 'highlight', seriesIndex: 0, dataIndex: newIdx })
            chartInstance!.dispatchAction({ type: 'highlight', seriesIndex: 1, dataIndex: newIdx })
          } else {
            chartInstance!.dispatchAction({ type: 'downplay', seriesIndex: 0 })
            chartInstance!.dispatchAction({ type: 'downplay', seriesIndex: 1 })
          }
        }
      })
      // 点击空白区域时取消选中
      chartInstance.getZr().on('click', (params: any) => {
        if (params.target == null) {
          selectedIndex.value = null
          chartInstance!.dispatchAction({ type: 'downplay', seriesIndex: 0 })
          chartInstance!.dispatchAction({ type: 'downplay', seriesIndex: 1 })
        }
      })
    }
    chartInstance!.setOption(
      {
        tooltip: {
          trigger: 'axis',
          axisPointer: { type: 'shadow', shadowStyle: { color: colorWithAlpha(textColor, 0.03) } },
          backgroundColor: surfaceColor,
          borderColor: 'transparent',
          borderWidth: 0,
          padding: [12, 16],
          extraCssText:
            'box-shadow: 0 12px 40px rgba(15, 23, 42, 0.18), 0 0 0 1px rgba(0,0,0,0.04); border-radius: 12px; backdrop-filter: blur(12px);',
          textStyle: { color: textColor, fontSize: 12, fontFamily: 'var(--font-mono)' },
          formatter: (params: any) => {
            if (!Array.isArray(params) || params.length === 0) return ''
            const date = params[0].axisValue
            // 计算合计
            let totalVal = 0
            params.forEach((p: any) => {
              totalVal += p.value as number
            })
            const totalDot = `<span style="display:inline-block;width:7px;height:7px;border-radius:2px;background:${brandColor};margin-right:8px;vertical-align:middle;"></span>`
            const totalRow = `${totalDot}<span style="color:var(--color-text-primary);font-size:12px;font-weight:600">${totalName}</span> <span style="float:right;font-weight:700;margin-left:16px">${formatFlowValue(totalVal)}</span>`
            const detailRows = params
              .map((p: any) => {
                const dot = `<span style="display:inline-block;width:7px;height:7px;border-radius:50%;background:${p.color};margin-right:8px;vertical-align:middle;"></span>`
                return `${dot}<span style="color:var(--color-text-secondary);font-size:12px">${p.seriesName}</span> <span style="float:right;font-weight:600;margin-left:16px">${formatFlowValue(p.value)}</span>`
              })
              .join('<div style="height:4px"></div>')
            return `<div style="margin-bottom:8px;color:var(--color-text-tertiary);font-size:11px;letter-spacing:0.02em">${date}</div>${totalRow}<div style="height:6px;border-bottom:1px solid var(--color-border-light);margin:6px 0"></div>${detailRows}`
          },
        },
        legend: {
          data: [shName, szName],
          bottom: 0,
          textStyle: { color: axisColor, fontSize: 11 },
          itemWidth: 14,
          itemHeight: 10,
          icon: 'roundRect',
          itemGap: 16,
        },
        grid: { top: 12, right: 16, bottom: 36, left: 52 },
        xAxis: {
          type: 'category',
          data: labels,
          boundaryGap: true,
          axisLine: { show: false },
          axisLabel: {
            color: axisColor,
            fontSize: 10,
            hideOverlap: true,
            interval: Math.max(0, Math.floor(labels.length / 5) - 1),
          },
          axisTick: { show: false },
        },
        yAxis: {
          type: 'value',
          scale: true,
          axisLine: { show: false },
          axisTick: { show: false },
          splitLine: {
            lineStyle: {
              color: gridColor,
              type: [3, 5],
              width: 1,
            },
          },
          axisLabel: {
            color: axisColor,
            fontSize: 10,
            formatter: (val: number) => formatFlowValue(val),
          },
        },
        series: [
          {
            name: shName,
            type: 'bar',
            stack: 'flow',
            data: shDataStyled,
            color: isNorth ? p1Color : upColor,
            z: 2,
            barMaxWidth: 24,
            emphasis: {
              itemStyle: {
                shadowBlur: 12,
                shadowColor: colorWithAlpha(brandColor, 0.35),
                borderColor: brandColor,
                borderWidth: 1,
              },
            },
            select: {
              itemStyle: {
                shadowBlur: 16,
                shadowColor: colorWithAlpha(brandColor, 0.4),
                borderColor: brandColor,
                borderWidth: 2,
              },
            },
          },
          {
            name: szName,
            type: 'bar',
            stack: 'flow',
            data: szDataStyled,
            color: isNorth ? p3Color : colorWithAlpha(upColor, 0.7),
            z: 1,
            barMaxWidth: 24,
            emphasis: {
              itemStyle: {
                shadowBlur: 12,
                shadowColor: colorWithAlpha(brandColor, 0.35),
                borderColor: brandColor,
                borderWidth: 1,
              },
            },
            select: {
              itemStyle: {
                shadowBlur: 16,
                shadowColor: colorWithAlpha(brandColor, 0.4),
                borderColor: brandColor,
                borderWidth: 2,
              },
            },
          },
        ],
        animation: !prefersReducedMotion,
        animationDuration: 600,
        animationEasing: 'cubicOut',
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
  () => [aggregatedData.value, isDark.value, direction.value],
  () => {
    selectedIndex.value = null
    if (aggregatedData.value.length) {
      nextTick(() => renderChart())
    }
  },
  { deep: true },
)

watch(
  () => props.data,
  () => {
    if (aggregatedData.value.length && chartRef.value) {
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
  if (animFrameTotal) cancelAnimationFrame(animFrameTotal)
  if (animFrameSh) cancelAnimationFrame(animFrameSh)
  if (animFrameSz) cancelAnimationFrame(animFrameSz)
  resizeRaf = null
  animFrameTotal = null
  animFrameSh = null
  animFrameSz = null
  resizeObserver?.disconnect()
  themeObserver?.disconnect()
  chartInstance?.dispose()
  resizeObserver = null
  themeObserver = null
  chartInstance = null
})
</script>

<style scoped>
/* ── Card Shell ── */
.hsgt-flow {
  padding: var(--sp-4) var(--sp-5);
  border-radius: var(--radius-lg);
  flex: 1;
  display: flex;
  flex-direction: column;
}

/* ── Header ── */
.flow-head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: var(--sp-3);
  margin-bottom: var(--sp-3);
}

.flow-title-group {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.flow-eyebrow {
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--color-brand-muted);
  font-family: var(--font-mono);
}

.flow-title {
  margin: 0;
  font-size: var(--fs-lg);
  font-weight: var(--fw-bold);
  color: var(--color-text-primary);
  letter-spacing: -0.01em;
}

/* ── Toolbar ── */
.flow-toolbar {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 3px;
  background: var(--color-bg-elevated);
  border-radius: 10px;
  border: 1px solid var(--color-border-light);
  box-shadow: inset 0 1px 0 color-mix(in srgb, var(--color-text-primary) 2%, transparent);
}

.toolbar-group {
  display: flex;
  align-items: center;
  gap: 2px;
}

.toolbar-divider {
  width: 1px;
  height: 14px;
  background: var(--color-border-light);
  margin: 0 2px;
}

.toolbar-btn {
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
  padding: 5px 10px;
  border-radius: 7px;
  color: var(--color-text-tertiary);
  background: transparent;
  border: none;
  cursor: pointer;
  transition:
    color 0.2s cubic-bezier(0.32, 0.72, 0, 1),
    background 0.2s cubic-bezier(0.32, 0.72, 0, 1),
    box-shadow 0.2s cubic-bezier(0.32, 0.72, 0, 1);
  white-space: nowrap;
  -webkit-user-select: none;
  user-select: none;
}

.toolbar-btn:hover {
  color: var(--color-text-secondary);
  background: color-mix(in srgb, var(--color-text-primary) 4%, transparent);
}

.toolbar-btn.active {
  font-weight: var(--fw-semibold);
  color: var(--color-text-primary);
  background: var(--color-bg-card);
  box-shadow:
    0 1px 3px rgba(15, 23, 42, 0.08),
    0 0 0 1px var(--color-border-light);
}

.toolbar-btn:active {
  transform: scale(0.97);
}

.dir-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 5px 12px;
  font-size: var(--fs-sm);
}

.dir-dot {
  display: inline-block;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  transition:
    box-shadow 0.25s cubic-bezier(0.32, 0.72, 0, 1),
    transform 0.25s cubic-bezier(0.32, 0.72, 0, 1);
}

.dir-dot.north {
  background: var(--color-brand);
}

.dir-dot.south {
  background: var(--color-chart-p2);
}

.dir-btn.active .dir-dot.north {
  box-shadow: 0 0 8px color-mix(in srgb, var(--color-brand) 60%, transparent);
  transform: scale(1.2);
}

.dir-btn.active .dir-dot.south {
  box-shadow: 0 0 8px color-mix(in srgb, var(--color-chart-p2) 60%, transparent);
  transform: scale(1.2);
}

/* ── Summary Row ── */
.flow-summary {
  display: flex;
  align-items: stretch;
  gap: var(--sp-2);
  margin-bottom: var(--sp-3);
}

.summary-card {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 3px;
  padding: var(--sp-2) var(--sp-3);
  border-radius: 12px;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border-light);
  transition:
    background 0.3s cubic-bezier(0.32, 0.72, 0, 1),
    border-color 0.3s cubic-bezier(0.32, 0.72, 0, 1);
}

.summary-card:hover {
  border-color: color-mix(in srgb, var(--color-brand) 20%, var(--color-border-light));
}

.summary-primary {
  flex: 1.4;
  background: color-mix(in srgb, var(--color-brand) 5%, var(--color-bg-elevated));
  border-color: color-mix(in srgb, var(--color-brand) 12%, var(--color-border-light));
}

.summary-selected {
  flex: 1.4;
  background: color-mix(in srgb, var(--color-brand) 8%, var(--color-bg-elevated));
  border-color: color-mix(in srgb, var(--color-brand) 30%, var(--color-border-light));
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--color-brand) 10%, transparent);
}

.summary-label {
  font-size: 10px;
  font-weight: 500;
  color: var(--color-text-tertiary);
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.summary-value {
  font-size: var(--fs-md);
  font-weight: var(--fw-bold);
  font-family: var(--font-mono);
  letter-spacing: -0.02em;
  line-height: 1.2;
  transition: color 0.3s cubic-bezier(0.32, 0.72, 0, 1);
}

.summary-value.up {
  color: var(--color-up);
}
.summary-value.down {
  color: var(--color-down);
}
.summary-value.neutral {
  color: var(--color-text-primary);
}

.summary-metric {
  font-size: 9px;
  color: var(--color-text-tertiary);
  margin-left: 2px;
}

.summary-date {
  display: flex;
  align-items: flex-end;
  font-size: 11px;
  font-family: var(--font-mono);
  color: var(--color-text-tertiary);
  padding-bottom: var(--sp-2);
  letter-spacing: 0.01em;
  white-space: nowrap;
}

/* ── Chart Area ── */
.flow-chart-wrap {
  border-radius: 12px;
  overflow: hidden;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border-light);
  padding: var(--sp-1);
  flex: 1;
  min-height: 0;
}

.flow-chart {
  width: 100%;
  height: 100%;
  min-height: 220px;
}

/* ── States ── */
.flow-state {
  padding: var(--sp-8);
  color: var(--color-text-secondary);
  font-size: var(--fs-sm);
  text-align: center;
}

.flow-state.error {
  color: var(--color-warning);
}

.empty-hint {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--sp-3);
  padding: var(--sp-12) 0;
  color: var(--color-text-tertiary);
  font-size: var(--fs-sm);
}

.empty-icon {
  width: 40px;
  height: 40px;
  opacity: 0.3;
  color: var(--color-text-tertiary);
}

/* ── Mobile ── */
@media (max-width: 768px) {
  .hsgt-flow {
    padding: var(--sp-3) var(--sp-4);
  }

  .flow-head {
    flex-direction: column;
    align-items: flex-start;
    gap: var(--sp-2);
  }

  .flow-chart {
    height: 220px;
  }

  .flow-summary {
    flex-wrap: wrap;
  }

  .summary-card {
    min-width: calc(50% - var(--sp-1));
  }

  .summary-date {
    width: 100%;
    padding-bottom: 0;
  }
}
</style>
