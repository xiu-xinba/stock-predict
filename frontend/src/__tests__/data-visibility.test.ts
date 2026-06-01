import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import MarketView from '@/views/MarketView.vue'
import WatchlistView from '@/views/WatchlistView.vue'
import FundDetailView from '@/views/FundDetailView.vue'
import StockDetailView from '@/views/StockDetailView.vue'

vi.mock('@/composables/useStaggerEntry', () => ({
  useStaggerEntry: vi.fn(),
}))

vi.mock('@/api/market', () => ({
  fetchMarketIndices: vi.fn(async () => ({
    code: 0,
    message: 'success',
    data: [
      {
        code: '000001',
        name: '上证指数',
        market: 'cn',
        value: 3100,
        change: 12,
        change_pct: 0.39,
        high: 3120,
        low: 3080,
        prev_close: 3088,
        volume: 100000,
        mini_chart_data: [1, 2, 3],
        update_time: '09:30',
        data_source: 'test',
      },
    ],
  })),
  fetchFundRanking: vi.fn(async (type: 'gainers' | 'losers') => ({
    code: 0,
    message: 'success',
    data: [
      {
        rank: 1,
        fund_code: type === 'gainers' ? '000001' : '000002',
        fund_name: type === 'gainers' ? '华夏成长混合' : '易方达蓝筹精选',
        fund_type: '混合型',
        change_pct: type === 'gainers' ? 2.3 : -1.2,
        estimated_nav: 1.2345,
        quote_date: '2026-06-01',
        quote_source: 'test',
      },
    ],
  })),
}))

vi.mock('@/api/stock', () => ({
  fetchStockList: vi.fn(async () => ({
    code: 0,
    message: 'success',
    data: {
      items: [
        {
          stock_code: '600519',
          stock_name: '贵州茅台',
          market: 'SH',
          industry: '白酒',
          list_date: '',
          total_shares: 0,
          float_shares: 0,
          current_price: 1307.02,
          change_pct: -1.43,
          volume: 32923,
          amount: 431438,
          turnover_rate: 0.26,
          pe_ratio: 0,
          pb_ratio: 0,
          total_mv: 0,
          pinyin: 'gzmt',
        },
      ],
      total: 1,
      page: 1,
      size: 20,
    },
  })),
  fetchStockRanking: vi.fn(async (type: string) => ({
    code: 0,
    message: 'success',
    data: [
      {
        rank: 1,
        stock_code: type === 'gainers' ? '600519' : '000858',
        stock_name: type === 'gainers' ? '贵州茅台' : '五粮液',
        change_pct: type === 'gainers' ? 1.2 : -1.1,
        current_price: 1307.02,
        volume: 32923,
        amount: 431438,
      },
    ],
  })),
  fetchStockDetail: vi.fn(async () => ({
    code: 0,
    message: 'success',
    data: {
      basic: {
        stock_code: '600519',
        stock_name: '贵州茅台',
        market: 'SH',
        industry: '白酒',
        list_date: '',
        total_shares: 0,
        float_shares: 0,
      },
      quote: {
        price: 1307.02,
        open: 1327,
        high: 1327,
        low: 1301.31,
        prev_close: 1326,
        volume: 32923,
        amount: 431438,
        turnover_rate: 0.26,
        change_pct: -1.43,
        change_amt: -18.98,
        bid_price: 1306.99,
        ask_price: 1307.01,
        quote_time: '2026-06-01 13:22:20',
      },
      kline: {
        period: 'daily',
        klines: [
          {
            date: '2026-06-01',
            open: 1327,
            close: 1307.02,
            high: 1327,
            low: 1301.31,
            volume: 32923,
            amount: 431438,
          },
        ],
      },
      capital_flow: { main_net_inflow: 0, retail_net_inflow: 0, flow_history: [] },
      financials: {
        pe_ratio: 0,
        pb_ratio: 0,
        roe: 0,
        revenue: 0,
        net_profit: 0,
        eps: 0,
        gross_margin: 0,
        net_margin: 0,
        quarterly: [],
      },
      shareholders: { top10: [], institution_count: 0, institution_ratio: 0 },
    },
  })),
}))

vi.mock('@/api/fundDetail', () => ({
  fetchFundDetail: vi.fn(async () => ({
    code: 0,
    message: 'success',
    data: {
      basic: {
        fund_code: '000001',
        fund_name: '华夏成长混合',
        fund_type: '混合型',
        company: '华夏基金',
        manager: '阳琨',
        latest_nav: 1.333,
        cumulative_nav: 3.906,
        risk_level: '中高',
        inception_date: '2001-12-18',
      },
      quote: {
        fund_code: '000001',
        fund_name: '华夏成长混合',
        fund_type: '混合型',
        latest_nav: 1.333,
        estimated_nav: 1.356,
        change_pct: 1.73,
        quote_date: '2026-06-01',
        quote_source: 'test',
      },
      performance: {
        nav_history: [{ date: '2026-06-01', nav: 1.333, cumulative_nav: 3.906, change_pct: 1.73 }],
        return_1m: 1,
        return_3m: 2,
        return_6m: 3,
        return_1y: 4,
        return_3y: 5,
      },
      manager: { name: '阳琨', tenure_days: 8931, managed_size: '', fund_count: 1, bio: '' },
      portfolio: { top_holdings: [], sector_allocation: [] },
      risk: { volatility_1y: 1, max_drawdown_1y: -1, sharpe_1y: 1, beta_1y: 1 },
    },
  })),
}))

function createTestRouter(path: string) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/market', component: MarketView },
      { path: '/watchlist', component: WatchlistView },
      { path: '/fund/:fundCode', component: FundDetailView },
      { path: '/stock/:stockCode', component: StockDetailView },
      { path: '/predict/:fundCode?', component: { template: '<div>Predict</div>' } },
    ],
  })
  router.push(path)
  return router
}

async function mountWithRouter(component: object, path: string) {
  const pinia = createPinia()
  setActivePinia(pinia)
  const router = createTestRouter(path)
  await router.isReady()
  const wrapper = mount(component, {
    global: {
      plugins: [pinia, router],
      stubs: {
        teleport: true,
        transition: false,
        'transition-group': false,
        ElIcon: true,
      },
    },
  })
  await flushPromises()
  return wrapper
}

describe('data visibility', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders fund rankings on market page', async () => {
    const wrapper = await mountWithRouter(MarketView, '/market')
    expect(wrapper.text()).toContain('华夏成长混合')
    expect(wrapper.text()).toContain('易方达蓝筹精选')
  })

  it('renders stock list on watchlist stock tab', async () => {
    const wrapper = await mountWithRouter(WatchlistView, '/watchlist?tab=stock')
    await flushPromises()
    expect(wrapper.text()).toContain('贵州茅台')
  })

  it('renders fund detail payload', async () => {
    const wrapper = await mountWithRouter(FundDetailView, '/fund/000001')
    expect(wrapper.text()).toContain('华夏成长混合')
    expect(wrapper.text()).toContain('华夏基金')
  })

  it('renders stock detail payload', async () => {
    const wrapper = await mountWithRouter(StockDetailView, '/stock/600519')
    expect(wrapper.text()).toContain('贵州茅台')
    expect(wrapper.text()).toContain('1307.02')
  })
})
