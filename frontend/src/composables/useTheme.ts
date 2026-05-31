import { ref, computed, watch } from 'vue'
import { invalidateCssVarCache } from '@/utils/format'

type ThemeMode = 'light' | 'dark'

const TIDE_DURATION = 500
const TIDE_EASING = 'cubic-bezier(0.4, 0, 0.2, 1)'

function getInitialMode(): ThemeMode {
  try {
    const saved = localStorage.getItem('theme-preference')
    if (saved === 'light' || saved === 'dark') {
      return saved
    }
  } catch {
    // ignore
  }
  if (window.matchMedia('(prefers-color-scheme: dark)').matches) {
    return 'dark'
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
    localStorage.setItem('theme-preference', newMode)
  } catch {
    // ignore
  }
  applyTheme(newMode)
  invalidateCssVarCache()
}, { immediate: true })

function toggleThemeWithTide(event?: MouseEvent) {
  const newMode = mode.value === 'dark' ? 'light' : 'dark'
  const prefersReduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches

  if (!('startViewTransition' in document) || prefersReduced) {
    mode.value = newMode
    return
  }

  let x = window.innerWidth
  let y = 0
  if (event) {
    x = event.clientX
    y = event.clientY
  } else {
    const fab = document.querySelector('.theme-fab') as HTMLElement | null
    if (fab) {
      const rect = fab.getBoundingClientRect()
      x = rect.left + rect.width / 2
      y = rect.top + rect.height / 2
    }
  }

  const endRadius = Math.hypot(
    Math.max(x, window.innerWidth - x),
    Math.max(y, window.innerHeight - y),
  )

  document.documentElement.style.setProperty('--tide-x', `${x}px`)
  document.documentElement.style.setProperty('--tide-y', `${y}px`)
  document.documentElement.style.setProperty('--tide-r', `${endRadius}px`)

  const transition = document.startViewTransition(async () => {
    mode.value = newMode
  })

  transition.ready.then(() => {
    document.documentElement.animate(
      [
        { clipPath: `circle(0px at ${x}px ${y}px)` },
        { clipPath: `circle(${endRadius}px at ${x}px ${y}px)` },
      ],
      {
        duration: TIDE_DURATION,
        easing: TIDE_EASING,
        pseudoElement: '::view-transition-new(root)',
      },
    )
  }).catch(() => {
    // Ignore animation failures; the theme state has already changed.
  })
}

export function useTheme() {
  const isDark = computed(() => mode.value === 'dark')

  function setMode(newMode: ThemeMode) {
    mode.value = newMode
  }

  function toggleTheme(event?: MouseEvent) {
    toggleThemeWithTide(event)
  }

  return {
    mode,
    isDark,
    setMode,
    toggleTheme,
  }
}
