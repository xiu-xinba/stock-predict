<template>
  <div class="sector-heat card card-spotlight" @mousemove="handleSpotlight">
    <!-- Header -->
    <div class="sector-head">
      <div class="sector-title-group">
        <span class="sector-eyebrow">SECTOR MAP</span>
        <h3 class="sector-title">行业板块</h3>
      </div>
    </div>

    <template v-if="loading">
      <div class="sector-body">
        <div class="sector-col sector-col--down">
          <div v-for="i in 5" :key="'ls-' + i" class="sector-row sector-skeleton sector-row--down">
            <span class="sector-pct skeleton-pulse" style="width: 52px"></span>
            <div class="sector-bar-wrap"><div class="sector-bar"></div></div>
            <span class="sector-name skeleton-pulse" style="width: 56px"></span>
          </div>
        </div>
        <div class="sector-col sector-col--up">
          <div v-for="i in 5" :key="'gs-' + i" class="sector-row sector-skeleton sector-row--up">
            <span class="sector-name skeleton-pulse" style="width: 56px"></span>
            <div class="sector-bar-wrap"><div class="sector-bar"></div></div>
            <span class="sector-pct skeleton-pulse" style="width: 52px"></span>
          </div>
        </div>
      </div>
    </template>
    <template v-else-if="error">
      <div class="sector-state error">{{ error }}</div>
    </template>
    <template v-else-if="sectors.length > 0">
      <div class="sector-body">
        <!-- 领跌 (左列：pct → bar← → name) -->
        <div class="sector-col sector-col--down">
          <div
            v-for="(sector, i) in losers"
            :key="'l-' + sector.name"
            class="sector-row sector-row--down"
            :style="{ '--i': i, '--pct': Math.min(Math.abs(sector.change_pct) / 3, 1) }"
          >
            <span class="sector-pct text-down">{{ sector.change_pct.toFixed(2) }}%</span>
            <div class="sector-bar-wrap">
              <div class="sector-bar bar-down"></div>
            </div>
            <span class="sector-name">{{ sector.name }}</span>
          </div>
        </div>
        <!-- 领涨 (右列：name → bar→ → pct) -->
        <div class="sector-col sector-col--up">
          <div
            v-for="(sector, i) in gainers"
            :key="'g-' + sector.name"
            class="sector-row sector-row--up"
            :style="{ '--i': i, '--pct': Math.min(Math.abs(sector.change_pct) / 5, 1) }"
          >
            <span class="sector-name">{{ sector.name }}</span>
            <div class="sector-bar-wrap">
              <div class="sector-bar bar-up"></div>
            </div>
            <span class="sector-pct text-up">+{{ sector.change_pct.toFixed(2) }}%</span>
          </div>
        </div>
      </div>
    </template>
    <div v-else class="empty-hint">
      <svg
        class="empty-icon"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="1.5"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        <rect x="3" y="3" width="7" height="7" rx="1" />
        <rect x="14" y="3" width="7" height="7" rx="1" />
        <rect x="3" y="14" width="7" height="7" rx="1" />
        <rect x="14" y="14" width="7" height="7" rx="1" />
      </svg>
      <span>暂无板块数据</span>
    </div>
  </div>
</template>

<script setup lang="ts">
/** 行业板块热力图组件，蝴蝶对称布局展示涨跌分布 */
import { computed } from 'vue'
import type { MarketSectorItem } from '@/features/market/types'

defineOptions({ name: 'SectorHeat' })

const props = defineProps<{
  sectors: MarketSectorItem[]
  loading?: boolean
  error?: string | null
}>()

const gainers = computed(() => props.sectors.filter((s) => s.change_pct >= 0).slice(0, 5))
const losers = computed(() => props.sectors.filter((s) => s.change_pct < 0).slice(0, 5))

/** 卡片聚光灯效果：追踪鼠标位置 */
function handleSpotlight(e: MouseEvent) {
  const el = e.currentTarget as HTMLElement
  const rect = el.getBoundingClientRect()
  el.style.setProperty('--mouse-x', `${e.clientX - rect.left}px`)
  el.style.setProperty('--mouse-y', `${e.clientY - rect.top}px`)
}
</script>

<style scoped>
/* ── Card Shell ── */
.sector-heat {
  padding: var(--sp-4) var(--sp-5);
  border-radius: var(--radius-lg);
  overflow: hidden;
  flex: 1;
  display: flex;
  flex-direction: column;
}

/* ── Header ── */
.sector-head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: var(--sp-3);
  margin-bottom: var(--sp-3);
}

.sector-title-group {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.sector-eyebrow {
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--color-brand-muted);
  font-family: var(--font-mono);
}

.sector-title {
  margin: 0;
  font-size: var(--fs-lg);
  font-weight: var(--fw-bold);
  color: var(--color-text-primary);
  letter-spacing: -0.01em;
}

/* ── Body: Butterfly Grid ── */
.sector-body {
  display: grid;
  grid-template-columns: 1fr 1fr;
  border-radius: 12px;
  overflow: hidden;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border-light);
  flex: 1;
}

.sector-col {
  display: flex;
  flex-direction: column;
  flex: 1;
}

.sector-col--down {
  border-right: 1px solid var(--color-border-light);
}

/* ── Row ── */
.sector-row {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  padding: 0 var(--sp-3);
  flex: 1;
  min-height: 36px;
  cursor: default;
  transition:
    background 0.25s cubic-bezier(0.32, 0.72, 0, 1),
    box-shadow 0.25s cubic-bezier(0.32, 0.72, 0, 1);
  position: relative;

  /* Stagger entry */
  opacity: 0;
  transform: translateY(3px);
  animation: row-reveal 0.4s cubic-bezier(0.16, 1, 0.3, 1) calc(var(--i) * 35ms) forwards;
}

.sector-row:hover {
  background: var(--color-bg-hover);
}

.sector-row + .sector-row {
  border-top: 1px solid var(--color-border-light);
}

.sector-row:active {
  background: color-mix(in srgb, var(--color-text-primary) 4%, var(--color-bg-hover));
}

/* ── Bar (diverging) ── */
.sector-bar-wrap {
  flex: 1;
  height: 8px;
  border-radius: 4px;
  background: color-mix(in srgb, var(--color-text-primary) 4%, transparent);
  overflow: hidden;
  min-width: 32px;
}

.sector-bar {
  height: 100%;
  border-radius: 4px;
  width: 0;
  animation: bar-grow 0.6s cubic-bezier(0.16, 1, 0.3, 1) calc(var(--i) * 35ms + 0.08s) forwards;
  transition: filter 0.25s ease;
}

.sector-row:hover .sector-bar {
  filter: brightness(1.2);
}

.bar-up {
  margin-left: auto;
  background: linear-gradient(
    90deg,
    color-mix(in srgb, var(--color-up) 30%, transparent),
    var(--color-up)
  );
}

.bar-down {
  margin-right: auto;
  background: linear-gradient(
    270deg,
    color-mix(in srgb, var(--color-down) 30%, transparent),
    var(--color-down)
  );
}

/* ── Name ── */
.sector-name {
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
  color: var(--color-text-primary);
  line-height: var(--lh-snug);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex-shrink: 0;
  max-width: 72px;
}

/* ── Pct ── */
.sector-pct {
  font-size: var(--fs-xs);
  font-weight: var(--fw-bold);
  font-family: var(--font-mono);
  line-height: var(--lh-snug);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
  flex-shrink: 0;
  padding: 2px 7px;
  border-radius: 6px;
  transition:
    box-shadow 0.25s cubic-bezier(0.32, 0.72, 0, 1),
    transform 0.25s cubic-bezier(0.32, 0.72, 0, 1);
}

.sector-row--up:hover .sector-pct {
  transform: translateX(-1px);
}

.sector-row--down:hover .sector-pct {
  transform: translateX(1px);
}

.text-up {
  color: var(--color-up);
  background: var(--color-up-bg);
}

.text-down {
  color: var(--color-down);
  background: var(--color-down-bg);
}

/* ── Keyframes ── */
@keyframes row-reveal {
  from {
    opacity: 0;
    transform: translateY(3px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes bar-grow {
  to {
    width: calc(var(--pct) * 100%);
  }
}

/* ── Reduced motion ── */
@media (prefers-reduced-motion: reduce) {
  .sector-row {
    opacity: 1;
    transform: none;
    animation: none;
  }
  .sector-bar {
    width: calc(var(--pct) * 100%);
    animation: none;
  }
  .sector-row:hover {
    transform: none;
  }
  .sector-row--up:hover .sector-pct,
  .sector-row--down:hover .sector-pct {
    transform: none;
  }
}

/* ── Skeleton & State ── */
.sector-skeleton {
  opacity: 1;
  transform: none;
  animation: none;
}
.sector-skeleton .sector-bar {
  animation: none;
  width: 0;
}
.sector-state {
  min-height: 54px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--sp-8);
  color: var(--color-text-secondary);
  font-size: var(--fs-sm);
}
.sector-state.error {
  color: var(--color-warning);
}

.empty-hint {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--sp-3);
  padding: var(--sp-12) 0;
  color: var(--color-text-tertiary);
  font-size: var(--fs-sm);
}

.empty-icon {
  width: 40px;
  height: 40px;
  opacity: 0.3;
  color: var(--color-text-tertiary);
}

/* ── Mobile ── */
@media (max-width: 480px) {
  .sector-heat {
    padding: var(--sp-3) var(--sp-4);
  }

  .sector-body {
    grid-template-columns: 1fr;
  }

  .sector-col--down {
    border-right: none;
    border-bottom: 1px solid var(--color-border-light);
  }

  /* 移动端领跌列恢复左→右布局 */
  .sector-row--down {
    flex-direction: row;
  }

  .sector-head {
    flex-direction: column;
    align-items: flex-start;
    gap: var(--sp-2);
  }
}
</style>
