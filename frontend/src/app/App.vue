<template>
  <div id="app" class="app-shell bg-mesh">
    <transition name="offline-banner">
      <div v-if="backendOffline" class="offline-banner">
        <span class="offline-banner-text">后端服务未启动，部分功能不可用</span>
        <button
          class="offline-banner-retry"
          type="button"
          :disabled="healthChecking || restarting"
          @click="handleRetryClick"
        >
          {{ restarting ? '重启中...' : healthChecking ? '检测中...' : '重启后端' }}
        </button>
        <button class="offline-banner-close" type="button" @click="backendOffline = false">
          &times;
        </button>
      </div>
    </transition>

    <header class="topbar">
      <div>
        <div class="topbar-kicker eyebrow">Realtime Fund Analytics</div>
        <div class="topbar-title">{{ currentTitle }}</div>
      </div>
      <div class="topbar-actions">
        <button
          class="topbar-search-btn"
          type="button"
          aria-label="搜索"
          title="搜索 (快捷键 /)"
          @click="searchOpen = true"
        >
          <svg viewBox="0 0 1024 1024" width="18" height="18" aria-hidden="true">
            <path
              fill="currentColor"
              d="m795.904 750.72 124.992 124.928a32 32 0 0 1-45.248 45.248L750.656 795.904a416 416 0 1 1 45.248-45.248zM480 832a352 352 0 1 0 0-704 352 352 0 0 0 0 704"
            />
          </svg>
        </button>
        <button
          class="topbar-search-btn"
          type="button"
          aria-label="设置"
          title="设置"
          @click="router.push('/settings')"
        >
          <svg viewBox="0 0 1024 1024" width="18" height="18" aria-hidden="true">
            <path
              fill="currentColor"
              d="M512 160c-27.4 0-50.2 18.6-56.4 44.4l-4.2 17.6c-4.8 20-20.4 35.6-40.4 40.4-4.8 1.2-9.6 2.6-14.2 4.2-19.2 6.4-40.4 2.8-56.2-10l-14.6-11.8c-21-17-51.2-17-72.2 0l-7.2 5.8c-21 17-27.2 46.4-14.8 70.4l8.6 16.6c9.8 18.8 8.8 41.4-2.6 59.2-2.6 4-5 8.2-7.2 12.4-10 18.6-29.4 30.8-50.8 30.8H160c-27.4 0-50.2 18.6-56.4 44.4l-2.8 11.6c-6.2 26.2 6.4 53.2 30 66l17.2 9.4c18.6 10.2 30.2 29.6 30.2 50.8 0 4.8 0.2 9.6 0.6 14.4 1.6 20.8-8.2 40.8-25.6 52.8l-14.6 10.2c-21.6 15.2-30.8 43-22.2 68l4.2 12.2c8.6 25 33 41.4 59.4 39.6l18.8-1.2c20.8-1.4 40.6 8.6 52.4 25.6 2.8 4 5.8 7.8 8.8 11.6 12.6 15.8 16.4 37.2 9.8 56.6l-6 17.6c-8.8 25.8 1.2 54.4 24 69.2l10.4 6.8c22.8 14.8 53 12.2 72.8-6.4l13.4-12.6c15-14 36.2-19.4 56.2-14.2 4.6 1.2 9.4 2.2 14.2 3 20.4 3.6 37.4 18 43.8 37.8l5.6 17.8c8 25.6 31.6 43 58.4 43h12.6c26.8 0 50.4-17.4 58.4-43l5.6-17.8c6.4-19.8 23.4-34.2 43.8-37.8 4.8-0.8 9.6-1.8 14.2-3 20-5.2 41.2 0.2 56.2 14.2l13.4 12.6c19.8 18.6 50 21.2 72.8 6.4l10.4-6.8c22.8-14.8 32.8-43.4 24-69.2l-6-17.6c-6.6-19.4-2.8-40.8 9.8-56.6 3-3.8 6-7.6 8.8-11.6 11.8-17 31.6-27 52.4-25.6l18.8 1.2c26.4 1.8 50.8-14.6 59.4-39.6l4.2-12.2c8.6-25-0.6-52.8-22.2-68l-14.6-10.2c-17.4-12-27.2-32-25.6-52.8 0.4-4.8 0.6-9.6 0.6-14.4 0-21.2 11.6-40.6 30.2-50.8l17.2-9.4c23.6-12.8 36.2-39.8 30-66l-2.8-11.6c-6.2-25.8-29-44.4-56.4-44.4h-18c-21.4 0-40.8-12.2-50.8-30.8-2.2-4.2-4.6-8.4-7.2-12.4-11.4-17.8-12.4-40.4-2.6-59.2l8.6-16.6c12.4-24 6.2-53.4-14.8-70.4l-7.2-5.8c-21-17-51.2-17-72.2 0l-14.6 11.8c-15.8 12.8-37 16.4-56.2 10-4.6-1.6-9.4-3-14.2-4.2-20-4.8-35.6-20.4-40.4-40.4l-4.2-17.6C562.2 178.6 539.4 160 512 160zm0 304a80 80 0 1 0 0 160 80 80 0 0 0 0-160z"
            />
          </svg>
        </button>
        <button class="theme-fab" type="button" :title="themeTitle" @click="toggleTheme($event)">
          <svg v-if="isDark" class="theme-fab-icon" viewBox="0 0 1024 1024" aria-hidden="true">
            <path
              fill="currentColor"
              d="M512 64h64v192h-64zm0 576h64v192h-64zM160 480v-64h192v64zm576 0v-64h192v64zM249.856 199.04l45.248-45.184L430.848 289.6 385.6 334.848 249.856 199.104zM657.152 606.4l45.248-45.248 135.744 135.744-45.248 45.248zM114.048 923.2 68.8 877.952l316.8-316.8 45.248 45.248zM702.4 334.848 657.152 289.6l135.744-135.744 45.248 45.248z"
            />
          </svg>
          <svg v-else class="theme-fab-icon" viewBox="0 0 1024 1024" aria-hidden="true">
            <path
              fill="currentColor"
              d="M240.448 240.448a384 384 0 1 0 559.424 525.696 448 448 0 0 1-542.016-542.08 391 391 0 0 0-17.408 16.384m181.056 362.048a384 384 0 0 0 525.632 16.384A448 448 0 1 1 405.056 76.8a384 384 0 0 0 16.448 525.696"
            />
          </svg>
        </button>
      </div>
    </header>

    <main class="main-content">
      <router-view v-slot="{ Component: RouteComponent, route: currentRoute }">
        <transition name="page" mode="out-in">
          <component :is="RouteComponent" :key="currentRoute.path" />
        </transition>
      </router-view>
    </main>

    <div
      class="dock-hotspot"
      aria-hidden="true"
      @mouseenter="showDock"
      @touchstart.passive="showDock"
    ></div>

    <transition name="pill">
      <button
        v-if="!dockVisible"
        class="dock-pill"
        type="button"
        aria-label="展开导航栏"
        @mouseenter="showDock"
        @focus="showDock"
        @click="showDock"
      >
        <span class="pill-bar"></span>
      </button>
    </transition>

    <transition name="dock">
      <nav
        v-if="dockVisible"
        class="dock"
        aria-label="主导航"
        @mouseenter="cancelHide"
        @mouseleave="scheduleHide"
        @focusin="cancelHide"
        @focusout="scheduleHide"
        @touchend="scheduleHide"
      >
        <router-link
          v-for="item in navItems"
          :key="item.path"
          :to="item.path"
          class="dock-item"
          :class="{ active: activeMenu === item.path }"
          :title="item.title"
        >
          <span class="dock-icon">
            <el-icon :size="24"><component :is="item.icon" /></el-icon>
          </span>
          <span class="dock-label">{{ item.label }}</span>
        </router-link>
      </nav>
    </transition>

    <RefreshFab position="bottom-right" :size="48" :opacity="0.92" />

    <SearchOverlay :open="searchOpen" @close="searchOpen = false" />

    <HealthWidget />
  </div>
</template>

<script setup lang="ts">
/**
 * 应用根组件
 *
 * 负责整体应用壳层布局，包括顶部导航栏、底部 Dock 导航、
 * 后端离线检测横幅、全局搜索弹层、市场指数浮窗及刷新按钮。
 * 同时管理主题切换与全局键盘快捷键（/ 打开搜索，Esc 关闭搜索）。
 */
import { computed, ref, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { MagicStick, Star, TrendCharts } from '@element-plus/icons-vue'
import type { Component } from 'vue'
import { useTheme } from '@/shared/composables/useTheme'
import { HealthWidget, useMarketStore } from '@/features/market'
import { SearchOverlay } from '@/features/search'
import { checkBackendHealth, restartBackend } from '@/shared/api/client'
import RefreshFab from '@/app/components/RefreshFab.vue'

defineOptions({ name: 'App' })

interface NavItem {
  path: string
  label: string
  icon: Component
  title: string
}

const navItems: NavItem[] = [
  { path: '/watchlist', label: '自选', icon: Star, title: '自选监控' },
  { path: '/market', label: '行情', icon: TrendCharts, title: '市场行情' },
  { path: '/predict', label: '预测', icon: MagicStick, title: '预测入口' },
]

const route = useRoute()
const router = useRouter()
const { isDark, toggleTheme } = useTheme()
const marketStore = useMarketStore()

const themeTitle = computed(() => (isDark.value ? '切换到日间模式' : '切换到夜间模式'))

const activeMenu = computed(() => {
  if (route.path.startsWith('/predict')) return '/predict'
  if (route.path.startsWith('/market')) return '/market'
  if (route.path.startsWith('/watchlist')) return '/watchlist'
  return route.path
})

const currentTitle = computed(() => {
  const item = navItems.find((n) => n.path === activeMenu.value)
  return item?.title ?? 'Stock Predict'
})

const dockVisible = ref(false)
const searchOpen = ref(false)
const backendOffline = ref(false)
const healthChecking = ref(false)
const restarting = ref(false)
let hideTimer: ReturnType<typeof setTimeout> | null = null
let initialTimer: ReturnType<typeof setTimeout> | null = null
let healthRetryTimer: ReturnType<typeof setInterval> | null = null

function showDock() {
  if (hideTimer) {
    clearTimeout(hideTimer)
    hideTimer = null
  }
  dockVisible.value = true
}

function cancelHide() {
  if (hideTimer) {
    clearTimeout(hideTimer)
    hideTimer = null
  }
}

function scheduleHide() {
  if (hideTimer) clearTimeout(hideTimer)
  hideTimer = setTimeout(() => {
    dockVisible.value = false
  }, 900)
}

function handleGlobalKeydown(e: KeyboardEvent) {
  if (e.key === '/' && !e.ctrlKey && !e.metaKey) {
    const active = document.activeElement
    if (active && (active.tagName === 'INPUT' || active.tagName === 'TEXTAREA')) return
    e.preventDefault()
    searchOpen.value = true
  }
  if (e.key === 'Escape') {
    searchOpen.value = false
  }
}

async function retryHealthCheck() {
  healthChecking.value = true
  const ok = await checkBackendHealth()
  backendOffline.value = !ok
  healthChecking.value = false
  if (!ok && !healthRetryTimer) {
    healthRetryTimer = setInterval(async () => {
      const retryOk = await checkBackendHealth()
      if (retryOk) {
        backendOffline.value = false
        if (healthRetryTimer) {
          clearInterval(healthRetryTimer)
          healthRetryTimer = null
        }
      }
    }, 10000)
  }
  if (ok && healthRetryTimer) {
    clearInterval(healthRetryTimer)
    healthRetryTimer = null
  }
}

/** 点击"重启后端"按钮：先尝试调用重启 API，然后轮询健康检查等待后端恢复 */
async function handleRetryClick() {
  restarting.value = true
  // 尝试调用后端重启 API（后端可能还活着但数据源异常）
  await restartBackend()
  // 等待后端重启完成，轮询健康检查
  let attempts = 0
  const maxAttempts = 30 // 最多等待 30 秒
  const poll = setInterval(async () => {
    attempts++
    const ok = await checkBackendHealth()
    if (ok) {
      backendOffline.value = false
      restarting.value = false
      clearInterval(poll)
      if (healthRetryTimer) {
        clearInterval(healthRetryTimer)
        healthRetryTimer = null
      }
    } else if (attempts >= maxAttempts) {
      restarting.value = false
      clearInterval(poll)
      // 重启失败，继续常规轮询
      retryHealthCheck()
    }
  }, 1000)
}

onMounted(() => {
  window.addEventListener('keydown', handleGlobalKeydown)
  showDock()
  initialTimer = setTimeout(scheduleHide, 1800)
  marketStore.fetchMarketData()
  retryHealthCheck()
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleGlobalKeydown)
  if (hideTimer) clearTimeout(hideTimer)
  if (initialTimer) clearTimeout(initialTimer)
  if (healthRetryTimer) {
    clearInterval(healthRetryTimer)
    healthRetryTimer = null
  }
})
</script>

<style>
#app {
  min-height: 100dvh;
}

.offline-banner {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--sp-3);
  padding: var(--sp-2) var(--sp-4);
  background: linear-gradient(
    135deg,
    var(--color-danger),
    color-mix(in srgb, var(--color-danger) 80%, var(--color-brand))
  );
  color: var(--color-brand-contrast);
  font-size: 13px;
  font-weight: 500;
}

.offline-banner-text {
  flex: 1;
  text-align: center;
}

.offline-banner-retry,
.offline-banner-close {
  padding: 2px 10px;
  border: 1px solid color-mix(in srgb, var(--color-brand-contrast) 30%, transparent);
  border-radius: 4px;
  background: transparent;
  color: var(--color-brand-contrast);
  font-size: 12px;
  cursor: pointer;
  transition: background-color 0.15s;
}

.offline-banner-retry:hover,
.offline-banner-close:hover {
  background: color-mix(in srgb, var(--color-brand-contrast) 15%, transparent);
}

.offline-banner-retry:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.offline-banner-close {
  padding: 2px 8px;
  font-size: 16px;
  line-height: 1;
}

.offline-banner-enter-active {
  transition:
    opacity 0.25s ease,
    transform 0.25s ease;
}

.offline-banner-leave-active {
  transition: opacity 0.15s ease;
}

.offline-banner-enter-from {
  opacity: 0;
  transform: translateY(-100%);
}

.offline-banner-leave-to {
  opacity: 0;
}

.app-shell {
  position: relative;
  z-index: var(--z-base);
  min-height: 100dvh;
  background: var(--color-bg-page);
  transition:
    background-color var(--transition-normal),
    color var(--transition-normal);
}

.page-enter-active {
  transition:
    opacity 0.25s var(--ease-out-expo),
    transform 0.25s var(--ease-out-expo);
}

.page-leave-active {
  transition:
    opacity 0.18s ease,
    transform 0.18s ease;
}

.page-enter-from {
  opacity: 0;
  transform: translateY(8px);
}

.page-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}

.topbar {
  position: sticky;
  top: 12px;
  z-index: var(--z-sticky);
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 56px;
  margin: 0 var(--sp-6);
  padding: 0 var(--sp-5);
  border: 1px solid color-mix(in srgb, var(--color-text-primary) 25%, transparent);
  border-radius: var(--radius-full);
  background: color-mix(in srgb, var(--color-bg-card) 60%, transparent);
  backdrop-filter: blur(24px) saturate(200%);
  -webkit-backdrop-filter: blur(24px) saturate(200%);
  box-shadow:
    inset 0 1px 1px color-mix(in srgb, var(--color-text-primary) 30%, transparent),
    var(--shadow-ambient);
  transition:
    background-color 0.3s ease,
    border-color 0.3s ease,
    color 0.3s ease;
}

html.dark .topbar {
  background: color-mix(in srgb, var(--color-bg-card) 65%, transparent);
  border-color: color-mix(in srgb, var(--color-text-primary) 8%, transparent);
  box-shadow:
    inset 0 1px 1px color-mix(in srgb, var(--color-text-primary) 5%, transparent),
    var(--shadow-ambient);
}

.topbar::before {
  content: '';
  position: absolute;
  left: 0;
  top: 25%;
  bottom: 25%;
  width: 2px;
  border-radius: 2px;
  background: var(--color-brand);
}

.topbar-title {
  margin: 0;
  color: var(--color-text-primary);
  font-size: var(--fs-xl);
  font-weight: var(--fw-bold);
  line-height: var(--lh-snug);
  letter-spacing: var(--ls-tight);
}

.topbar-actions {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
}

.topbar-search-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 42px;
  height: 42px;
  padding: 0;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-full);
  background: var(--color-bg-topbar);
  color: var(--color-text-regular);
  cursor: pointer;
  box-shadow:
    inset 0 1px 1px color-mix(in srgb, var(--color-text-primary) 10%, transparent),
    var(--shadow-ambient);
  transition: all 0.2s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.topbar-search-btn:hover {
  color: var(--color-brand);
  border-color: var(--color-brand-muted);
  background: var(--color-bg-card);
  box-shadow: var(--shadow-elevated);
  transform: translateY(-2px) scale(1.05);
}

.topbar-search-btn:active {
  transform: translateY(0) scale(0.95);
}

.theme-fab {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 42px;
  height: 42px;
  padding: 0;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-full);
  background: var(--color-bg-topbar);
  color: var(--color-text-regular);
  cursor: pointer;
  box-shadow:
    inset 0 1px 1px color-mix(in srgb, var(--color-text-primary) 10%, transparent),
    var(--shadow-ambient);
  transition: all 0.2s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.theme-fab:hover {
  color: var(--color-brand);
  border-color: var(--color-brand-muted);
  background: var(--color-bg-card);
  box-shadow: var(--shadow-elevated);
  transform: translateY(-2px) scale(1.05);
}

.theme-fab:active {
  transform: translateY(0) scale(0.95);
}

.theme-fab-icon {
  width: 18px;
  height: 18px;
}

.main-content {
  width: min(100%, 1400px);
  margin: 0 auto;
  padding: var(--sp-8) var(--sp-8) 116px;
  box-sizing: border-box;
  position: relative;
  z-index: var(--z-base);
}

.dock-hotspot {
  position: fixed;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: var(--z-dock);
  height: 28px;
}

.dock-pill {
  position: fixed;
  bottom: var(--sp-2);
  left: 50%;
  z-index: var(--z-dock-pill);
  display: flex;
  align-items: center;
  justify-content: center;
  width: 54px;
  height: 18px;
  padding: 0;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-full);
  background: var(--color-bg-topbar);
  cursor: pointer;
  box-shadow:
    var(--shadow-ambient),
    0 0 12px var(--color-brand-soft);
  transform: translateX(-50%);
  transition:
    background-color 0.3s ease,
    border-color 0.3s ease,
    color 0.3s ease;
}

.pill-bar {
  width: 22px;
  height: 3px;
  border-radius: var(--radius-full);
  background: linear-gradient(90deg, var(--color-brand), var(--color-brand-muted));
}

.dock {
  position: fixed;
  bottom: var(--sp-3);
  left: 50%;
  z-index: var(--z-dock-expanded);
  display: flex;
  align-items: flex-end;
  gap: var(--sp-2);
  padding: var(--sp-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-xl);
  background: var(--color-bg-topbar);
  backdrop-filter: blur(20px) saturate(180%);
  -webkit-backdrop-filter: blur(20px) saturate(180%);
  box-shadow:
    inset 0 1px 1px color-mix(in srgb, var(--color-text-primary) 10%, transparent),
    var(--shadow-floating);
  transform: translateX(-50%);
  transition:
    background-color 0.3s ease,
    border-color 0.3s ease,
    color 0.3s ease;
}

.dock::before {
  content: '';
  position: absolute;
  top: 0;
  left: 20%;
  right: 20%;
  height: 1px;
  background: linear-gradient(90deg, transparent, var(--color-brand-muted), transparent);
  opacity: 0.3;
}

.dock-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--sp-0_5);
  width: 58px;
  min-height: 58px;
  padding: var(--sp-1);
  border: 1px solid transparent;
  border-radius: 16px;
  color: var(--color-text-regular);
  text-decoration: none;
  transition:
    transform var(--transition-spring),
    background-color var(--transition-fast),
    color var(--transition-fast),
    border-color var(--transition-fast);
}

.dock-item:hover {
  color: var(--color-text-primary);
  background: var(--color-bg-hover);
  transform: translateY(-6px) scale(1.05);
}

.dock-item.active {
  color: var(--color-brand);
  background: var(--color-brand-soft);
  border-color: var(--color-brand-muted);
}

.dock-item:active {
  transform: translateY(-2px) scale(0.98);
}

.dock-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.dock-label {
  color: currentColor;
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  line-height: var(--lh-tight);
  letter-spacing: var(--ls-wide);
}

.dock-enter-active {
  transition:
    opacity 0.22s ease,
    transform 0.3s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.dock-leave-active {
  transition:
    opacity var(--transition-fast),
    transform var(--transition-fast);
}

.dock-enter-from {
  opacity: 0;
  transform: translateX(-50%) translateY(18px) scale(0.96);
}

.dock-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(14px) scale(0.98);
}

.pill-enter-active,
.pill-leave-active {
  transition:
    opacity var(--transition-fast),
    transform var(--transition-fast);
}

.pill-enter-from,
.pill-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(8px);
}

@media (max-width: 768px) {
  .topbar {
    top: 8px;
    margin: 0 var(--sp-3);
    min-height: 52px;
    backdrop-filter: blur(16px) saturate(180%);
    -webkit-backdrop-filter: blur(16px) saturate(180%);
  }

  .topbar-kicker {
    display: none;
  }

  .topbar-title {
    font-size: var(--fs-lg);
  }

  .topbar-search-btn,
  .theme-fab {
    width: 38px;
    height: 38px;
  }

  .main-content {
    padding: var(--sp-4) var(--sp-4) 106px;
  }

  .dock {
    right: var(--sp-3);
    left: var(--sp-3);
    justify-content: space-around;
    transform: none;
  }

  .dock-item {
    width: 31%;
    min-height: 54px;
  }

  .dock-enter-from {
    transform: translateY(18px) scale(0.96);
  }

  .dock-leave-to {
    transform: translateY(14px) scale(0.98);
  }
}

@media (prefers-reduced-motion: reduce) {
  .dock,
  .dock-item,
  .dock-enter-active,
  .dock-leave-active,
  .pill-enter-active,
  .pill-leave-active,
  .fade-enter-active,
  .fade-leave-active,
  .page-enter-active,
  .page-leave-active {
    transition-duration: 0.01ms !important;
  }
}

@media (prefers-reduced-transparency: reduce) {
  .topbar,
  .dock {
    backdrop-filter: none;
    -webkit-backdrop-filter: none;
    background: var(--color-bg-card);
  }
}
</style>
