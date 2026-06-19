import { describe, expect, it } from 'vitest'
import { isMarketInSession } from '@/features/market/utils/marketTime'

describe('isMarketInSession', () => {
  it('uses New York daylight-saving time for US sessions', () => {
    expect(isMarketInSession('us', new Date('2026-01-12T15:00:00Z'))).toBe(true)
    expect(isMarketInSession('us', new Date('2026-06-10T14:00:00Z'))).toBe(true)
    expect(isMarketInSession('us', new Date('2026-06-13T14:00:00Z'))).toBe(false)
  })
})
