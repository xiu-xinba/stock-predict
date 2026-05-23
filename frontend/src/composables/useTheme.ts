import { ref, computed, watch } from 'vue'

type ThemeMode = 'light' | 'dark'

function getInitialMode(): ThemeMode {
  try {
    const saved = localStorage.getItem('theme-mode')
    if (saved === 'light' || saved === 'dark') {
      return saved
    }
  } catch {
    // ignore
  }
  return 'light'
}

function applyTheme(newMode: ThemeMode) {
  const isDark = newMode === 'dark'
  if (isDark) {
    document.documentElement.classList.add('dark')
  } else {
    document.documentElement.classList.remove('dark')
  }
  document.documentElement.style.colorScheme = newMode
}

const mode = ref<ThemeMode>(getInitialMode())

watch(mode, (newMode) => {
  try {
    localStorage.setItem('theme-mode', newMode)
  } catch {
    // ignore
  }
  applyTheme(newMode)
}, { immediate: true })

export function useTheme() {
  const isDark = computed(() => mode.value === 'dark')

  function setMode(newMode: ThemeMode) {
    mode.value = newMode
  }

  function toggleTheme() {
    mode.value = isDark.value ? 'light' : 'dark'
  }

  function cycleMode() {
    const order: ThemeMode[] = ['light', 'dark']
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
