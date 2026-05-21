import axios from 'axios'
import type { AxiosRequestConfig, AxiosResponse } from 'axios'

export class CancelError extends Error {
  constructor(message: string = 'Request cancelled') {
    super(message)
    this.name = 'CancelError'
  }
}

const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  timeout: 15000,
  withCredentials: true,
})

const pendingRequests = new Map<string, AbortController>()

function getRequestKey(config: AxiosRequestConfig): string {
  // Include request body for POST dedup key so different payloads aren't treated as same request
  const bodyKey = config.method === 'post' || config.method === 'put' || config.method === 'patch'
    ? JSON.stringify(config.data ?? {})
    : ''
  return `${config.method}:${config.baseURL}${config.url}?${JSON.stringify(config.params ?? {})}${bodyKey}`
}

api.interceptors.request.use((config) => {
  const csrfMatch = document.cookie.match(/csrftoken=([^;]+)/)
  if (csrfMatch && config.headers) {
    config.headers['X-CSRFToken'] = csrfMatch[1]
  }

  if (config.method === 'get' || config.method === 'post') {
    const key = getRequestKey(config)
    const existing = pendingRequests.get(key)
    if (existing) {
      existing.abort()
    }
    const controller = new AbortController()
    config.signal = controller.signal
    pendingRequests.set(key, controller)
  }

  return config
})

api.interceptors.response.use(
  (response: AxiosResponse) => {
    const key = getRequestKey(response.config)
    pendingRequests.delete(key)
    return response
  },
  (error) => {
    if (axios.isCancel(error)) {
      return Promise.reject(new CancelError('Request cancelled'))
    }
    if (error.config) {
      const key = getRequestKey(error.config)
      pendingRequests.delete(key)
    }

    const status = error.response?.status
    let safeMessage: string

    if (!status) {
      safeMessage = '网络错误，请检查网络连接后重试'
    } else if (status >= 500) {
      safeMessage = import.meta.env.DEV
        ? `服务器错误 (${status}): ${error.response?.data?.message || error.message}`
        : '服务器繁忙，请稍后重试'
    } else if (status === 403) {
      safeMessage = '请求被拒绝，请检查权限或刷新页面后重试'
    } else {
      safeMessage = error.response?.data?.message || error.message || '请求失败'
    }

    const wrappedError = new Error(safeMessage)
    wrappedError.cause = { status, url: error.config?.url }
    return Promise.reject(wrappedError)
  }
)

export default api
