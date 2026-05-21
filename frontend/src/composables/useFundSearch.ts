import { ref, onUnmounted } from 'vue'
import { usePredictionStore } from '@/stores/prediction'
import type { FundItem } from '@/types'

export function useFundSearch(getFilters?: () => Record<string, string | undefined>) {
  const predictionStore = usePredictionStore()
  const keyword = ref('')
  const suggestions = ref<FundItem[]>([])
  const showDropdown = ref(false)
  const focused = ref(false)

  let debounceTimer: ReturnType<typeof setTimeout> | null = null
  let blurTimer: ReturnType<typeof setTimeout> | null = null
  let disposed = false

  function onInput() {
    if (debounceTimer) clearTimeout(debounceTimer)
    const query = keyword.value.trim()
    if (!query) {
      suggestions.value = []
      showDropdown.value = false
      return
    }
    debounceTimer = setTimeout(async () => {
      if (disposed) return
      if (query !== keyword.value.trim()) return
      try {
        const filters = getFilters?.()
        await predictionStore.search(query, 1, filters)
        if (disposed) return // BUG-01: check again after await
        suggestions.value = predictionStore.searchResults
        showDropdown.value = true
      } catch {
        suggestions.value = []
      }
    }, 300)
  }

  function onBlur() {
    // BUG-02: Store blur timer so onFocus can cancel it
    blurTimer = setTimeout(() => {
      showDropdown.value = false
      focused.value = false
      blurTimer = null
    }, 150)
  }

  function onFocus() {
    // BUG-02: Cancel pending blur timer to prevent dropdown from closing
    if (blurTimer) {
      clearTimeout(blurTimer)
      blurTimer = null
    }
    showDropdown.value = true
    focused.value = true
  }

  function clearSuggestions() {
    suggestions.value = []
    showDropdown.value = false
  }

  function selectItem(item: FundItem) {
    keyword.value = `${item.fund_name} (${item.fund_code})`
    clearSuggestions()
    return item
  }

  onUnmounted(() => {
    disposed = true
    if (debounceTimer) clearTimeout(debounceTimer)
    if (blurTimer) clearTimeout(blurTimer)
  })

  return {
    keyword,
    suggestions,
    showDropdown,
    focused,
    onInput,
    onFocus,
    onBlur,
    clearSuggestions,
    selectItem,
  }
}
