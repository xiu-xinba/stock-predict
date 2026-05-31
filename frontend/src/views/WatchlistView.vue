<template>
  <div class="watchlist-page card-container">
    <div class="tab-bar">
      <div class="tab-group">
        <button
          :class="['tab-btn', { active: activeTab === 'fund' }]"
          type="button"
          @click="activeTab = 'fund'"
        >基金</button>
        <button
          :class="['tab-btn', { active: activeTab === 'stock' }]"
          type="button"
          @click="activeTab = 'stock'"
        >股票</button>
      </div>
    </div>

    <template v-if="activeTab === 'fund'">
      <div v-if="store.items.length > 0" class="toolbar">
        <div class="toolbar-left">
          <span class="live-dot"></span>
          <span v-if="store.lastRefresh" class="refresh-time">{{ store.lastRefresh }}</span>
          <span v-else class="refresh-time">同步中</span>
        </div>
        <div class="sort-group">
          <button
            v-for="opt in sortOptions"
            :key="opt.value"
            :class="['sort-btn', { active: store.sortBy === opt.value }]"
            type="button"
            @click="store.setSort(opt.value as typeof store.sortBy)"
          >
            {{ opt.label }}
            <svg v-if="store.sortBy === opt.value" class="sort-dir" :class="{ asc: store.sortOrder === 'asc' }" viewBox="0 0 1024 1024" aria-hidden="true"><path fill="currentColor" d="M384 192v640l-320-320zm256 0v640l320-320z"/></svg>
          </button>
        </div>
      </div>

      <div v-if="store.loading && !store.lastRefresh && store.items.length > 0" class="skeleton-strip">
        <div v-for="i in 5" :key="i" class="skeleton-row">
          <div class="sk-cell sk-code skeleton-pulse"></div>
          <div class="sk-cell sk-name skeleton-pulse"></div>
          <div class="sk-cell sk-nav skeleton-pulse"></div>
          <div class="sk-cell sk-pct skeleton-pulse"></div>
        </div>
      </div>

      <ErrorState
        v-else-if="store.error && store.items.length > 0"
        :message="store.error"
        compact
      />

      <template v-else-if="store.items.length > 0">
        <section class="metrics-strip card card-accent-top" aria-label="自选统计">
          <div class="metric-cell">
            <span class="metric-label">Total</span>
            <strong>{{ store.items.length }}</strong>
          </div>
          <div class="metric-cell text-up">
            <span class="metric-label">Up</span>
            <strong>{{ upCount }}</strong>
          </div>
          <div class="metric-cell text-down">
            <span class="metric-label">Down</span>
            <strong>{{ downCount }}</strong>
          </div>
          <div class="metric-cell text-flat">
            <span class="metric-label">→ 持平</span>
            <strong>{{ flatCount }}</strong>
          </div>
        </section>

        <section class="fund-table card card-accent-top" aria-label="自选基金列表">
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
              <span class="fund-nav">{{ hasQuote(item) ? item.estimated_nav.toFixed(4) : '--' }}</span>
              <span :class="['fund-change', getChangeClass(item.change_pct)]">
                <template v-if="hasQuote(item) && item.change_pct > 0">▲</template>
                <template v-else-if="hasQuote(item) && item.change_pct < 0">▼</template>
                {{ hasQuote(item) ? (item.change_pct > 0 ? '+' : '') + item.change_pct.toFixed(2) + '%' : '--' }}
              </span>
              <button
                class="remove-btn"
                aria-label="移除自选"
                title="移除自选"
                type="button"
                @click.stop="handleRemove(item.fund_code)"
              >
                <svg viewBox="0 0 1024 1024" aria-hidden="true"><path fill="currentColor" d="M512 64a448 448 0 1 1 0 896 448 448 0 0 1 0-896zM288 512a38.4 38.4 0 0 0 0 76.8h448a38.4 38.4 0 0 0 0-76.8H288z"/></svg>
              </button>
            </article>
          </transition-group>
        </section>
      </template>

      <WatchlistEmpty v-else />
    </template>

    <template v-if="activeTab === 'stock'">
      <div v-if="stockLoading" class="skeleton-strip">
        <div v-for="i in 5" :key="i" class="skeleton-row">
          <div class="sk-cell sk-code skeleton-pulse"></div>
          <div class="sk-cell sk-name skeleton-pulse"></div>
          <div class="sk-cell sk-nav skeleton-pulse"></div>
          <div class="sk-cell sk-pct skeleton-pulse"></div>
        </div>
      </div>

      <ErrorState
        v-else-if="stockError"
        :message="stockError"
        compact
      />

      <section v-else-if="hotStocks.length > 0" class="fund-table card card-accent-top" aria-label="热门股票">
        <div class="table-head stock-table-head">
          <span>股票</span>
          <span>行业</span>
          <span>价格</span>
          <span>涨跌幅</span>
        </div>
        <div class="table-body">
          <article
            v-for="item in hotStocks"
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
            <span class="fund-nav">{{ item.current_price ? item.current_price.toFixed(2) : '--' }}</span>
            <span :class="['fund-change', getChangeClass(item.change_pct)]">
              <template v-if="item.change_pct > 0">▲</template>
              <template v-else-if="item.change_pct < 0">▼</template>
              {{ item.change_pct !== 0 ? (item.change_pct > 0 ? '+' : '') + item.change_pct.toFixed(2) + '%' : '--' }}
            </span>
          </article>
        </div>
      </section>

      <div v-else class="stock-empty">
        <svg viewBox="0 0 48 48" class="empty-svg" aria-hidden="true">
          <rect x="6" y="8" width="36" height="32" rx="4" fill="none" stroke="currentColor" stroke-width="1.5" opacity="0.3"/>
          <line x1="14" y1="18" x2="34" y2="18" stroke="currentColor" stroke-width="1.5" opacity="0.2" stroke-linecap="round"/>
          <line x1="14" y1="24" x2="28" y2="24" stroke="currentColor" stroke-width="1.5" opacity="0.2" stroke-linecap="round"/>
          <line x1="14" y1="30" x2="22" y2="30" stroke="currentColor" stroke-width="1.5" opacity="0.2" stroke-linecap="round"/>
          <circle cx="36" cy="14" r="6" fill="currentColor" opacity="0.08"/>
          <path d="M34 14l2 2 4-4" stroke="currentColor" stroke-width="1.5" fill="none" stroke-linecap="round" stroke-linejoin="round" opacity="0.4"/>
        </svg>
        <h3 class="empty-title">暂无热门股票</h3>
        <p class="empty-desc">点击右上角搜索图标，搜索股票后即可查看详情。</p>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useWatchlistStore } from '@/stores/watchlist'
import { useStaggerEntry } from '@/composables/useStaggerEntry'
import { getDirection } from '@/utils/format'
import { fetchStockList } from '@/api/stock'
import ErrorState from '@/components/ErrorState.vue'
import WatchlistEmpty from '@/components/watchlist/WatchlistEmpty.vue'
import type { WatchlistItem } from '@/types'
import type { StockItem } from '@/types/stock'

defineOptions({ name: 'WatchlistView' })

useStaggerEntry('.fund-row', { staggerMs: 40, translateY: 8 })

const store = useWatchlistStore()
const router = useRouter()
const route = useRoute()

const activeTab = ref<'fund' | 'stock'>((route.query.tab === 'stock' ? 'stock' : 'fund') as 'fund' | 'stock')
const hotStocks = ref<StockItem[]>([])
const stockLoading = ref(false)
const stockError = ref<string | null>(null)

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

function handleRemove(fundCode: string) {
  store.removeItem(fundCode)
  ElMessage.success('已从自选移除')
}

async function fetchHotStocks() {
  stockLoading.value = true
  stockError.value = null
  try {
    const res = await fetchStockList({ size: 20 })
    if (res.code === 0 && res.data) {
      hotStocks.value = res.data.items || []
    } else {
      stockError.value = res.message || '获取热门股票失败'
    }
  } catch {
    stockError.value = '网络异常，请稍后重试'
  } finally {
    stockLoading.value = false
  }
}

onMounted(() => {
  store.refreshQuotes()
  fetchHotStocks()
})
</script>

<style scoped>
.watchlist-page {
  display: flex;
  flex-direction: column;
  gap: var(--sp-4);
}

.tab-bar {
  display: flex;
  align-items: center;
}

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

.remove-btn svg {
  width: 15px;
  height: 15px;
}

.sort-group {
  display: inline-flex;
  flex-wrap: wrap;
  gap: var(--sp-1);
  padding: var(--sp-1);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-bg-card);
}

.sort-btn {
  display: inline-flex;
  align-items: center;
  gap: var(--sp-0_5);
  min-height: 30px;
  padding: 0 var(--sp-3);
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-regular);
  font-size: var(--fs-sm);
  cursor: pointer;
  transition: background-color var(--transition-fast), color var(--transition-fast), border-color var(--transition-fast);
}

.sort-btn:hover {
  color: var(--color-brand);
  background: var(--color-bg-hover);
}

.sort-btn.active {
  color: var(--color-brand);
  background: var(--color-brand-soft);
  border-color: var(--color-brand-muted);
  font-weight: var(--fw-semibold);
}

.sort-dir {
  width: 11px;
  height: 11px;
  transition: transform var(--transition-fast);
}

.sort-dir.asc {
  transform: rotate(180deg);
}

.skeleton-strip {
  display: flex;
  flex-direction: column;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  overflow: hidden;
  background: var(--color-bg-card);
}

.skeleton-row {
  display: grid;
  grid-template-columns: 72px minmax(0, 1fr) 80px 72px;
  align-items: center;
  gap: var(--sp-3);
  min-height: 64px;
  padding: var(--sp-2) var(--sp-4);
  border-bottom: 1px solid var(--color-border-light);
}

.skeleton-row:last-child {
  border-bottom: 0;
}

.sk-cell {
  height: 16px;
  border-radius: var(--radius-sm);
}

.sk-code { width: 56px; }
.sk-name { width: 60%; }
.sk-nav { width: 64px; }
.sk-pct { width: 52px; }

.metrics-strip {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  overflow: hidden;
  position: relative;
}

.metrics-strip::before {
  display: none;
}

.metric-cell {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: var(--sp-2);
  padding: var(--sp-3) var(--sp-4);
  border-right: 1px solid var(--color-border-light);
}

.metric-cell:last-child {
  border-right: 0;
}

.metric-label {
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
}

.metric-cell strong {
  color: var(--color-text-primary);
  font-size: var(--fs-xl);
  line-height: var(--lh-tight);
}

.fund-table {
  overflow: hidden;
  position: relative;
}

.fund-table::before {
  display: none;
}

.table-head,
.fund-row {
  display: grid;
  grid-template-columns: minmax(220px, 1.5fr) minmax(100px, 0.7fr) minmax(96px, 0.6fr) minmax(96px, 0.6fr) 44px;
  align-items: center;
  gap: var(--sp-3);
}

.stock-table-head,
.stock-table-head + .table-body .fund-row {
  grid-template-columns: minmax(220px, 1.5fr) minmax(100px, 0.7fr) minmax(96px, 0.6fr) minmax(96px, 0.6fr);
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
  padding: var(--sp-2) var(--sp-4);
  border-bottom: 1px solid var(--color-border-light);
  cursor: pointer;
  transition: background-color var(--transition-fast), transform 0.18s var(--ease-out-quart);
}

.fund-row::before {
  content: '';
  position: absolute;
  top: 0;
  bottom: 0;
  left: 0;
  width: 1px;
  background: var(--color-flat);
}

.fund-row.up::before { background: var(--color-up); }
.fund-row.down::before { background: var(--color-down); }
.fund-row:last-child { border-bottom: 0; }
.fund-row:hover { background: var(--color-bg-hover); box-shadow: inset 2px 0 0 var(--color-brand); }

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

.fund-change.text-up { color: var(--color-up); }
.fund-change.text-down { color: var(--color-down); }
.fund-change.text-flat { color: var(--color-flat); }

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
  transition: background-color var(--transition-fast), border-color var(--transition-fast), color var(--transition-fast);
}

.remove-btn:hover {
  color: var(--color-up);
  background: var(--color-up-bg);
  border-color: var(--color-up-border);
}

.stock-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 320px;
  padding: var(--sp-8) var(--sp-4);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-bg-card);
  text-align: center;
  color: var(--color-text-tertiary);
  font-size: var(--fs-sm);
}

.empty-svg {
  width: 48px;
  height: 48px;
  color: var(--color-text-secondary);
  margin-bottom: var(--sp-4);
}

.empty-title {
  margin: 0 0 var(--sp-2);
  color: var(--color-text-primary);
  font-size: var(--fs-md);
  font-weight: var(--fw-semibold);
}

.empty-desc {
  margin: 0;
  color: var(--color-text-secondary);
  font-size: var(--fs-sm);
  line-height: var(--lh-relaxed);
  max-width: 260px;
}

.row-enter-active,
.row-leave-active,
.row-move {
  transition: opacity var(--transition-normal), transform var(--transition-normal);
}

.row-enter-from {
  opacity: 0;
  transform: translateY(8px);
}

.row-leave-to {
  opacity: 0;
  transform: translateX(24px);
}

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

@media (prefers-reduced-motion: reduce) {
  .fund-row,
  .metrics-strip,
  .fund-table,
  .skeleton-strip,
  .sk-cell {
    transition-duration: 0.01ms !important;
    animation-duration: 0.01ms !important;
  }
}
</style>
