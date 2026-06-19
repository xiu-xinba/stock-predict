/** @module settings/store — 设置模块 Pinia store */
import { defineStore } from 'pinia'
import { ref } from 'vue'

const REFRESH_STORAGE_KEY = 'settings-refresh-interval'
const ALLOWED_REFRESH_INTERVALS = new Set([30, 60, 300])

function loadRefreshInterval(): number {
  const parsed = Number(localStorage.getItem(REFRESH_STORAGE_KEY))
  return ALLOWED_REFRESH_INTERVALS.has(parsed) ? parsed : 60
}

/** useSettingsStore - 设置 store */
export const useSettingsStore = defineStore('settings', () => {
  const refreshIntervalSeconds = ref(loadRefreshInterval())

  /**
   * 设置行情自动刷新间隔
   * @param seconds - 刷新间隔秒数，仅允许 30/60/300
   */
  function setRefreshInterval(seconds: number) {
    if (!ALLOWED_REFRESH_INTERVALS.has(seconds)) return
    refreshIntervalSeconds.value = seconds
    localStorage.setItem(REFRESH_STORAGE_KEY, String(seconds))
  }

  return { refreshIntervalSeconds, setRefreshInterval }
})
