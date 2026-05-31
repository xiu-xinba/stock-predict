import { onUnmounted } from 'vue'
import echarts from '@/utils/echarts'
import { colorWithAlpha, cssVar } from '@/utils/format'
import type { MarketIndex } from '@/types'

interface SparklineOptions {
  height?: number
  lineWidth?: number
  smooth?: number
}

export function useSparkline(options: SparklineOptions = {}) {
  const { lineWidth = 1.5, smooth = 0.4 } = options

  const chartInstances = new Map<string, echarts.ECharts>()
  const chartEls = new Map<string, HTMLElement>()
  const resizeRafs = new Map<string, number>()

  const sharedRO = new ResizeObserver((entries) => {
    for (const entry of entries) {
      const code = (entry.target as HTMLElement).dataset.sparklineCode
      if (!code) continue
      const prev = resizeRafs.get(code)
      if (prev) cancelAnimationFrame(prev)
      resizeRafs.set(code, requestAnimationFrame(() => {
        resizeRafs.delete(code)
        chartInstances.get(code)?.resize()
      }))
    }
  })

  function setChartRef(code: string, el: Element | null) {
    if (!el || !(el instanceof HTMLElement)) {
      chartEls.delete(code)
      const raf = resizeRafs.get(code)
      if (raf) cancelAnimationFrame(raf)
      resizeRafs.delete(code)
      chartInstances.get(code)?.dispose()
      chartInstances.delete(code)
      return
    }
    chartEls.set(code, el)
  }

  function initChart(code: string, el: HTMLElement, idx: MarketIndex) {
    const existing = chartInstances.get(code)
    if (existing) existing.dispose()

    const isUp = idx.change_pct >= 0
    const lineColor = isUp ? cssVar('--color-up') : cssVar('--color-down')
    const chartData = Array.isArray(idx.mini_chart_data) ? idx.mini_chart_data : []

    try {
      const chart = echarts.init(el)
      chart.setOption({
        grid: { top: 2, right: 2, bottom: 2, left: 2 },
        xAxis: {
          show: false,
          type: 'category',
          data: chartData.map((_: number, i: number) => i),
          boundaryGap: false,
        },
        yAxis: { show: false, type: 'value', min: 'dataMin' },
        series: [{
          type: 'line',
          data: chartData,
          smooth,
          showSymbol: false,
          lineStyle: { width: lineWidth, color: lineColor },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: colorWithAlpha(lineColor, 0.18) },
              { offset: 1, color: colorWithAlpha(lineColor, 0.02) },
            ]),
          },
        }],
        tooltip: { show: false },
        animation: true,
        animationDuration: 520,
        animationEasing: 'cubicOut',
      })
      chartInstances.set(code, chart)

      el.dataset.sparklineCode = code
      sharedRO.observe(el)
    } catch {
      chartInstances.get(code)?.dispose()
      chartInstances.delete(code)
    }
  }

  function initCharts(indices: MarketIndex[]) {
    indices.forEach(idx => {
      const el = chartEls.get(idx.code)
      if (el && el.clientWidth > 0 && el.clientHeight > 0) {
        initChart(idx.code, el, idx)
      }
    })
  }

  function disposeCharts(indices: MarketIndex[]) {
    indices.forEach(idx => {
      const el = chartEls.get(idx.code)
      if (el) sharedRO.unobserve(el)
      const raf = resizeRafs.get(idx.code)
      if (raf) cancelAnimationFrame(raf)
      resizeRafs.delete(idx.code)
      chartInstances.get(idx.code)?.dispose()
      chartInstances.delete(idx.code)
    })
  }

  function disposeAll() {
    chartEls.forEach((el) => sharedRO.unobserve(el))
    chartInstances.forEach(chart => chart.dispose())
    chartInstances.clear()
    resizeRafs.forEach(raf => cancelAnimationFrame(raf))
    resizeRafs.clear()
    chartEls.clear()
  }

  onUnmounted(disposeAll)

  return {
    setChartRef,
    initCharts,
    disposeCharts,
    disposeAll,
    chartInstances,
  }
}
