/** @module shared/composables/useAnimatedNumber - 数字滚动动画组合式函数
 * 基于 requestAnimationFrame + easeOutCubic 缓动实现数字平滑过渡，
 * 支持 prefers-reduced-motion 降级为瞬时跳变。
 */
import { ref, watch, type Ref } from 'vue'

function easeOutCubic(t: number): number {
  return 1 - Math.pow(1 - t, 3)
}

function prefersReducedMotion(): boolean {
  return (
    typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches
  )
}

/** 让一个数字在变化时以 easeOutCubic 缓动平滑过渡到新值
 * @param target - 目标值的响应式引用
 * @param duration - 动画时长（毫秒），默认 400
 * @returns 显示值的响应式引用（动画过程中的中间值）
 */
export function useAnimatedNumber(target: Ref<number>, duration = 400): Ref<number> {
  const display = ref(target.value)
  let rafId: number | null = null

  watch(
    target,
    (newVal, oldVal) => {
      if (rafId !== null) {
        cancelAnimationFrame(rafId)
        rafId = null
      }

      // 降级：直接跳变
      if (prefersReducedMotion()) {
        display.value = newVal
        return
      }

      // 值未变化或非有限数，直接赋值
      if (!Number.isFinite(newVal) || !Number.isFinite(oldVal) || newVal === oldVal) {
        display.value = newVal
        return
      }

      const start = oldVal
      const delta = newVal - oldVal
      const startTime = performance.now()

      const tick = (now: number) => {
        const elapsed = now - startTime
        const progress = Math.min(elapsed / duration, 1)
        const eased = easeOutCubic(progress)
        display.value = start + delta * eased

        if (progress < 1) {
          rafId = requestAnimationFrame(tick)
        } else {
          display.value = newVal
          rafId = null
        }
      }

      rafId = requestAnimationFrame(tick)
    },
    { flush: 'post' },
  )

  return display
}
