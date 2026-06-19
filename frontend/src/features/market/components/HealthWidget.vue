<template>
  <div
    ref="widgetRef"
    :class="[
      'health-widget',
      { expanded: isExpanded, dragging: isDragging, 'edge-mode': isEdgeMode },
    ]"
    :style="widgetStyle"
    @mousedown="onDragStart"
    @touchstart.passive="onDragStart"
  >
    <!-- 药丸内容（收起态可见） -->
    <div class="health-pill-content" @click.stop="toggleExpand">
      <span
        v-for="(info, key) in sortedHealthData"
        :key="key"
        :class="['pill-dot', info.status]"
        :title="formatTooltip(info)"
      ></span>
    </div>

    <!-- 卡片内容（展开态可见） -->
    <div class="health-card-content">
      <div
        class="health-card-head"
        @mousedown.stop="onDragStart"
        @touchstart.stop.passive="onDragStart"
      >
        <span class="health-card-title">数据源状态</span>
        <div class="health-card-actions">
          <span v-if="updateTime" class="health-card-time">{{ updateTime }}</span>
          <button
            class="health-card-btn"
            type="button"
            title="刷新"
            :disabled="loading"
            @pointerdown.stop
            @click.stop="refresh"
          >
            <svg :class="{ spinning: loading }" viewBox="0 0 24 24" width="14" height="14">
              <path
                fill="currentColor"
                d="M17.65 6.35A7.958 7.958 0 0012 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08A5.99 5.99 0 0112 18c-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z"
              />
            </svg>
          </button>
          <button
            class="health-card-btn"
            type="button"
            title="贴边收起"
            @pointerdown.stop
            @click.stop="dockToEdge"
          >
            <svg viewBox="0 0 24 24" width="14" height="14">
              <path
                fill="currentColor"
                d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"
              />
            </svg>
          </button>
        </div>
      </div>

      <div class="health-card-body">
        <template v-if="loading && !healthData">
          <div class="health-loading">
            <span class="health-spinner"></span>
            <span>加载中...</span>
          </div>
        </template>
        <template v-else-if="error && !healthData">
          <div class="health-error">
            <span>数据加载失败</span>
            <button class="health-retry-btn" type="button" @click="refresh">重试</button>
          </div>
        </template>
        <template v-else-if="healthData && Object.keys(healthData).length === 0">
          <div class="health-empty">暂无数据源信息</div>
        </template>
        <template v-else>
          <div
            v-for="(info, key) in sortedHealthData"
            :key="key"
            :class="['health-source-row', info.status]"
          >
            <div class="health-source-main">
              <span :class="['health-source-dot', info.status]"></span>
              <span
                class="health-source-name"
                :class="{ 'has-error': info.last_error && info.status !== 'healthy' }"
                @mouseenter="showTooltip($event, info)"
                @mouseleave="hideTooltip"
                >{{ sourceLabel(info.name) }}</span
              >
              <span
                v-if="info.status !== 'healthy' && info.fail_count > 0"
                class="health-source-fail"
                >{{ info.fail_count }}次</span
              >
              <span :class="['health-source-status', info.status]">{{
                statusLabel(info.status)
              }}</span>
            </div>
          </div>
        </template>
      </div>

      <div class="health-card-footer">
        <span class="health-footer-text">5分钟自动恢复探测</span>
      </div>
    </div>

    <!-- Error tooltip (teleported to body) -->
    <Teleport to="body">
      <Transition name="tooltip">
        <div v-if="tooltip.visible" class="health-tooltip" :style="tooltip.style">
          <div class="health-tooltip-body">{{ tooltip.error }}</div>
          <div v-if="tooltip.failCount > 1" class="health-tooltip-meta">
            连续失败 {{ tooltip.failCount }} 次
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
/** 数据源健康状态小组件，展示各数据源延迟和可用性 */
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { getMarketHealth, type SourceHealthInfo } from '@/features/market/api/market'

defineOptions({ name: 'HealthWidget' })

interface Props {
  /** Idle timeout in seconds before auto-docking. 0 = disabled. Default: 30 */
  idleTimeout?: number
  /** Distance in px from screen edge when docked. Default: 4 */
  edgeOffset?: number
}

const props = withDefaults(defineProps<Props>(), {
  idleTimeout: 30,
  edgeOffset: 4,
})

const STORAGE_KEY = 'health-widget-pos'
const STORAGE_EDGE_KEY = 'health-widget-edge'
const CARD_WIDTH = 240

// Tooltip state
const tooltip = ref({
  visible: false,
  source: '',
  status: '',
  statusText: '',
  error: '',
  failCount: 0,
  style: {} as Record<string, string>,
})
const ANIM_DURATION = 250
const EXPAND_MARGIN = 12 // min distance from screen edge when expanded

const widgetRef = ref<HTMLElement | null>(null)
const isExpanded = ref(false)
const loading = ref(false)
const error = ref(false)
const healthData = ref<Record<string, SourceHealthInfo> | null>(null)
const isAnimating = ref(false)
const isEdgeMode = ref(false)

// Reactive viewport size for computed style recalculation on resize
const viewportW = ref(window.innerWidth)
const viewportH = ref(window.innerHeight)

// Position using left/top for consistent 4-direction support
let cachedPillWidth = 0
const isDragging = ref(false)
const pos = ref({ left: 0, top: 0 }) // left/top in px

let healthTimer: ReturnType<typeof setInterval> | null = null
let dragStartClientX = 0
let dragStartClientY = 0
let dragStartPosX = 0
let dragStartPosY = 0
let currentAnim: Animation | null = null
let idleTimer: ReturnType<typeof setTimeout> | null = null

const widgetStyle = computed(() => {
  if (isEdgeMode.value) {
    const eo = props.edgeOffset
    return { left: `${eo}px`, top: `${pos.value.top}px` }
  }
  return {
    left: `${pos.value.left}px`,
    top: `${pos.value.top}px`,
  }
})

const statusOrder: Record<string, number> = { healthy: 0, degraded: 1, unhealthy: 2 }

const sortedHealthData = computed(() => {
  if (!healthData.value) return null
  const entries = Object.entries(healthData.value)
  entries.sort((a, b) => (statusOrder[a[1].status] ?? 9) - (statusOrder[b[1].status] ?? 9))
  return Object.fromEntries(entries)
})

const updateTime = computed(() => {
  if (!healthData.value) return ''
  const entries = Object.values(healthData.value)
  if (entries.length === 0) return ''
  // Find the most recent non-zero last_check
  let latest = ''
  for (const e of entries) {
    if (e.last_check && e.last_check !== '0001-01-01T00:00:00Z' && e.last_check > latest) {
      latest = e.last_check
    }
  }
  if (!latest) return ''
  try {
    const d = new Date(latest)
    return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
  } catch {
    return ''
  }
})

// === Expand position calculation ===

function calcSafeExpandPosition(
  startLeft: number,
  startTop: number,
): { left: number; top: number } {
  const vw = viewportW.value
  const margin = EXPAND_MARGIN
  const cardW = CARD_WIDTH

  let left = startLeft
  let top = startTop

  // Clamp horizontal: card must fit with margin on both sides
  if (left + cardW + margin > vw) left = Math.max(margin, vw - cardW - margin)
  if (left < margin) left = margin

  // Clamp vertical top
  if (top < margin) top = margin

  return { left, top }
}

// === Edge snap logic ===

function animateToEdge() {
  const el = widgetRef.value
  if (!el) return
  isAnimating.value = true

  const rect = el.getBoundingClientRect()
  const startLeft = rect.left
  const startTop = rect.top

  const eo = props.edgeOffset
  const targetLeft = eo

  if (currentAnim) currentAnim.cancel()
  currentAnim = el.animate(
    [
      { left: `${startLeft}px`, top: `${startTop}px` },
      { left: `${targetLeft}px`, top: `${startTop}px` },
    ],
    {
      duration: ANIM_DURATION,
      easing: 'cubic-bezier(0.4, 0, 0.2, 1)',
    },
  )

  currentAnim.onfinish = () => {
    currentAnim?.cancel()
    currentAnim = null
    // Update pos to the final left/top so widgetStyle is consistent
    pos.value = { left: targetLeft, top: startTop }
    isEdgeMode.value = true
    isAnimating.value = false
    saveEdgeState()
  }
}

// === Expand / Collapse animation ===

function measurePillWidth(): number {
  const el = widgetRef.value
  if (!el) return 40
  if (el.classList.contains('expanded') || isAnimating.value) return cachedPillWidth || 40
  const rect = el.getBoundingClientRect()
  cachedPillWidth = rect.width
  return cachedPillWidth
}

function animateExpand() {
  const el = widgetRef.value
  if (!el || isAnimating.value) return
  isAnimating.value = true

  // Capture current visual position before any state changes
  const rect = el.getBoundingClientRect()
  const startLeft = rect.left
  const startTop = rect.top

  // Exit edge mode and sync pos to current visual position (prevents jump)
  isEdgeMode.value = false
  pos.value = { left: startLeft, top: startTop }

  // Calculate safe position for expanded card
  const targetPos = calcSafeExpandPosition(startLeft, startTop)

  const pillWidth = measurePillWidth()
  const pill = el.querySelector('.health-pill-content') as HTMLElement
  const card = el.querySelector('.health-card-content') as HTMLElement

  el.style.width = `${pillWidth}px`
  el.style.borderRadius = 'var(--radius-lg)'
  el.style.overflow = 'hidden'
  if (pill) pill.style.opacity = '1'
  if (card) {
    card.style.opacity = '0'
    card.style.display = 'none'
  }

  if (currentAnim) currentAnim.cancel()
  currentAnim = el.animate(
    [
      {
        width: `${pillWidth}px`,
        borderRadius: 'var(--radius-lg)',
        boxShadow: '0 4px 16px color-mix(in srgb, var(--color-text-primary) 30%, transparent)',
        left: `${startLeft}px`,
        top: `${startTop}px`,
      },
      {
        width: `${CARD_WIDTH}px`,
        borderRadius: 'var(--radius-md)',
        boxShadow: '0 8px 32px color-mix(in srgb, var(--color-text-primary) 40%, transparent)',
        left: `${targetPos.left}px`,
        top: `${targetPos.top}px`,
      },
    ],
    {
      duration: ANIM_DURATION,
      easing: 'cubic-bezier(0.4, 0, 0.2, 1)',
    },
  )

  setTimeout(() => {
    if (pill) pill.animate([{ opacity: 1 }, { opacity: 0 }], { duration: 80 })
  }, ANIM_DURATION * 0.25)

  setTimeout(() => {
    if (card) {
      card.style.display = ''
      card.animate([{ opacity: 0 }, { opacity: 1 }], { duration: 120 })
    }
    el.classList.add('expanded')
  }, ANIM_DURATION * 0.55)

  currentAnim.onfinish = () => {
    currentAnim?.cancel()
    currentAnim = null
    el.style.width = ''
    el.style.borderRadius = ''
    el.style.overflow = ''
    if (pill) {
      pill.style.opacity = ''
      pill.style.display = 'none'
    }
    if (card) card.style.opacity = ''

    // Update pos to target position
    pos.value = { left: targetPos.left, top: targetPos.top }

    // Post-expansion: adjust vertical if card overflows bottom
    requestAnimationFrame(() => {
      const expandedRect = el.getBoundingClientRect()
      let adjustedTop = targetPos.top
      if (expandedRect.bottom + EXPAND_MARGIN > viewportH.value) {
        adjustedTop = Math.max(EXPAND_MARGIN, viewportH.value - expandedRect.height - EXPAND_MARGIN)
      }
      if (adjustedTop !== targetPos.top) {
        pos.value = { left: targetPos.left, top: adjustedTop }
      }
      savePosition()
    })

    isAnimating.value = false
  }
}

function animateCollapse() {
  const el = widgetRef.value
  if (!el || isAnimating.value) return
  isAnimating.value = true

  const pillWidth = cachedPillWidth || 40
  const pill = el.querySelector('.health-pill-content') as HTMLElement
  const card = el.querySelector('.health-card-content') as HTMLElement

  el.style.width = `${CARD_WIDTH}px`
  el.style.borderRadius = 'var(--radius-md)'
  el.style.overflow = 'hidden'
  if (card) card.style.opacity = '1'
  if (pill) {
    pill.style.display = ''
    pill.style.opacity = '0'
  }

  if (currentAnim) currentAnim.cancel()
  currentAnim = el.animate(
    [
      {
        width: `${CARD_WIDTH}px`,
        borderRadius: 'var(--radius-md)',
        boxShadow: '0 8px 32px color-mix(in srgb, var(--color-text-primary) 40%, transparent)',
      },
      {
        width: `${pillWidth}px`,
        borderRadius: 'var(--radius-lg)',
        boxShadow: '0 4px 16px color-mix(in srgb, var(--color-text-primary) 30%, transparent)',
      },
    ],
    {
      duration: ANIM_DURATION,
      easing: 'cubic-bezier(0.4, 0, 0.2, 1)',
    },
  )

  setTimeout(() => {
    if (card) card.animate([{ opacity: 1 }, { opacity: 0 }], { duration: 80 })
  }, ANIM_DURATION * 0.15)

  setTimeout(() => {
    if (pill) pill.animate([{ opacity: 0 }, { opacity: 1 }], { duration: 100 })
  }, ANIM_DURATION * 0.65)

  currentAnim.onfinish = () => {
    currentAnim?.cancel()
    currentAnim = null
    el.classList.remove('expanded')
    el.style.width = ''
    el.style.borderRadius = ''
    el.style.overflow = ''
    if (pill) {
      pill.style.opacity = ''
      pill.style.display = ''
    }
    if (card) {
      card.style.opacity = ''
      card.style.display = 'none'
    }
    isAnimating.value = false
  }
}

watch(isExpanded, (newVal) => {
  if (newVal) animateExpand()
  else animateCollapse()
})

// === Utility functions ===

function loadPosition() {
  try {
    const saved = localStorage.getItem(STORAGE_KEY)
    if (saved) {
      const p = JSON.parse(saved)
      pos.value = { left: p.left ?? viewportW.value - 60, top: p.top ?? viewportH.value - 180 }
      return
    }
  } catch {
    /* ignore */
  }
  // Default: bottom-right
  pos.value = { left: viewportW.value - 60, top: viewportH.value - 180 }
}

function savePosition() {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(pos.value))
  } catch {
    /* ignore */
  }
}

function loadEdgeState() {
  try {
    const saved = localStorage.getItem(STORAGE_EDGE_KEY)
    if (saved) {
      const state = JSON.parse(saved)
      isEdgeMode.value = state.isEdge ?? false
      // Always left edge now; ignore saved side
    }
  } catch {
    /* ignore */
  }
}

function saveEdgeState() {
  try {
    localStorage.setItem(
      STORAGE_EDGE_KEY,
      JSON.stringify({
        isEdge: isEdgeMode.value,
        side: 'left',
      }),
    )
  } catch {
    /* ignore */
  }
}

async function fetchHealth() {
  loading.value = true
  error.value = false
  try {
    const res = await getMarketHealth()
    healthData.value = res.data?.sources ?? null
  } catch {
    error.value = true
  } finally {
    loading.value = false
  }
}

function refresh() {
  fetchHealth()
  resetIdleTimer()
}

function toggleExpand() {
  if (!isAnimating.value) {
    isExpanded.value = !isExpanded.value
    resetIdleTimer()
  }
}

function dockToEdge() {
  // Collapse then snap to nearest edge
  if (isExpanded.value) {
    isExpanded.value = false
    setTimeout(() => animateToEdge(), ANIM_DURATION + 50)
  } else {
    animateToEdge()
  }
}

// === Idle auto-dock timer ===

function resetIdleTimer() {
  if (idleTimer) clearTimeout(idleTimer)
  if (props.idleTimeout <= 0) return // disabled
  if (isEdgeMode.value && !isExpanded.value) return // already docked, no need
  idleTimer = setTimeout(() => {
    // Only auto-dock if not expanded and not already in edge mode
    if (!isExpanded.value && !isEdgeMode.value && !isAnimating.value) {
      animateToEdge()
    }
  }, props.idleTimeout * 1000)
}

function clearIdleTimer() {
  if (idleTimer) {
    clearTimeout(idleTimer)
    idleTimer = null
  }
}

function onDragStart(e: MouseEvent | TouchEvent) {
  if (isAnimating.value) return
  if (isExpanded.value && e.target instanceof HTMLElement && e.target.closest('.health-card-body'))
    return
  e.preventDefault()

  // Exit edge mode on drag
  if (isEdgeMode.value) {
    isEdgeMode.value = false
    const el = widgetRef.value
    if (el) {
      const rect = el.getBoundingClientRect()
      pos.value = { left: rect.left, top: rect.top }
    }
  }

  const clientX = 'touches' in e ? e.touches[0].clientX : e.clientX
  const clientY = 'touches' in e ? e.touches[0].clientY : e.clientY
  dragStartClientX = clientX
  dragStartClientY = clientY
  dragStartPosX = pos.value.left
  dragStartPosY = pos.value.top
  isDragging.value = true

  const moveHandler = (ev: MouseEvent | TouchEvent) => {
    if (!isDragging.value) return
    ev.preventDefault()
    const cx = 'touches' in ev ? ev.touches[0].clientX : ev.clientX
    const cy = 'touches' in ev ? ev.touches[0].clientY : ev.clientY
    pos.value = {
      left: Math.max(0, Math.min(viewportW.value - 40, dragStartPosX + (cx - dragStartClientX))),
      top: Math.max(0, Math.min(viewportH.value - 40, dragStartPosY + (cy - dragStartClientY))),
    }
  }

  const endHandler = () => {
    isDragging.value = false
    savePosition()
    resetIdleTimer()
    document.removeEventListener('mousemove', moveHandler)
    document.removeEventListener('mouseup', endHandler)
    document.removeEventListener('touchmove', moveHandler)
    document.removeEventListener('touchend', endHandler)
  }

  document.addEventListener('mousemove', moveHandler)
  document.addEventListener('mouseup', endHandler)
  document.addEventListener('touchmove', moveHandler, { passive: false })
  document.addEventListener('touchend', endHandler)
}

function sourceLabel(name: string): string {
  switch (name) {
    case 'tencent':
      return '腾讯行情'
    case 'tdx':
      return '通达信'
    case 'biyingapi':
      return '必盈'
    case 'eastmoney':
      return '东方财富'
    case 'ths':
      return '同花顺'
    case 'sina':
      return '新浪财经'
    case 'akshare':
      return 'AkShare'
    default:
      return name
  }
}

function statusLabel(status: string): string {
  switch (status) {
    case 'healthy':
      return '正常'
    case 'degraded':
      return '降级'
    case 'unhealthy':
      return '不可用'
    default:
      return status
  }
}

// Tooltip positioning
function showTooltip(e: MouseEvent, info: SourceHealthInfo) {
  if (!info.last_error || info.status === 'healthy') return
  const el = e.currentTarget as HTMLElement
  const rect = el.getBoundingClientRect()
  const gap = 8

  // Position below the name element, left-aligned
  let left = rect.left
  let top = rect.bottom + gap

  // Clamp to viewport
  const maxW = 260
  if (left + maxW > window.innerWidth - 12) {
    left = Math.max(12, window.innerWidth - maxW - 12)
  }
  if (top + 120 > window.innerHeight) {
    top = rect.top - gap - 80 // flip above
  }

  tooltip.value = {
    visible: true,
    source: sourceLabel(info.name),
    status: info.status,
    statusText: statusLabel(info.status),
    error: info.last_error,
    failCount: info.fail_count,
    style: {
      left: `${left}px`,
      top: `${top}px`,
    },
  }
}

function hideTooltip() {
  tooltip.value.visible = false
}

function formatTooltip(info: SourceHealthInfo): string {
  const label = sourceLabel(info.name)
  const status = statusLabel(info.status)
  return `${label}: ${status}${info.last_error ? ' - ' + info.last_error : ''}`
}

function clampPosition() {
  const el = widgetRef.value
  const w = el?.offsetWidth ?? 40
  const h = el?.offsetHeight ?? 40
  let { left, top } = pos.value
  left = Math.max(0, Math.min(viewportW.value - w, left))
  top = Math.max(0, Math.min(viewportH.value - h, top))
  pos.value = { left, top }
}

function onResize() {
  viewportW.value = window.innerWidth
  viewportH.value = window.innerHeight
  clampPosition()
}

onMounted(() => {
  loadEdgeState()
  loadPosition()
  clampPosition()
  fetchHealth()
  healthTimer = setInterval(fetchHealth, 30000)
  window.addEventListener('resize', onResize)
  requestAnimationFrame(() => {
    measurePillWidth()
  })
  resetIdleTimer()
})

onUnmounted(() => {
  if (healthTimer) {
    clearInterval(healthTimer)
    healthTimer = null
  }
  clearIdleTimer()
  window.removeEventListener('resize', onResize)
})
</script>

<style scoped>
/* === Widget container === */
.health-widget {
  position: fixed;
  z-index: var(--z-dock, 70);
  user-select: none;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--color-border);
  border-radius: 20px;
  background: var(--color-bg-topbar);
  backdrop-filter: blur(20px) saturate(180%);
  -webkit-backdrop-filter: blur(20px) saturate(180%);
  box-shadow: 0 4px 16px color-mix(in srgb, var(--color-text-primary) 30%, transparent);
  cursor: pointer;
  transition: box-shadow 0.25s ease-in-out;
}

.health-widget.expanded {
  width: 240px;
  border-radius: var(--radius-lg, 20px);
  box-shadow: 0 8px 32px color-mix(in srgb, var(--color-text-primary) 40%, transparent);
  cursor: default;
  overflow: hidden;
}

.health-widget.dragging {
  z-index: 9999;
  box-shadow: 0 12px 40px color-mix(in srgb, var(--color-text-primary) 30%, transparent);
}

/* Edge mode: slim vertical strip */
.health-widget.edge-mode {
  border-radius: 10px;
  box-shadow: 0 2px 8px color-mix(in srgb, var(--color-text-primary) 20%, transparent);
  transition:
    box-shadow 0.25s ease-in-out,
    border-radius 0.25s ease-in-out;
}

.health-widget.edge-mode:hover {
  box-shadow: 0 4px 16px color-mix(in srgb, var(--color-text-primary) 30%, transparent);
}

/* === Pill content (vertical layout) === */
.health-pill-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 8px 6px;
}

.health-widget.expanded .health-pill-content {
  display: none;
}

.pill-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
  transition:
    background-color 0.3s,
    box-shadow 0.3s;
}

.pill-dot.healthy {
  background-color: var(--color-success);
  box-shadow: 0 0 4px color-mix(in srgb, var(--color-success) 40%, transparent);
}
.pill-dot.degraded {
  background-color: var(--color-warning);
  box-shadow: 0 0 4px color-mix(in srgb, var(--color-warning) 40%, transparent);
}
.pill-dot.unhealthy {
  background-color: var(--color-danger);
  box-shadow: 0 0 4px color-mix(in srgb, var(--color-danger) 40%, transparent);
}

/* Hover effect on pill */
.health-widget:not(.expanded):not(.edge-mode):hover {
  transform: translateY(-1px);
  box-shadow: 0 6px 20px color-mix(in srgb, var(--color-text-primary) 25%, transparent);
  border-color: var(--color-brand-muted, color-mix(in srgb, var(--color-brand) 30%, transparent));
}

/* === Card content === */
.health-card-content {
  display: none;
}

.health-widget.expanded .health-card-content {
  display: block;
}

.health-card-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid var(--color-border-light);
  cursor: grab;
}

.health-card-head:active {
  cursor: grabbing;
}

.health-card-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.health-card-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.health-card-time {
  font-size: 11px;
  color: var(--color-text-disabled);
  margin-right: 4px;
}

.health-card-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  padding: 0;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--color-text-disabled);
  cursor: pointer;
  transition:
    background-color 0.15s,
    color 0.15s;
}

.health-card-btn:hover {
  background: var(--color-bg-hover);
  color: var(--color-text-primary);
}

.health-card-btn.close:hover {
  background: color-mix(in srgb, var(--color-danger) 15%, transparent);
  color: var(--color-danger);
}

.health-card-btn:focus-visible {
  outline: 2px solid var(--color-brand);
  outline-offset: 2px;
}

.health-card-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.spinning {
  animation: spin 0.8s linear infinite;
}
@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

/* Card body */
.health-card-body {
  padding: 4px 0;
  min-height: 80px;
}

.health-source-row {
  padding: 6px 12px;
  border-bottom: 1px solid var(--color-border-light);
  animation: row-fade-in 0.25s ease-in-out both;
}

.health-source-row:nth-child(1) {
  animation-delay: 0.06s;
}
.health-source-row:nth-child(2) {
  animation-delay: 0.1s;
}
.health-source-row:nth-child(3) {
  animation-delay: 0.14s;
}

@keyframes row-fade-in {
  from {
    opacity: 0;
    transform: translateY(6px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.health-source-row:last-child {
  border-bottom: none;
}

/* Horizontal layout for each source row */
.health-source-main {
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 6px;
}

.health-source-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
}

.health-source-dot.healthy {
  background-color: var(--color-success);
  box-shadow: 0 0 6px color-mix(in srgb, var(--color-success) 40%, transparent);
}
.health-source-dot.degraded {
  background-color: var(--color-warning);
  box-shadow: 0 0 6px color-mix(in srgb, var(--color-warning) 40%, transparent);
}
.health-source-dot.unhealthy {
  background-color: var(--color-danger);
  box-shadow: 0 0 6px color-mix(in srgb, var(--color-danger) 40%, transparent);
}

.health-source-name {
  font-size: 12px;
  font-weight: 500;
  color: var(--color-text-primary);
  flex: 1;
  min-width: 0;
}

.health-source-name.has-error {
  cursor: help;
  border-bottom: 1px dashed color-mix(in srgb, var(--color-text-primary) 15%, transparent);
  transition: border-color 0.2s;
}

.health-source-name.has-error:hover {
  border-bottom-color: color-mix(in srgb, var(--color-text-primary) 35%, transparent);
}

.health-source-status {
  font-size: 10px;
  padding: 1px 6px;
  border-radius: 8px;
  font-weight: 500;
  flex-shrink: 0;
}

.health-source-status.healthy {
  background: color-mix(in srgb, var(--color-success) 12%, transparent);
  color: var(--color-success);
}
.health-source-status.degraded {
  background: color-mix(in srgb, var(--color-warning) 12%, transparent);
  color: var(--color-warning);
}
.health-source-status.unhealthy {
  background: color-mix(in srgb, var(--color-danger) 12%, transparent);
  color: var(--color-danger);
}

.health-source-fail {
  font-size: 10px;
  color: var(--color-text-disabled);
  flex-shrink: 0;
}

/* Loading / Error / Empty */
.health-loading,
.health-error,
.health-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 24px 16px;
  color: var(--color-text-secondary);
  font-size: 13px;
}

.health-spinner {
  width: 16px;
  height: 16px;
  border: 2px solid var(--color-border);
  border-top-color: var(--color-brand);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.health-retry-btn {
  padding: 2px 10px;
  border: 1px solid var(--color-border);
  border-radius: 6px;
  background: transparent;
  color: var(--color-brand);
  font-size: 12px;
  cursor: pointer;
  transition: background-color 0.15s;
}

.health-retry-btn:hover {
  background: var(--color-bg-hover);
}

/* Footer */
.health-card-footer {
  padding: 6px 16px 8px;
  border-top: 1px solid var(--color-border-light);
  text-align: center;
}

.health-footer-text {
  font-size: 10px;
  color: var(--color-text-disabled);
}

/* === Reduced motion === */
@media (prefers-reduced-motion: reduce) {
  .health-source-row {
    animation-duration: 0.01ms !important;
    animation-delay: 0s !important;
  }
  .spinning,
  .health-spinner {
    animation-duration: 0.01ms !important;
  }
}
</style>

<!-- Unscoped styles for teleported tooltip -->
<style>
.health-tooltip {
  position: fixed;
  z-index: 9999;
  width: max-content;
  max-width: 260px;
  border-radius: 8px;
  background: color-mix(in srgb, var(--color-bg-card) 94%, transparent);
  border: 1px solid var(--color-border-light);
  box-shadow:
    inset 0 1px 0 color-mix(in srgb, var(--color-text-primary) 5%, transparent),
    0 8px 24px color-mix(in srgb, var(--color-text-primary) 55%, transparent),
    0 0 0 1px color-mix(in srgb, var(--color-text-primary) 10%, transparent);
  backdrop-filter: blur(16px) saturate(140%);
  -webkit-backdrop-filter: blur(16px) saturate(140%);
  overflow: hidden;
}

.health-tooltip-body {
  padding: 8px 10px;
  font-size: 11px;
  line-height: 1.45;
  color: var(--color-text-secondary);
  font-family: 'SF Mono', 'Cascadia Code', 'Fira Code', monospace;
  word-break: break-all;
}

.health-tooltip-meta {
  padding: 5px 10px 7px;
  border-top: 1px solid var(--color-border-light);
  font-size: 10px;
  color: var(--color-text-disabled);
}

/* Tooltip transition */
.tooltip-enter-active {
  transition:
    opacity 0.15s ease-out,
    transform 0.15s ease-out;
}
.tooltip-leave-active {
  transition:
    opacity 0.1s ease-in,
    transform 0.1s ease-in;
}
.tooltip-enter-from {
  opacity: 0;
  transform: translateY(-4px);
}
.tooltip-leave-to {
  opacity: 0;
  transform: translateY(-2px);
}

@media (prefers-reduced-motion: reduce) {
  .tooltip-enter-active,
  .tooltip-leave-active {
    transition-duration: 0.01ms !important;
  }
}
</style>
