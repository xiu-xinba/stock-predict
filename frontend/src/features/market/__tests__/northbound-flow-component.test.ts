import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import NorthboundFlow from '../components/NorthboundFlow.vue'
import type { NorthboundFlow as NorthboundFlowData } from '../types'

describe('NorthboundFlow', () => {
  it('shows disclosure notice instead of zero summary when intraday data is unavailable', () => {
    const flow: NorthboundFlowData = {
      sh_net_buy: 0,
      sz_net_buy: 0,
      total_net_buy: 0,
      timeline: [],
      status: 'intraday_unavailable',
      data_source: 'compliance',
      notice: '官方不再公开北向实时分时，未接入授权数据源',
    }

    const wrapper = mount(NorthboundFlow, {
      props: { flow, loading: false, error: null },
    })

    expect(wrapper.text()).toContain('官方不再公开北向实时分时')
    expect(wrapper.text()).toContain('未接入授权数据源')
    expect(wrapper.text()).not.toContain('总净买入')
  })
})
