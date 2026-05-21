<template>
  <div class="watchlist-page">
    <section class="page-hero">
      <h1 class="page-title">我的自选</h1>
      <p class="page-desc">追踪关注的基金，实时掌握涨跌动态</p>
      <div v-if="store.items.length > 0" class="refresh-info">
        <span v-if="store.lastRefresh" class="refresh-time">更新于 {{ store.lastRefresh }}</span>
        <span v-else class="refresh-time">加载中...</span>
        <button class="refresh-btn" :class="{ spinning: store.loading }" @click="handleRefresh" title="刷新数据">
          <svg viewBox="0 0 1024 1024"><path fill="currentColor" d="M771.776 794.88A384 384 0 0 1 128 512h64a320 320 0 0 0 555.712 216.448H654.72a32 32 0 1 1 0-64h149.44a32 32 0 0 1 32 32v148.16a32 32 0 1 1-64 0v-50.048zM296.064 229.12A384 384 0 0 1 896 512h-64a320 320 0 0 0-555.712-216.448h72.832a32 32 0 0 1 0 64H199.04a32 32 0 0 1-32-32V179.52a32 32 0 0 1 64 0v49.6z"/></svg>
        </button>
      </div>
    </section>

    <section class="add-section">
      <WatchlistAdd />
    </section>

    <template v-if="store.items.length > 0">
      <section class="stats-section">
        <div class="stat-card total">
          <div class="stat-number">{{ store.items.length }}</div>
          <div class="stat-label">自选基金</div>
        </div>
        <div class="stat-card up">
          <div class="stat-number">{{ upCount }}</div>
          <div class="stat-label">上涨</div>
        </div>
        <div class="stat-card down">
          <div class="stat-number">{{ downCount }}</div>
          <div class="stat-label">下跌</div>
        </div>
        <div class="stat-card flat">
          <div class="stat-number">{{ flatCount }}</div>
          <div class="stat-label">平盘</div>
        </div>
      </section>

      <section class="toolbar">
        <div class="sort-group">
          <button
            v-for="opt in sortOptions"
            :key="opt.value"
            :class="['sort-btn', { active: store.sortBy === opt.value }]"
            @click="store.setSort(opt.value as typeof store.sortBy)"
          >
            {{ opt.label }}
            <svg v-if="store.sortBy === opt.value" class="sort-dir" :class="{ asc: store.sortOrder === 'asc' }" viewBox="0 0 1024 1024"><path fill="currentColor" d="M384 192v640l-320-320zm256 0v640l320-320z"/></svg>
          </button>
        </div>
      </section>

      <section class="fund-list">
        <transition-group name="card" tag="div" class="list-container">
          <article
            v-for="item in store.sortedItems"
            :key="item.fund_code"
            class="fund-card"
            tabindex="0"
            @click="goToPredict(item.fund_code)"
            @keydown.enter="goToPredict(item.fund_code)"
          >
            <div :class="['card-indicator', item.direction]"></div>
            <div class="card-body">
              <div class="card-row-top">
                <h3 class="card-name">{{ item.fund_name }}</h3>
                <span class="card-type">{{ item.fund_type }}</span>
              </div>
              <div class="card-row-bottom">
                <span class="card-code">{{ item.fund_code }}</span>
                <div class="card-figures">
                  <span class="card-nav">{{ item.estimated_nav != null ? item.estimated_nav.toFixed(4) : '--' }}</span>
                  <span :class="['card-change', getChangeClass(item.change_pct)]">
                    <template v-if="item.change_pct > 0">▲</template>
                    <template v-else-if="item.change_pct < 0">▼</template>
                    {{ item.change_pct != null ? (item.change_pct > 0 ? '+' : '') + item.change_pct.toFixed(2) : '--' }}%
                  </span>
                </div>
              </div>
            </div>
            <button
              class="card-remove"
              aria-label="移除自选"
              title="移除自选"
              @click.stop="handleRemove(item.fund_code)"
            >
              <svg viewBox="0 0 1024 1024"><path fill="currentColor" d="M512 64a448 448 0 1 1 0 896 448 448 0 0 1 0-896zM288 512a38.4 38.4 0 0 0 0 76.8h448a38.4 38.4 0 0 0 0-76.8H288z"/></svg>
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

/** 基于涨跌幅数值决定红涨绿跌颜色类（中国A股标准） */
function getChangeClass(pct: number | null | undefined): string {
  if (pct == null) return 'flat'
  if (pct > 0) return 'up'    // 红色 - 上涨
  if (pct < 0) return 'down'  // 绿色 - 下跌
  return 'flat'                // 灰色 - 平盘
}

function goToPredict(fundCode: string) {
  router.push(`/predict/${fundCode}`)
}

function handleRemove(fundCode: string) {
  store.removeItem(fundCode)
  ElMessage.success('已从自选移除')
}

function handleRefresh() {
  // Prevent rapid clicks while already loading
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
  padding: var(--sp-6) 0;
  max-width: 720px;
  margin: 0 auto;
}

.page-hero {
  text-align: center;
  padding: 40px 0 24px;
}
.page-title {
  font-size: var(--fs-4xl);
  font-weight: 700;
  color: var(--color-text-primary);
  margin-bottom: var(--sp-2);
  letter-spacing: -0.5px;
}
.page-desc {
  font-size: var(--fs-md);
  color: var(--color-text-secondary);
}

.refresh-info {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--sp-2);
  margin-top: var(--sp-2);
}
.refresh-time {
  font-size: var(--fs-xs);
  color: var(--color-text-secondary);
}
.refresh-btn {
  width: 24px;
  height: 24px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  transition: color 0.15s, background 0.15s;
}
.refresh-btn svg {
  width: 14px;
  height: 14px;
}
.refresh-btn:hover {
  color: var(--color-brand);
  background: var(--color-bg-hover);
}
.refresh-btn.spinning svg {
  animation: spin 1s linear infinite;
}
@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.add-section {
  margin-bottom: var(--sp-6);
}

.stats-section {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--sp-3);
  margin-bottom: var(--sp-5);
}
.stat-card {
  background: var(--color-bg-card);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: var(--sp-4);
  text-align: center;
  transition: transform 0.2s, box-shadow 0.2s;
}
.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}
.stat-number {
  font-size: var(--fs-2xl);
  font-weight: 700;
  color: var(--color-text-primary);
  line-height: 1.2;
}
.stat-card.up .stat-number { color: var(--color-up); }
.stat-card.down .stat-number { color: var(--color-down); }
.stat-card.flat .stat-number { color: var(--color-text-secondary); }
.stat-label {
  font-size: var(--fs-xs);
  color: var(--color-text-secondary);
  margin-top: var(--sp-1);
  font-weight: 500;
}

.toolbar {
  margin-bottom: var(--sp-4);
}
.sort-group {
  display: flex;
  gap: var(--sp-2);
  flex-wrap: wrap;
}
.sort-btn {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 5px 14px;
  border: 1px solid var(--color-border);
  border-radius: 9999px;
  background: var(--color-bg-card);
  color: var(--color-text-regular);
  font-size: var(--fs-sm);
  font-family: inherit;
  cursor: pointer;
  transition: all 0.15s;
  user-select: none;
  outline: none;
}
.sort-btn:hover {
  border-color: var(--color-brand);
  color: var(--color-brand);
}
.sort-btn.active {
  background: var(--color-brand-light);
  border-color: var(--color-brand);
  color: var(--color-brand);
  font-weight: 600;
}
.sort-dir {
  width: 12px;
  height: 12px;
  transition: transform 0.2s;
}
.sort-dir.asc {
  transform: rotate(180deg);
}

.fund-list {
  margin-bottom: var(--sp-6);
}
.list-container {
  display: flex;
  flex-direction: column;
  gap: var(--sp-3);
}

.fund-card {
  display: flex;
  align-items: center;
  background: var(--color-bg-card);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  overflow: hidden;
  cursor: pointer;
  transition: transform 0.2s, box-shadow 0.2s;
}
.fund-card:hover {
  transform: translateY(-1px);
  box-shadow: var(--shadow-md);
}
.fund-card:focus-visible {
  outline: 2px solid var(--color-brand);
  outline-offset: 2px;
}

.card-indicator {
  width: 4px;
  align-self: stretch;
  flex-shrink: 0;
}
.card-indicator.up { background: var(--color-up); }
.card-indicator.down { background: var(--color-down); }
.card-indicator.flat { background: var(--color-text-secondary); }

.card-body {
  flex: 1;
  min-width: 0;
  padding: var(--sp-4) var(--sp-4) var(--sp-4) var(--sp-5);
}
.card-row-top {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  margin-bottom: 4px;
}
.card-name {
  font-size: var(--fs-base);
  font-weight: 600;
  color: var(--color-text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  margin: 0;
}
.card-type {
  font-size: var(--fs-xs);
  padding: 1px 8px;
  border-radius: 4px;
  background: var(--color-bg-hover);
  color: var(--color-text-secondary);
  flex-shrink: 0;
  font-weight: 500;
}
.card-row-bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.card-code {
  font-size: var(--fs-sm);
  color: var(--color-text-regular);
}
.card-figures {
  display: flex;
  align-items: baseline;
  gap: var(--sp-3);
}
.card-nav {
  font-size: var(--fs-md);
  font-weight: 600;
  color: var(--color-text-primary);
}
.card-change {
  font-size: var(--fs-sm);
  font-weight: 600;
}
.card-change.up { color: var(--color-up); }
.card-change.down { color: var(--color-down); }
.card-change.flat { color: var(--color-text-regular); }

.card-remove {
  width: 36px;
  height: 36px;
  border: none;
  border-radius: 10px;
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  margin-right: var(--sp-3);
  transition: all 0.15s;
  outline: none;
  padding: 0;
}
.card-remove svg {
  width: 16px;
  height: 16px;
}
.card-remove:hover {
  background: rgba(228, 57, 60, 0.08);
  color: var(--color-up);
}
.card-remove:focus-visible {
  box-shadow: 0 0 0 2px var(--color-brand);
}

.card-enter-active,
.card-leave-active {
  transition: all 0.3s ease;
}
.card-enter-from {
  opacity: 0;
  transform: translateY(12px);
}
.card-leave-to {
  opacity: 0;
  transform: translateX(40px);
}
.card-move {
  transition: transform 0.3s ease;
}

@media (max-width: 767px) {
  .stats-section {
    grid-template-columns: repeat(2, 1fr);
  }
  .card-body {
    padding: var(--sp-3) var(--sp-3) var(--sp-3) var(--sp-4);
  }
  .card-figures {
    gap: var(--sp-2);
  }
}
</style>
