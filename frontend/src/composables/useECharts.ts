import { onMounted, onUnmounted, watch } from 'vue'
import type { Ref } from 'vue'
import echarts from '@/utils/echarts'
import type { EChartsType } from 'echarts/core'

export function useECharts(
  chartRef: Ref<HTMLElement | undefined>,
  getOption: () => Record<string, unknown>,
  watchSource?: () => unknown
) {
  let chartInstance: EChartsType | null = null
  let resizeObserver: ResizeObserver | null = null
  let resizeRaf: number | null = null
  let pendingInit = false

  function renderChart() {
    if (!chartRef.value) return
    const el = chartRef.value
    if (el.clientWidth === 0 || el.clientHeight === 0) {
      pendingInit = true
      return
    }
    try {
      if (!chartInstance) {
        chartInstance = echarts.init(el)
      }
      chartInstance.setOption(getOption(), true)
      pendingInit = false
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
      if (pendingInit && chartRef.value && chartRef.value.clientWidth > 0 && chartRef.value.clientHeight > 0) {
        renderChart()
        return
      }
      chartInstance?.resize()
    })
  }

  onMounted(() => {
    renderChart()
    if (chartRef.value) {
      resizeObserver = new ResizeObserver(handleResize)
      resizeObserver.observe(chartRef.value)
    }
  })

  if (watchSource) {
    watch(watchSource, () => renderChart())
  }

  onUnmounted(() => {
    if (resizeRaf) cancelAnimationFrame(resizeRaf)
    resizeRaf = null
    resizeObserver?.disconnect()
    resizeObserver = null
    chartInstance?.dispose()
    chartInstance = null
  })

  return { renderChart }
}
