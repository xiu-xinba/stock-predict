/** @module shared/utils/format - 通用格式化工具函数，涵盖数值、成交量、颜色、CSS 变量与涨跌方向判断 */

/**
 * 格式化指数/基金数值
 * 大于1万使用千分位，否则保留2位小数
 * @param val - 待格式化的数值
 * @returns 格式化后的字符串，无效值返回 '--'
 */
export function formatValue(val: number): string {
  if (val == null || isNaN(val)) return '--'
  if (val >= 10000) return val.toLocaleString('zh-CN', { maximumFractionDigits: 2 })
  return val.toFixed(2)
}

/**
 * 格式化成交量/成交额
 * @param vol - 成交量或成交额数值
 * @returns 格式化后的字符串，亿/万为单位，无效值返回 '--'
 */
export function formatVolume(vol: number): string {
  if (vol == null || isNaN(vol)) return '--'
  if (vol >= 1e8) return (vol / 1e8).toFixed(2) + '亿'
  if (vol >= 1e4) return (vol / 1e4).toFixed(0) + '万'
  return vol.toLocaleString()
}

/**
 * 将颜色值转换为带透明度的 rgba 格式
 * 支持 hex (#rrggbb) 和 rgb(r, g, b) 输入
 * @param color - 原始颜色值，支持 hex 或 rgb 格式
 * @param alpha - 透明度，范围 0-1
 * @returns rgba 格式颜色字符串，无法解析时返回原值
 */
export function colorWithAlpha(color: string, alpha: number): string {
  // Handle hex format
  if (color.startsWith('#')) {
    const hex = color.replace('#', '')
    const r = parseInt(hex.substring(0, 2), 16)
    const g = parseInt(hex.substring(2, 4), 16)
    const b = parseInt(hex.substring(4, 6), 16)
    return `rgba(${r}, ${g}, ${b}, ${alpha})`
  }
  // Handle rgb() format
  const rgbMatch = color.match(/rgb\((\d+),\s*(\d+),\s*(\d+)\)/)
  if (rgbMatch) {
    return `rgba(${rgbMatch[1]}, ${rgbMatch[2]}, ${rgbMatch[3]}, ${alpha})`
  }
  // Fallback: return as-is
  return color
}

const cssVarCache = new Map<string, string>()
let cssVarCacheTheme: boolean | null = null

/** 读取 CSS 自定义属性值，带主题感知缓存
 * @param name - CSS 变量名（如 '--color-brand'）
 * @param fallback - 变量不存在时的回退值，默认空字符串
 * @returns CSS 变量值或回退值
 */
export function cssVar(name: string, fallback: string = ''): string {
  if (typeof document === 'undefined') return fallback
  const isDark = document.documentElement.classList.contains('dark')
  if (cssVarCacheTheme !== isDark) {
    cssVarCache.clear()
    cssVarCacheTheme = isDark
  }
  const cached = cssVarCache.get(name)
  if (cached !== undefined) return cached
  const value = getComputedStyle(document.documentElement).getPropertyValue(name).trim() || fallback
  cssVarCache.set(name, value)
  return value
}

/** 使 CSS 变量缓存失效，应在主题切换后调用 */
export function invalidateCssVarCache() {
  cssVarCache.clear()
  cssVarCacheTheme = null
}

/** 根据数值正负返回涨跌方向 CSS 类名
 * @param val - 数值，正数为涨、负数为跌、零为平
 * @returns 对应的 CSS 类名 'text-up' | 'text-down' | 'text-flat'
 */
export function getDirection(
  val: number | null | undefined,
): 'text-up' | 'text-down' | 'text-flat' {
  if (val == null) return 'text-flat'
  if (val > 0) return 'text-up'
  if (val < 0) return 'text-down'
  return 'text-flat'
}

/** 将方向字符串转换为 CSS 类名
 * @param dir - 方向标识 'up' | 'down' 或其他
 * @returns 对应的 CSS 类名 'text-up' | 'text-down' | 'text-flat'
 */
export function dirClass(dir: string | null | undefined): 'text-up' | 'text-down' | 'text-flat' {
  if (dir === 'up') return 'text-up'
  if (dir === 'down') return 'text-down'
  return 'text-flat'
}

/** 格式化带正负号的百分比字符串
 * @param val - 百分比数值
 * @param digits - 小数位数，默认 4
 * @returns 带正负号的百分比字符串，无效值返回 '--%'
 */
export function formatSignedPct(val: number | null | undefined, digits: number = 4): string {
  if (val == null) return '--%'
  const sign = val >= 0 ? '+' : ''
  return `${sign}${Number(val).toFixed(digits)}%`
}
