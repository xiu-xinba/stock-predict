<template>
  <div class="watchlist-page card-container">
    <div class="tab-bar fade-slide-up" style="--delay: 0">
      <div class="tab-group">
        <button
          :class="['tab-btn', { active: activeTab === 'fund' }]"
          type="button"
          @click="activeTab = 'fund'"
        >
          基金
        </button>
        <button
          :class="['tab-btn', { active: activeTab === 'stock' }]"
          type="button"
          @click="activeTab = 'stock'"
        >
          股票
        </button>
      </div>
    </div>

    <template v-if="activeTab === 'fund'">
      <div v-if="store.items.length > 0" class="toolbar fade-slide-up" style="--delay: 2">
        <div class="toolbar-left">
          <span class="live-dot"></span>
          <span v-if="store.lastRefresh" class="refresh-time">{{ store.lastRefresh }}</span>
          <span v-else class="refresh-time">同步中</span>
        </div>
        <div class="tab-group sort-group">
          <button
            v-for="opt in sortOptions"
            :key="opt.value"
            :class="['tab-btn', 'sort-btn', { active: store.sortBy === opt.value }]"
            type="button"
            @click="store.setSort(opt.value as typeof store.sortBy)"
          >
            {{ opt.label }}
            <svg
              v-if="store.sortBy === opt.value"
              class="sort-dir"
              :class="{ asc: store.sortOrder === 'asc' }"
              viewBox="0 0 1024 1024"
              aria-hidden="true"
            >
              <path fill="currentColor" d="M384 192v640l-320-320zm256 0v640l320-320z" />
            </svg>
          </button>
        </div>
      </div>

      <SkeletonTable
        v-if="store.loading && !store.lastRefresh && store.items.length > 0"
        :row-count="5"
      />

      <ErrorState
        v-else-if="store.error && store.items.length > 0"
        :message="store.error"
        compact
      />

      <template v-else-if="store.items.length > 0">
        <section
          class="metrics-strip card-glass fade-slide-up"
          style="--delay: 3"
          aria-label="自选统计"
        >
          <div class="metrics-strip-gradient" aria-hidden="true"></div>
          <div class="metric-cell metric-cell--total">
            <span class="metric-label">Total</span>
            <strong>{{ store.items.length }}</strong>
          </div>
          <div class="metric-cell metric-cell--up">
            <span class="metric-label">Up</span>
            <strong>{{ upCount }}</strong>
          </div>
          <div class="metric-cell metric-cell--down">
            <span class="metric-label">Down</span>
            <strong>{{ downCount }}</strong>
          </div>
          <div class="metric-cell metric-cell--flat">
            <span class="metric-label">→ 持平</span>
            <strong>{{ flatCount }}</strong>
          </div>
        </section>

        <section
          class="fund-table card card-spotlight fade-slide-up"
          style="--delay: 4"
          aria-label="自选基金列表"
        >
          <div class="table-head">
            <span>基金</span>
            <span>类型</span>
            <span>估算净值</span>
            <span>涨跌幅</span>
            <span></span>
          </div>
          <transition-group name="row" tag="div" class="table-body">
            <article
              v-for="item in store.sortedItems"
              :key="item.fund_code"
              :class="['fund-row', item.direction]"
              tabindex="0"
              @click="goToFund(item.fund_code)"
              @keydown.enter="goToFund(item.fund_code)"
            >
              <div class="fund-main">
                <span class="fund-code">{{ item.fund_code }}</span>
                <h3 class="fund-name">{{ item.fund_name }}</h3>
              </div>
              <span class="fund-type">{{ item.fund_type || '--' }}</span>
              <span class="fund-nav">{{
                hasQuote(item) ? item.estimated_nav.toFixed(4) : '--'
              }}</span>
              <span :class="['fund-change', getChangeClass(item.change_pct)]">
                <template v-if="hasQuote(item) && item.change_pct > 0">▲</template>
                <template v-else-if="hasQuote(item) && item.change_pct < 0">▼</template>
                {{
                  hasQuote(item)
                    ? (item.change_pct > 0 ? '+' : '') + item.change_pct.toFixed(2) + '%'
                    : '--'
                }}
              </span>
              <button
                class="remove-btn"
                aria-label="移除自选"
                title="移除自选"
                type="button"
                @click.stop="handleRemove(item.fund_code)"
              >
                <svg viewBox="0 0 1024 1024" aria-hidden="true">
                  <path
                    fill="currentColor"
                    d="M512 64a448 448 0 1 1 0 896 448 448 0 0 1 0-896zM288 512a38.4 38.4 0 0 0 0 76.8h448a38.4 38.4 0 0 0 0-76.8H288z"
                  />
                </svg>
              </button>
            </article>
          </transition-group>
        </section>
      </template>

      <WatchlistEmpty v-else @search="openSearch" />
    </template>

    <template v-if="activeTab === 'stock'">
      <div v-if="stockStore.stockItems.length > 0" class="toolbar fade-slide-up" style="--delay: 2">
        <div class="toolbar-left">
          <span class="live-dot"></span>
          <span v-if="stockStore.lastRefresh" class="refresh-time">{{
            stockStore.lastRefresh
          }}</span>
          <span v-else class="refresh-time">同步中</span>
        </div>
      </div>

      <SkeletonTable
        v-if="stockStore.loading && !stockStore.lastRefresh && stockStore.stockItems.length > 0"
        :row-count="5"
      />

      <ErrorState
        v-else-if="stockStore.error && stockStore.stockItems.length > 0"
        :message="stockStore.error"
        compact
      />

      <section
        v-else-if="stockStore.stockItems.length > 0"
        class="fund-table card card-spotlight fade-slide-up"
        style="--delay: 3"
        aria-label="自选股票列表"
      >
        <div class="table-head stock-table-head">
          <span>股票</span>
          <span>行业</span>
          <span>价格</span>
          <span>涨跌幅</span>
          <span></span>
        </div>
        <transition-group name="row" tag="div" class="table-body">
          <article
            v-for="item in stockStore.stockItems"
            :key="item.stock_code"
            :class="['fund-row', getDirection(item.change_pct)]"
            tabindex="0"
            @click="goToStock(item.stock_code)"
            @keydown.enter="goToStock(item.stock_code)"
          >
            <div class="fund-main">
              <span class="fund-code">{{ item.stock_code }}</span>
              <h3 class="fund-name">{{ item.stock_name }}</h3>
            </div>
            <span class="fund-type">{{ item.industry || '--' }}</span>
            <span class="fund-nav">{{
              item.current_price ? item.current_price.toFixed(2) : '--'
            }}</span>
            <span :class="['fund-change', getChangeClass(item.change_pct)]">
              <template v-if="item.change_pct > 0">▲</template>
              <template v-else-if="item.change_pct < 0">▼</template>
              {{
                Math.abs(item.change_pct) > 0.005
                  ? (item.change_pct > 0 ? '+' : '') + item.change_pct.toFixed(2) + '%'
                  : '--'
              }}
            </span>
            <button
              class="remove-btn"
              aria-label="移除自选"
              title="移除自选"
              type="button"
              @click.stop="handleRemoveStock(item.stock_code)"
            >
              <svg viewBox="0 0 1024 1024" aria-hidden="true">
                <path
                  fill="currentColor"
                  d="M512 64a448 448 0 1 1 0 896 448 448 0 0 1 0-896zM288 512a38.4 38.4 0 0 0 0 76.8h448a38.4 38.4 0 0 0 0-76.8H288z"
                />
              </svg>
            </button>
          </article>
        </transition-group>
      </section>

      <WatchlistEmpty v-else @search="openSearch" />
    </template>
  </div>
</template>

<script setup lang="ts">
/** 自选列表页面，支持基金/股票 Tab 切换、排序、实时行情刷新 */
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useWatchlistStore } from '@/features/watchlist'
import { useStockWatchlistStore } from '@/features/watchlist'
import { useStaggerEntry } from '@/shared/composables/useStaggerEntry'
import { getDirection } from '@/shared/utils/format'
import ErrorState from '@/shared/components/ErrorState.vue'
import SkeletonTable from '@/shared/components/SkeletonTable.vue'
import WatchlistEmpty from '@/features/watchlist/components/WatchlistEmpty.vue'
import type { WatchlistItem } from '@/features/watchlist/types'

defineOptions({ name: 'WatchlistView' })

useStaggerEntry('.fund-row', { staggerMs: 40, translateY: 8 })

const store = useWatchlistStore()
const stockStore = useStockWatchlistStore()
const router = useRouter()
const route = useRoute()

const activeTab = ref<'fund' | 'stock'>(
  (route.query.tab === 'stock' ? 'stock' : 'fund') as 'fund' | 'stock',
)

watch(activeTab, (tab) => {
  router.replace({ query: { ...route.query, tab } })
})

const sortOptions = [
  { label: '时间', value: 'added_at' },
  { label: '涨跌幅', value: 'change_pct' },
  { label: '净值', value: 'estimated_nav' },
  { label: '名称', value: 'fund_name' },
]

const upCount = computed(() => store.directionCounts.up)
const downCount = computed(() => store.directionCounts.down)
const flatCount = computed(() => store.directionCounts.flat)

function getChangeClass(pct: number | null | undefined): string {
  return getDirection(pct)
}

function hasQuote(item: WatchlistItem): boolean {
  return Boolean(item.quote_source)
}

function goToFund(fundCode: string) {
  router.push(`/fund/${fundCode}`)
}

function goToStock(stockCode: string) {
  router.push(`/stock/${stockCode}`)
}

function openSearch() {
  document.dispatchEvent(new KeyboardEvent('keydown', { key: '/' }))
}

function handleRemove(fundCode: string) {
  store.removeItem(fundCode)
  ElMessage.success('已从自选移除')
}

function handleRemoveStock(stockCode: string) {
  stockStore.removeStockItem(stockCode)
  ElMessage.success('已从自选移除')
}

onMounted(() => {
  store.refreshQuotes()
  stockStore.refreshStockQuotes()
})
</script>

<style scoped>
.watchlist-page {
  display: flex;
  flex-direction: column;
  gap: var(--sp-8);
}

/* ── Tab Bar ── */
.tab-bar {
  display: flex;
  align-items: center;
}

/* ── Toolbar ── */
.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--sp-3);
}

.toolbar-left {
  display: inline-flex;
  align-items: center;
  gap: var(--sp-2);
  padding: var(--sp-1);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-bg-card);
}

.refresh-time {
  padding-left: var(--sp-2);
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
}

/* ── Sort Group (uses global tab-group + tab-btn) ── */
.sort-group {
  flex-wrap: wrap;
}

.sort-btn {
  gap: var(--sp-0_5);
}

.sort-dir {
  width: 11px;
  height: 11px;
  transition: transform var(--transition-fast);
}

.sort-dir.asc {
  transform: rotate(180deg);
}

/* ── Metrics Strip ── */
.metrics-strip {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  overflow: hidden;
  position: relative;
  padding: var(--sp-5) var(--sp-6);
}

.metrics-strip-gradient {
  position: absolute;
  inset: 0;
  background: linear-gradient(
    135deg,
    var(--color-brand-soft) 0%,
    transparent 50%,
    var(--color-up-bg) 100%
  );
  pointer-events: none;
  border-radius: inherit;
  animation: gradient-shift 8s ease infinite alternate;
}

html.dark .metrics-strip-gradient {
  background: linear-gradient(
    135deg,
    var(--color-brand-light) 0%,
    transparent 50%,
    var(--color-up-bg) 100%
  );
}

@keyframes gradient-shift {
  0% {
    background-position: 0% 50%;
  }
  100% {
    background-position: 100% 50%;
  }
}

.metric-cell {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: var(--sp-2);
  padding: var(--sp-3) var(--sp-4);
  border-right: 1px solid var(--color-border-light);
  border-left: 2px solid transparent;
}

.metric-cell:last-child {
  border-right: 0;
}

.metric-cell--total {
  border-left-color: var(--color-brand);
}

.metric-cell--up {
  border-left-color: var(--color-up);
}

.metric-cell--down {
  border-left-color: var(--color-down);
}

.metric-cell--flat {
  border-left-color: var(--color-flat);
}

.metric-label {
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
}

.metric-cell strong {
  color: var(--color-text-primary);
  font-size: var(--fs-4xl);
  line-height: var(--lh-tight);
}

.metric-cell--up strong {
  color: var(--color-up);
}

.metric-cell--down strong {
  color: var(--color-down);
}

.metric-cell--flat strong {
  color: var(--color-flat);
}

/* ── Fund Table ── */
.fund-table {
  overflow: hidden;
  position: relative;
  border-radius: var(--radius-lg);
}

.fund-table::before {
  display: none;
}

.table-head,
.fund-row {
  display: grid;
  grid-template-columns:
    minmax(220px, 1.5fr) minmax(100px, 0.7fr) minmax(96px, 0.6fr) minmax(96px, 0.6fr)
    44px;
  align-items: center;
  gap: var(--sp-3);
}

.stock-table-head,
.stock-table-head + .table-body .fund-row {
  grid-template-columns:
    minmax(220px, 1.5fr) minmax(100px, 0.7fr) minmax(96px, 0.6fr) minmax(96px, 0.6fr)
    44px;
}

.table-head {
  min-height: 36px;
  padding: 0 var(--sp-4);
  border-bottom: 1px solid var(--color-border);
  background: var(--color-bg-hover);
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
}

.table-body {
  display: flex;
  flex-direction: column;
}

.fund-row {
  position: relative;
  min-height: 64px;
  padding: var(--sp-2) var(--sp-4) var(--sp-2) calc(var(--sp-4) + 2px);
  border-bottom: 1px solid var(--color-border-light);
  cursor: pointer;
  transition:
    background-color var(--transition-fast),
    transform 0.25s var(--ease-out-quart),
    box-shadow 0.25s var(--ease-out-quart);
}

.fund-row::before {
  content: '';
  position: absolute;
  top: 8px;
  bottom: 8px;
  left: 0;
  width: 2px;
  border-radius: 0 1px 1px 0;
  background: var(--color-flat);
  transition: background var(--transition-fast);
}

.fund-row.up::before {
  background: var(--color-up);
}
.fund-row.down::before {
  background: var(--color-down);
}
.fund-row:last-child {
  border-bottom: 0;
}
.fund-row:hover {
  background: var(--color-bg-hover);
  box-shadow: inset 2px 0 0 var(--color-brand);
  border-left: 2px solid var(--color-brand);
  transform: scale(1.005);
}

.fund-row:active {
  transform: scale(0.998);
}

.fund-row:focus-visible {
  outline: 2px solid var(--color-brand);
  outline-offset: -2px;
}

.fund-main {
  min-width: 0;
}

.fund-code {
  display: block;
  margin-bottom: var(--sp-0_5);
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
}

.fund-name {
  margin: 0;
  overflow: hidden;
  color: var(--color-text-primary);
  font-size: var(--fs-base);
  font-weight: var(--fw-semibold);
  line-height: var(--lh-snug);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.fund-type {
  overflow: hidden;
  color: var(--color-text-secondary);
  font-size: var(--fs-sm);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.fund-nav {
  color: var(--color-text-primary);
  font-size: var(--fs-base);
  font-weight: var(--fw-semibold);
  font-family: var(--font-mono);
}

.fund-change {
  font-size: var(--fs-base);
  font-weight: var(--fw-bold);
}

.fund-change.text-up {
  color: var(--color-up);
}
.fund-change.text-down {
  color: var(--color-down);
}
.fund-change.text-flat {
  color: var(--color-flat);
}

.remove-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: 1px solid transparent;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition:
    background-color var(--transition-fast),
    border-color var(--transition-fast),
    color var(--transition-fast);
}

.remove-btn svg {
  width: 15px;
  height: 15px;
}

.remove-btn:hover {
  color: var(--color-up);
  background: var(--color-up-bg);
  border-color: var(--color-up-border);
}

.remove-btn:active {
  transform: scale(0.9);
}

.remove-btn:focus-visible {
  outline: 2px solid var(--color-brand);
  outline-offset: 2px;
}

/* ── Row Transition ── */
.row-enter-active,
.row-leave-active,
.row-move {
  transition:
    opacity var(--transition-normal),
    transform var(--transition-normal);
}

.row-enter-from {
  opacity: 0;
  transform: translateY(8px);
}

.row-leave-to {
  opacity: 0;
  transform: translateX(24px);
}

/* ── Responsive: Tablet ── */
@media (max-width: 768px) {
  .toolbar {
    align-items: flex-start;
    flex-direction: column;
  }

  .toolbar-left,
  .sort-group {
    width: 100%;
  }

  .metrics-strip {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .metric-cell:nth-child(2) {
    border-right: 0;
  }

  .metric-cell:nth-child(-n + 2) {
    border-bottom: 1px solid var(--color-border-light);
  }

  .table-head {
    display: none;
  }

  .fund-row {
    grid-template-columns: minmax(0, 1fr) auto;
    gap: var(--sp-2);
    min-height: 78px;
  }

  .fund-main {
    grid-row: 1 / span 2;
  }

  .fund-type,
  .fund-nav {
    display: none;
  }

  .fund-change {
    grid-column: 2;
    grid-row: 1;
    justify-self: end;
  }

  .remove-btn {
    grid-column: 2;
    grid-row: 2;
    justify-self: end;
  }
}

/* ── Responsive: Small phone ── */
@media (max-width: 480px) {
  .watchlist-page {
    gap: var(--sp-3);
  }

  .metric-cell strong {
    font-size: var(--fs-xl);
  }

  .metric-cell {
    padding: var(--sp-2) var(--sp-3);
  }
}

@media (prefers-reduced-motion: reduce) {
  .fund-row,
  .metrics-strip,
  .fund-table,
  .fade-slide-up {
    transition-duration: 0.01ms !important;
    animation-duration: 0.01ms !important;
  }
}
</style>
