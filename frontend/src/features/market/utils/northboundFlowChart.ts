import type { NorthboundFlow, NorthboundPoint } from '@/features/market/types'

export interface NorthboundChartPoint extends NorthboundPoint {
  total_flow: number
}

function isValidTime(value: string): boolean {
  return /^\d{2}:\d{2}$/.test(value.trim())
}

function safeNumber(value: number): number {
  return Number.isFinite(value) ? value : 0
}

export function normalizeNorthboundTimeline(
  points: NorthboundPoint[] = [],
): NorthboundChartPoint[] {
  return points
    .filter((point) => isValidTime(point.time))
    .map((point) => {
      const shFlow = safeNumber(point.sh_flow)
      const szFlow = safeNumber(point.sz_flow)
      return {
        time: point.time.trim(),
        sh_flow: shFlow,
        sz_flow: szFlow,
        total_flow: shFlow + szFlow,
      }
    })
    .sort((a, b) => a.time.localeCompare(b.time))
}

export function getNorthboundLatestValues(
  points: NorthboundChartPoint[],
): NorthboundChartPoint | null {
  return points.at(-1) ?? null
}

export function buildNorthboundChartData(
  flow: NorthboundFlow | null | undefined,
): NorthboundChartPoint[] {
  if (!flow) return []

  const normalized = normalizeNorthboundTimeline(flow.timeline)
  const shFlow = safeNumber(flow.sh_net_buy)
  const szFlow = safeNumber(flow.sz_net_buy)
  const totalFlow = Number.isFinite(flow.total_net_buy) ? flow.total_net_buy : shFlow + szFlow
  const hasTimelineSignal = normalized.some((point) => point.sh_flow !== 0 || point.sz_flow !== 0)
  const hasSummarySignal = shFlow !== 0 || szFlow !== 0 || totalFlow !== 0

  if (!hasTimelineSignal && !hasSummarySignal) return []
  if (normalized.length >= 2) return normalized
  if (normalized.length > 0 && !hasSummarySignal) return normalized

  return [
    { time: '09:30', sh_flow: 0, sz_flow: 0, total_flow: 0 },
    { time: '15:00', sh_flow: shFlow, sz_flow: szFlow, total_flow: totalFlow },
  ]
}
