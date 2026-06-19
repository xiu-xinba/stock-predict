/** @module shared/api/client - HTTP 客户端封装，基于 Axios 提供请求去重、CSRF 防护、自动重试与统一错误处理 */
import axios from 'axios'
import type { AxiosRequestConfig, AxiosResponse } from 'axios'
import type { ApiResponse } from '@/shared/api/types'
import type { AppError } from '@/shared/types/errors'
import { API_ROUTES } from '@/shared/api/routes'

declare module 'axios' {
  interface AxiosRequestConfig {
    __retryCount?: number
    skipDedup?: boolean
  }
}

/** 请求被取消时抛出的错误类型 */
export class CancelError extends Error {
  constructor(message: string = 'Request cancelled') {
    super(message)
    this.name = 'CancelError'
  }
}

/** 预配置的 Axios 实例，包含 baseURL、超时、凭证及请求/响应拦截器 */
const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  timeout: 15000,
  withCredentials: import.meta.env.VITE_API_WITH_CREDENTIALS !== 'false',
})

interface PendingRequest {
  controller: AbortController
  cleanup?: () => void
}

const pendingRequests = new Map<string, PendingRequest>()

function getRequestKey(config: AxiosRequestConfig): string {
  // Include request body for POST dedup key so different payloads aren't treated as same request
  const method = (config.method ?? 'get').toLowerCase()
  const bodyKey =
    method === 'post' || method === 'put' || method === 'patch'
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

function linkAbortSignals(
  source: AxiosRequestConfig['signal'],
  target: AbortController,
): (() => void) | undefined {
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

const AUTH_TOKEN_KEY = 'auth_token'
let csrfToken = ''

/** 从浏览器 cookie 中读取指定名称的值
 * @param name - Cookie 名称
 * @returns Cookie 值，不存在时返回空字符串
 */
export function getCookie(name: string): string {
  const escapedName = name.replace(/([.*+?^${}()|[\]\\])/g, '\\$1')
  const match = document.cookie.match(new RegExp(`(?:^|;\\s*)${escapedName}=([^;]*)`))
  return match ? decodeURIComponent(match[1]) : ''
}

api.interceptors.request.use((config) => {
  const token = localStorage.getItem(AUTH_TOKEN_KEY)
  if (token && config.headers) {
    config.headers['Authorization'] = `Bearer ${token}`
  }

  const method = (config.method ?? 'get').toLowerCase()
  if (['post', 'put', 'delete', 'patch'].includes(method)) {
    if (csrfToken && config.headers) {
      config.headers['X-CSRF-Token'] = csrfToken
    }
  }

  if ((method === 'get' || method === 'post') && !config.skipDedup) {
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
    const responseCSRFToken = response.headers['x-csrf-token']
    if (typeof responseCSRFToken === 'string' && responseCSRFToken) {
      csrfToken = responseCSRFToken
    }
    return response
  },
  async (error) => {
    const config = error.config

    if (config) {
      cleanupPendingRequest(config)
    }

    if (axios.isCancel(error) || error.code === 'ERR_CANCELED') {
      return Promise.reject(new CancelError())
    }

    const status = error.response?.status

    if (status === 401) {
      localStorage.removeItem(AUTH_TOKEN_KEY)
      window.dispatchEvent(new CustomEvent('auth:expired'))
    }
    const method = (config?.method ?? 'get').toLowerCase()
    const isRetryable = method === 'get' && status >= 500 && status < 600
    const retryCount = config?.__retryCount ?? 0
    const MAX_RETRIES = 2

    if (isRetryable && retryCount < MAX_RETRIES) {
      config.__retryCount = retryCount + 1
      const delay = 500 * config.__retryCount
      await new Promise<void>((resolve) => setTimeout(resolve, delay))
      return api(config)
    }

    let safeMessage: string
    let errorType: AppError['type']
    let retryable = false

    if (!status) {
      safeMessage = '网络连接失败，请检查网络'
      errorType =
        error.code === 'ECONNABORTED' || error.code === 'ERR_CANCELED' ? 'timeout' : 'network'
      retryable = true
    } else if (status === 401) {
      safeMessage = '登录已过期'
      errorType = 'business'
    } else if (status === 403) {
      safeMessage = '没有权限'
      errorType = 'business'
    } else if (status === 404) {
      safeMessage = '请求的资源不存在'
      errorType = 'business'
    } else if (status === 429) {
      safeMessage = '请求过于频繁，请稍后重试'
      errorType = 'business'
      retryable = true
    } else if (status >= 500) {
      safeMessage = import.meta.env.DEV
        ? `服务器错误 (${status}): ${error.response?.data?.message || error.message}`
        : '服务器繁忙，请稍后重试'
      errorType = 'server'
      retryable = true
    } else {
      safeMessage = error.response?.data?.message || error.message || '请求失败'
      errorType = 'unknown'
    }

    const appError: AppError = {
      code: status || 0,
      message: safeMessage,
      retryable,
      type: errorType,
    }
    return Promise.reject(appError)
  },
)

export default api

/** 请求后端重启进程，先返回成功响应后异步执行重启 */
export async function restartBackend(): Promise<boolean> {
  try {
    await api.post(API_ROUTES.admin.restart, {}, { timeout: 5000, skipDedup: true })
    return true
  } catch {
    return false
  }
}

/** 检测后端服务健康状态
 * @returns 后端是否正常运行，请求失败时返回 false
 */
export async function checkBackendHealth(): Promise<boolean> {
  try {
    const { data } = await api.get<ApiResponse<{ status: string }>>(API_ROUTES.health, {
      timeout: 5000,
      skipDedup: true,
    })
    // 容错：支持标准 ApiResponse（code===0）和裸格式（status==='ok'）
    const payload = data as ApiResponse<{ status: string }> & { status?: string }
    return data.code === 0 || payload.status === 'ok'
  } catch {
    return false
  }
}
