/**
 * Vite 环境变量类型声明
 *
 * 本文件为 Vite 项目提供全局类型补充：
 * 1. 声明 .vue 单文件组件模块类型，使 TypeScript 能正确识别 import
 * 2. 扩展 ImportMetaEnv 接口，为自定义环境变量提供类型提示
 *
 * 注意：VITE_ 前缀的环境变量通过 import.meta.env 访问，
 * 只有在此声明过的变量才具有类型提示和类型检查
 */
/// <reference types="vite/client" />

/** 声明 .vue 单文件组件的模块类型，避免 TS2307 报错 */
declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{}, {}, any>
  export default component
}

/** 自定义环境变量类型声明（VITE_ 前缀） */
interface ImportMetaEnv {
  /** 后端 API 基础地址，如不设置则默认使用 Vite proxy 转发到 localhost:8000 */
  readonly VITE_API_BASE_URL?: string
  /** 是否为 API 请求携带 Cookie 凭据，只有配置为 "true" 时启用 */
  readonly VITE_API_WITH_CREDENTIALS?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
