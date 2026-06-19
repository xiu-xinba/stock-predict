<script setup lang="ts">
/** 资产头部组件，展示资产名称、代码、实时报价、涨跌幅及自选操作 */
import { computed } from 'vue'
import { getDirection } from '@/shared/utils/format'

defineOptions({ name: 'AssetHeader' })

interface BadgeItem {
  text: string
  type: 'primary' | 'secondary'
}

const props = withDefaults(
  defineProps<{
    name: string
    code: string
    price: string | number
    change: string | number
    changePercent: string | number
    isUp: boolean
    infoItems: Array<{ label: string; value: string }>
    isInWatchlist: boolean
    watchlistLoading: boolean
    gridColumns?: number
    badges?: BadgeItem[]
    liveDotTitle?: string
  }>(),
  {
    gridColumns: 3,
    badges: () => [],
    liveDotTitle: '实时行情',
  },
)

const emit = defineEmits<{
  toggleWatchlist: []
}>()

const dirClass = computed(() => getDirection(Number(props.changePercent)))
</script>

<template>
  <section class="asset-header card card-container card-spotlight fade-slide-up" style="--delay: 0">
    <div class="header-top">
      <div class="asset-identity">
        <h1 class="asset-name">{{ name }}</h1>
        <div class="asset-meta">
          <span class="asset-code">{{ code }}</span>
          <span
            v-for="(badge, idx) in badges"
            :key="idx"
            :class="badge.type === 'primary' ? 'type-badge' : 'secondary-badge'"
            >{{ badge.text }}</span
          >
        </div>
      </div>
      <div class="header-actions">
        <button
          type="button"
          class="icon-btn"
          :class="{ active: isInWatchlist }"
          :title="isInWatchlist ? '移出自选' : '加入自选'"
          @click="emit('toggleWatchlist')"
        >
          <svg
            width="20"
            height="20"
            viewBox="0 0 24 24"
            :fill="isInWatchlist ? 'var(--color-brand)' : 'none'"
            stroke="currentColor"
            stroke-width="2"
          >
            <path
              d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"
            />
          </svg>
        </button>
        <slot name="actions" />
      </div>
    </div>

    <div class="quote-row" :class="dirClass">
      <span class="price-value">{{ price }}</span>
      <span class="change-pct pct-badge">
        {{ change }}
      </span>
      <span class="live-dot" :title="liveDotTitle"></span>
    </div>

    <div class="info-grid" :style="{ gridTemplateColumns: `repeat(${gridColumns}, 1fr)` }">
      <div v-for="(item, idx) in infoItems" :key="idx" class="kv-item">
        <span class="kv-label">{{ item.label }}</span>
        <span class="kv-value">{{ item.value }}</span>
      </div>
    </div>
  </section>
</template>

<style scoped>
.fade-slide-up {
  animation: fade-slide-up 0.5s var(--ease-out-expo) both;
  animation-delay: calc(var(--delay, 0) * 60ms);
}

.asset-header {
  padding: var(--sp-4);
}

.header-top {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: var(--sp-3);
  margin-bottom: var(--sp-4);
}

.asset-identity {
  flex: 1;
  min-width: 0;
}

.asset-name {
  font-size: var(--fs-2xl);
  font-weight: var(--fw-bold);
  color: var(--color-text-primary);
  margin: 0 0 var(--sp-1) 0;
  line-height: var(--lh-tight);
}

.asset-meta {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  flex-wrap: wrap;
}

.asset-code {
  font-size: var(--fs-sm);
  color: var(--color-text-tertiary);
  font-family: var(--font-mono, monospace);
}

.type-badge {
  font-size: var(--fs-xs);
  padding: 2px var(--sp-2);
  border-radius: var(--radius-full);
  background: var(--color-brand);
  color: var(--color-brand-contrast);
  opacity: 0.85;
}

.secondary-badge {
  font-size: var(--fs-xs);
  padding: 2px var(--sp-2);
  border-radius: var(--radius-full);
  background: var(--color-bg-elevated);
  color: var(--color-text-secondary);
}

.header-actions {
  display: flex;
  gap: var(--sp-2);
  align-items: center;
  flex-shrink: 0;
}

.icon-btn {
  transition: transform var(--transition-fast);
}

.icon-btn.active {
  color: var(--color-brand);
}

.icon-btn:hover svg {
  transform: scale(1.1);
  transition: transform var(--transition-fast);
}

.icon-btn:active {
  transform: scale(0.9);
}

.icon-btn:focus-visible {
  outline: 2px solid var(--color-brand);
  outline-offset: 2px;
  border-radius: var(--radius-sm);
}

.quote-row {
  display: flex;
  align-items: baseline;
  gap: var(--sp-3);
  margin-bottom: var(--sp-6);
  padding: var(--sp-2) var(--sp-3);
  border-radius: var(--radius-md);
  background: linear-gradient(135deg, var(--color-bg-elevated) 0%, transparent 100%);
}

.price-value {
  font-size: var(--fs-5xl);
  font-weight: var(--fw-bold);
  font-family: var(--font-mono);
  line-height: var(--lh-tight);
}

.change-pct {
  font-size: var(--fs-lg);
  font-weight: var(--fw-semibold);
  font-family: var(--font-mono);
}

.text-up .price-value,
.text-up .change-pct {
  color: var(--color-up);
}

.text-down .price-value,
.text-down .change-pct {
  color: var(--color-down);
}

.text-flat .price-value,
.text-flat .change-pct {
  color: var(--color-flat);
}

.info-grid {
  display: grid;
  gap: var(--sp-3);
}

.kv-item {
  position: relative;
}

.kv-item + .kv-item::before {
  content: '';
  position: absolute;
  left: calc(-1 * var(--sp-3) / 2);
  top: 15%;
  height: 70%;
  width: 1px;
  background: var(--color-border);
}

@media (max-width: 768px) {
  .info-grid {
    grid-template-columns: repeat(2, 1fr) !important;
  }
  .price-value {
    font-size: var(--fs-2xl);
  }
  .kv-item + .kv-item::before {
    display: none;
  }
}
</style>
