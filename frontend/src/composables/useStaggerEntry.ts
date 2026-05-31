import { getCurrentInstance, onMounted, onUnmounted } from 'vue'

export function useStaggerEntry(selector: string, options?: {
  threshold?: number
  rootMargin?: string
  staggerMs?: number
  translateY?: number
}) {
  const {
    threshold = 0.08,
    rootMargin = '0px 0px -40px 0px',
    staggerMs = 60,
    translateY = 12,
  } = options ?? {}

  let observer: IntersectionObserver | null = null

  onMounted(() => {
    const instance = getCurrentInstance()
    const root = instance?.proxy?.$el as HTMLElement | undefined
    const elements = root
      ? Array.from(root.querySelectorAll(selector))
      : []
    if (!elements.length) return

    const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    if (prefersReducedMotion) return

    elements.forEach((el, i) => {
      const htmlEl = el as HTMLElement
      htmlEl.style.opacity = '0'
      htmlEl.style.transform = `translateY(${translateY}px) scale(0.98)`
      htmlEl.style.transition = `opacity 0.5s cubic-bezier(0.16, 1, 0.3, 1) ${i * staggerMs}ms, transform 0.5s cubic-bezier(0.16, 1, 0.3, 1) ${i * staggerMs}ms`
    })

    observer = new IntersectionObserver((entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          const htmlEl = entry.target as HTMLElement
          htmlEl.style.opacity = '1'
          htmlEl.style.transform = 'translateY(0) scale(1)'
          observer?.unobserve(entry.target)
        }
      })
    }, { threshold, rootMargin })

    elements.forEach((el) => observer?.observe(el))
  })

  onUnmounted(() => {
    observer?.disconnect()
    observer = null
  })
}
