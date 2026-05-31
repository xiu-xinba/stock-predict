<template>
  <div class="search-overlay" :data-state="open ? 'open' : 'closed'" role="dialog" aria-modal="true" :aria-hidden="!open" :inert="!open || undefined">
    <div class="search-backdrop" @click="close" @wheel.prevent></div>
    <div class="search-container" @click.self="close">
      <div class="search-panel" @click.stop>
        <div class="search-header">
          <div class="search-input-wrap">
            <svg class="search-icon" viewBox="0 0 1024 1024" width="18" height="18" aria-hidden="true"><path fill="currentColor" d="m795.904 750.72 124.992 124.928a32 32 0 0 1-45.248 45.248L750.656 795.904a416 416 0 1 1 45.248-45.248zM480 832a352 352 0 1 0 0-704 352 352 0 0 0 0 704"/></svg>
            <input
              ref="inputRef"
              v-model="store.query"
              type="text"
              class="search-input"
              placeholder="搜索基金、股票..."
              autocomplete="off"
              role="combobox"
              :aria-expanded="showDropdown"
              aria-controls="search-results"
              aria-activedescendant=""
              @input="onInput(($event.target as HTMLInputElement).value)"
              @focus="onFocus"
              @keydown="onKeydown"
            >
            <button v-if="store.query" class="search-clear" type="button" @click="clearInput" aria-label="清除搜索">
              <svg viewBox="0 0 1024 1024" width="16" height="16"><path fill="currentColor" d="M512 64C264.6 64 64 264.6 64 512s200.6 448 448 448 448-200.6 448-448S759.4 64 512 64zm165.4 618.2-44.2 44.2L512 555.2 390.8 726.4l-44.2-44.2L467.8 512 346.6 340.8l44.2-44.2L512 468.8l121.2-171.2 44.2 44.2L556.2 512z"/></svg>
            </button>
            <kbd class="search-kbd">ESC</kbd>
          </div>
        </div>

        <div v-if="showDropdown && !store.query.trim() && store.history.length > 0" class="search-history">
          <div class="search-history-header">
            <span class="search-history-title">
              <svg class="history-clock-icon" viewBox="0 0 1024 1024" width="14" height="14" aria-hidden="true"><path fill="currentColor" d="M512 64C264.6 64 64 264.6 64 512s200.6 448 448 448 448-200.6 448-448S759.4 64 512 64zm0 820c-205.4 0-372-166.6-372-372s166.6-372 372-372 372 166.6 372 372-166.6 372-372 372z"/><path fill="currentColor" d="M686.7 638.6 544.1 535.5V288c0-4.4-3.6-8-8-8h-48c-4.4 0-8 3.6-8 8v275.4c0 2.6 1.2 5 3.3 6.5l165.4 120.6c3.6 2.6 8.6 1.8 11.2-1.7l28.6-39c2.6-3.7 1.8-8.7-1.8-11.2z"/></svg>
              搜索历史
            </span>
            <button class="search-history-clear" type="button" @click="store.clearHistory()">清除全部</button>
          </div>
          <div class="search-history-list">
            <div v-for="h in store.history.slice(0, 10)" :key="h.keyword" class="search-history-item-wrap">
              <button class="search-history-item" type="button" @click="selectSuggestion(h.keyword)">
                <span class="history-keyword">{{ h.keyword }}</span>
                <span v-if="h.type !== 'all'" class="history-type-badge">{{ h.type === 'funds' ? '基金' : '股票' }}</span>
                <span class="history-time">{{ formatHistoryTime(h.timestamp) }}</span>
              </button>
              <button class="history-remove" type="button" title="删除此记录" @click.stop="store.removeHistory(h.keyword)">
                <svg viewBox="0 0 1024 1024" width="12" height="12"><path fill="currentColor" d="M512 64C264.6 64 64 264.6 64 512s200.6 448 448 448 448-200.6 448-448S759.4 64 512 64zm165.4 618.2-44.2 44.2L512 555.2 390.8 726.4l-44.2-44.2L467.8 512 346.6 340.8l44.2-44.2L512 468.8l121.2-171.2 44.2 44.2L556.2 512z"/></svg>
              </button>
            </div>
          </div>
        </div>

        <div v-if="store.query.trim() && (store.loading || store.hasResults || store.error)" id="search-results" class="search-body" role="listbox">
          <div class="search-tabs">
            <button class="search-tab" :class="{ active: store.activeTab === 'all' }" type="button" @click="switchTab('all')">全部</button>
            <button class="search-tab" :class="{ active: store.activeTab === 'funds' }" type="button" @click="switchTab('funds')">基金 <span v-if="store.fundTotal" class="tab-count">{{ store.fundTotal }}</span></button>
            <button class="search-tab" :class="{ active: store.activeTab === 'stocks' }" type="button" @click="switchTab('stocks')">股票 <span v-if="store.stockTotal" class="tab-count">{{ store.stockTotal }}</span></button>
          </div>

          <div class="search-results" @keydown.escape.stop>
            <div v-if="store.loading" class="search-loading">
              <div class="search-spinner"></div>
              <span>搜索中...</span>
            </div>

            <div v-else-if="store.error" class="search-error">
              <span>搜索失败，请重试</span>
              <button class="search-retry" type="button" @click="store.search()">重试</button>
            </div>

            <template v-else>
              <template v-if="store.activeTab !== 'stocks'">
                <div v-if="store.fundResults.length > 0" class="result-section">
                  <div v-if="store.activeTab === 'all'" class="result-section-title">基金</div>
                  <div v-for="fund in store.fundResults" :key="fund.fund_code" class="result-item" role="option" @click="goFund(fund.fund_code)">
                    <div class="result-item-main">
                      <span class="result-item-name">{{ fund.fund_name }}</span>
                      <span class="result-item-code">{{ fund.fund_code }}</span>
                    </div>
                    <div class="result-item-meta">
                      <span v-if="fund.fund_type" class="result-badge">{{ fund.fund_type }}</span>
                      <span v-if="fund.change_pct != null" class="result-change" :class="getDirection(fund.change_pct)">{{ formatSignedPct(fund.change_pct, 2) }}</span>
                    </div>
                  </div>
                </div>
              </template>

              <template v-if="store.activeTab !== 'funds'">
                <div v-if="store.stockResults.length > 0" class="result-section">
                  <div v-if="store.activeTab === 'all'" class="result-section-title">股票</div>
                  <div v-for="stock in store.stockResults" :key="stock.stock_code" class="result-item" role="option" @click="goStock(stock.stock_code)">
                    <div class="result-item-main">
                      <span class="result-item-name">{{ stock.stock_name }}</span>
                      <span class="result-item-code">
                        <span v-if="stock.market" class="market-tag" :data-market="stock.market">{{ marketLabel(stock.market) }}</span>
                        {{ stock.stock_code }}
                      </span>
                    </div>
                    <div class="result-item-meta">
                      <span v-if="stock.industry" class="result-badge">{{ stock.industry }}</span>
                      <span v-if="stock.change_pct != null && stock.change_pct !== 0" class="result-change" :class="getDirection(stock.change_pct)">{{ formatSignedPct(stock.change_pct, 2) }}</span>
                    </div>
                  </div>
                </div>
              </template>

              <div v-if="!store.hasResults" class="search-empty">
                <span>未找到相关结果</span>
              </div>
            </template>
          </div>

          <div v-if="!store.loading && !store.error && store.hasResults && totalPages > 1" class="search-pagination">
            <button class="page-btn" :disabled="store.page <= 1" type="button" @click="store.search(undefined, store.page - 1)">上一页</button>
            <span class="page-info">{{ store.page }} / {{ totalPages }}</span>
            <button class="page-btn" :disabled="store.page >= totalPages" type="button" @click="store.search(undefined, store.page + 1)">下一页</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useSearch } from '@/composables/useSearch'
import { getDirection, formatSignedPct } from '@/utils/format'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ close: [] }>()

const router = useRouter()
const { store, inputRef, showDropdown, onInput, onFocus, selectSuggestion, clearInput } = useSearch()

const totalPages = computed(() => {
  const total = store.activeTab === 'funds' ? store.fundTotal
    : store.activeTab === 'stocks' ? store.stockTotal
    : Math.max(store.fundTotal, store.stockTotal)
  return Math.max(1, Math.ceil(total / store.size))
})

let savedScrollY = 0

function lockBodyScroll() {
  savedScrollY = window.scrollY
  const scrollbarWidth = window.innerWidth - document.documentElement.clientWidth
  document.body.style.overflow = 'hidden'
  document.body.style.paddingRight = `${scrollbarWidth}px`
  document.body.style.top = `-${savedScrollY}px`
  document.body.style.position = 'fixed'
  document.body.style.width = '100%'
}

function unlockBodyScroll() {
  document.body.style.overflow = ''
  document.body.style.paddingRight = ''
  document.body.style.top = ''
  document.body.style.position = ''
  document.body.style.width = ''
  window.scrollTo(0, savedScrollY)
}

function close() {
  emit('close')
}

function switchTab(tab: 'all' | 'funds' | 'stocks') {
  store.activeTab = tab
  if (store.query.trim()) {
    store.search()
  }
}

function goFund(code: string) {
  close()
  router.push({ name: 'Predict', query: { fundCode: code } })
}

function goStock(code: string) {
  close()
  router.push({ name: 'Predict', query: { stockCode: code } })
}

function marketLabel(market: string): string {
  switch (market) {
    case 'sh': return '沪'
    case 'sz': return '深'
    case 'bj': return '北'
    default: return market.toUpperCase()
  }
}

function formatHistoryTime(timestamp: number): string {
  const now = Date.now()
  const diff = now - timestamp
  if (diff < 60_000) return '刚刚'
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}分钟前`
  const date = new Date(timestamp)
  const today = new Date(now)
  const isToday = date.toDateString() === today.toDateString()
  const yesterday = new Date(now - 86_400_000)
  const isYesterday = date.toDateString() === yesterday.toDateString()
  if (isToday) return `${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
  if (isYesterday) return `昨天 ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
  return `${date.getMonth() + 1}/${date.getDate()} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    e.preventDefault()
    close()
  }
  if (e.key === 'Enter' && store.query.trim()) {
    store.search()
  }
}

watch(() => props.open, (val) => {
  if (val) {
    store.reset()
    store.loadHistory()
    store.loadFilters()
    lockBodyScroll()
    nextTick(() => {
      inputRef.value?.focus()
    })
  } else {
    showDropdown.value = false
    unlockBodyScroll()
  }
})

onUnmounted(() => {
  if (props.open) {
    unlockBodyScroll()
  }
})
</script>

<style scoped>
.search-overlay {
  position: fixed;
  inset: 0;
  z-index: 1000;
  pointer-events: none;
  visibility: hidden;
  transition: visibility 0s 0.2s;
}

.search-overlay[data-state="open"] {
  pointer-events: auto;
  visibility: visible;
  transition: visibility 0s 0s;
}

.search-backdrop {
  position: absolute;
  inset: 0;
  background: var(--color-bg-overlay);
  opacity: 0;
  transition: opacity 0.2s ease;
}

.search-overlay[data-state="open"] .search-backdrop {
  opacity: 1;
}

.search-container {
  position: relative;
  display: flex;
  justify-content: center;
  padding-top: max(80px, 18vh);
  width: 100%;
  height: 100%;
}

.search-panel {
  position: relative;
  width: min(600px, 92vw);
  max-height: 70vh;
  display: flex;
  flex-direction: column;
  background: var(--color-bg-card);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-lg);
  opacity: 0;
  transform: translateY(-12px) scale(0.98);
  transition: opacity 0.2s ease, transform 0.2s cubic-bezier(0.16, 1, 0.3, 1);
  overflow: hidden;
}

.search-overlay[data-state="open"] .search-panel {
  opacity: 1;
  transform: translateY(0) scale(1);
}

.search-header {
  padding: var(--sp-4) var(--sp-5);
  border-bottom: 1px solid var(--color-border-light);
}

.search-input-wrap {
  display: flex;
  align-items: center;
  gap: var(--sp-3);
  padding: var(--sp-2) var(--sp-4);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-bg-elevated);
  transition: border-color var(--transition-fast), box-shadow var(--transition-fast);
}

.search-input-wrap:focus-within {
  border-color: var(--color-brand-muted);
  box-shadow: 0 0 0 3px var(--color-brand-light);
}

.search-icon {
  flex-shrink: 0;
  color: var(--color-text-secondary);
}

.search-input {
  flex: 1;
  border: none;
  outline: none;
  background: transparent;
  color: var(--color-text-primary);
  font-size: var(--fs-base);
  font-family: inherit;
  line-height: var(--lh-normal);
}

.search-input::placeholder {
  color: var(--color-text-disabled);
}

.search-clear {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  border: none;
  background: none;
  color: var(--color-text-secondary);
  cursor: pointer;
  border-radius: var(--radius-full);
  transition: color var(--transition-fast), background-color var(--transition-fast);
}

.search-clear:hover {
  color: var(--color-text-primary);
  background: var(--color-bg-hover);
}

.search-kbd {
  flex-shrink: 0;
  padding: var(--sp-0_5) var(--sp-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg-card);
  color: var(--color-text-disabled);
  font-size: var(--fs-xs);
  font-family: inherit;
  line-height: 1;
}

.search-history {
  padding: var(--sp-3) var(--sp-5);
  border-bottom: 1px solid var(--color-border-light);
}

.search-history-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--sp-2);
}

.search-history-title {
  display: flex;
  align-items: center;
  gap: var(--sp-1);
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
  color: var(--color-text-secondary);
}

.history-clock-icon {
  color: var(--color-text-disabled);
}

.search-history-clear {
  border: none;
  background: none;
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
  cursor: pointer;
  transition: color var(--transition-fast);
}

.search-history-clear:hover {
  color: var(--color-danger);
}

.search-history-list {
  display: flex;
  flex-direction: column;
  gap: var(--sp-0_5);
}

.search-history-item-wrap {
  display: flex;
  align-items: center;
  gap: var(--sp-1);
}

.search-history-item {
  flex: 1;
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  padding: var(--sp-2) var(--sp-3);
  border: none;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--color-text-regular);
  font-size: var(--fs-sm);
  cursor: pointer;
  text-align: left;
  transition: background-color var(--transition-fast), color var(--transition-fast);
}

.search-history-item:hover {
  background: var(--color-bg-hover);
  color: var(--color-brand);
}

.history-keyword {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.history-type-badge {
  flex-shrink: 0;
  padding: 0 var(--sp-2);
  border-radius: var(--radius-full);
  background: var(--color-brand-soft);
  color: var(--color-brand);
  font-size: 10px;
  font-weight: var(--fw-medium);
  line-height: 18px;
}

.history-time {
  flex-shrink: 0;
  color: var(--color-text-disabled);
  font-size: var(--fs-xs);
  font-variant-numeric: tabular-nums;
}

.history-remove {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  padding: 0;
  border: none;
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--color-text-disabled);
  cursor: pointer;
  opacity: 0;
  transition: opacity var(--transition-fast), color var(--transition-fast), background-color var(--transition-fast);
}

.search-history-item-wrap:hover .history-remove {
  opacity: 1;
}

.history-remove:hover {
  color: var(--color-danger);
  background: var(--color-bg-hover);
}

.search-body {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}

.search-tabs {
  display: flex;
  gap: var(--sp-1);
  padding: var(--sp-2) var(--sp-5);
  border-bottom: 1px solid var(--color-border-light);
}

.search-tab {
  padding: var(--sp-1) var(--sp-3);
  border: none;
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--color-text-secondary);
  font-size: var(--fs-sm);
  font-family: inherit;
  cursor: pointer;
  transition: background-color var(--transition-fast), color var(--transition-fast);
}

.search-tab:hover {
  background: var(--color-bg-hover);
  color: var(--color-text-primary);
}

.search-tab.active {
  background: var(--color-brand-light);
  color: var(--color-brand);
  font-weight: var(--fw-medium);
}

.tab-count {
  font-size: var(--fs-xs);
  opacity: 0.7;
}

.search-results {
  overflow-y: auto;
  flex: 1;
  min-height: 0;
  overscroll-behavior: contain;
  padding: var(--sp-2) 0;
}

.result-section {
  padding: 0 var(--sp-5);
}

.result-section-title {
  padding: var(--sp-2) 0;
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.result-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--sp-3) var(--sp-3);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: background-color var(--transition-fast);
}

.result-item:hover {
  background: var(--color-bg-hover);
}

.result-item-main {
  display: flex;
  flex-direction: column;
  gap: var(--sp-0_5);
  min-width: 0;
}

.result-item-name {
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
  color: var(--color-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.result-item-code {
  font-size: var(--fs-xs);
  color: var(--color-text-secondary);
  font-variant-numeric: tabular-nums;
}

.market-tag {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border-radius: 3px;
  font-size: 10px;
  font-weight: 600;
  line-height: 1;
  margin-right: 4px;
  vertical-align: middle;
}

.market-tag[data-market="sh"] {
  background: rgba(239, 68, 68, 0.12);
  color: #ef4444;
}

.market-tag[data-market="sz"] {
  background: rgba(34, 197, 94, 0.12);
  color: #22c55e;
}

.market-tag[data-market="bj"] {
  background: rgba(249, 115, 22, 0.12);
  color: #f97316;
}

.result-item-meta {
  display: flex;
  align-items: center;
  gap: var(--sp-2);
  flex-shrink: 0;
}

.result-badge {
  padding: var(--sp-0_5) var(--sp-2);
  border-radius: var(--radius-full);
  background: var(--color-bg-elevated);
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
}

.result-change {
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
  font-variant-numeric: tabular-nums;
}

.result-change.text-up { color: var(--color-up); }
.result-change.text-down { color: var(--color-down); }

.search-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--sp-10);
  gap: var(--sp-3);
  color: var(--color-text-secondary);
  font-size: var(--fs-sm);
}

.search-spinner {
  width: 24px;
  height: 24px;
  border: 2px solid var(--color-border);
  border-top-color: var(--color-brand);
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.search-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--sp-10);
  gap: var(--sp-3);
  color: var(--color-text-secondary);
  font-size: var(--fs-sm);
}

.search-retry {
  padding: var(--sp-1) var(--sp-4);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-full);
  background: var(--color-bg-card);
  color: var(--color-text-regular);
  font-size: var(--fs-sm);
  cursor: pointer;
  transition: border-color var(--transition-fast), color var(--transition-fast);
}

.search-retry:hover {
  border-color: var(--color-brand-muted);
  color: var(--color-brand);
}

.search-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--sp-10);
  color: var(--color-text-disabled);
  font-size: var(--fs-sm);
}

.search-pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--sp-4);
  padding: var(--sp-3) var(--sp-5);
  border-top: 1px solid var(--color-border-light);
}

.page-btn {
  padding: var(--sp-1) var(--sp-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg-card);
  color: var(--color-text-regular);
  font-size: var(--fs-xs);
  cursor: pointer;
  transition: border-color var(--transition-fast), color var(--transition-fast);
}

.page-btn:hover:not(:disabled) {
  border-color: var(--color-brand-muted);
  color: var(--color-brand);
}

.page-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.page-info {
  font-size: var(--fs-xs);
  color: var(--color-text-secondary);
  font-variant-numeric: tabular-nums;
}

@media (max-width: 768px) {
  .search-container {
    padding-top: max(60px, 12vh);
  }

  .search-panel {
    width: 96vw;
    max-height: 80vh;
    border-radius: var(--radius-lg);
  }

  .search-kbd {
    display: none;
  }
}

@media (prefers-reduced-motion: reduce) {
  .search-panel {
    transition: none;
  }
  .search-spinner {
    animation: none;
  }
}
</style>
