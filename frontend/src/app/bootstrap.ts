/** @module app/bootstrap - 应用启动引导
 *
 * 负责创建 Vue 应用实例、注册 Pinia 状态管理与路由插件，
 * 并抑制 ResizeObserver 循环溢出的控制台错误，最终挂载到 DOM。
 */
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from '@/app/App.vue'
import router from '@/app/router'
import '@/app/styles/index.css'

/**
 * 启动应用
 *
 * 初始化 Vue 实例并注册核心插件（Pinia、Router），
 * 同时屏蔽 ResizeObserver loop completed with undelivered notifications 错误，
 * 最后将应用挂载到 `#app` 节点。
 */
export function bootstrap() {
  const debounceResizeObserverErr = (() => {
    let timer: ReturnType<typeof setTimeout> | null = null
    return (e: ErrorEvent) => {
      if (e.message === 'ResizeObserver loop completed with undelivered notifications.') {
        e.stopImmediatePropagation()
        if (timer) clearTimeout(timer)
        timer = setTimeout(() => {
          timer = null
        }, 100)
      }
    }
  })()
  window.addEventListener('error', debounceResizeObserverErr)

  const app = createApp(App)
  app.use(createPinia())
  app.use(router)
  app.mount('#app')
}
