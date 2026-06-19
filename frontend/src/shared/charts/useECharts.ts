/** @module shared/charts/useECharts - ECharts 组合式函数，提供图表实例生命周期管理与自适应 resize */
import { onMounted, onUnmounted, watch } from 'vue'
import type { Ref } from 'vue'
import type { EChartsType } from 'echarts/core'

import type echarts from '@/shared/charts/echarts'
type EChartsLib = typeof echarts

let echartsLoader: Promise<EChartsLib> | null = null

function loadECharts(): Promise<EChartsLib> {
  if (!echartsLoader) {
    echartsLoader = import('@/shared/charts/echarts').then((m) => m.default)
  }
  return echartsLoader
}

/** ECharts 组合式函数，自动管理图表初始化、数据更新、容器 resize 及实例销毁
 * @param chartRef - 绑定图表容器的模板引用
 * @param getOption - 返回 ECharts option 的工厂函数
 * @param watchSource - 可选的响应式数据源，变化时自动重新渲染图表
 * @returns 包含 renderChart 手动渲染方法的对象
 */
export function useECharts(
  chartRef: Ref<HTMLElement | undefined>,
  getOption: () => Record<string, unknown>,
  watchSource?: () => unknown,
) {
  let chartInstance: EChartsType | null = null
  let resizeObserver: ResizeObserver | null = null
  let resizeRaf: number | null = null
  let pendingInit = false

  async function renderChart() {
    if (!chartRef.value) return
    const el = chartRef.value
    if (el.clientWidth === 0 || el.clientHeight === 0) {
      pendingInit = true
      return
    }
    const echarts = await loadECharts()
    try {
      if (!chartInstance) {
        chartInstance = echarts.init(el)
      }
      chartInstance!.setOption(getOption(), true)
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
      if (
        pendingInit &&
        chartRef.value &&
        chartRef.value.clientWidth > 0 &&
        chartRef.value.clientHeight > 0
      ) {
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
