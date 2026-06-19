import { describe, expect, it } from 'vitest'
import {
  buildNorthboundChartData,
  normalizeNorthboundTimeline,
  getNorthboundLatestValues,
} from '../utils/northboundFlowChart'
import type { NorthboundFlow, NorthboundPoint } from '../types'

describe('northbound flow chart data', () => {
  it('sorts valid timeline points and derives total flow from Shanghai and Shenzhen channels', () => {
    const points: NorthboundPoint[] = [
      { time: '09:33', sh_flow: 120000000, sz_flow: -20000000 },
      { time: '', sh_flow: 1, sz_flow: 1 },
      { time: '09:31', sh_flow: 30000000, sz_flow: 40000000 },
      { time: '09:32', sh_flow: -10000000, sz_flow: 5000000 },
    ]

    const normalized = normalizeNorthboundTimeline(points)

    expect(normalized.map((point) => point.time)).toEqual(['09:31', '09:32', '09:33'])
    expect(normalized.map((point) => point.total_flow)).toEqual([70000000, -5000000, 100000000])
    expect(normalized[1]).toMatchObject({
      sh_flow: -10000000,
      sz_flow: 5000000,
      total_flow: -5000000,
    })
  })

  it('returns latest chart values from the final normalized point', () => {
    const latest = getNorthboundLatestValues([
      { time: '09:31', sh_flow: 30000000, sz_flow: 40000000, total_flow: 70000000 },
      { time: '09:32', sh_flow: -10000000, sz_flow: 5000000, total_flow: -5000000 },
    ])

    expect(latest).toEqual({
      time: '09:32',
      sh_flow: -10000000,
      sz_flow: 5000000,
      total_flow: -5000000,
    })
  })

  it('builds a summary fallback curve when the API only returns a daily placeholder point', () => {
    const flow: NorthboundFlow = {
      sh_net_buy: 273757367.45,
      sz_net_buy: 251431468.29,
      total_net_buy: 525188835.74,
      timeline: [{ time: '2026-06-16', sh_flow: 0, sz_flow: 0 }],
    }

    const chartData = buildNorthboundChartData(flow)

    expect(chartData).toEqual([
      { time: '09:30', sh_flow: 0, sz_flow: 0, total_flow: 0 },
      {
        time: '15:00',
        sh_flow: 273757367.45,
        sz_flow: 251431468.29,
        total_flow: 525188835.74,
      },
    ])
  })

  it('does not draw all-zero placeholder minute data as an intraday chart', () => {
    const flow: NorthboundFlow = {
      sh_net_buy: 0,
      sz_net_buy: 0,
      total_net_buy: 0,
      timeline: [
        { time: '10:00', sh_flow: 0, sz_flow: 0 },
        { time: '10:01', sh_flow: 0, sz_flow: 0 },
        { time: '10:02', sh_flow: 0, sz_flow: 0 },
      ],
    }

    expect(buildNorthboundChartData(flow)).toEqual([])
  })
})
