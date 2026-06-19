/** @module shared/types/errors - 应用级错误类型定义 */

/** 应用统一错误结构，用于 HTTP 拦截器与业务层的错误传递 */
export interface AppError {
  /** HTTP 状态码或自定义错误码，无响应时为 0 */
  code: number
  /** 面向用户的错误消息 */
  message: string
  /** 该错误是否可重试 */
  retryable: boolean
  /** 错误类型分类 */
  type: 'network' | 'server' | 'business' | 'timeout' | 'unknown'
}
