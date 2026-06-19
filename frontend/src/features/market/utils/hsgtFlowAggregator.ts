/** @module market/utils/hsgtFlowAggregator — 沪深港通资金流向数据聚合工具 */
import type { HSGTFlowDaily, HSGTAggregatedPoint, HSGTTimeRange, HSGTDirection } from '../types'

/** 聚合字段名 */
type FlowField =
  | 'north_total_buy'
  | 'north_sh_buy'
  | 'north_sz_buy'
  | 'north_total_amt'
  | 'north_sh_amt'
  | 'north_sz_amt'
  | 'south_total_buy'
  | 'south_sh_buy'
  | 'south_sz_buy'

/** 聚合后的字段名 */
type AggField =
  | 'north_total'
  | 'north_sh'
  | 'north_sz'
  | 'north_total_amt'
  | 'north_sh_amt'
  | 'north_sz_amt'
  | 'south_total'
  | 'south_sh'
  | 'south_sz'

const fieldMap: Record<FlowField, AggField> = {
  north_total_buy: 'north_total',
  north_sh_buy: 'north_sh',
  north_sz_buy: 'north_sz',
  north_total_amt: 'north_total_amt',
  north_sh_amt: 'north_sh_amt',
  north_sz_amt: 'north_sz_amt',
  south_total_buy: 'south_total',
  south_sh_buy: 'south_sh',
  south_sz_buy: 'south_sz',
}

/** 获取日期所属的周标识（ISO周） */
function getWeekKey(dateStr: string): string {
  const d = new Date(dateStr)
  // ISO week: get Thursday of the same week to determine week number
  const day = d.getDay()
  const thursday = new Date(d)
  thursday.setDate(d.getDate() - ((day + 6) % 7) + 3)
  const year = thursday.getFullYear()
  const jan1 = new Date(year, 0, 1)
  const weekNum = Math.ceil(
    ((thursday.getTime() - jan1.getTime()) / 86400000 + jan1.getDay() + 1) / 7,
  )
  return `${year}-W${String(weekNum).padStart(2, '0')}`
}

/** 获取日期所属的月标识 */
function getMonthKey(dateStr: string): string {
  return dateStr.substring(0, 7) // YYYY-MM
}

/** 按key聚合数据，对每个字段求和 */
function aggregateByKey(
  data: HSGTFlowDaily[],
  keyFn: (d: HSGTFlowDaily) => string,
  labelFn: (key: string) => string,
): HSGTAggregatedPoint[] {
  const groups = new Map<string, HSGTFlowDaily[]>()
  for (const item of data) {
    const key = keyFn(item)
    if (!groups.has(key)) groups.set(key, [])
    groups.get(key)!.push(item)
  }

  const result: HSGTAggregatedPoint[] = []
  for (const [key, items] of groups) {
    const point: HSGTAggregatedPoint = {
      label: labelFn(key),
      north_total: 0,
      north_sh: 0,
      north_sz: 0,
      north_total_amt: 0,
      north_sh_amt: 0,
      north_sz_amt: 0,
      south_total: 0,
      south_sh: 0,
      south_sz: 0,
    }
    for (const item of items) {
      for (const [srcField, dstField] of Object.entries(fieldMap) as [FlowField, AggField][]) {
        point[dstField] += item[srcField] ?? 0
      }
    }
    result.push(point)
  }
  return result
}

/** 日/周/月对应的数据切片天数 */
export const HSGT_SLICE_DAYS: Record<HSGTTimeRange, number> = {
  daily: 30,
  weekly: 90,
  monthly: 365,
}

/** 根据时间维度切片并聚合 HSGT 数据 */
export function aggregateHSGTData(
  data: HSGTFlowDaily[],
  range: HSGTTimeRange,
): HSGTAggregatedPoint[] {
  const sliceDays = HSGT_SLICE_DAYS[range]
  const sliced = data.slice(-sliceDays)

  if (range === 'daily') {
    return sliced.map((d) => ({
      label: d.date,
      north_total: d.north_total_buy ?? 0,
      north_sh: d.north_sh_buy ?? 0,
      north_sz: d.north_sz_buy ?? 0,
      north_total_amt: d.north_total_amt ?? 0,
      north_sh_amt: d.north_sh_amt ?? 0,
      north_sz_amt: d.north_sz_amt ?? 0,
      south_total: d.south_total_buy ?? 0,
      south_sh: d.south_sh_buy ?? 0,
      south_sz: d.south_sz_buy ?? 0,
    }))
  }

  if (range === 'weekly') {
    return aggregateByKey(
      sliced,
      (d) => getWeekKey(d.date),
      (key) => key,
    )
  }

  // monthly
  return aggregateByKey(
    sliced,
    (d) => getMonthKey(d.date),
    (key) => key,
  )
}

/** 获取最新一天的数据摘要 */
export function getLatestHSGTSummary(data: HSGTFlowDaily[]): {
  northTotal: number
  southTotal: number
  date: string
} | null {
  if (data.length === 0) return null
  const latest = data[data.length - 1]
  return {
    northTotal: latest.north_total_buy ?? 0,
    southTotal: latest.south_total_buy ?? 0,
    date: latest.date,
  }
}

/** 获取指定方向的最新数据摘要 */
export function getLatestDirectionSummary(
  data: HSGTFlowDaily[],
  direction: HSGTDirection,
): { total: number; sh: number; sz: number; date: string; metric: string } | null {
  if (data.length === 0) return null
  const latest = data[data.length - 1]
  if (direction === 'north') {
    return {
      total: latest.north_total_amt ?? 0,
      sh: latest.north_sh_amt ?? 0,
      sz: latest.north_sz_amt ?? 0,
      date: latest.date,
      metric: '成交额',
    }
  }
  return {
    total: latest.south_total_buy ?? 0,
    sh: latest.south_sh_buy ?? 0,
    sz: latest.south_sz_buy ?? 0,
    date: latest.date,
    metric: '净买额',
  }
}
