<template>
  <div class="watchlist-page">
    <section class="page-head">
      <div>
        <p class="page-kicker">Watchlist</p>
        <h1 class="page-title">自选监控</h1>
        <p class="page-desc">关注基金实时涨跌与净值状态</p>
      </div>
      <div v-if="store.items.length > 0" class="head-actions">
        <span v-if="store.lastRefresh" class="refresh-time">{{ store.lastRefresh }}</span>
        <span v-else class="refresh-time">同步中</span>
        <button class="icon-btn" :class="{ spinning: store.loading }" type="button" @click="handleRefresh" title="刷新数据">
          <svg viewBox="0 0 1024 1024" aria-hidden="true"><path fill="currentColor" d="M771.776 794.88A384 384 0 0 1 128 512h64a320 320 0 0 0 555.712 216.448H654.72a32 32 0 1 1 0-64h149.44a32 32 0 0 1 32 32v148.16a32 32 0 1 1-64 0v-50.048zM296.064 229.12A384 384 0 0 1 896 512h-64a320 320 0 0 0-555.712-216.448h72.832a32 32 0 0 1 0 64H199.04a32 32 0 0 1-32-32V179.52a32 32 0 0 1 64 0v49.6z"/></svg>
        </button>
      </div>
    </section>

    <section class="command-panel">
      <WatchlistAdd />
    </section>

    <template v-if="store.items.length > 0">
      <section class="metrics-strip" aria-label="自选统计">
        <div class="metric-cell">
          <span class="metric-label">Total</span>
          <strong>{{ store.items.length }}</strong>
        </div>
        <div class="metric-cell up">
          <span class="metric-label">Up</span>
          <strong>{{ upCount }}</strong>
        </div>
        <div class="metric-cell down">
          <span class="metric-label">Down</span>
          <strong>{{ downCount }}</strong>
        </div>
        <div class="metric-cell flat">
          <span class="metric-label">Flat</span>
          <strong>{{ flatCount }}</strong>
        </div>
      </section>

      <section class="toolbar">
        <span class="toolbar-label">排序</span>
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
      </section>

      <section class="fund-table" aria-label="自选基金列表">
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
            @click="goToPredict(item.fund_code)"
            @keydown.enter="goToPredict(item.fund_code)"
          >
            <div class="fund-main">
              <span class="fund-code">{{ item.fund_code }}</span>
              <h3 class="fund-name">{{ item.fund_name }}</h3>
            </div>
            <span class="fund-type">{{ item.fund_type || '--' }}</span>
            <span class="fund-nav">{{ item.estimated_nav != null ? item.estimated_nav.toFixed(4) : '--' }}</span>
            <span :class="['fund-change', getChangeClass(item.change_pct)]">
              <template v-if="item.change_pct > 0">▲</template>
              <template v-else-if="item.change_pct < 0">▼</template>
              {{ item.change_pct != null ? (item.change_pct > 0 ? '+' : '') + item.change_pct.toFixed(2) : '--' }}%
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
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useWatchlistStore } from '@/stores/watchlist'
import WatchlistAdd from '@/components/watchlist/WatchlistAdd.vue'
import WatchlistEmpty from '@/components/watchlist/WatchlistEmpty.vue'

const store = useWatchlistStore()
const router = useRouter()

let refreshTimer: ReturnType<typeof setInterval> | null = null

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
  if (pct == null) return 'flat'
  if (pct > 0) return 'up'
  if (pct < 0) return 'down'
  return 'flat'
}

function goToPredict(fundCode: string) {
  router.push(`/predict/${fundCode}`)
}

function handleRemove(fundCode: string) {
  store.removeItem(fundCode)
  ElMessage.success('已从自选移除')
}

function handleRefresh() {
  if (store.loading) return
  store.refreshQuotes()
}

onMounted(() => {
  store.refreshQuotes()
  refreshTimer = setInterval(() => {
    if (!store.loading) store.refreshQuotes()
  }, 30000)
})

onUnmounted(() => {
  if (refreshTimer) {
    clearInterval(refreshTimer)
    refreshTimer = null
  }
})
</script>

<style scoped>
.watchlist-page {
  display: flex;
  flex-direction: column;
  gap: var(--sp-4);
}

.page-head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: var(--sp-4);
  padding: var(--sp-2) 0 var(--sp-1);
}

.page-kicker {
  margin: 0 0 var(--sp-1);
  color: var(--color-brand);
  font-size: var(--fs-xs);
  font-weight: var(--fw-bold);
  line-height: var(--lh-tight);
}

.page-title {
  margin: 0;
  color: var(--color-text-primary);
  font-size: var(--fs-3xl);
  font-weight: var(--fw-extrabold);
  line-height: var(--lh-snug);
}

.page-desc {
  margin: var(--sp-1) 0 0;
  color: var(--color-text-secondary);
  font-size: var(--fs-sm);
}

.head-actions {
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

.icon-btn,
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

.icon-btn svg,
.remove-btn svg {
  width: 15px;
  height: 15px;
}

.icon-btn:hover {
  color: var(--color-brand);
  background: var(--color-brand-soft);
  border-color: var(--color-brand-muted);
}

.icon-btn.spinning svg {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.command-panel {
  padding: var(--sp-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-bg-card);
}

.metrics-strip {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  overflow: hidden;
  background: var(--color-bg-card);
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

.metric-cell.up strong { color: var(--color-up); }
.metric-cell.down strong { color: var(--color-down); }
.metric-cell.flat strong { color: var(--color-flat); }

.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--sp-3);
}

.toolbar-label {
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
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
  gap: 2px;
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

.fund-table {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  overflow: hidden;
  background: var(--color-bg-card);
}

.table-head,
.fund-row {
  display: grid;
  grid-template-columns: minmax(220px, 1.5fr) minmax(100px, 0.7fr) minmax(96px, 0.6fr) minmax(96px, 0.6fr) 44px;
  align-items: center;
  gap: var(--sp-3);
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
  transition: background-color var(--transition-fast);
}

.fund-row::before {
  content: '';
  position: absolute;
  top: var(--sp-2);
  bottom: var(--sp-2);
  left: 0;
  width: 3px;
  background: var(--color-flat);
}

.fund-row.up::before { background: var(--color-up); }
.fund-row.down::before { background: var(--color-down); }
.fund-row:last-child { border-bottom: 0; }
.fund-row:hover { background: var(--color-bg-hover); }

.fund-main {
  min-width: 0;
}

.fund-code {
  display: block;
  margin-bottom: 2px;
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
}

.fund-change {
  font-size: var(--fs-base);
  font-weight: var(--fw-bold);
}

.fund-change.up { color: var(--color-up); }
.fund-change.down { color: var(--color-down); }
.fund-change.flat { color: var(--color-flat); }

.remove-btn:hover {
  color: var(--color-up);
  background: var(--color-up-bg);
  border-color: var(--color-up-border);
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

@media (max-width: 760px) {
  .page-head,
  .toolbar {
    align-items: flex-start;
    flex-direction: column;
  }

  .head-actions,
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
</style>
