import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import './style.css'

const debounceResizeObserverErr = (() => {
  let timer: ReturnType<typeof setTimeout> | null = null
  return (e: ErrorEvent) => {
    if (e.message === 'ResizeObserver loop completed with undelivered notifications.') {
      e.stopImmediatePropagation()
      if (timer) clearTimeout(timer)
      timer = setTimeout(() => { timer = null }, 100)
      return
    }
  }
})()
window.addEventListener('error', debounceResizeObserverErr)

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.mount('#app')
