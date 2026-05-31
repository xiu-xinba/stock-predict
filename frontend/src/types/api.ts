/** Back-end unified response wrapper. */
export interface ApiResponse<T> {
  /** Business status code, 0 means success. */
  code: number
  /** User-facing message or error description. */
  message: string
  /** Response payload; null when the request fails. */
  data: T | null
}
