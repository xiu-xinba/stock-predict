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
    meta: { title: '预测入口', icon: 'MagicStick' },
  },
  {
    path: '/predict/:fundCode',
    name: 'PredictDetail',
    component: () => import('@/views/PredictView.vue'),
    meta: { title: '预测入口', icon: 'MagicStick' },
  },
  {
    path: '/fund/:fundCode',
    name: 'FundDetail',
    component: () => import('@/views/FundDetailView.vue'),
    meta: { title: '基金详情' },
  },
  {
    path: '/stock/:stockCode',
    name: 'StockDetail',
    component: () => import('@/views/StockDetailView.vue'),
    meta: { title: '股票详情' },
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: () => import('@/views/NotFoundView.vue'),
    meta: { title: '页面未找到' },
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior() {
    return { top: 0 }
  },
})

router.beforeEach((to, _from, next) => {
  const title = (to.meta.title as string) || 'Stock Predict'
  document.title = `${title} · Stock Predict`

  // Auth check point - currently no auth required
  // if (to.meta.requiresAuth && !isAuthenticated()) {
  //   return next({ name: 'Login', query: { redirect: to.fullPath } })
  // }

  next()
})

export default router
