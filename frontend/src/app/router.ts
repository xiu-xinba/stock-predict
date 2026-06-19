/** @module app/router - 应用路由配置
 *
 * 定义全局路由表与导航守卫。所有路由均采用懒加载策略，
 * 根路径默认重定向至自选页。导航守卫负责动态设置页面标题。
 */
import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'

/** 应用路由表 */
const routes: RouteRecordRaw[] = [
  {
    /** 根路径重定向至自选页 */
    path: '/',
    redirect: '/watchlist',
  },
  {
    /** 自选监控页 */
    path: '/watchlist',
    name: 'Watchlist',
    component: () => import('@/features/watchlist/WatchlistView.vue'),
    meta: { title: '自选', icon: 'Star' },
  },
  {
    /** 市场行情页 */
    path: '/market',
    name: 'Market',
    component: () => import('@/features/market/MarketView.vue'),
    meta: { title: '行情', icon: 'TrendCharts' },
  },
  {
    /** 预测入口页（已迁移至独立服务） */
    path: '/predict',
    name: 'Predict',
    component: () => import('@/features/prediction/PredictView.vue'),
    meta: { title: '预测入口', icon: 'MagicStick' },
  },
  {
    /** 设置页 */
    path: '/settings',
    name: 'Settings',
    component: () => import('@/features/settings/SettingsView.vue'),
    meta: { title: '设置' },
  },
  {
    /** 基金详情页 */
    path: '/fund/:fundCode',
    name: 'FundDetail',
    component: () => import('@/features/funds/FundDetailView.vue'),
    meta: { title: '基金详情' },
  },
  {
    /** 股票详情页 */
    path: '/stock/:stockCode',
    name: 'StockDetail',
    component: () => import('@/features/stocks/StockDetailView.vue'),
    meta: { title: '股票详情' },
  },
  {
    /** 404 兜底路由 */
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: () => import('@/app/NotFoundView.vue'),
    meta: { title: '页面未找到' },
  },
]

/** 路由器实例，使用 HTML5 History 模式 */
const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior(to, from, savedPosition) {
    // 同一路由仅 query 变化时不重置滚动位置
    if (to.path === from.path) return false
    if (savedPosition) return savedPosition
    return { top: 0 }
  },
})

/** 全局前置守卫：设置页面标题 */
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
