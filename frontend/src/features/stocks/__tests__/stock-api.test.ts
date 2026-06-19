import { beforeEach, describe, expect, it, vi } from 'vitest'

const postMock = vi.hoisted(() => vi.fn())

vi.mock('@/shared/api/client', () => ({
  default: {
    post: postMock,
  },
}))

import { fetchStockQuotes } from '@/features/stocks/api/stocks'

describe('stock api', () => {
  beforeEach(() => {
    postMock.mockReset()
    postMock.mockResolvedValue({ data: { code: 0, message: 'ok', data: {} } })
  })

  it('sends realtime freshness for stock quotes when requested', async () => {
    await fetchStockQuotes(['600519'], 'realtime')

    expect(postMock).toHaveBeenCalledWith('/stocks/quotes', {
      codes: ['600519'],
      freshness: 'realtime',
    })
  })
})
