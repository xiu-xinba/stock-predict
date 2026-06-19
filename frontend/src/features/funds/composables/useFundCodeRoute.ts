/** @module funds/composables/useFundCodeRoute — 从路由参数提取并校验基金代码 */
import { computed } from 'vue'
import { useRoute } from 'vue-router'

/**
 * 从当前路由参数中提取基金代码，并校验其为六位数字格式
 * @returns 包含 fundCode 响应式引用的对象
 */
export function useFundCodeRoute() {
  const route = useRoute()

  const fundCode = computed(() => {
    const raw = route.params.fundCode
    const code = Array.isArray(raw) ? raw[0] : raw
    return code && /^\d{6}$/.test(code) ? code : ''
  })

  return { fundCode }
}
