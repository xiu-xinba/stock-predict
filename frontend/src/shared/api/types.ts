/** @module shared/api/types - API 响应类型定义 */

/** 后端统一响应包装结构 */
export interface ApiResponse<T> {
  /** 业务状态码，0 表示成功 */
  code: number
  /** 面向用户的消息或错误描述 */
  message: string
  /** 响应数据载荷，请求失败时为 null */
  data: T | null
}
