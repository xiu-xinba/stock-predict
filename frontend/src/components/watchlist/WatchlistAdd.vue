<template>
  <div class="add-row">
    <div class="add-field">
      <svg class="add-icon" viewBox="0 0 1024 1024" aria-hidden="true"><path fill="currentColor" d="m795.904 750.72 124.992 124.928a32 32 0 0 1-45.248 45.248L750.656 795.904a416 416 0 1 1 45.248-45.248zM480 832a352 352 0 1 0 0-704 352 352 0 0 0 0 704"/></svg>
      <input
        v-model="keyword"
        type="text"
        class="add-input"
        placeholder="输入基金代码或名称"
        aria-label="搜索基金添加到自选"
        @input="onInput"
        @keydown.enter="addFirst"
        @focus="onFocus"
        @blur="onBlur"
      />
      <span class="add-meta">{{ watchlistStore.items.length }}/50</span>
      <transition name="drop">
        <div v-if="showDropdown && suggestions.length > 0" class="add-dropdown">
          <div
            v-for="item in suggestions"
            :key="item.fund_code"
            class="drop-item"
            @mousedown.prevent="onSelectItem(item)"
          >
            <span class="drop-code">{{ item.fund_code }}</span>
            <span class="drop-name">{{ item.fund_name }}</span>
            <span class="drop-type">{{ item.fund_type }}</span>
          </div>
        </div>
      </transition>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { useWatchlistStore } from '@/stores/watchlist'
import { useFundSearch } from '@/composables/useFundSearch'

const watchlistStore = useWatchlistStore()
const { keyword, suggestions, showDropdown, onInput, onFocus, onBlur, selectItem } = useFundSearch()

function onSelectItem(item: { fund_code: string; fund_name: string; fund_type: string }) {
  const selected = selectItem(item)
  const result = watchlistStore.addItem({
    fund_code: selected.fund_code,
    fund_name: selected.fund_name,
    fund_type: selected.fund_type,
  })
  if (result === 'added') {
    ElMessage.success(`已添加 ${selected.fund_name}`)
  } else if (result === 'duplicate') {
    ElMessage.info(`${selected.fund_name} 已在自选中`)
  } else {
    ElMessage.warning('自选基金最多支持 50 只')
  }
  keyword.value = ''
}

function addFirst() {
  if (suggestions.value.length > 0) {
    onSelectItem(suggestions.value[0])
  }
}
</script>

<style scoped>
.add-row {
  display: flex;
}

.add-field {
  position: relative;
  width: 100%;
  height: 44px;
}

.add-icon {
  position: absolute;
  left: 14px;
  top: 50%;
  z-index: 1;
  width: 16px;
  height: 16px;
  color: var(--color-text-secondary);
  pointer-events: none;
  transform: translateY(-50%);
}

.add-input {
  width: 100%;
  height: 44px;
  margin: 0;
  padding: 0 70px 0 40px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  outline: none;
  background: var(--color-bg-page);
  color: var(--color-text-primary);
  font-size: var(--fs-base);
  line-height: 42px;
  transition: border-color var(--transition-fast), box-shadow var(--transition-fast), background-color var(--transition-fast);
}

.add-input::placeholder {
  color: var(--color-text-secondary);
}

.add-input:hover {
  border-color: var(--color-brand-muted);
}

.add-input:focus {
  border-color: var(--color-brand);
  background: var(--color-bg-card);
  box-shadow: 0 0 0 3px var(--color-brand-light);
}

.add-meta {
  position: absolute;
  top: 50%;
  right: 12px;
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
  transform: translateY(-50%);
}

.add-dropdown {
  position: absolute;
  top: calc(100% + 6px);
  right: 0;
  left: 0;
  z-index: 100;
  max-height: 260px;
  overflow-y: auto;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-bg-card);
  box-shadow: var(--shadow-md);
}

.drop-item {
  display: grid;
  grid-template-columns: 72px minmax(0, 1fr) auto;
  align-items: center;
  gap: var(--sp-2);
  min-height: 42px;
  padding: 0 var(--sp-3);
  border-bottom: 1px solid var(--color-border-light);
  cursor: pointer;
  transition: background-color var(--transition-fast);
}

.drop-item:last-child {
  border-bottom: 0;
}

.drop-item:hover {
  background: var(--color-bg-hover);
}

.drop-code {
  color: var(--color-text-regular);
  font-size: var(--fs-sm);
  font-weight: var(--fw-bold);
}

.drop-name {
  overflow: hidden;
  color: var(--color-text-primary);
  font-size: var(--fs-sm);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.drop-type {
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
}

.drop-enter-active,
.drop-leave-active {
  transition: opacity var(--transition-fast), transform var(--transition-fast);
}

.drop-enter-from,
.drop-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
</style>
