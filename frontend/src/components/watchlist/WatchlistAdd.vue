<template>
  <div class="add-row">
    <div class="add-field">
      <svg class="add-icon" viewBox="0 0 1024 1024"><path fill="currentColor" d="m795.904 750.72 124.992 124.928a32 32 0 0 1-45.248 45.248L750.656 795.904a416 416 0 1 1 45.248-45.248zM480 832a352 352 0 1 0 0-704 352 352 0 0 0 0 704"/></svg>
      <input
        v-model="keyword"
        type="text"
        class="add-input"
        placeholder="搜索基金代码或名称添加到自选"
        aria-label="搜索基金添加到自选"
        @input="onInput"
        @keydown.enter="addFirst"
        @focus="onFocus"
        @blur="onBlur"
      />
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
  const added = watchlistStore.addItem({
    fund_code: selected.fund_code,
    fund_name: selected.fund_name,
    fund_type: selected.fund_type,
  })
  if (added) {
    ElMessage.success(`已添加 ${selected.fund_name}`)
  } else {
    ElMessage.info(`${selected.fund_name} 已在自选中`)
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
  justify-content: center;
}
.add-field {
  position: relative;
  width: 100%;
  max-width: 480px;
  height: 44px;
}
.add-icon {
  position: absolute;
  left: 14px;
  top: 50%;
  transform: translateY(-50%);
  width: 16px;
  height: 16px;
  color: var(--color-text-secondary);
  pointer-events: none;
  z-index: 1;
}
.add-input {
  width: 100%;
  height: 44px;
  padding: 0 14px 0 40px;
  margin: 0;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-bg-card);
  color: var(--color-text-primary);
  font-size: var(--fs-base);
  font-family: inherit;
  line-height: 42px;
  box-sizing: border-box;
  outline: none;
  transition: border-color 0.2s, box-shadow 0.2s;
}
.add-input::placeholder {
  color: var(--color-text-secondary);
}
.add-input:hover {
  border-color: var(--color-brand);
}
.add-input:focus {
  border-color: var(--color-brand);
  box-shadow: 0 0 0 3px rgba(51, 102, 255, 0.1);
}
.add-dropdown {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  right: 0;
  background: var(--color-bg-card);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
  z-index: 100;
  max-height: 240px;
  overflow-y: auto;
}
.drop-item {
  display: flex;
  align-items: center;
  padding: 10px 14px;
  cursor: pointer;
  transition: background 0.15s;
  gap: var(--sp-2);
}
.drop-item:hover {
  background: var(--color-bg-hover);
}
.drop-item:first-child { border-radius: var(--radius-md) var(--radius-md) 0 0; }
.drop-item:last-child { border-radius: 0 0 var(--radius-md) var(--radius-md); }
.drop-code {
  color: var(--color-text-regular);
  font-size: var(--fs-sm);
  font-weight: 600;
  min-width: 60px;
}
.drop-name {
  flex: 1;
  color: var(--color-text-primary);
  font-size: var(--fs-sm);
}
.drop-type {
  color: var(--color-text-secondary);
  font-size: var(--fs-xs);
}
.drop-enter-active,
.drop-leave-active {
  transition: opacity 0.15s, transform 0.15s;
}
.drop-enter-from,
.drop-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
</style>
