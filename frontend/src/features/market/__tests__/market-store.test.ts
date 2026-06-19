import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useMarketStore } from '@/features/market'
import {
  fetchIndexKline,
  fetchIndexMinute,
  fetchMarketIndices,
  fetchFundRanking,
  fetchSectorRanking,
  fetchNorthboundFlow,
  fetchStockRanking,
} from '@/features/market/api/market'

vi.mock('element-plus', () => ({
  ElMessage: {
    warning: vi.fn(),
    error: vi.fn(),
  },
}))

vi.mock('@/features/market/api/market', () => ({
  fetchMarketIndices: vi.fn(),
  fetchFundRanking: vi.fn(),
  fetchIndexKline: vi.fn(),
  fetchIndexMinute: vi.fn(),
  fetchSectorRanking: vi.fn(),
  fetchNorthboundFlow: vi.fn(),
  fetchStockRanking: vi.fn(),
}))

describe('market store real-data state', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.mocked(fetchMarketIndices).mockResolvedValue({ code: 0, message: 'success', data: [] })
    vi.mocked(fetchFundRanking).mockResolvedValue({ code: 0, message: 'success', data: [] })
    vi.mocked(fetchSectorRanking).mockResolvedValue({ code: 0, message: 'success', data: [] })
    vi.mocked(fetchNorthboundFlow).mockResolvedValue({ code: 0, message: 'success', data: null })
  })

  it('fetches stock rankings with size and exposes loading state', async () => {
    vi.mocked(fetchStockRanking)
      .mockResolvedValueOnce({
        code: 0,
        message: 'success',
        data: [
          {
            rank: 1,
            stock_code: '600519',
            stock_name: 'Kweichow Moutai',
            change_pct: 2.3,
            current_price: 100,
            volume: 10,
            amount: 20,
          },
        ],
      })
      .mockResolvedValueOnce({
        code: 0,
        message: 'success',
        data: [
          {
            rank: 1,
            stock_code: '000858',
            stock_name: 'Wuliangye',
            change_pct: -1.2,
            current_price: 50,
            volume: 11,
            amount: 21,
          },
        ],
      })
    const store = useMarketStore()

    const pending = store.fetchStockRankingData(8)

    expect(store.stockRankingLoading).toBe(true)
    await pending
    expect(fetchStockRanking).toHaveBeenNthCalledWith(1, 'gainers', 8, expect.any(AbortSignal))
    expect(fetchStockRanking).toHaveBeenNthCalledWith(2, 'losers', 8, expect.any(AbortSignal))
    expect(store.stockGainers).toHaveLength(1)
    expect(store.stockLosers).toHaveLength(1)
    expect(store.stockRankingError).toBeNull()
    expect(store.stockRankingLoading).toBe(false)
  })

  it('records stock ranking error instead of leaving an indefinite loading state', async () => {
    vi.mocked(fetchStockRanking).mockRejectedValue(new Error('upstream unavailable'))
    const store = useMarketStore()

    await store.fetchStockRankingData(5)

    expect(store.stockRankingLoading).toBe(false)
    expect(store.stockRankingError).toBe('股票排行数据暂不可用')
    expect(store.stockGainers).toEqual([])
    expect(store.stockLosers).toEqual([])
  })

  it('tracks minute curve loading and errors by index code', async () => {
    vi.mocked(fetchIndexMinute).mockRejectedValue(new Error('minute failed'))
    const store = useMarketStore()

    const pending = store.fetchIndexMinuteData('000001')

    expect(store.indexMinuteLoading.get('000001')).toBe(true)
    await pending
    expect(store.indexMinuteLoading.get('000001')).toBe(false)
    expect(store.indexMinuteError.get('000001')).toBe('指数分时数据暂不可用')
  })

  it('fetches Shanghai Composite historical K-line through dedicated state', async () => {
    vi.mocked(fetchIndexKline).mockResolvedValue({
      code: 0,
      message: 'success',
      data: [
        {
          date: '2026-06-01',
          open: 3000,
          close: 3100,
          high: 3110,
          low: 2990,
          volume: 1,
          amount: 2,
        },
      ],
    })
    const store = useMarketStore()

    await store.fetchShanghaiCompositeKline()

    expect(fetchIndexKline).toHaveBeenCalledWith('000001', 120, expect.any(AbortSignal))
    expect(store.indexKline.get('000001')).toHaveLength(1)
    expect(store.indexKlineLoading.get('000001')).toBe(false)
    expect(store.indexKlineError.get('000001')).toBeNull()
  })

  it('keeps northbound unavailable status as data instead of an error', async () => {
    vi.mocked(fetchNorthboundFlow).mockResolvedValue({
      code: 0,
      message: 'success',
      data: {
        sh_net_buy: 0,
        sz_net_buy: 0,
        total_net_buy: 0,
        timeline: [],
        status: 'intraday_unavailable',
        data_source: 'compliance',
        notice: '官方不再公开北向实时分时，未接入授权数据源',
      },
    })
    const store = useMarketStore()

    await store.fetchNorthboundData()

    expect(store.northboundError).toBeNull()
    expect(store.northbound?.status).toBe('intraday_unavailable')
    expect(store.northbound?.notice).toContain('未接入授权数据源')
  })
})
