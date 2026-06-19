/** @module search/utils/highlight — 搜索关键词高亮工具 */

/** 高亮文本片段 */
export interface HighlightSegment {
  /** 片段文本 */
  text: string
  /** 是否为高亮匹配部分 */
  highlighted: boolean
}

/**
 * 将文本按搜索关键词拆分为高亮片段数组
 * @param text - 原始文本
 * @param query - 搜索关键词
 * @returns 高亮片段数组
 */
export function highlightSegments(text: string, query: string): HighlightSegment[] {
  const needle = query.trim()
  if (!needle) return [{ text, highlighted: false }]

  const lowerText = text.toLocaleLowerCase()
  const lowerNeedle = needle.toLocaleLowerCase()
  const segments: HighlightSegment[] = []
  let cursor = 0

  while (cursor < text.length) {
    const matchIndex = lowerText.indexOf(lowerNeedle, cursor)
    if (matchIndex < 0) {
      segments.push({ text: text.slice(cursor), highlighted: false })
      break
    }
    if (matchIndex > cursor) {
      segments.push({ text: text.slice(cursor, matchIndex), highlighted: false })
    }
    const end = matchIndex + needle.length
    segments.push({ text: text.slice(matchIndex, end), highlighted: true })
    cursor = end
  }

  return segments.length > 0 ? segments : [{ text, highlighted: false }]
}
