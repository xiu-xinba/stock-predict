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
  let searchSeq = 0

  function onInput() {
    if (debounceTimer) clearTimeout(debounceTimer)
    const query = keyword.value.trim()
    if (!query) {
      searchSeq++
      suggestions.value = []
      showDropdown.value = false
      return
    }
    const seq = ++searchSeq
    debounceTimer = setTimeout(async () => {
      if (disposed) return
      if (seq !== searchSeq) return
      if (query !== keyword.value.trim()) return
      try {
        const filters = getFilters?.()
        await predictionStore.search(query, 1, filters)
        if (disposed) return
        if (seq !== searchSeq) return
        if (query !== keyword.value.trim()) return
        suggestions.value = predictionStore.searchResults
        showDropdown.value = true
      } catch {
        if (seq === searchSeq) suggestions.value = []
      }
    }, 300)
  }

  function onBlur() {
    blurTimer = setTimeout(() => {
      showDropdown.value = false
      focused.value = false
      blurTimer = null
    }, 150)
  }

  function onFocus() {
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
