<template>
  <div class="hero-search">
    <h1 class="hero-title">基金涨跌预测</h1>
    <p class="hero-subtitle">输入基金代码或名称，AI 模型实时预测当日涨跌方向</p>
    <div class="search-box">
      <div class="search-field">
        <svg class="search-icon" viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg"><path fill="currentColor" d="m795.904 750.72 124.992 124.928a32 32 0 0 1-45.248 45.248L750.656 795.904a416 416 0 1 1 45.248-45.248zM480 832a352 352 0 1 0 0-704 352 352 0 0 0 0 704"/></svg>
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
          title="筛选"
          @mousedown.prevent="toggleFilterPanel"
        >
          <svg viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg"><path fill="currentColor" d="M384 523.392V928a32 32 0 0 0 46.336 28.608l192-96A32 32 0 0 0 640 832V523.392l280.768-343.104a32 32 0 0 0-24.768-52.288H128a32 32 0 0 0-24.768 52.288L384 523.392z"/></svg>
        </button>
        <transition name="dropdown">
          <div v-if="showDropdown && (suggestions.length > 0 || searchHistory.length > 0)" class="search-dropdown">
            <!-- Search history -->
            <div v-if="searchHistory.length > 0 && !keyword.trim()" class="dropdown-section">
              <div class="dropdown-section-header">
                <span>搜索历史</span>
                <button class="clear-history-btn" @mousedown.prevent="clearHistory">清除</button>
              </div>
              <div
                v-for="item in searchHistory"
                :key="'history-' + item.code"
                class="dropdown-item"
                @mousedown.prevent="onSelectHistory(item)"
              >
                <svg class="history-icon" viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg"><path fill="currentColor" d="M512 896a384 384 0 1 0 0-768 384 384 0 0 0 0 768zm0 64a448 448 0 1 1 0-896 448 448 0 0 1 0 896z"/><path fill="currentColor" d="M480 256a32 32 0 0 1 32 32v224h160a32 32 0 0 1 0 64H480a32 32 0 0 1-32-32V288a32 32 0 0 1 32-32z"/></svg>
                <span class="dropdown-code">{{ item.code }}</span>
                <span class="dropdown-name">{{ item.name }}</span>
              </div>
            </div>
            <!-- Search suggestions -->
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
          </div>
        </transition>
      </div>
      <button
        class="predict-btn"
        :disabled="store.loading"
        @click="doPredict"
      >
        <svg v-if="!store.loading" class="btn-icon" viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg"><path fill="currentColor" d="M512 64h64v192h-64zm0 576h64v192h-64zM160 480v-64h192v64zm576 0v-64h192v64zM249.856 199.04l45.248-45.184L430.848 289.6 385.6 334.848 249.856 199.104zM657.152 606.4l45.248-45.248 135.744 135.744-45.248 45.248zM114.048 923.2 68.8 877.952l316.8-316.8 45.248 45.248zM702.4 334.848 657.152 289.6l135.744-135.744 45.248 45.248z"/></svg>
        <svg v-else class="btn-icon spinning" viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg"><path fill="currentColor" d="M512 64a32 32 0 0 1 32 32v192a32 32 0 0 1-64 0V96a32 32 0 0 1 32-32zm0 640a32 32 0 0 1 32 32v192a32 32 0 0 1-64 0V736a32 32 0 0 1 32-32zM195.2 195.2a32 32 0 0 1 45.248 0L376.32 331.008a32 32 0 0 1-45.248 45.248L195.2 240.448a32 32 0 0 1 0-45.248zm407.424 407.424a32 32 0 0 1 45.248 0l135.808 135.872a32 32 0 0 1-45.248 45.248l-135.808-135.872a32 32 0 0 1 0-45.248zM64 512a32 32 0 0 1 32-32h192a32 32 0 0 1 0 64H96a32 32 0 0 1-32-32zm640 0a32 32 0 0 1 32-32h192a32 32 0 0 1 0 64H736a32 32 0 0 1-32-32zM195.2 828.8a32 32 0 0 1 0-45.248l135.872-135.808a32 32 0 0 1 45.248 45.248L240.448 828.8a32 32 0 0 1-45.248 0zm407.424-407.424a32 32 0 0 1 0-45.248l135.872-135.808a32 32 0 0 1 45.248 45.248L647.872 421.376a32 32 0 0 1-45.248 0z"/></svg>
        预测
      </button>
    </div>

    <!-- Filter panel -->
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
          <button class="filter-reset-btn" @click="resetFilters">重置筛选</button>
        </div>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { usePredictionStore } from '@/stores/prediction'
import { useFundSearch } from '@/composables/useFundSearch'
import type { FundItem } from '@/types'

const store = usePredictionStore()
const { keyword, suggestions, showDropdown, onInput, onBlur, onFocus, selectItem, clearSuggestions } = useFundSearch(
  () => ({
    type: activeFilters.type || undefined,
    company: activeFilters.company || undefined,
    risk_level: activeFilters.risk_level || undefined,
    sort_by: activeFilters.sort_by,
    sort_order: activeFilters.sort_order,
  })
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
        searchHistory.value = parsed.filter(
          (h: any) => h && typeof h.code === 'string' && typeof h.name === 'string'
        )
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

function onFilterChange() {
  const codeMatch = keyword.value.match(/\((\d{6})\)/)
  const searchKey = codeMatch ? codeMatch[1] : keyword.value.trim()
  if (searchKey) {
    store.search(searchKey, 1, {
      type: activeFilters.type || undefined,
      company: activeFilters.company || undefined,
      risk_level: activeFilters.risk_level || undefined,
      sort_by: activeFilters.sort_by,
      sort_order: activeFilters.sort_order,
    })
  }
}

function resetFilters() {
  activeFilters.type = ''
  activeFilters.company = ''
  activeFilters.risk_level = ''
  activeFilters.sort_by = 'relevance'
  activeFilters.sort_order = 'desc'
  if (keyword.value.trim()) {
    store.search(keyword.value.trim())
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
  // Save history after predict (fundName will be set asynchronously)
  const nameMatch = keyword.value.match(/^(.+?)\s*\(\d{6}\)/)
  const name = nameMatch ? nameMatch[1] : code
  saveHistory(code, name)
}
</script>

<style scoped>
.hero-search {
  text-align: center;
  padding: 48px 0 40px;
}
.hero-title {
  font-size: var(--fs-4xl);
  font-weight: 700;
  color: var(--color-text-primary);
  margin-bottom: var(--sp-2);
  letter-spacing: -0.5px;
}
.hero-subtitle {
  font-size: var(--fs-md);
  color: var(--color-text-secondary);
  margin-bottom: var(--sp-8);
}
.search-box {
  display: flex;
  gap: var(--sp-3);
  max-width: 600px;
  margin: 0 auto;
  align-items: stretch;
  height: 52px;
}
.search-field {
  flex: 1;
  min-width: 0;
  position: relative;
  height: 52px;
}
.search-icon {
  position: absolute;
  left: 16px;
  top: 50%;
  transform: translateY(-50%);
  width: 18px;
  height: 18px;
  color: var(--color-text-secondary);
  pointer-events: none;
  z-index: 1;
}
.search-input {
  width: 100%;
  height: 52px;
  padding: 0 44px 0 44px;
  margin: 0;
  border: 2px solid transparent;
  border-radius: 14px;
  background: var(--color-bg-card);
  color: var(--color-text-primary);
  font-size: var(--fs-md);
  font-family: inherit;
  line-height: 48px;
  box-sizing: border-box;
  outline: none;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.06);
  transition: border-color 0.2s, box-shadow 0.2s;
}
.search-input::placeholder {
  color: var(--color-text-secondary);
}
.search-input:hover {
  border-color: var(--color-brand);
}
.search-input:focus {
  border-color: var(--color-brand);
  box-shadow: 0 2px 16px rgba(51, 102, 255, 0.15);
}
html.dark .search-input {
  background: var(--color-bg-card);
  color: var(--color-text-primary);
}
.filter-toggle-btn {
  position: absolute;
  right: 8px;
  top: 50%;
  transform: translateY(-50%);
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s, color 0.15s;
  padding: 0;
}
.filter-toggle-btn svg {
  width: 16px;
  height: 16px;
}
.filter-toggle-btn:hover {
  background: var(--color-bg-hover);
  color: var(--color-brand);
}
.filter-toggle-btn.active {
  color: var(--color-brand);
  background: rgba(51, 102, 255, 0.1);
}
.search-dropdown {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  right: 0;
  background: var(--color-bg-card);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
  z-index: 100;
  max-height: 320px;
  overflow-y: auto;
}
.dropdown-section {
  padding: 4px 0;
}
.dropdown-section + .dropdown-section {
  border-top: 1px solid var(--color-border);
}
.dropdown-section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 6px 16px;
  font-size: var(--fs-xs);
  color: var(--color-text-secondary);
}
.clear-history-btn {
  border: none;
  background: none;
  color: var(--color-brand);
  font-size: var(--fs-xs);
  cursor: pointer;
  padding: 0;
}
.clear-history-btn:hover {
  text-decoration: underline;
}
.dropdown-item {
  display: flex;
  align-items: center;
  padding: 10px 16px;
  cursor: pointer;
  transition: background 0.15s;
  gap: var(--sp-2);
}
.dropdown-item:hover {
  background: var(--color-bg-hover);
}
.dropdown-item:first-child {
  border-radius: var(--radius-md) var(--radius-md) 0 0;
}
.dropdown-item:last-child {
  border-radius: 0 0 var(--radius-md) var(--radius-md);
}
.history-icon {
  width: 14px;
  height: 14px;
  color: var(--color-text-secondary);
  flex-shrink: 0;
}
.dropdown-code {
  color: var(--color-text-regular);
  font-size: var(--fs-sm);
  font-weight: 600;
  min-width: 60px;
}
.dropdown-name {
  flex: 1;
  color: var(--color-text-primary);
  font-size: var(--fs-sm);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.dropdown-risk {
  font-size: var(--fs-xs);
  padding: 1px 6px;
  border-radius: 4px;
  font-weight: 500;
  flex-shrink: 0;
}
.dropdown-risk.risk-low { background: #e8f5e9; color: #2e7d32; }
.dropdown-risk.risk-medium-low { background: #f1f8e9; color: #558b2f; }
.dropdown-risk.risk-medium { background: #fff8e1; color: #f57f17; }
.dropdown-risk.risk-medium-high { background: #fff3e0; color: #e65100; }
.dropdown-risk.risk-high { background: #fce4ec; color: #c62828; }
html.dark .dropdown-risk.risk-low { background: #1b3a1b; color: #66bb6a; }
html.dark .dropdown-risk.risk-medium-low { background: #2a3a1b; color: #9ccc65; }
html.dark .dropdown-risk.risk-medium { background: #3a3510; color: #ffca28; }
html.dark .dropdown-risk.risk-medium-high { background: #3a2510; color: #ff9800; }
html.dark .dropdown-risk.risk-high { background: #3a1b1b; color: #ef5350; }
.dropdown-type {
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
  flex-shrink: 0;
}
.dropdown-enter-active,
.dropdown-leave-active {
  transition: opacity 0.15s, transform 0.15s;
}
.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
.predict-btn {
  height: 52px;
  min-width: 120px;
  border: 2px solid transparent;
  border-radius: 14px;
  padding: 0 28px;
  margin: 0;
  box-sizing: border-box;
  background: var(--color-brand);
  color: #ffffff;
  font-size: var(--fs-md);
  font-weight: 600;
  font-family: inherit;
  letter-spacing: 2px;
  line-height: 48px;
  cursor: pointer;
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  box-shadow: 0 4px 16px rgba(51, 102, 255, 0.3);
  transition: transform 0.15s ease, box-shadow 0.15s ease, background-color 0.15s ease;
  user-select: none;
  -webkit-tap-highlight-color: transparent;
  outline: none;
}
.predict-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(51, 102, 255, 0.4);
  background: #2952e6;
}
html.dark .predict-btn:hover {
  box-shadow: 0 6px 20px rgba(91, 138, 255, 0.4);
  background: #4d7aff;
}
.predict-btn:active {
  transform: translateY(0);
  box-shadow: 0 4px 16px rgba(51, 102, 255, 0.3);
}
.predict-btn:disabled {
  opacity: 0.7;
  cursor: not-allowed;
  transform: none;
}
.predict-btn:focus-visible {
  border-color: #ffffff;
  box-shadow: 0 4px 16px rgba(51, 102, 255, 0.3), 0 0 0 2px var(--color-brand);
}
.btn-icon {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
}
.btn-icon.spinning {
  animation: spin 1s linear infinite;
}
@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

/* Filter panel */
.filter-panel {
  max-width: 600px;
  margin: var(--sp-3) auto 0;
  background: var(--color-bg-card);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: var(--sp-4);
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.06);
  z-index: 10;
  position: relative;
}
.filter-row {
  display: flex;
  gap: var(--sp-3);
  flex-wrap: wrap;
}
.filter-group {
  flex: 1;
  min-width: 120px;
}
.filter-label {
  display: block;
  font-size: var(--fs-xs);
  color: var(--color-text-secondary);
  margin-bottom: 4px;
  font-weight: 500;
}
.filter-select {
  width: 100%;
  height: 36px;
  padding: 0 8px;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  background: var(--color-bg-primary);
  color: var(--color-text-primary);
  font-size: var(--fs-sm);
  font-family: inherit;
  outline: none;
  cursor: pointer;
  transition: border-color 0.15s;
  box-sizing: border-box;
}
.filter-select:focus {
  border-color: var(--color-brand);
}
.filter-actions {
  margin-top: var(--sp-3);
  display: flex;
  justify-content: flex-end;
}
.filter-reset-btn {
  border: 1px solid var(--color-border);
  border-radius: 8px;
  background: transparent;
  color: var(--color-text-secondary);
  font-size: var(--fs-sm);
  padding: 4px 16px;
  cursor: pointer;
  transition: color 0.15s, border-color 0.15s;
}
.filter-reset-btn:hover {
  color: var(--color-brand);
  border-color: var(--color-brand);
}
.filter-panel-enter-active,
.filter-panel-leave-active {
  transition: opacity 0.2s, transform 0.2s;
}
.filter-panel-enter-from,
.filter-panel-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}

@media (max-width: 767px) {
  .hero-search {
    padding: 32px 0 28px;
  }
  .hero-title {
    font-size: var(--fs-2xl);
  }
  .hero-subtitle {
    font-size: var(--fs-base);
  }
  .search-box {
    flex-direction: column;
    height: auto;
  }
  .search-field {
    height: 48px;
  }
  .search-input {
    height: 48px;
    line-height: 44px;
  }
  .predict-btn {
    width: 100%;
    min-width: unset;
    height: 48px;
    line-height: 44px;
  }
  .filter-row {
    flex-direction: column;
  }
  .filter-group {
    min-width: 100%;
  }
}
</style>
