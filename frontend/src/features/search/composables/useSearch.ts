/** @module search/composables/useSearch — 搜索交互 composable，封装输入防抖、下拉控制等逻辑 */
import { ref, watch, onUnmounted } from 'vue'
import { useSearchStore } from '@/features/search/store/search'

/**
 * 搜索交互 composable，封装输入防抖、下拉展示/隐藏、建议选择等交互逻辑
 * @returns 搜索交互相关的响应式状态和方法
 */
export function useSearch() {
  const store = useSearchStore()
  const inputRef = ref<HTMLInputElement | null>(null)
  const showDropdown = ref(false)
  let debounceTimer: ReturnType<typeof setTimeout> | null = null

  function onInput(value: string) {
    store.query = value
    if (debounceTimer) clearTimeout(debounceTimer)
    if (!value.trim()) {
      store.fundResults = []
      store.stockResults = []
      store.suggestions = []
      showDropdown.value = false
      return
    }
    showDropdown.value = true
    debounceTimer = setTimeout(() => {
      store.search(value, undefined, false) // 防抖实时搜索不记录历史
    }, 300)
  }

  function onFocus() {
    if (store.query.trim() || store.history.length > 0) {
      showDropdown.value = true
    }
  }

  function onBlur() {
    setTimeout(() => {
      showDropdown.value = false
    }, 200)
  }

  function selectSuggestion(keyword: string) {
    store.query = keyword
    showDropdown.value = false
    store.search(keyword, undefined, true) // 点击建议视为明确搜索，记录历史
  }

  function clearInput() {
    store.query = ''
    showDropdown.value = false
    store.fundResults = []
    store.stockResults = []
    store.suggestions = []
    inputRef.value?.focus()
  }

  watch(
    () => store.query,
    (val) => {
      if (!val.trim()) {
        showDropdown.value = false
      }
    },
  )

  onUnmounted(() => {
    if (debounceTimer) clearTimeout(debounceTimer)
  })

  return {
    store,
    inputRef,
    showDropdown,
    onInput,
    onFocus,
    onBlur,
    selectSuggestion,
    clearInput,
  }
}
