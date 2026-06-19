import { describe, expect, it } from 'vitest'
import { highlightSegments } from '@/features/search/utils/highlight'

describe('highlightSegments', () => {
  it('preserves HTML-like input as literal text', () => {
    const input = '<img src=x onerror=alert(1)>'
    const segments = highlightSegments(input, 'img')

    expect(segments.map((segment) => segment.text).join('')).toBe(input)
    expect(
      segments.filter((segment) => segment.highlighted).map((segment) => segment.text),
    ).toEqual(['img'])
  })
})
