import { ref, computed, watch, onUnmounted } from 'vue'

type ThemeMode = 'light' | 'dark' | 'system'

function getSystemPreference(): boolean {
  return window.matchMedia('(prefers-color-scheme: dark)').matches
}

function getInitialMode(): ThemeMode {
  try {
    const saved = localStorage.getItem('theme-mode')
    if (saved === 'light' || saved === 'dark' || saved === 'system') {
      return saved
    }
  } catch {
    // ignore
  }
  return 'system'
}

function applyDark(isDark: boolean) {
  if (isDark) {
    document.documentElement.classList.add('dark')
  } else {
    document.documentElement.classList.remove('dark')
  }
}

const mode = ref<ThemeMode>(getInitialMode())

let mediaQueryListenerCount = 0
const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')

function handleMediaChange(e: MediaQueryListEvent) {
  if (mode.value === 'system') {
    applyDark(e.matches)
  }
}

watch(mode, (newMode) => {
  try {
    localStorage.setItem('theme-mode', newMode)
  } catch {
    // ignore
  }
  applyDark(newMode === 'dark' || (newMode === 'system' && getSystemPreference()))
}, { immediate: true })

export function useTheme() {
  const isDark = computed(() =>
    mode.value === 'dark' || (mode.value === 'system' && getSystemPreference())
  )

  if (mediaQueryListenerCount === 0) {
    mediaQuery.addEventListener('change', handleMediaChange)
  }
  mediaQueryListenerCount++

  onUnmounted(() => {
    mediaQueryListenerCount--
    if (mediaQueryListenerCount <= 0) {
      mediaQueryListenerCount = 0
      mediaQuery.removeEventListener('change', handleMediaChange)
    }
  })

  function setMode(newMode: ThemeMode) {
    mode.value = newMode
  }

  function toggleTheme() {
    mode.value = isDark.value ? 'light' : 'dark'
  }

  function cycleMode() {
    const order: ThemeMode[] = ['light', 'dark', 'system']
    const idx = order.indexOf(mode.value)
    mode.value = order[(idx + 1) % order.length]
  }

  return {
    mode,
    isDark,
    setMode,
    toggleTheme,
    cycleMode,
  }
}
