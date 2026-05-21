/**
 * 格式化指数/基金数值
 * 大于1万使用千分位，否则保留2位小数
 */
export function formatValue(val: number): string {
  if (val == null || isNaN(val)) return '--'
  if (val >= 10000) return val.toLocaleString('zh-CN', { maximumFractionDigits: 2 })
  return val.toFixed(2)
}

/**
 * 格式化成交量/成交额
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
