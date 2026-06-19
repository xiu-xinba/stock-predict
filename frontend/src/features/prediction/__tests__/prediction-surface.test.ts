import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import PredictionPlaceholder from '@/features/prediction/components/PredictionPlaceholder.vue'
import FundHeader from '@/features/funds/components/FundHeader.vue'
import PredictView from '@/features/prediction/PredictView.vue'
import type { FundItem } from '@/features/funds'

describe('prediction surface', () => {
  it('shows only the migrated-service notice on the prediction page', () => {
    const wrapper = mount(PredictView)

    expect(wrapper.text()).toContain('预测服务已迁移')
    expect(wrapper.text()).toContain('当前行情项目不再提供预测结果')
    expect(wrapper.text()).not.toContain('RSI')
    expect(wrapper.text()).not.toContain('MACD')
    expect(wrapper.text()).not.toContain('历史准确率')
    expect(wrapper.text()).not.toContain('模型选择')
    expect(wrapper.text()).not.toContain('风险评估参数')
    expect(wrapper.find('input[type="range"]').exists()).toBe(false)
  })

  it('does not offer the removed prediction flow from the fund header', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/', component: { template: '<div />' } }],
    })
    router.push('/')
    await router.isReady()

    const basic: FundItem = {
      fund_code: '000001',
      fund_name: '华夏成长混合',
      fund_type: '混合型',
    }
    const quote: FundItem = {
      ...basic,
      latest_nav: 1.2345,
      change_pct: 1.2,
    }

    const wrapper = mount(FundHeader, {
      props: { basic, quote },
      global: {
        plugins: [createPinia(), router],
        stubs: {
          AssetHeader: {
            template: '<section><slot name="actions" /></section>',
          },
        },
      },
    })

    expect(wrapper.text()).not.toContain('查看预测')
    expect(wrapper.find('.predict-btn').exists()).toBe(false)
  })

  it('uses the same migrated-service contract on detail pages', () => {
    const wrapper = mount(PredictionPlaceholder, {
      props: { code: '000001', type: 'fund' },
    })

    expect(wrapper.text()).toContain('预测服务已迁移')
    expect(wrapper.text()).toContain('当前行情项目不再提供预测结果')
    expect(wrapper.text()).toContain('410 Gone')
  })
})
