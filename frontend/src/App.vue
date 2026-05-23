<template>
  <div id="app" class="app-shell">
    <header class="topbar">
      <div>
        <p class="topbar-kicker">Realtime Fund Analytics</p>
        <h1 class="topbar-title">{{ currentTitle }}</h1>
      </div>
    </header>

    <button class="theme-fab" type="button" :title="themeTitle" @click="toggleTheme">
      <svg v-if="isDark" class="theme-fab-icon" viewBox="0 0 1024 1024" aria-hidden="true">
        <path fill="currentColor" d="M512 64h64v192h-64zm0 576h64v192h-64zM160 480v-64h192v64zm576 0v-64h192v64zM249.856 199.04l45.248-45.184L430.848 289.6 385.6 334.848 249.856 199.104zM657.152 606.4l45.248-45.248 135.744 135.744-45.248 45.248zM114.048 923.2 68.8 877.952l316.8-316.8 45.248 45.248zM702.4 334.848 657.152 289.6l135.744-135.744 45.248 45.248z"/>
      </svg>
      <svg v-else class="theme-fab-icon" viewBox="0 0 1024 1024" aria-hidden="true">
        <path fill="currentColor" d="M240.448 240.448a384 384 0 1 0 559.424 525.696 448 448 0 0 1-542.016-542.08 391 391 0 0 0-17.408 16.384m181.056 362.048a384 384 0 0 0 525.632 16.384A448 448 0 1 1 405.056 76.8a384 384 0 0 0 16.448 525.696"/>
      </svg>
    </button>

    <main class="main-content">
      <router-view v-slot="{ Component }">
        <transition name="fade" mode="out-in">
          <component :is="Component" />
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
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { MagicStick, Star, TrendCharts } from '@element-plus/icons-vue'
import type { Component } from 'vue'
import { useTheme } from '@/composables/useTheme'

interface NavItem {
  path: string
  label: string
  icon: Component
  title: string
}

const navItems: NavItem[] = [
  { path: '/watchlist', label: '自选', icon: Star, title: '自选监控' },
  { path: '/market', label: '行情', icon: TrendCharts, title: '市场行情' },
  { path: '/predict', label: '预测', icon: MagicStick, title: '模型预测' },
]

const route = useRoute()
const { isDark, toggleTheme } = useTheme()

const themeTitle = computed(() => isDark.value ? '切换到日间模式' : '切换到夜间模式')

const activeMenu = computed(() => {
  if (route.path.startsWith('/predict')) return '/predict'
  if (route.path.startsWith('/market')) return '/market'
  if (route.path.startsWith('/watchlist')) return '/watchlist'
  return route.path
})

const currentTitle = computed(() => {
  const item = navItems.find((n) => n.path === activeMenu.value)
  return item?.title ?? '基金预测'
})

const dockVisible = ref(false)
let hideTimer: ReturnType<typeof setTimeout> | null = null
let initialTimer: ReturnType<typeof setTimeout> | null = null

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

onMounted(() => {
  showDock()
  initialTimer = setTimeout(scheduleHide, 1800)
})

onUnmounted(() => {
  if (hideTimer) clearTimeout(hideTimer)
  if (initialTimer) clearTimeout(initialTimer)
})
</script>

<style>
#app {
  min-height: 100vh;
}

.app-shell {
  min-height: 100vh;
  background: var(--color-bg-page);
  transition: background-color var(--transition-normal), color var(--transition-normal);
}

.topbar {
  position: sticky;
  top: 0;
  z-index: 30;
  display: flex;
  align-items: center;
  min-height: 64px;
  padding: 0 var(--sp-8);
  border-bottom: 1px solid var(--color-border);
  background: var(--color-bg-topbar);
  box-sizing: border-box;
  box-shadow: var(--shadow-sm);
  transition: background-color var(--transition-normal), border-color var(--transition-normal), box-shadow var(--transition-normal);
}

.topbar-kicker {
  margin: 0 0 2px;
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
  line-height: var(--lh-tight);
}

.topbar-title {
  margin: 0;
  color: var(--color-text-primary);
  font-size: var(--fs-xl);
  font-weight: var(--fw-bold);
  line-height: var(--lh-snug);
}

.theme-fab {
  position: fixed;
  top: 14px;
  right: 24px;
  z-index: 80;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 42px;
  height: 42px;
  padding: 0;
  border: 1px solid var(--color-border);
  border-radius: 50%;
  background: var(--color-bg-topbar);
  color: var(--color-text-regular);
  cursor: pointer;
  box-shadow: var(--shadow-sm);
  transition: transform var(--transition-fast), border-color var(--transition-fast), color var(--transition-fast), background-color var(--transition-fast), box-shadow var(--transition-fast);
}

.theme-fab:hover {
  color: var(--color-brand);
  border-color: var(--color-brand-muted);
  background: var(--color-bg-card);
  box-shadow: var(--shadow-md);
  transform: translateY(-1px);
}

.theme-fab:active {
  transform: translateY(0) scale(0.96);
}

.theme-fab-icon {
  width: 18px;
  height: 18px;
}

.main-content {
  width: min(100%, 1180px);
  margin: 0 auto;
  padding: var(--sp-6) var(--sp-8) 116px;
  box-sizing: border-box;
}

.dock-hotspot {
  position: fixed;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 50;
  height: 28px;
}

.dock-pill {
  position: fixed;
  bottom: var(--sp-2);
  left: 50%;
  z-index: 70;
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
  box-shadow: var(--shadow-sm);
  transform: translateX(-50%);
}

.pill-bar {
  width: 22px;
  height: 3px;
  border-radius: var(--radius-full);
  background: var(--color-text-secondary);
}

.dock {
  position: fixed;
  bottom: var(--sp-3);
  left: 50%;
  z-index: 75;
  display: flex;
  align-items: flex-end;
  gap: var(--sp-2);
  padding: var(--sp-2);
  border: 1px solid var(--color-border);
  border-radius: 18px;
  background: var(--color-bg-topbar);
  box-shadow: var(--shadow-lg);
  transform: translateX(-50%);
}

.dock-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 3px;
  width: 58px;
  min-height: 58px;
  padding: var(--sp-1);
  border: 1px solid transparent;
  border-radius: 14px;
  color: var(--color-text-regular);
  text-decoration: none;
  transition: transform 0.18s cubic-bezier(0.2, 0.8, 0.2, 1), background-color var(--transition-fast), color var(--transition-fast), border-color var(--transition-fast);
}

.dock-item:hover {
  color: var(--color-text-primary);
  background: var(--color-bg-hover);
  transform: translateY(-7px);
}

.dock-item.active {
  color: var(--color-brand);
  background: var(--color-brand-soft);
  border-color: var(--color-brand-muted);
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
}

.dock-enter-active {
  transition: opacity 0.22s ease, transform 0.3s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.dock-leave-active {
  transition: opacity 0.16s ease, transform 0.18s ease;
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
  transition: opacity 0.18s ease, transform 0.18s ease;
}

.pill-enter-from,
.pill-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(8px);
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.12s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

@media (max-width: 900px) {
  .topbar {
    min-height: 60px;
    padding: 0 var(--sp-4);
  }

  .topbar-kicker {
    display: none;
  }

  .topbar-title {
    font-size: var(--fs-lg);
  }

  .theme-fab {
    top: 10px;
    right: 14px;
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
  .fade-leave-active {
    transition-duration: 0.01ms !important;
  }
}
</style>
