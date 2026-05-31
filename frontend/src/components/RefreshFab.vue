<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useWatchlistStore } from '@/stores/watchlist'
import { useMarketStore } from '@/stores/market'
import { useFundDetailStore } from '@/stores/fundDetail'
import { useStockDetailStore } from '@/stores/stockDetail'

defineOptions({ name: 'RefreshFab' })

type FabPosition = 'bottom-right' | 'bottom-left' | 'top-right' | 'top-left'

const props = withDefaults(defineProps<{
  position?: FabPosition
  size?: number
  opacity?: number
  idleTimeout?: number
}>(), {
  position: 'bottom-right',
  size: 48,
  opacity: 0.92,
  idleTimeout: 900,
})

const route = useRoute()
const watchlistStore = useWatchlistStore()
const marketStore = useMarketStore()
const fundDetailStore = useFundDetailStore()
const stockDetailStore = useStockDetailStore()

const spinning = ref(false)
const lastRefresh = ref('')
const showTooltip = ref(false)
const isSnapped = ref(false)
const isHovering = ref(false)
let tooltipTimer: ReturnType<typeof setTimeout> | null = null
let idleTimer: ReturnType<typeof setTimeout> | null = null

const isLoading = computed(() => {
  return watchlistStore.loading || marketStore.loading || fundDetailStore.loading || stockDetailStore.loading
})

watch(isLoading, (loading, _, onCleanup) => {
  if (loading) {
    spinning.value = true
    resetIdleTimer()
  } else {
    const delay = setTimeout(() => {
      spinning.value = false
    }, 400)
    onCleanup(() => clearTimeout(delay))
  }
})

const positionStyle = computed(() => {
  const offset = 'var(--sp-6)'
  const dockOffset = 'calc(var(--dock-height, 80px) + var(--sp-4))'
  const positions: Record<FabPosition, Record<string, string>> = {
    'bottom-right': { right: offset, bottom: dockOffset },
    'bottom-left': { left: offset, bottom: dockOffset },
    'top-right': { right: offset, top: 'calc(64px + var(--sp-4))' },
    'top-left': { left: offset, top: 'calc(64px + var(--sp-4))' },
  }
  return positions[props.position]
})

const fabSize = computed(() => `${props.size}px`)

const compactSize = computed(() => `${Math.round(props.size * 0.6)}px`)

const isRightEdge = computed(() => props.position === 'bottom-right' || props.position === 'top-right')

function resetIdleTimer() {
  if (idleTimer) clearTimeout(idleTimer)
  isSnapped.value = false
  idleTimer = setTimeout(() => {
    if (!isHovering.value && !isLoading.value && !showTooltip.value) {
      isSnapped.value = true
    }
  }, props.idleTimeout)
}

function onFabMouseEnter() {
  isHovering.value = true
  isSnapped.value = false
  if (idleTimer) clearTimeout(idleTimer)
}

function onFabMouseLeave() {
  isHovering.value = false
  resetIdleTimer()
}

function refresh() {
  if (isLoading.value) return

  const path = route.path
  if (path.startsWith('/watchlist')) {
    watchlistStore.refreshQuotes()
    lastRefresh.value = new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
  } else if (path.startsWith('/market')) {
    marketStore.fetchMarketData(true)
    lastRefresh.value = new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
  } else if (path.startsWith('/fund/')) {
    const code = Array.isArray(route.params.fundCode) ? route.params.fundCode[0] : route.params.fundCode
    if (code) {
      fundDetailStore.fetchDetail(code)
      lastRefresh.value = new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
    }
  } else if (path.startsWith('/stock/')) {
    const code = Array.isArray(route.params.stockCode) ? route.params.stockCode[0] : route.params.stockCode
    if (code) {
      stockDetailStore.fetchDetail(code)
      lastRefresh.value = new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
    }
  }

  showTooltip.value = true
  resetIdleTimer()
  if (tooltipTimer) clearTimeout(tooltipTimer)
  tooltipTimer = setTimeout(() => {
    showTooltip.value = false
    resetIdleTimer()
  }, 2000)
}

function handleKeydown(e: KeyboardEvent) {
  if (e.altKey && e.key === 'r') {
    e.preventDefault()
    refresh()
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleKeydown)
  resetIdleTimer()
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
  if (tooltipTimer) clearTimeout(tooltipTimer)
  if (idleTimer) clearTimeout(idleTimer)
})
</script>

<template>
  <div
    :class="['refresh-fab-wrap', { snapped: isSnapped, 'snap-right': isRightEdge, 'snap-left': !isRightEdge }]"
    :style="{ ...positionStyle, '--fab-size': fabSize, '--fab-compact': compactSize, '--fab-opacity': opacity }"
    @mouseenter="onFabMouseEnter"
    @mouseleave="onFabMouseLeave"
  >
    <transition name="fab-tooltip">
      <span v-if="showTooltip && lastRefresh" class="fab-tooltip">
        已刷新 {{ lastRefresh }}
      </span>
    </transition>

    <button
      class="refresh-fab"
      :class="{ spinning, loading: isLoading }"
      type="button"
      :aria-label="isLoading ? '正在刷新' : '刷新当前页面'"
      :disabled="isLoading"
      @click="refresh"
    >
      <svg class="fab-icon" viewBox="0 0 1024 1024" aria-hidden="true">
        <path fill="currentColor" d="M771.776 794.88A384 384 0 0 1 128 512h64a320 320 0 0 0 555.712 216.448H654.72a32 32 0 1 1 0-64h149.44a32 32 0 0 1 32 32v148.16a32 32 0 1 1-64 0v-50.048zM296.064 229.12A384 384 0 0 1 896 512h-64a320 320 0 0 0-555.712-216.448h72.832a32 32 0 0 1 0 64H199.04a32 32 0 0 1-32-32V179.52a32 32 0 0 1 64 0v49.6z"/>
      </svg>
    </button>
  </div>
</template>

<style scoped>
.refresh-fab-wrap {
  position: fixed;
  z-index: var(--z-fab);
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  transition: right 0.22s ease, left 0.22s ease;
}

.refresh-fab-wrap.snapped.snap-right {
  right: calc(-1 * var(--fab-compact) * 0.35) !important;
}

.refresh-fab-wrap.snapped.snap-left {
  left: calc(-1 * var(--fab-compact) * 0.35) !important;
}

.refresh-fab {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: var(--fab-size);
  height: var(--fab-size);
  padding: 0;
  border: 1px solid var(--color-border);
  border-radius: 50%;
  background: var(--color-bg-topbar);
  backdrop-filter: blur(20px) saturate(180%);
  -webkit-backdrop-filter: blur(20px) saturate(180%);
  color: var(--color-text-regular);
  cursor: pointer;
  box-shadow: var(--shadow-md);
  opacity: var(--fab-opacity);
  transition:
    width 0.22s ease,
    height 0.22s ease,
    transform 0.3s cubic-bezier(0.2, 0.8, 0.2, 1),
    border-color var(--transition-fast),
    color var(--transition-fast),
    background-color var(--transition-fast),
    box-shadow var(--transition-fast),
    opacity var(--transition-fast);
  -webkit-tap-highlight-color: transparent;
  touch-action: manipulation;
}

.snapped .refresh-fab {
  width: var(--fab-compact);
  height: var(--fab-compact);
  opacity: 0.55;
  box-shadow: var(--shadow-sm);
}

.refresh-fab:hover {
  color: var(--color-brand);
  border-color: var(--color-brand-muted);
  background: var(--color-bg-card);
  box-shadow: var(--shadow-lg);
  opacity: 1;
  transform: translateY(-2px);
}

.refresh-fab:active {
  transform: translateY(0) scale(0.94);
  opacity: 1;
}

.refresh-fab.loading {
  cursor: not-allowed;
  color: var(--color-brand);
  border-color: var(--color-brand-muted);
}

.fab-icon {
  width: 55%;
  height: 55%;
  transition: transform 0.6s ease;
}

.refresh-fab.spinning .fab-icon {
  animation: fab-spin 0.8s ease infinite;
}

@keyframes fab-spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.fab-tooltip {
  position: absolute;
  right: calc(var(--fab-size) + var(--sp-2));
  white-space: nowrap;
  padding: var(--sp-1) var(--sp-3);
  border-radius: var(--radius-md);
  background: var(--color-bg-topbar);
  backdrop-filter: blur(20px) saturate(180%);
  -webkit-backdrop-filter: blur(20px) saturate(180%);
  border: 1px solid var(--color-border);
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
  box-shadow: var(--shadow-sm);
  pointer-events: none;
}

.snapped.snap-right .fab-tooltip {
  right: calc(var(--fab-compact) + var(--sp-2));
}

.snapped.snap-left .fab-tooltip {
  right: auto;
  left: calc(var(--fab-compact) + var(--sp-2));
}

.fab-tooltip-enter-active {
  transition: opacity 0.2s ease, transform 0.2s var(--ease-out-expo);
}

.fab-tooltip-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}

.fab-tooltip-enter-from {
  opacity: 0;
  transform: translateX(8px);
}

.fab-tooltip-leave-to {
  opacity: 0;
  transform: translateX(4px);
}

@media (max-width: 768px) {
  .refresh-fab-wrap {
    transition: right 0.22s ease, left 0.22s ease;
  }

  .refresh-fab-wrap .fab-tooltip {
    right: auto;
    left: 50%;
    bottom: calc(var(--fab-size) + var(--sp-2));
    transform: translateX(-50%);
  }

  .snapped .fab-tooltip {
    bottom: calc(var(--fab-compact) + var(--sp-2));
  }
}

@media (prefers-reduced-motion: reduce) {
  .refresh-fab-wrap,
  .refresh-fab,
  .fab-icon,
  .fab-tooltip-enter-active,
  .fab-tooltip-leave-active {
    transition-duration: 0.01ms !important;
    animation-duration: 0.01ms !important;
  }
}

@media (prefers-reduced-transparency: reduce) {
  .refresh-fab,
  .fab-tooltip {
    backdrop-filter: none;
    -webkit-backdrop-filter: none;
    background: var(--color-bg-card);
  }
}
</style>
