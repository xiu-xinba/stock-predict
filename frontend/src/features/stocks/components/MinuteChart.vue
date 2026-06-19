<script setup lang="ts">
/** 股票走势图组件，支持分时、日K、周K、月K 切换展示。
 * 阶段3增强功能：
 * 1. 十字光标 — 鼠标移动显示价格/时间十字线 + 数值标签
 * 2. 缩放拖拽 — 滚轮缩放 + 拖拽平移
 * 3. 技术指标参数自定义 — MACD/KDJ 参数可调
 * 4. 复权处理 — 前复权/后复权/不复权切换
 * 5. 对比叠加 — 叠加大盘指数对比走势
 */
import { ref, computed, watch, nextTick } from 'vue'
import { useECharts } from '@/shared/charts/useECharts'
import { cssVar, colorWithAlpha } from '@/shared/utils/format'
import { useTheme } from '@/shared/composables/useTheme'
import { fetchStockKline } from '@/features/stocks/api/stocks'
import { fetchIndexKline } from '@/features/market/api/market'
import type { IndexKlinePoint } from '@/features/market/types'
import type { StockQuote, StockKlineData, MinutePoint, KlinePoint } from '@/features/stocks/types'

const props = defineProps<{
  stockCode: string
  quote: StockQuote
  kline: StockKlineData
  minuteData: MinutePoint[]
}>()

const { isDark } = useTheme()

// === 周期与指标 ===
const tabs = [
  { label: '分时', value: 'minute' as const },
  { label: '日K', value: 'daily' as const },
  { label: '周K', value: 'weekly' as const },
  { label: '月K', value: 'monthly' as const },
]

const indicators = [
  { label: 'VOL', value: 'vol' as const },
  { label: 'MACD', value: 'macd' as const },
  { label: 'KDJ', value: 'kdj' as const },
]

const activeTab = ref<'minute' | 'daily' | 'weekly' | 'monthly'>('minute')
const activeIndicator = ref<'vol' | 'macd' | 'kdj'>('vol')
const minuteChartRef = ref<HTMLElement>()
const klineChartRef = ref<HTMLElement>()

// === 复权处理 ===
const adjustTypes = [
  { label: '前复权', value: 1 as const },
  { label: '后复权', value: 2 as const },
  { label: '不复权', value: 0 as const },
]
const adjustType = ref(1)

// === 对比叠加 ===
const compareOptions = [
  { label: '无', value: '' },
  { label: '上证指数', value: '000001' },
  { label: '沪深300', value: '399300' },
  { label: '深证成指', value: '399001' },
]
const compareIndex = ref('')

// === 技术指标参数自定义 ===
const showParamPanel = ref(false)
const macdParams = ref({ fast: 12, slow: 26, signal: 9 })
const kdjParams = ref({ n: 9 })

// === 复权 K 线数据获取 ===
// 当复权类型变化时，从后端获取新的 K 线数据
const customKline = ref<StockKlineData | null>(null)
const klineLoading = ref(false)

watch(
  [() => props.stockCode, adjustType],
  async ([code, fq]) => {
    if (!code || activeTab.value === 'minute') return
    klineLoading.value = true
    try {
      const res = await fetchStockKline(code, 'daily', fq)
      if (res.code === 0 && res.data) {
        customKline.value = res.data
      }
    } catch {
      // 获取失败时回退到 prop 数据
      customKline.value = null
    } finally {
      klineLoading.value = false
    }
  },
  { immediate: false },
)

// 切换到 K 线模式时，如果复权类型不是默认的前复权，需要获取数据
watch(activeTab, (tab) => {
  if (tab !== 'minute' && adjustType.value !== 1 && !customKline.value && props.stockCode) {
    fetchStockKline(props.stockCode, 'daily', adjustType.value).then((res) => {
      if (res.code === 0 && res.data) customKline.value = res.data
    })
  }
})

// === 对比指数 K 线数据 ===
const compareKlineData = ref<IndexKlinePoint[]>([])

watch(
  [compareIndex, () => props.stockCode],
  async ([code]) => {
    if (!code) {
      compareKlineData.value = []
      return
    }
    try {
      const res = await fetchIndexKline(code, 120)
      if (res.code === 0 && res.data) {
        compareKlineData.value = res.data
      }
    } catch {
      compareKlineData.value = []
    }
  },
  { immediate: false },
)

// === 实际使用的 K 线数据（优先使用复权数据） ===
const effectiveKline = computed<StockKlineData>(() => {
  if (customKline.value && customKline.value.klines?.length) {
    return customKline.value
  }
  return props.kline
})

// === 分时图 option ===
function getMinuteOption() {
  const data = props.minuteData
  if (!data || data.length === 0) return {}

  const times = data.map((d) => d.time)
  const prices = data.map((d) => d.price)
  const avgPrices = data.map((d) => d.avg_price)
  const volumes = data.map((d) => d.volume)
  const prevClose = props.quote.prev_close || data[0]?.price || 1

  const textColor = cssVar('--color-chart-axis')
  const gridLineColor = cssVar('--color-chart-grid')
  const upColor = cssVar('--color-up')
  const downColor = cssVar('--color-down')

  const maxDev = Math.max(
    Math.abs(Math.max(...prices) - prevClose),
    Math.abs(Math.min(...prices) - prevClose),
  )
  const yMin = prevClose - maxDev * 1.1
  const yMax = prevClose + maxDev * 1.1

  const volumeColors = data.map((d, i) => {
    const prevPrice = i > 0 ? data[i - 1].price : prevClose
    return d.price >= prevPrice ? upColor : downColor
  })

  return {
    animation: false,
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'cross',
        crossStyle: { color: cssVar('--color-text-disabled') },
        label: {
          backgroundColor: cssVar('--color-brand'),
          color: '#fff',
          fontSize: 10,
        },
      },
      formatter: (params: any[]) => {
        if (!params || !params.length) return ''
        const time = params[0].axisValue
        let html = `<div style="font-size:10px;color:${cssVar('--color-text-tertiary')};margin-bottom:4px">${time}</div>`
        params.forEach((p) => {
          const val = typeof p.value === 'number' ? p.value.toFixed(2) : p.value
          html += `<div style="font-size:11px"><span style="display:inline-block;width:8px;height:8px;border-radius:50%;background:${p.color};margin-right:6px"></span>${p.seriesName}: <span style="font-family:monospace;font-weight:600">${val}</span></div>`
        })
        return html
      },
    },
    axisPointer: { link: [{ xAxisIndex: 'all' }] },
    grid: [
      { left: 56, right: 56, top: 30, height: '55%' },
      { left: 56, right: 56, top: '78%', height: '20%' },
    ],
    xAxis: [
      {
        type: 'category',
        data: times,
        gridIndex: 0,
        axisLine: { lineStyle: { color: gridLineColor } },
        axisLabel: { color: textColor, fontSize: 10 },
        axisTick: { show: false },
        splitLine: { show: false },
        axisPointer: {
          label: { formatter: (v: any) => v.value },
        },
      },
      {
        type: 'category',
        data: times,
        gridIndex: 1,
        axisLine: { lineStyle: { color: gridLineColor } },
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
        axisLabel: {
          color: textColor,
          fontSize: 10,
          formatter: (v: number) => {
            const pct = (((v - prevClose) / prevClose) * 100).toFixed(2)
            return `${v.toFixed(2)}\n${v >= prevClose ? '+' : ''}${pct}%`
          },
        },
        splitLine: { lineStyle: { color: gridLineColor, type: 'dashed' } },
        axisPointer: {
          label: {
            formatter: (v: any) => Number(v.value).toFixed(2),
          },
        },
      },
      {
        type: 'value',
        gridIndex: 1,
        axisLine: { show: false },
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
        lineStyle: { width: 1.5, color: cssVar('--color-brand') },
        areaStyle: {
          color: {
            type: 'linear',
            x: 0,
            y: 0,
            x2: 0,
            y2: 1,
            colorStops: [
              { offset: 0, color: colorWithAlpha(cssVar('--color-brand'), 0.35) },
              { offset: 1, color: colorWithAlpha(cssVar('--color-brand'), 0) },
            ],
          },
        },
        markLine: {
          silent: true,
          symbol: 'none',
          data: [
            {
              yAxis: prevClose,
              lineStyle: { color: cssVar('--color-text-disabled'), type: 'dashed', width: 1 },
            },
          ],
          label: { show: false },
        },
      },
      {
        name: '均价',
        type: 'line',
        xAxisIndex: 0,
        yAxisIndex: 0,
        data: avgPrices,
        symbol: 'none',
        lineStyle: { width: 1, color: cssVar('--color-chart-ma5'), type: 'dashed' },
      },
      {
        name: '成交量',
        type: 'bar',
        xAxisIndex: 1,
        yAxisIndex: 1,
        data: volumes.map((v, i) => ({
          value: v,
          itemStyle: { color: volumeColors[i], opacity: 0.6 },
        })),
        barWidth: '60%',
      },
    ],
  }
}

// === K 线聚合 ===
const filteredKlines = computed<KlinePoint[]>(() => {
  const klines = effectiveKline.value.klines
  if (!klines || klines.length === 0) return []

  switch (activeTab.value) {
    case 'weekly':
      return aggregateKlines(klines, (d) => {
        const dt = new Date(d.date)
        const start = new Date(dt)
        start.setDate(dt.getDate() - dt.getDay() + 1)
        return `${start.getFullYear()}-${String(start.getMonth() + 1).padStart(2, '0')}-W${getWeekNumber(dt)}`
      })
    case 'monthly':
      return aggregateKlines(klines, (d) => d.date.substring(0, 7))
    default:
      return klines
  }
})

function getWeekNumber(d: Date): number {
  const start = new Date(d.getFullYear(), 0, 1)
  const diff = d.getTime() - start.getTime()
  return Math.ceil((diff / 86400000 + start.getDay() + 1) / 7)
}

function aggregateKlines(klines: KlinePoint[], keyFn: (d: KlinePoint) => string): KlinePoint[] {
  const groups = new Map<string, KlinePoint[]>()
  klines.forEach((k) => {
    const key = keyFn(k)
    if (!groups.has(key)) groups.set(key, [])
    groups.get(key)!.push(k)
  })
  const result: KlinePoint[] = []
  groups.forEach((items) => {
    const first = items[0]
    const last = items[items.length - 1]
    result.push({
      date: first.date,
      open: first.open,
      close: last.close,
      high: Math.max(...items.map((i) => i.high)),
      low: Math.min(...items.map((i) => i.low)),
      volume: items.reduce((s, i) => s + i.volume, 0),
      amount: items.reduce((s, i) => s + i.amount, 0),
    })
  })
  return result
}

function calculateMA(data: KlinePoint[], dayCount: number) {
  return data.map((_, i) => {
    if (i < dayCount - 1) return null
    let sum = 0
    for (let j = 0; j < dayCount; j++) sum += data[i - j].close
    return +(sum / dayCount).toFixed(2)
  })
}

// === MACD 计算（支持自定义参数） ===
function calculateEMA(data: number[], period: number): number[] {
  const k = 2 / (period + 1)
  const ema: number[] = []
  data.forEach((v, i) => {
    if (i === 0) ema.push(v)
    else ema.push(v * k + ema[i - 1] * (1 - k))
  })
  return ema
}

function calculateMACD(data: KlinePoint[], fast = 12, slow = 26, signal = 9) {
  const closes = data.map((d) => d.close)
  const emaFast = calculateEMA(closes, fast)
  const emaSlow = calculateEMA(closes, slow)
  const dif = closes.map((_, i) => +(emaFast[i] - emaSlow[i]).toFixed(4))
  const dea = calculateEMA(dif, signal).map((v) => +v.toFixed(4))
  const macd = dif.map((d, i) => +((d - dea[i]) * 2).toFixed(4))
  return { dif, dea, macd }
}

// === KDJ 计算（支持自定义参数） ===
function calculateKDJ(data: KlinePoint[], n = 9) {
  let k = 50
  let d = 50
  const kArr: number[] = []
  const dArr: number[] = []
  const jArr: number[] = []
  data.forEach((item, i) => {
    const start = Math.max(0, i - n + 1)
    const slice = data.slice(start, i + 1)
    const hn = Math.max(...slice.map((s) => s.high))
    const ln = Math.min(...slice.map((s) => s.low))
    const rsv = hn === ln ? 50 : ((item.close - ln) / (hn - ln)) * 100
    k = (2 / 3) * k + (1 / 3) * rsv
    d = (2 / 3) * d + (1 / 3) * k
    const j = 3 * k - 2 * d
    kArr.push(+k.toFixed(2))
    dArr.push(+d.toFixed(2))
    jArr.push(+j.toFixed(2))
  })
  return { k: kArr, d: dArr, j: jArr }
}

// === 对比指数归一化（百分比涨跌幅） ===
function normalizeCompareIndex(
  stockData: KlinePoint[],
  compareData: IndexKlinePoint[],
): (number | null)[] {
  if (!compareData.length || !stockData.length) return []
  const compareMap = new Map<string, number>()
  compareData.forEach((d) => compareMap.set(d.date, d.close))
  const firstCompare = compareData[0]?.close
  if (!firstCompare) return []
  return stockData.map((d) => {
    const cmpClose = compareMap.get(d.date)
    if (cmpClose == null) return null
    const cmpPct = ((cmpClose - firstCompare) / firstCompare) * 100
    return +cmpPct.toFixed(2)
  })
}

// === K 线图 option ===
function getKlineOption() {
  const data = filteredKlines.value
  if (!data || data.length === 0) return {}

  const textColor = cssVar('--color-chart-axis')
  const gridLineColor = cssVar('--color-chart-grid')
  const upColor = cssVar('--color-up')
  const downColor = cssVar('--color-down')

  const dates = data.map((d) => d.date)
  const ohlc = data.map((d) => [d.open, d.close, d.low, d.high])
  const volumes = data.map((d) => d.volume)
  const ma5 = calculateMA(data, 5)
  const ma10 = calculateMA(data, 10)
  const ma20 = calculateMA(data, 20)

  // 是否有对比数据
  const hasCompare = compareIndex.value !== '' && compareKlineData.value.length > 0
  const compareIndexPct = hasCompare ? normalizeCompareIndex(data, compareKlineData.value) : []
  const compareLabel =
    compareOptions.find((o) => o.value === compareIndex.value)?.label || '对比指数'

  // 根据指标类型构建子图
  const indicatorHeight = '14%'
  const mainHeight = activeIndicator.value === 'vol' ? '52%' : '46%'
  const volHeight = activeIndicator.value === 'vol' ? '18%' : '12%'
  const indicatorTop = activeIndicator.value === 'vol' ? '74%' : '64%'

  const grids: object[] = [
    { left: 56, right: 56, top: 30, height: mainHeight },
    { left: 56, right: 56, top: indicatorTop, height: volHeight },
  ]

  const xAxes: object[] = [
    {
      type: 'category',
      data: dates,
      gridIndex: 0,
      axisLine: { lineStyle: { color: gridLineColor } },
      axisLabel: { color: textColor, fontSize: 10 },
      axisTick: { show: false },
      axisPointer: {
        label: { formatter: (v: any) => v.value },
      },
    },
    {
      type: 'category',
      data: dates,
      gridIndex: 1,
      axisLabel: { show: false },
      axisTick: { show: false },
    },
  ]

  const yAxes: object[] = [
    {
      type: 'value',
      gridIndex: 0,
      scale: true,
      axisLine: { show: false },
      axisLabel: { color: textColor, fontSize: 10 },
      splitLine: { lineStyle: { color: gridLineColor, type: 'dashed' } },
      axisPointer: {
        label: { formatter: (v: any) => Number(v.value).toFixed(2) },
      },
    },
    {
      type: 'value',
      gridIndex: 1,
      axisLine: { show: false },
      axisLabel: { show: false },
      splitLine: { show: false },
    },
  ]

  const series: object[] = [
    {
      name: 'K线',
      type: 'candlestick',
      xAxisIndex: 0,
      yAxisIndex: 0,
      data: ohlc,
      itemStyle: {
        color: upColor,
        color0: downColor,
        borderColor: upColor,
        borderColor0: downColor,
      },
    },
    {
      name: 'MA5',
      type: 'line',
      xAxisIndex: 0,
      yAxisIndex: 0,
      data: ma5,
      symbol: 'none',
      lineStyle: { width: 1 },
    },
    {
      name: 'MA10',
      type: 'line',
      xAxisIndex: 0,
      yAxisIndex: 0,
      data: ma10,
      symbol: 'none',
      lineStyle: { width: 1 },
    },
    {
      name: 'MA20',
      type: 'line',
      xAxisIndex: 0,
      yAxisIndex: 0,
      data: ma20,
      symbol: 'none',
      lineStyle: { width: 1 },
    },
    {
      name: '成交量',
      type: 'bar',
      xAxisIndex: 1,
      yAxisIndex: 1,
      data: volumes.map((v, i) => ({
        value: v,
        itemStyle: { color: data[i].close >= data[i].open ? upColor : downColor, opacity: 0.6 },
      })),
      barWidth: '60%',
    },
  ]

  const legendData = ['MA5', 'MA10', 'MA20']

  // 对比叠加：使用百分比涨跌幅，添加到主图的副 Y 轴
  if (hasCompare) {
    yAxes[0] = {
      type: 'value',
      gridIndex: 0,
      scale: true,
      axisLine: { show: false },
      axisLabel: { color: textColor, fontSize: 10 },
      splitLine: { lineStyle: { color: gridLineColor, type: 'dashed' } },
      axisPointer: {
        label: { formatter: (v: any) => Number(v.value).toFixed(2) },
      },
    }
    // 添加对比指数的百分比涨跌幅线
    series.push({
      name: compareLabel,
      type: 'line',
      xAxisIndex: 0,
      yAxisIndex: 0,
      data: compareIndexPct,
      symbol: 'none',
      lineStyle: { width: 1.5, color: cssVar('--color-chart-p4'), type: 'dashed' },
      connectNulls: true,
    })
    legendData.push(compareLabel)
  }

  // 添加 MACD 或 KDJ 子图
  if (activeIndicator.value === 'macd') {
    const { dif, dea, macd } = calculateMACD(
      data,
      macdParams.value.fast,
      macdParams.value.slow,
      macdParams.value.signal,
    )
    grids.push({ left: 56, right: 56, top: '82%', height: indicatorHeight })
    xAxes.push({
      type: 'category',
      data: dates,
      gridIndex: 2,
      axisLine: { lineStyle: { color: gridLineColor } },
      axisLabel: { color: textColor, fontSize: 10 },
      axisTick: { show: false },
    })
    yAxes.push({
      type: 'value',
      gridIndex: 2,
      axisLine: { show: false },
      axisLabel: { color: textColor, fontSize: 10 },
      splitLine: { show: false },
    })
    series.push(
      {
        name: 'MACD',
        type: 'bar',
        xAxisIndex: 2,
        yAxisIndex: 2,
        data: macd.map((v) => ({
          value: v,
          itemStyle: { color: v >= 0 ? upColor : downColor, opacity: 0.7 },
        })),
        barWidth: '40%',
      },
      {
        name: 'DIF',
        type: 'line',
        xAxisIndex: 2,
        yAxisIndex: 2,
        data: dif,
        symbol: 'none',
        lineStyle: { width: 1, color: cssVar('--color-chart-ma5') },
      },
      {
        name: 'DEA',
        type: 'line',
        xAxisIndex: 2,
        yAxisIndex: 2,
        data: dea,
        symbol: 'none',
        lineStyle: { width: 1, color: cssVar('--color-chart-ma10') },
      },
    )
    legendData.push('DIF', 'DEA')
  } else if (activeIndicator.value === 'kdj') {
    const { k, d, j } = calculateKDJ(data, kdjParams.value.n)
    grids.push({ left: 56, right: 56, top: '82%', height: indicatorHeight })
    xAxes.push({
      type: 'category',
      data: dates,
      gridIndex: 2,
      axisLine: { lineStyle: { color: gridLineColor } },
      axisLabel: { color: textColor, fontSize: 10 },
      axisTick: { show: false },
    })
    yAxes.push({
      type: 'value',
      gridIndex: 2,
      axisLine: { show: false },
      axisLabel: { color: textColor, fontSize: 10 },
      splitLine: { show: false },
    })
    series.push(
      {
        name: 'K',
        type: 'line',
        xAxisIndex: 2,
        yAxisIndex: 2,
        data: k,
        symbol: 'none',
        lineStyle: { width: 1, color: cssVar('--color-chart-ma5') },
      },
      {
        name: 'D',
        type: 'line',
        xAxisIndex: 2,
        yAxisIndex: 2,
        data: d,
        symbol: 'none',
        lineStyle: { width: 1, color: cssVar('--color-chart-ma10') },
      },
      {
        name: 'J',
        type: 'line',
        xAxisIndex: 2,
        yAxisIndex: 2,
        data: j,
        symbol: 'none',
        lineStyle: { width: 1, color: cssVar('--color-chart-p4') },
      },
    )
    legendData.push('K', 'D', 'J')
  }

  return {
    animation: true,
    animationDuration: 600,
    animationEasing: 'cubicOut',
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'cross',
        crossStyle: { color: cssVar('--color-text-disabled') },
        label: {
          backgroundColor: cssVar('--color-brand'),
          color: '#fff',
          fontSize: 10,
        },
      },
      formatter: (params: any[]) => {
        if (!params || !params.length) return ''
        const date = params[0].axisValue
        let html = `<div style="font-size:10px;color:${cssVar('--color-text-tertiary')};margin-bottom:4px">${date}</div>`
        params.forEach((p) => {
          const val = typeof p.value === 'number' ? p.value.toFixed(2) : (p.value ?? '--')
          html += `<div style="font-size:11px"><span style="display:inline-block;width:8px;height:8px;border-radius:50%;background:${p.color};margin-right:6px"></span>${p.seriesName}: <span style="font-family:monospace;font-weight:600">${val}</span></div>`
        })
        return html
      },
    },
    legend: {
      data: legendData,
      top: 0,
      textStyle: { color: textColor, fontSize: 10 },
    },
    grid: grids,
    xAxis: xAxes,
    yAxis: yAxes,
    dataZoom: [
      {
        type: 'inside',
        xAxisIndex: xAxes.map((_, i) => i),
        start: 70,
        end: 100,
        zoomOnMouseWheel: true,
        moveOnMouseMove: true,
        moveOnMouseWheel: false,
      },
      {
        type: 'slider',
        xAxisIndex: xAxes.map((_, i) => i),
        bottom: 4,
        height: 16,
        borderColor: 'transparent',
        textStyle: { fontSize: 10 },
      },
    ],
    series,
  }
}

// ECharts instances
const { renderChart: renderMinuteChart } = useECharts(minuteChartRef, getMinuteOption, () => [
  props.minuteData,
  props.quote.prev_close,
  isDark.value,
])
const { renderChart: renderKlineChart } = useECharts(klineChartRef, getKlineOption, () => [
  filteredKlines.value,
  isDark.value,
  activeIndicator.value,
  compareIndex.value,
  compareKlineData.value,
  macdParams.value,
  kdjParams.value,
])

// Tab 切换时触发 resize
watch(activeTab, async (tab) => {
  await nextTick()
  if (tab === 'minute') {
    renderMinuteChart()
  } else {
    renderKlineChart()
  }
})

function priceClass(price: number, prevClose: number) {
  return price > prevClose ? 'up' : price < prevClose ? 'down' : ''
}

function formatPrice(v: number) {
  return v.toFixed(2)
}

function formatVolume(v: number) {
  if (v >= 100000000) return (v / 100000000).toFixed(1) + '亿'
  if (v >= 10000) return (v / 10000).toFixed(0) + '万'
  return String(v)
}

function formatAmount(v: number) {
  if (v >= 100000000) return (v / 100000000).toFixed(1) + '亿'
  if (v >= 10000) return (v / 10000).toFixed(0) + '万'
  return String(v)
}

function toggleParamPanel() {
  showParamPanel.value = !showParamPanel.value
}
</script>

<template>
  <section class="card card-tier-main">
    <div class="card-header">
      <div class="card-title-wrap">
        <span class="card-eyebrow">PRICE TREND</span>
        <h2 class="card-title">走势图</h2>
      </div>
      <div class="chart-controls">
        <!-- 周期切换 -->
        <div class="chart-tabs">
          <button
            v-for="tab in tabs"
            :key="tab.value"
            :class="['chart-tab', { active: activeTab === tab.value }]"
            type="button"
            @click="activeTab = tab.value"
          >
            {{ tab.label }}
          </button>
        </div>

        <!-- K 线模式专属控件 -->
        <template v-if="activeTab !== 'minute'">
          <!-- 指标切换 -->
          <div class="indicator-tabs">
            <button
              v-for="ind in indicators"
              :key="ind.value"
              :class="['ind-tab', { active: activeIndicator === ind.value }]"
              type="button"
              @click="activeIndicator = ind.value"
            >
              {{ ind.label }}
            </button>
            <!-- 指标参数按钮 -->
            <button
              v-if="activeIndicator !== 'vol'"
              class="ind-param-btn"
              type="button"
              title="指标参数设置"
              @click="toggleParamPanel"
            >
              <svg width="14" height="14" viewBox="0 0 16 16" fill="none">
                <path
                  d="M8 1l1.5 3L13 5l-2.5 2.5L11 11l-3-1.5L5 11l.5-3.5L3 5l3.5-1L8 1z"
                  stroke="currentColor"
                  stroke-width="1.2"
                  stroke-linejoin="round"
                />
              </svg>
            </button>
          </div>

          <!-- 复权切换 -->
          <div class="adjust-tabs">
            <button
              v-for="adj in adjustTypes"
              :key="adj.value"
              :class="['adj-tab', { active: adjustType === adj.value }]"
              type="button"
              @click="adjustType = adj.value"
            >
              {{ adj.label }}
            </button>
          </div>

          <!-- 对比指数选择 -->
          <div class="compare-select">
            <select v-model="compareIndex" class="compare-dropdown">
              <option v-for="opt in compareOptions" :key="opt.value" :value="opt.value">
                {{ opt.label }}
              </option>
            </select>
          </div>
        </template>
      </div>

      <!-- 指标参数面板 -->
      <div v-if="showParamPanel && activeIndicator !== 'vol'" class="param-panel">
        <template v-if="activeIndicator === 'macd'">
          <div class="param-row">
            <label>FAST</label>
            <input v-model.number="macdParams.fast" type="number" min="2" max="30" />
          </div>
          <div class="param-row">
            <label>SLOW</label>
            <input v-model.number="macdParams.slow" type="number" min="5" max="60" />
          </div>
          <div class="param-row">
            <label>SIGNAL</label>
            <input v-model.number="macdParams.signal" type="number" min="2" max="30" />
          </div>
        </template>
        <template v-if="activeIndicator === 'kdj'">
          <div class="param-row">
            <label>N</label>
            <input v-model.number="kdjParams.n" type="number" min="1" max="30" />
          </div>
        </template>
      </div>
    </div>
    <div class="card-body">
      <!-- Minute chart -->
      <div v-show="activeTab === 'minute'" ref="minuteChartRef" class="chart-wrap" />

      <!-- Kline chart -->
      <div v-show="activeTab !== 'minute'" ref="klineChartRef" class="chart-wrap" />

      <!-- Loading hint -->
      <div v-if="klineLoading" class="loading-hint">加载复权数据中...</div>

      <!-- Info bar (minute mode) -->
      <div v-if="activeTab === 'minute' && minuteData.length > 0" class="minute-info">
        <span
          >开
          <em :class="priceClass(quote.open, quote.prev_close)">{{
            formatPrice(quote.open)
          }}</em></span
        >
        <span
          >高
          <em :class="priceClass(quote.high, quote.prev_close)">{{
            formatPrice(quote.high)
          }}</em></span
        >
        <span
          >低
          <em :class="priceClass(quote.low, quote.prev_close)">{{
            formatPrice(quote.low)
          }}</em></span
        >
        <span>量 {{ formatVolume(quote.volume) }}</span>
        <span>额 {{ formatAmount(quote.amount) }}</span>
      </div>
    </div>
  </section>
</template>

<style scoped>
.chart-controls {
  display: flex;
  align-items: center;
  gap: var(--sp-3);
  flex-wrap: wrap;
}
.chart-tabs {
  display: flex;
  gap: var(--sp-0_5);
}
.indicator-tabs {
  display: flex;
  gap: var(--sp-0_5);
  padding-left: var(--sp-3);
  border-left: 1px solid var(--color-border);
  align-items: center;
}
.adjust-tabs {
  display: flex;
  gap: var(--sp-0_5);
  padding-left: var(--sp-3);
  border-left: 1px solid var(--color-border);
}
.compare-select {
  padding-left: var(--sp-3);
  border-left: 1px solid var(--color-border);
}
.compare-dropdown {
  padding: var(--sp-1) var(--sp-2);
  font-size: var(--fs-xs);
  border: 1px solid var(--color-border);
  border-radius: var(--sp-1);
  background: var(--color-bg-card);
  color: var(--color-text-primary);
  cursor: pointer;
  outline: none;
}
.compare-dropdown:hover {
  border-color: var(--color-brand);
}
.chart-tab {
  padding: var(--sp-1) var(--fs-xs);
  font-size: var(--fs-xs);
  border: none;
  border-radius: var(--sp-1);
  background: transparent;
  color: var(--color-text-disabled);
  cursor: pointer;
  transition: all 0.15s;
}
.chart-tab.active {
  background: var(--color-brand);
  color: var(--color-brand-contrast);
}
.chart-tab:hover:not(.active) {
  background: var(--color-bg-hover, var(--color-border-light));
  color: var(--color-text-primary, var(--color-text-secondary));
}
.chart-tab:active {
  transform: scale(0.95);
}
.ind-tab {
  padding: var(--sp-1) var(--sp-2);
  font-size: var(--fs-xs);
  font-family: var(--font-mono);
  border: none;
  border-radius: var(--sp-1);
  background: transparent;
  color: var(--color-text-disabled);
  cursor: pointer;
  transition: all 0.15s;
  letter-spacing: var(--ls-wide);
}
.ind-tab.active {
  background: var(--color-brand-soft);
  color: var(--color-brand);
  font-weight: var(--fw-semibold);
}
.ind-tab:hover:not(.active) {
  background: var(--color-bg-hover, var(--color-border-light));
  color: var(--color-text-secondary);
}
.ind-tab:active {
  transform: scale(0.95);
}
.ind-param-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border: none;
  border-radius: var(--sp-1);
  background: transparent;
  color: var(--color-text-tertiary);
  cursor: pointer;
  transition: all 0.15s;
}
.ind-param-btn:hover {
  background: var(--color-bg-hover);
  color: var(--color-brand);
}
.adj-tab {
  padding: var(--sp-1) var(--sp-2);
  font-size: var(--fs-xs);
  border: none;
  border-radius: var(--sp-1);
  background: transparent;
  color: var(--color-text-disabled);
  cursor: pointer;
  transition: all 0.15s;
}
.adj-tab.active {
  background: var(--color-brand-soft);
  color: var(--color-brand);
  font-weight: var(--fw-semibold);
}
.adj-tab:hover:not(.active) {
  background: var(--color-bg-hover, var(--color-border-light));
  color: var(--color-text-secondary);
}
.param-panel {
  display: flex;
  gap: var(--sp-3);
  padding: var(--sp-2) var(--sp-3);
  margin-top: var(--sp-2);
  background: var(--color-bg-elevated);
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border-light);
}
.param-row {
  display: flex;
  align-items: center;
  gap: var(--sp-1);
}
.param-row label {
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  color: var(--color-text-tertiary);
  letter-spacing: var(--ls-wide);
  font-family: var(--font-mono);
}
.param-row input {
  width: 50px;
  padding: 2px var(--sp-1);
  font-size: var(--fs-xs);
  font-family: var(--font-mono);
  border: 1px solid var(--color-border);
  border-radius: var(--sp-1);
  background: var(--color-bg-card);
  color: var(--color-text-primary);
  outline: none;
  text-align: center;
}
.param-row input:focus {
  border-color: var(--color-brand);
}
.chart-wrap {
  height: 400px;
  padding: var(--sp-2) 0;
}
@media (max-width: 768px) {
  .chart-wrap {
    height: 300px;
  }
}
.loading-hint {
  text-align: center;
  font-size: var(--fs-xs);
  color: var(--color-text-tertiary);
  padding: var(--sp-1);
}
.minute-info {
  display: flex;
  gap: var(--sp-4);
  padding: var(--sp-2) 0;
  font-size: var(--fs-xs);
  color: var(--color-text-tertiary);
  border-top: 1px solid var(--color-border-light);
  font-variant-numeric: tabular-nums;
  letter-spacing: var(--ls-wide);
}
.minute-info em {
  font-style: normal;
  font-weight: var(--fw-semibold);
  font-family: var(--font-mono);
  margin-left: var(--sp-1);
}
.minute-info .up {
  color: var(--color-up);
}
.minute-info .down {
  color: var(--color-down);
}
</style>
