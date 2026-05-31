import { ref, watch } from 'vue'
import { useSearchStore } from '@/stores/search'

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
      store.search(value)
    }, 300)
  }

  function onFocus() {
    if (store.query.trim() || store.history.length > 0) {
      showDropdown.value = true
    }
  }

  function onBlur() {
    setTimeout(() => { showDropdown.value = false }, 200)
  }

  function selectSuggestion(keyword: string) {
    store.query = keyword
    showDropdown.value = false
    store.search(keyword)
  }

  function clearInput() {
    store.query = ''
    showDropdown.value = false
    store.fundResults = []
    store.stockResults = []
    store.suggestions = []
    inputRef.value?.focus()
  }

  watch(() => store.query, (val) => {
    if (!val.trim()) {
      showDropdown.value = false
    }
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
