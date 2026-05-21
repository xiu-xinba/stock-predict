import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    redirect: '/watchlist',
  },
  {
    path: '/watchlist',
    name: 'Watchlist',
    component: () => import('@/views/WatchlistView.vue'),
    meta: { title: '自选', icon: 'Star' },
  },
  {
    path: '/market',
    name: 'Market',
    component: () => import('@/views/MarketView.vue'),
    meta: { title: '行情', icon: 'TrendCharts' },
  },
  {
    path: '/predict',
    name: 'Predict',
    component: () => import('@/views/PredictView.vue'),
    meta: { title: '预测', icon: 'MagicStick' },
  },
  {
    path: '/predict/:fundCode',
    name: 'PredictDetail',
    component: () => import('@/views/PredictView.vue'),
    meta: { title: '预测', icon: 'MagicStick' },
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: '/watchlist',
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior() {
    return { top: 0 }
  },
})

router.beforeEach((to) => {
  const title = (to.meta.title as string) || '基金预测'
  document.title = title
})

export default router
