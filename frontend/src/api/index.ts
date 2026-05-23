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
  withCredentials: import.meta.env.VITE_API_WITH_CREDENTIALS === 'true',
})

interface PendingRequest {
  controller: AbortController
  cleanup?: () => void
}

const pendingRequests = new Map<string, PendingRequest>()

function getRequestKey(config: AxiosRequestConfig): string {
  // Include request body for POST dedup key so different payloads aren't treated as same request
  const method = (config.method ?? 'get').toLowerCase()
  const bodyKey = method === 'post' || method === 'put' || method === 'patch'
    ? JSON.stringify(config.data ?? {})
    : ''
  return `${method}:${config.baseURL}${config.url}?${JSON.stringify(config.params ?? {})}${bodyKey}`
}

function cleanupPendingRequest(config: AxiosRequestConfig) {
  const key = getRequestKey(config)
  const pending = pendingRequests.get(key)
  if (pending && config.signal && pending.controller.signal !== config.signal) return
  pending?.cleanup?.()
  pendingRequests.delete(key)
}

function linkAbortSignals(source: AxiosRequestConfig['signal'], target: AbortController): (() => void) | undefined {
  if (!source) return undefined
  if (source.aborted) {
    target.abort()
    return undefined
  }
  const addAbortListener = source.addEventListener
  const removeAbortListener = source.removeEventListener
  if (!addAbortListener || !removeAbortListener) return undefined
  const abortTarget = () => target.abort()
  addAbortListener.call(source, 'abort', abortTarget, { once: true })
  return () => removeAbortListener.call(source, 'abort', abortTarget)
}

api.interceptors.request.use((config) => {
  const csrfMatch = document.cookie.match(/csrftoken=([^;]+)/)
  if (csrfMatch && config.headers) {
    config.headers['X-CSRFToken'] = csrfMatch[1]
  }

  const method = (config.method ?? 'get').toLowerCase()
  if (method === 'get' || method === 'post') {
    const key = getRequestKey(config)
    const existing = pendingRequests.get(key)
    if (existing) {
      existing.cleanup?.()
      existing.controller.abort()
    }

    const callerSignal = config.signal
    const controller = new AbortController()
    const cleanup = linkAbortSignals(callerSignal, controller)
    config.signal = controller.signal
    pendingRequests.set(key, { controller, cleanup })
  }

  return config
})

api.interceptors.response.use(
  (response: AxiosResponse) => {
    cleanupPendingRequest(response.config)
    return response
  },
  (error) => {
    if (error.config) {
      cleanupPendingRequest(error.config)
    }

    if (axios.isCancel(error) || error.code === 'ERR_CANCELED') {
      return Promise.reject(new CancelError('Request cancelled'))
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
