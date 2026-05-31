import * as echarts from 'echarts/core'
import { BarChart, CandlestickChart, LineChart, PieChart } from 'echarts/charts'
import { TitleComponent, TooltipComponent, GridComponent, LegendComponent, DataZoomComponent, MarkLineComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import { cssVar } from '@/utils/format'

echarts.use([BarChart, CandlestickChart, LineChart, PieChart, TitleComponent, TooltipComponent, GridComponent, LegendComponent, DataZoomComponent, MarkLineComponent, CanvasRenderer])

export function getBaseChartOption() {
  return {
    tooltip: {
      trigger: 'axis',
      backgroundColor: cssVar('--color-bg-card'),
      borderColor: cssVar('--color-border'),
      textStyle: { color: cssVar('--color-text-primary'), fontSize: 12 },
    },
    grid: { top: 20, right: 56, bottom: 28, left: 56 },
    xAxis: {
      type: 'category',
      axisLine: { lineStyle: { color: cssVar('--color-border') } },
      axisLabel: { color: cssVar('--color-chart-axis'), fontSize: 10 },
      axisTick: { show: false },
    },
    yAxis: {
      type: 'value',
      scale: true,
      axisLine: { show: false },
      axisTick: { show: false },
      splitLine: { lineStyle: { color: cssVar('--color-chart-grid'), type: 'dashed' } },
      axisLabel: { color: cssVar('--color-chart-axis'), fontSize: 10 },
    },
  }
}

export default echarts
