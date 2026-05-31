import { describe, it, expect } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createRouter, createWebHistory } from 'vue-router'
import { createPinia } from 'pinia'
import { nextTick } from 'vue'
import App from '@/App.vue'

function createTestRouter() {
  return createRouter({
    history: createWebHistory(),
    routes: [
      { path: '/', redirect: '/watchlist' },
      { path: '/watchlist', name: 'Watchlist', component: { template: '<div>Watchlist</div>' } },
      { path: '/market', name: 'Market', component: { template: '<div>Market</div>' } },
      { path: '/predict', name: 'Predict', component: { template: '<div>Predict</div>' } },
    ],
  })
}

describe('App', () => {
  it('mounts successfully', async () => {
    const router = createTestRouter()
    router.push('/watchlist')
    await router.isReady()

    const wrapper = mount(App, {
      global: {
        plugins: [createPinia(), router],
        stubs: {
          RefreshFab: true,
          SearchOverlay: true,
          MarketDock: true,
        },
      },
    })

    expect(wrapper.find('#app').exists()).toBe(true)
    expect(wrapper.find('.topbar').exists()).toBe(true)
    expect(wrapper.find('.main-content').exists()).toBe(true)
  })

  it('renders navigation items after dock becomes visible', async () => {
    const router = createTestRouter()
    router.push('/watchlist')
    await router.isReady()

    const wrapper = mount(App, {
      global: {
        plugins: [createPinia(), router],
        stubs: {
          RefreshFab: true,
          SearchOverlay: true,
          MarketDock: true,
        },
      },
    })

    await nextTick()
    await flushPromises()

    const dockItems = wrapper.findAll('.dock-item')
    expect(dockItems.length).toBe(3)
  })
})
