/** @module shared/composables/useSpotlight - 卡片边框光效组合式函数
 * 监听鼠标移动，更新 --mouse-x / --mouse-y CSS 变量，
 * 配合 .card-spotlight 类实现径向高光边框效果。
 */
import { onMounted, onUnmounted, type Ref } from 'vue'

/** 为指定元素绑定 mousemove 事件，更新 spotlight 所需的 CSS 变量
 * @param elRef - 目标元素的引用
 */
export function useSpotlight(elRef: Ref<HTMLElement | undefined>) {
  function handleMouseMove(e: MouseEvent) {
    const el = elRef.value
    if (!el) return
    const rect = el.getBoundingClientRect()
    const x = ((e.clientX - rect.left) / rect.width) * 100
    const y = ((e.clientY - rect.top) / rect.height) * 100
    el.style.setProperty('--mouse-x', `${x}%`)
    el.style.setProperty('--mouse-y', `${y}%`)
  }

  onMounted(() => {
    const el = elRef.value
    if (!el) return
    el.addEventListener('mousemove', handleMouseMove)
  })

  onUnmounted(() => {
    const el = elRef.value
    if (!el) return
    el.removeEventListener('mousemove', handleMouseMove)
  })
}
