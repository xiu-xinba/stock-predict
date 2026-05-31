import { computed } from 'vue'
import { useRoute } from 'vue-router'

export function useFundCodeRoute() {
  const route = useRoute()

  const fundCode = computed(() => {
    const raw = route.params.fundCode
    const code = Array.isArray(raw) ? raw[0] : raw
    return code && /^\d{6}$/.test(code) ? code : ''
  })

  return { fundCode }
}
