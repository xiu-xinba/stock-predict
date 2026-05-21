<template>
  <div id="app">
    <header class="topbar">
      <h1 class="topbar-title">
        <span aria-hidden="true" class="topbar-logo">📊</span>
        {{ currentTitle }}
      </h1>
    </header>
    <div class="theme-fab" @click="toggleTheme" :title="themeTitle" role="button" tabindex="0" @keydown.enter="toggleTheme">
      <svg v-if="isDark" class="theme-fab-icon" viewBox="0 0 1024 1024"><path fill="currentColor" d="M240.448 240.448a384 384 0 1 0 559.424 525.696 448 448 0 0 1-542.016-542.08 391 391 0 0 0-17.408 16.384m181.056 362.048a384 384 0 0 0 525.632 16.384A448 448 0 1 1 405.056 76.8a384 384 0 0 0 16.448 525.696"/></svg>
      <svg v-else class="theme-fab-icon" viewBox="0 0 1024 1024"><path fill="currentColor" d="M512 64h64v192h-64zm0 576h64v192h-64zM160 480v-64h192v64zm576 0v-64h192v64zM249.856 199.04l45.248-45.184L430.848 289.6 385.6 334.848 249.856 199.104zM657.152 606.4l45.248-45.248 135.744 135.744-45.248 45.248zM114.048 923.2 68.8 877.952l316.8-316.8 45.248 45.248zM702.4 334.848 657.152 289.6l135.744-135.744 45.248 45.248z"/></svg>
    </div>
    <main class="main-content">
      <router-view v-slot="{ Component }">
        <transition name="fade" mode="out-in">
          <component :is="Component" />
        </transition>
      </router-view>
    </main>
    <transition name="pill">
      <div
        v-if="!dockVisible"
        class="dock-pill"
        @mouseenter="showDock"
        @touchstart.prevent="showDock"
        role="button"
        aria-label="展开导航栏"
        tabindex="0"
        @keydown.enter="showDock"
      >
        <span class="pill-bar"></span>
      </div>
    </transition>
    <transition name="dock">
      <nav
        v-if="dockVisible"
        class="dock"
        @mouseenter="cancelHide"
        @mouseleave="scheduleHide"
        @touchend="scheduleHide"
      >
        <div class="dock-track">
          <router-link
            v-for="item in navItems"
            :key="item.path"
            :to="item.path"
            class="dock-item"
            :class="{ active: activeMenu === item.path }"
            @mouseenter="onItemEnter($event, item)"
            @mouseleave="onItemLeave"
          >
            <div class="dock-icon-wrap">
              <el-icon :size="28"><component :is="item.icon" /></el-icon>
            </div>
            <span class="dock-label">{{ item.label }}</span>
          </router-link>
        </div>
        <transition name="tooltip">
          <div v-if="tooltipText" class="dock-tooltip" :style="tooltipStyle">
            {{ tooltipText }}
          </div>
        </transition>
      </nav>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { Star, TrendCharts, MagicStick } from '@element-plus/icons-vue'
import type { Component } from 'vue'
import { useTheme } from '@/composables/useTheme'

interface NavItem {
  path: string
  label: string
  icon: Component
  title: string
}

const navItems: NavItem[] = [
  { path: '/watchlist', label: '自选', icon: Star, title: '我的自选' },
  { path: '/market', label: '行情', icon: TrendCharts, title: '市场行情' },
  { path: '/predict', label: '预测', icon: MagicStick, title: '基金预测' },
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
  hideTimer = setTimeout(() => {
    dockVisible.value = false
  }, 800)
}

const tooltipText = ref('')
const tooltipStyle = ref<{ left: string }>({ left: '0px' })

function onItemEnter(e: MouseEvent, item: NavItem | null) {
  cancelHide()
  const text = item ? item.title : ''
  tooltipText.value = text
  const el = (e.currentTarget as HTMLElement)
  const rect = el.getBoundingClientRect()
  const dockEl = el.closest('.dock')
  if (dockEl) {
    const dockRect = dockEl.getBoundingClientRect()
    tooltipStyle.value = {
      left: `${rect.left - dockRect.left + rect.width / 2}px`,
    }
  }
}

function onItemLeave() {
  tooltipText.value = ''
}

onUnmounted(() => {
  if (hideTimer) clearTimeout(hideTimer)
})
</script>

<style>
#app {
  background-color: var(--color-bg-page);
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.topbar {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  background: var(--color-bg-card);
  border-bottom: 1px solid var(--color-border);
  padding: 0 var(--sp-6);
  height: 56px;
  flex-shrink: 0;
  position: sticky;
  top: 0;
  z-index: 50;
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  background-color: rgba(255, 255, 255, 0.85);
  box-shadow: 0 1px 8px rgba(0, 0, 0, 0.06);
}

html.dark .topbar {
  background-color: rgba(35, 35, 38, 0.85);
  box-shadow: 0 1px 8px rgba(0, 0, 0, 0.25);
}

.topbar-logo {
  font-size: var(--fs-lg);
  flex-shrink: 0;
}

.topbar-title {
  flex: 1;
  font-size: var(--fs-lg);
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0;
  display: flex;
  align-items: center;
  gap: var(--sp-2);
}

.topbar-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-bg-card);
  color: var(--color-text-regular);
  cursor: pointer;
  transition: background-color 0.15s ease, color 0.15s ease, border-color 0.15s ease;
  flex-shrink: 0;
}

.topbar-btn:hover {
  background: var(--color-bg-hover);
  color: var(--color-text-primary);
  border-color: var(--color-brand);
}

.theme-fab {
  position: fixed;
  top: 72px;
  right: 24px;
  z-index: 90;
  width: 44px;
  height: 44px;
  border-radius: 50%;
  border: 1px solid var(--color-border);
  background: var(--color-bg-card);
  color: var(--color-text-regular);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  transition: transform 0.2s ease, box-shadow 0.2s ease, background-color 0.15s ease, color 0.15s ease;
  outline: none;
  -webkit-tap-highlight-color: transparent;
}

.theme-fab:hover {
  transform: scale(1.1);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);
  color: var(--color-brand);
  border-color: var(--color-brand);
}

.theme-fab:active {
  transform: scale(0.95);
}

.theme-fab:focus-visible {
  box-shadow: 0 0 0 3px rgba(51, 102, 255, 0.2);
}

.theme-fab-icon {
  width: 20px;
  height: 20px;
  transition: transform 0.3s ease;
}

.theme-fab:hover .theme-fab-icon {
  transform: rotate(30deg);
}

.main-content {
  flex: 1;
  padding: 0 var(--sp-6);
  max-width: 1000px;
  width: 100%;
  margin: 0 auto;
  box-sizing: border-box;
  padding-bottom: 100px;
}

.dock-pill {
  position: fixed;
  bottom: var(--sp-2);
  left: 50%;
  transform: translateX(-50%);
  z-index: 99;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 20px;
  border-radius: var(--radius-full);
  background: rgba(255, 255, 255, 0.6);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border: 1px solid rgba(0, 0, 0, 0.06);
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08);
  cursor: pointer;
  transition: background 0.2s ease, box-shadow 0.2s ease, transform 0.2s ease;
  -webkit-tap-highlight-color: transparent;
}

html.dark .dock-pill {
  background: rgba(60, 60, 65, 0.6);
  border-color: rgba(255, 255, 255, 0.06);
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.3);
}

.dock-pill:hover {
  background: rgba(255, 255, 255, 0.85);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.12);
  transform: translateX(-50%) scale(1.08);
}

html.dark .dock-pill:hover {
  background: rgba(70, 70, 75, 0.85);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.4);
}

.pill-bar {
  width: 20px;
  height: 4px;
  border-radius: 2px;
  background: var(--color-text-secondary);
  transition: background 0.2s ease, width 0.2s ease;
}

.dock-pill:hover .pill-bar {
  background: var(--color-brand);
  width: 24px;
}

.pill-enter-active {
  transition: opacity 0.2s ease, transform 0.25s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.pill-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}

.pill-enter-from {
  opacity: 0;
  transform: translateX(-50%) scale(0.8);
}

.pill-leave-to {
  opacity: 0;
  transform: translateX(-50%) scale(0.6);
}

.dock {
  position: fixed;
  bottom: var(--sp-3);
  left: 50%;
  transform: translateX(-50%);
  z-index: 100;
  display: flex;
  flex-direction: column;
  align-items: center;
  will-change: transform, opacity;
  contain: layout style;
}

.dock-track {
  display: flex;
  align-items: flex-end;
  gap: var(--sp-1);
  padding: var(--sp-2) var(--sp-3);
  background: rgba(255, 255, 255, 0.72);
  backdrop-filter: blur(20px) saturate(180%);
  -webkit-backdrop-filter: blur(20px) saturate(180%);
  border-radius: 20px;
  border: 1px solid rgba(255, 255, 255, 0.3);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.12), 0 2px 8px rgba(0, 0, 0, 0.08);
}

html.dark .dock-track {
  background: rgba(50, 50, 55, 0.72);
  border-color: rgba(255, 255, 255, 0.08);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4), 0 2px 8px rgba(0, 0, 0, 0.3);
}

.dock-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-decoration: none;
  color: var(--color-text-regular);
  cursor: pointer;
  transition: transform 0.2s cubic-bezier(0.34, 1.56, 0.64, 1);
  padding: var(--sp-1);
  border-radius: var(--radius-md);
  position: relative;
  -webkit-tap-highlight-color: transparent;
  will-change: transform;
  contain: layout style;
}

.dock-item:hover {
  transform: translateY(-8px) scale(1.25);
  color: var(--color-text-primary);
}

.dock-item.active {
  color: var(--color-brand);
}

.dock-item.active .dock-icon-wrap {
  background: rgba(51, 102, 255, 0.1);
}

html.dark .dock-item.active .dock-icon-wrap {
  background: rgba(91, 138, 255, 0.15);
}

.dock-icon-wrap {
  width: 48px;
  height: 48px;
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background-color 0.15s ease, transform 0.2s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.dock-item:hover .dock-icon-wrap {
  background: var(--color-bg-hover);
}

.dock-label {
  font-size: var(--fs-xs);
  font-weight: 500;
  margin-top: 2px;
  opacity: 0;
  transition: opacity 0.15s ease;
  white-space: nowrap;
}

.dock-item:hover .dock-label {
  opacity: 1;
}

.dock-item.active .dock-label {
  opacity: 1;
}

.dock-separator {
  width: 1px;
  height: 32px;
  background: var(--color-border);
  margin: 0 var(--sp-1);
  align-self: center;
}

.dock-action {
  background: none;
  border: none;
  font-family: inherit;
}

.dock-tooltip {
  position: absolute;
  bottom: 100%;
  margin-bottom: var(--sp-2);
  padding: var(--sp-1) var(--sp-3);
  background: var(--color-bg-overlay);
  color: #ffffff;
  font-size: var(--fs-sm);
  font-weight: 500;
  border-radius: var(--radius-sm);
  white-space: nowrap;
  transform: translateX(-50%);
  pointer-events: none;
  box-shadow: var(--shadow-md);
  z-index: 200;
}

.dock-tooltip::after {
  content: '';
  position: absolute;
  top: 100%;
  left: 50%;
  transform: translateX(-50%);
  border: 5px solid transparent;
  border-top-color: var(--color-bg-overlay);
}

.dock-enter-active {
  transition: opacity 0.25s ease, transform 0.35s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.dock-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.dock-enter-from {
  opacity: 0;
  transform: translateX(-50%) translateY(20px);
}

.dock-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(10px);
}

.tooltip-enter-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}

.tooltip-leave-active {
  transition: opacity 0.1s ease;
}

.tooltip-enter-from {
  opacity: 0;
  transform: translateX(-50%) translateY(4px);
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.15s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

@media (max-width: 767px) {
  .topbar {
    padding: 0 var(--sp-4);
  }
  .topbar-title {
    font-size: var(--fs-md);
  }
  .main-content {
    padding: 0 var(--sp-4) 100px;
  }
  .dock-pill {
    width: 48px;
    height: 18px;
    bottom: var(--sp-1);
  }
  .pill-bar {
    width: 16px;
  }
  .dock-pill:hover .pill-bar {
    width: 20px;
  }
  .dock-track {
    padding: var(--sp-2) var(--sp-2);
    gap: 2px;
  }
  .dock-icon-wrap {
    width: 44px;
    height: 44px;
    border-radius: 12px;
  }
  .dock-item:hover {
    transform: translateY(-6px) scale(1.15);
  }
}

@media (prefers-reduced-motion: reduce) {
  .dock-item,
  .dock-enter-active,
  .dock-leave-active,
  .pill-enter-active,
  .pill-leave-active,
  .tooltip-enter-active,
  .tooltip-leave-active,
  .fade-enter-active,
  .fade-leave-active {
    transition-duration: 0.01ms !important;
  }
}
</style>
