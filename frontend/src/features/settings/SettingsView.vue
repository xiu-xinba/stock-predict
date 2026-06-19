<template>
  <div class="settings-page">
    <header class="settings-header fade-slide-up" style="--delay: 0">
      <span class="eyebrow">偏好设置</span>
      <h1 class="settings-title">设置中心</h1>
      <p class="settings-subtitle">个性化您的使用体验</p>
    </header>

    <!-- Appearance -->
    <section class="card settings-section fade-slide-up" style="--delay: 1">
      <h2 class="section-title">外观</h2>
      <div class="setting-row">
        <div class="setting-info">
          <span class="setting-label">主题模式</span>
          <span class="setting-desc">选择界面配色方案</span>
        </div>
        <div class="tab-group">
          <button
            :class="['tab-btn', { active: themeMode === 'light' }]"
            type="button"
            @click="setThemeMode('light')"
          >
            日间
          </button>
          <button
            :class="['tab-btn', { active: themeMode === 'dark' }]"
            type="button"
            @click="setThemeMode('dark')"
          >
            夜间
          </button>
          <button
            :class="['tab-btn', { active: themeMode === 'system' }]"
            type="button"
            @click="setThemeMode('system')"
          >
            系统
          </button>
        </div>
      </div>
    </section>

    <!-- Data -->
    <section class="card settings-section fade-slide-up" style="--delay: 2">
      <h2 class="section-title">数据</h2>
      <div class="setting-row">
        <div class="setting-info">
          <span class="setting-label">自动刷新间隔</span>
          <span class="setting-desc">行情数据自动更新频率</span>
        </div>
        <div class="tab-group">
          <button
            v-for="opt in refreshOptions"
            :key="opt.value"
            :class="['tab-btn', { active: refreshIntervalSeconds === opt.value }]"
            type="button"
            @click="settings.setRefreshInterval(opt.value)"
          >
            {{ opt.label }}
          </button>
        </div>
      </div>
    </section>

    <!-- About -->
    <section class="card settings-section fade-slide-up" style="--delay: 3">
      <h2 class="section-title">关于</h2>
      <div class="setting-row">
        <div class="setting-info">
          <span class="setting-label">版本</span>
        </div>
        <span class="setting-value">1.0.0</span>
      </div>
      <div class="setting-row">
        <div class="setting-info">
          <span class="setting-label">技术栈</span>
        </div>
        <span class="setting-value">Vue 3 + Go</span>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
/** 设置页面，提供主题切换、行情刷新间隔配置等偏好设置 */
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { storeToRefs } from 'pinia'
import { useTheme } from '@/shared/composables/useTheme'
import { useSettingsStore } from '@/features/settings'

defineOptions({ name: 'SettingsView' })

const { setMode } = useTheme()

type ThemePreference = 'light' | 'dark' | 'system'

const THEME_STORAGE_KEY = 'theme-mode-preference'
const settings = useSettingsStore()
const { refreshIntervalSeconds } = storeToRefs(settings)

const storedPreference = localStorage.getItem(THEME_STORAGE_KEY) as ThemePreference | null
const preference = ref<ThemePreference>(
  storedPreference === 'light' || storedPreference === 'dark' || storedPreference === 'system'
    ? storedPreference
    : 'system',
)

const themeMode = computed<ThemePreference>(() => preference.value)

function applySystemTheme() {
  const isDark = window.matchMedia('(prefers-color-scheme: dark)').matches
  setMode(isDark ? 'dark' : 'light')
}

function setThemeMode(mode: ThemePreference) {
  preference.value = mode
}

watch(
  preference,
  (newPref) => {
    localStorage.setItem(THEME_STORAGE_KEY, newPref)
    if (newPref === 'system') {
      applySystemTheme()
    } else {
      setMode(newPref)
    }
  },
  { immediate: true },
)

let mediaQuery: MediaQueryList | null = null
function onMediaChange() {
  if (preference.value === 'system') {
    applySystemTheme()
  }
}

onMounted(() => {
  mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
  mediaQuery.addEventListener('change', onMediaChange)
})

onUnmounted(() => {
  mediaQuery?.removeEventListener('change', onMediaChange)
})

const refreshOptions = [
  { label: '30秒', value: 30 },
  { label: '1分钟', value: 60 },
  { label: '5分钟', value: 300 },
]
</script>

<style scoped>
.fade-slide-up {
  animation: fade-slide-up 0.5s var(--ease-out-expo) both;
  animation-delay: calc(var(--delay, 0) * 60ms);
}

.settings-page {
  display: flex;
  flex-direction: column;
  gap: var(--sp-8);
}

.settings-header {
  margin-bottom: var(--sp-2);
}

.settings-title {
  font-size: var(--fs-5xl);
  font-weight: var(--fw-bold);
  letter-spacing: var(--ls-tighter);
  line-height: 1.05;
  color: var(--color-text-primary);
  margin: var(--sp-2) 0 0;
}

.settings-subtitle {
  font-size: var(--fs-sm);
  color: var(--color-text-secondary);
  margin: var(--sp-1) 0 0;
}

.settings-section {
  padding: var(--sp-6);
}

.section-title {
  font-size: var(--fs-md);
  font-weight: var(--fw-bold);
  color: var(--color-text-primary);
  margin: 0 0 var(--sp-5);
}

.setting-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--sp-4);
  padding: var(--sp-4) 0;
  border-bottom: 1px solid var(--color-border-light);
}

.setting-row:last-child {
  border-bottom: none;
}

.setting-info {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.setting-label {
  font-size: var(--fs-base);
  font-weight: var(--fw-medium);
  color: var(--color-text-primary);
}

.setting-desc {
  font-size: var(--fs-xs);
  color: var(--color-text-secondary);
  margin-top: var(--sp-0_5);
}

.setting-value {
  font-size: var(--fs-sm);
  color: var(--color-text-secondary);
  font-variant-numeric: tabular-nums;
  flex-shrink: 0;
}

.tab-btn:active:not(:disabled) {
  transform: scale(0.95);
}

@media (max-width: 640px) {
  .setting-row {
    flex-direction: column;
    align-items: flex-start;
    gap: var(--sp-3);
  }

  .settings-title {
    font-size: var(--fs-2xl);
  }
}

@media (prefers-reduced-motion: reduce) {
  .fade-slide-up {
    transition-duration: 0.01ms !important;
    animation-duration: 0.01ms !important;
  }
}
</style>
