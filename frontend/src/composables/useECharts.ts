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

  function renderChart() {
    if (!chartRef.value) return
    try {
      if (!chartInstance) {
        chartInstance = echarts.init(chartRef.value)
      }
      chartInstance.setOption(getOption(), true)
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
      chartInstance?.resize()
      resizeRaf = null
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
    resizeObserver?.disconnect()
    resizeObserver = null
    chartInstance?.dispose()
    chartInstance = null
  })

  return { renderChart }
}
