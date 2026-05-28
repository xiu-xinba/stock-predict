<template>
  <div class="search-panel">
    <div class="search-box">
      <div class="search-field">
        <svg class="search-icon" viewBox="0 0 1024 1024" aria-hidden="true"><path fill="currentColor" d="m795.904 750.72 124.992 124.928a32 32 0 0 1-45.248 45.248L750.656 795.904a416 416 0 1 1 45.248-45.248zM480 832a352 352 0 1 0 0-704 352 352 0 0 0 0 704"/></svg>
        <input
          v-model="keyword"
          type="text"
          class="search-input"
          placeholder="输入基金名称或代码，如 110011"
          aria-label="搜索基金"
          @input="onInput"
          @keydown.enter="doPredict"
          @focus="onFocus"
          @blur="onBlur"
        />
        <button
          class="filter-toggle-btn"
          :class="{ active: showFilterPanel }"
          type="button"
          title="筛选"
          @mousedown.prevent="toggleFilterPanel"
        >
          <svg viewBox="0 0 1024 1024" aria-hidden="true"><path fill="currentColor" d="M384 523.392V928a32 32 0 0 0 46.336 28.608l192-96A32 32 0 0 0 640 832V523.392l280.768-343.104a32 32 0 0 0-24.768-52.288H128a32 32 0 0 0-24.768 52.288L384 523.392z"/></svg>
        </button>
        <transition name="dropdown">
          <div v-if="showDropdown && (suggestions.length > 0 || searchHistory.length > 0)" class="search-dropdown">
            <div v-if="searchHistory.length > 0 && !keyword.trim()" class="dropdown-section">
              <div class="dropdown-section-header">
                <span>搜索历史</span>
                <button class="clear-history-btn" type="button" @mousedown.prevent="clearHistory">清除</button>
              </div>
              <div
                v-for="item in searchHistory"
                :key="'history-' + item.code"
                class="dropdown-item history-row"
                @mousedown.prevent="onSelectHistory(item)"
              >
                <svg class="history-icon" viewBox="0 0 1024 1024" aria-hidden="true"><path fill="currentColor" d="M512 896a384 384 0 1 0 0-768 384 384 0 0 0 0 768zm0 64a448 448 0 1 1 0-896 448 448 0 0 1 0 896z"/><path fill="currentColor" d="M480 256a32 32 0 0 1 32 32v224h160a32 32 0 0 1 0 64H480a32 32 0 0 1-32-32V288a32 32 0 0 1 32-32z"/></svg>
                <span class="dropdown-code">{{ item.code }}</span>
                <span class="dropdown-name">{{ item.name }}</span>
              </div>
            </div>
            <div v-if="suggestions.length > 0" class="dropdown-section">
              <div
                v-for="item in suggestions"
                :key="item.fund_code"
                class="dropdown-item"
                @mousedown.prevent="onSelectItem(item)"
              >
                <span class="dropdown-code">{{ item.fund_code }}</span>
                <span class="dropdown-name">{{ item.fund_name }}</span>
                <span v-if="item.risk_level" class="dropdown-risk" :class="riskClass(item.risk_level)">{{ item.risk_level }}</span>
                <span class="dropdown-type">{{ item.fund_type }}</span>
              </div>
            </div>
            <div v-if="totalPages > 1" class="dropdown-pagination">
              <button
                type="button"
                class="page-btn"
                :disabled="store.searchPage <= 1 || store.loading"
                @mousedown.prevent="changeSearchPage(store.searchPage - 1)"
              >
                上一页
              </button>
              <span class="page-info">{{ store.searchPage }} / {{ totalPages }}</span>
              <button
                type="button"
                class="page-btn"
                :disabled="store.searchPage >= totalPages || store.loading"
                @mousedown.prevent="changeSearchPage(store.searchPage + 1)"
              >
                下一页
              </button>
            </div>
          </div>
        </transition>
      </div>
      <button class="predict-btn" type="button" :disabled="store.loading" @click="doPredict">
        <svg v-if="!store.loading" class="btn-icon" viewBox="0 0 1024 1024" aria-hidden="true"><path fill="currentColor" d="M512 64h64v192h-64zm0 576h64v192h-64zM160 480v-64h192v64zm576 0v-64h192v64zM249.856 199.04l45.248-45.184L430.848 289.6 385.6 334.848 249.856 199.104zM657.152 606.4l45.248-45.248 135.744 135.744-45.248 45.248zM114.048 923.2 68.8 877.952l316.8-316.8 45.248 45.248zM702.4 334.848 657.152 289.6l135.744-135.744 45.248 45.248z"/></svg>
        <svg v-else class="btn-icon spinning" viewBox="0 0 1024 1024" aria-hidden="true"><path fill="currentColor" d="M512 64a32 32 0 0 1 32 32v192a32 32 0 0 1-64 0V96a32 32 0 0 1 32-32zm0 640a32 32 0 0 1 32 32v192a32 32 0 0 1-64 0V736a32 32 0 0 1 32-32zM195.2 195.2a32 32 0 0 1 45.248 0L376.32 331.008a32 32 0 0 1-45.248 45.248L195.2 240.448a32 32 0 0 1 0-45.248zm407.424 407.424a32 32 0 0 1 45.248 0l135.808 135.872a32 32 0 0 1-45.248 45.248l-135.808-135.872a32 32 0 0 1 0-45.248zM64 512a32 32 0 0 1 32-32h192a32 32 0 0 1 0 64H96a32 32 0 0 1-32-32zm640 0a32 32 0 0 1 32-32h192a32 32 0 0 1 0 64H736a32 32 0 0 1-32-32zM195.2 828.8a32 32 0 0 1 0-45.248l135.872-135.808a32 32 0 0 1 45.248 45.248L240.448 828.8a32 32 0 0 1-45.248 0zm407.424-407.424a32 32 0 0 1 0-45.248l135.872-135.808a32 32 0 0 1 45.248 45.248L647.872 421.376a32 32 0 0 1-45.248 0z"/></svg>
        <span>预测</span>
      </button>
    </div>

    <transition name="filter-panel">
      <div v-if="showFilterPanel" class="filter-panel">
        <div class="filter-row">
          <div class="filter-group">
            <label class="filter-label">基金类型</label>
            <select v-model="activeFilters.type" class="filter-select" @change="onFilterChange">
              <option value="">全部</option>
              <option v-for="t in store.filters.types" :key="t" :value="t">{{ t }}</option>
            </select>
          </div>
          <div class="filter-group">
            <label class="filter-label">基金公司</label>
            <select v-model="activeFilters.company" class="filter-select" @change="onFilterChange">
              <option value="">全部</option>
              <option v-for="c in store.filters.companies" :key="c" :value="c">{{ c }}</option>
            </select>
          </div>
          <div class="filter-group">
            <label class="filter-label">风险等级</label>
            <select v-model="activeFilters.risk_level" class="filter-select" @change="onFilterChange">
              <option value="">全部</option>
              <option v-for="r in store.filters.risk_levels" :key="r" :value="r">{{ r }}</option>
            </select>
          </div>
          <div class="filter-group">
            <label class="filter-label">排序</label>
            <select v-model="activeFilters.sort_by" class="filter-select" @change="onFilterChange">
              <option value="relevance">相关度</option>
              <option value="return_1y">近1年收益</option>
              <option value="return_3y">近3年收益</option>
              <option value="latest_nav">最新净值</option>
              <option value="inception_date">成立日期</option>
            </select>
          </div>
        </div>
        <div class="filter-actions">
          <button class="filter-reset-btn" type="button" @click="resetFilters">重置筛选</button>
        </div>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { usePredictionStore } from '@/stores/prediction'
import { useFundSearch } from '@/composables/useFundSearch'
import type { FundItem } from '@/types'

const store = usePredictionStore()
const { keyword, suggestions, showDropdown, onInput, onBlur, onFocus, selectItem, clearSuggestions } = useFundSearch(
  () => currentFilters()
)

const showFilterPanel = ref(false)
const activeFilters = reactive({
  type: '',
  company: '',
  risk_level: '',
  sort_by: 'relevance',
  sort_order: 'desc' as string,
})

const HISTORY_KEY = 'fund-search-history'
const MAX_HISTORY = 10

interface HistoryItem {
  code: string
  name: string
}

const searchHistory = ref<HistoryItem[]>([])
const totalPages = computed(() => Math.max(1, Math.ceil(store.searchTotal / store.searchSize)))

function isHistoryItem(value: unknown): value is HistoryItem {
  if (!value || typeof value !== 'object') return false
  const candidate = value as Partial<Record<keyof HistoryItem, unknown>>
  return typeof candidate.code === 'string' && typeof candidate.name === 'string'
}

onMounted(() => {
  loadHistory()
  store.loadFilters()
})

function loadHistory() {
  try {
    const raw = localStorage.getItem(HISTORY_KEY)
    if (raw) {
      const parsed = JSON.parse(raw)
      if (Array.isArray(parsed)) {
        searchHistory.value = parsed.filter(isHistoryItem)
      }
    }
  } catch { /* ignore */ }
}

function saveHistory(code: string, name: string) {
  const list = searchHistory.value.filter(h => h.code !== code)
  list.unshift({ code, name })
  if (list.length > MAX_HISTORY) list.length = MAX_HISTORY
  searchHistory.value = list
  localStorage.setItem(HISTORY_KEY, JSON.stringify(list))
}

function clearHistory() {
  searchHistory.value = []
  localStorage.removeItem(HISTORY_KEY)
}

function onSelectHistory(item: HistoryItem) {
  keyword.value = `${item.name} (${item.code})`
  saveHistory(item.code, item.name)
  clearSuggestions()
  store.predict(item.code)
}

function toggleFilterPanel() {
  showFilterPanel.value = !showFilterPanel.value
}

function currentFilters() {
  return {
    type: activeFilters.type || undefined,
    company: activeFilters.company || undefined,
    risk_level: activeFilters.risk_level || undefined,
    sort_by: activeFilters.sort_by,
    sort_order: activeFilters.sort_order,
  }
}

async function onFilterChange() {
  const codeMatch = keyword.value.match(/\((\d{6})\)/)
  const searchKey = codeMatch ? codeMatch[1] : keyword.value.trim()
  if (searchKey) {
    await store.search(searchKey, 1, currentFilters())
    suggestions.value = store.searchResults
    showDropdown.value = true
  }
}

async function changeSearchPage(page: number) {
  const nextPage = Math.min(Math.max(1, page), totalPages.value)
  if (nextPage === store.searchPage || store.loading) return
  const codeMatch = keyword.value.match(/\((\d{6})\)/)
  const searchKey = codeMatch ? codeMatch[1] : keyword.value.trim()
  if (!searchKey) return
  await store.search(searchKey, nextPage, currentFilters())
  suggestions.value = store.searchResults
  showDropdown.value = true
}

async function resetFilters() {
  activeFilters.type = ''
  activeFilters.company = ''
  activeFilters.risk_level = ''
  activeFilters.sort_by = 'relevance'
  activeFilters.sort_order = 'desc'
  if (keyword.value.trim()) {
    await store.search(keyword.value.trim())
    suggestions.value = store.searchResults
    showDropdown.value = true
  }
}

function onSelectItem(item: FundItem) {
  selectItem(item)
  saveHistory(item.fund_code, item.fund_name)
  store.predict(item.fund_code)
}

function riskClass(level: string) {
  const map: Record<string, string> = {
    '低': 'risk-low',
    '中低': 'risk-medium-low',
    '中': 'risk-medium',
    '中高': 'risk-medium-high',
    '高': 'risk-high',
  }
  return map[level] || 'risk-medium'
}

function doPredict() {
  const codeMatch = keyword.value.match(/\((\d{6})\)/)
  const code = codeMatch ? codeMatch[1] : keyword.value.trim()
  if (!/^\d{6}$/.test(code)) {
    ElMessage.warning('请输入6位基金代码')
    return
  }
  clearSuggestions()
  store.predict(code)
  const nameMatch = keyword.value.match(/^(.+?)\s*\(\d{6}\)/)
  const name = nameMatch ? nameMatch[1] : code
  saveHistory(code, name)
}
</script>

<style scoped>
.search-panel {
  padding: var(--sp-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-bg-card);
}

.search-box {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 112px;
  gap: var(--sp-3);
  align-items: stretch;
}

.search-field {
  position: relative;
  min-width: 0;
  height: 44px;
}

.search-icon {
  position: absolute;
  top: 50%;
  left: 14px;
  z-index: 1;
  width: 16px;
  height: 16px;
  color: var(--color-text-secondary);
  pointer-events: none;
  transform: translateY(-50%);
}

.search-input {
  width: 100%;
  height: 44px;
  margin: 0;
  padding: 0 46px 0 40px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  outline: none;
  background: var(--color-bg-page);
  color: var(--color-text-primary);
  font-size: var(--fs-base);
  line-height: 42px;
  transition: border-color var(--transition-fast), box-shadow var(--transition-fast), background-color var(--transition-fast);
}

.search-input::placeholder {
  color: var(--color-text-secondary);
}

.search-input:hover {
  border-color: var(--color-brand-muted);
}

.search-input:focus {
  border-color: var(--color-brand);
  background: var(--color-bg-card);
  box-shadow: 0 0 0 3px var(--color-brand-light);
}

.filter-toggle-btn {
  position: absolute;
  top: 50%;
  right: 7px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  padding: 0;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
  transform: translateY(-50%);
  transition: background-color var(--transition-fast), border-color var(--transition-fast), color var(--transition-fast);
}

.filter-toggle-btn svg {
  width: 15px;
  height: 15px;
}

.filter-toggle-btn:hover,
.filter-toggle-btn.active {
  color: var(--color-brand);
  background: var(--color-brand-soft);
  border-color: var(--color-brand-muted);
}

.predict-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--sp-2);
  height: 44px;
  padding: 0 var(--sp-4);
  border: 1px solid var(--color-brand);
  border-radius: var(--radius-md);
  background: var(--color-brand);
  color: var(--color-brand-contrast);
  cursor: pointer;
  font-size: var(--fs-base);
  font-weight: var(--fw-semibold);
  line-height: 1;
  transition: background-color var(--transition-fast), border-color var(--transition-fast), opacity var(--transition-fast);
}

.predict-btn:hover {
  background: var(--color-brand-hover);
  border-color: var(--color-brand-hover);
}

.predict-btn:disabled {
  cursor: not-allowed;
  opacity: 0.65;
}

.btn-icon {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}

.btn-icon.spinning {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.search-dropdown {
  position: absolute;
  top: calc(100% + 6px);
  right: 0;
  left: 0;
  z-index: 100;
  max-height: 320px;
  overflow-y: auto;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-bg-card);
  box-shadow: var(--shadow-md);
}

.dropdown-section {
  padding: var(--sp-1) 0;
}

.dropdown-section + .dropdown-section {
  border-top: 1px solid var(--color-border-light);
}

.dropdown-section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--sp-1) var(--sp-3);
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
}

.clear-history-btn {
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--color-brand);
  cursor: pointer;
  font-size: var(--fs-xs);
}

.dropdown-item {
  display: grid;
  grid-template-columns: 70px minmax(0, 1fr) auto auto;
  align-items: center;
  gap: var(--sp-2);
  min-height: 42px;
  padding: 0 var(--sp-3);
  cursor: pointer;
  transition: background-color var(--transition-fast);
}

.dropdown-item.history-row {
  grid-template-columns: 22px 70px minmax(0, 1fr);
}

.dropdown-item:hover {
  background: var(--color-bg-hover);
}

.history-icon {
  width: 14px;
  height: 14px;
  color: var(--color-text-secondary);
}

.dropdown-code {
  color: var(--color-text-regular);
  font-size: var(--fs-sm);
  font-weight: var(--fw-bold);
}

.dropdown-name {
  overflow: hidden;
  color: var(--color-text-primary);
  font-size: var(--fs-sm);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dropdown-risk {
  flex-shrink: 0;
  padding: 1px 6px;
  border-radius: var(--radius-sm);
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
}

.dropdown-risk.risk-low,
.dropdown-risk.risk-medium-low {
  color: var(--color-down);
  background: var(--color-down-bg);
}

.dropdown-risk.risk-medium {
  color: var(--color-brand);
  background: var(--color-brand-soft);
}

.dropdown-risk.risk-medium-high,
.dropdown-risk.risk-high {
  color: var(--color-up);
  background: var(--color-up-bg);
}

.dropdown-type {
  flex-shrink: 0;
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
}

.dropdown-pagination {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--sp-2);
  padding: var(--sp-2) var(--sp-3);
  border-top: 1px solid var(--color-border-light);
  background: var(--color-bg-page);
}

.page-btn {
  min-width: 64px;
  height: 28px;
  padding: 0 var(--sp-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-bg-card);
  color: var(--color-text-primary);
  cursor: pointer;
  font-size: var(--fs-xs);
  transition: border-color var(--transition-fast), color var(--transition-fast), opacity var(--transition-fast);
}

.page-btn:hover:not(:disabled) {
  border-color: var(--color-brand);
  color: var(--color-brand);
}

.page-btn:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.page-info {
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
  white-space: nowrap;
}

.dropdown-enter-active,
.dropdown-leave-active,
.filter-panel-enter-active,
.filter-panel-leave-active {
  transition: opacity var(--transition-fast), transform var(--transition-fast);
}

.dropdown-enter-from,
.dropdown-leave-to,
.filter-panel-enter-from,
.filter-panel-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}

.filter-panel {
  margin-top: var(--sp-3);
  padding-top: var(--sp-3);
  border-top: 1px solid var(--color-border-light);
}

.filter-row {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: var(--sp-3);
}

.filter-group {
  min-width: 0;
}

.filter-label {
  display: block;
  margin-bottom: var(--sp-1);
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
}

.filter-select {
  width: 100%;
  height: 34px;
  padding: 0 var(--sp-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  outline: none;
  background: var(--color-bg-page);
  color: var(--color-text-primary);
  cursor: pointer;
  font-size: var(--fs-sm);
}

.filter-select:focus {
  border-color: var(--color-brand);
  box-shadow: 0 0 0 3px var(--color-brand-light);
}

.filter-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: var(--sp-3);
}

.filter-reset-btn {
  min-height: 30px;
  padding: 0 var(--sp-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
  font-size: var(--fs-sm);
}

.filter-reset-btn:hover {
  color: var(--color-brand);
  border-color: var(--color-brand-muted);
  background: var(--color-brand-soft);
}

@media (max-width: 760px) {
  .search-box {
    grid-template-columns: 1fr;
  }

  .predict-btn {
    width: 100%;
  }

  .filter-row {
    grid-template-columns: 1fr;
  }

  .dropdown-item {
    grid-template-columns: 62px minmax(0, 1fr);
  }

  .dropdown-item.history-row {
    grid-template-columns: 22px 62px minmax(0, 1fr);
  }

  .dropdown-risk,
  .dropdown-type {
    display: none;
  }
}
</style>
