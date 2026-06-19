import axios, { AxiosHeaders, type AxiosAdapter, type AxiosResponse } from 'axios'
import { afterEach, describe, expect, it } from 'vitest'
import api, { CancelError } from '@/shared/api/client'

const originalAdapter = api.defaults.adapter

afterEach(() => {
  api.defaults.adapter = originalAdapter
})

describe('api client', () => {
  it('rejects cancelled requests with CancelError', async () => {
    api.defaults.adapter = (async (_config) => {
      throw new axios.CanceledError('cancelled')
    }) as AxiosAdapter

    await expect(api.get('/cancelled')).rejects.toBeInstanceOf(CancelError)
  })

  it('reuses the CSRF response header for mutations', async () => {
    let mutationToken = ''
    api.defaults.adapter = (async (config) => {
      if (config.method === 'post') {
        mutationToken = String(config.headers?.get('X-CSRF-Token') ?? '')
      }
      return {
        config,
        data: { code: 0, message: 'success', data: null },
        headers: new AxiosHeaders(
          config.method === 'get' ? { 'x-csrf-token': 'header-token' } : {},
        ),
        status: 200,
        statusText: 'OK',
      } satisfies AxiosResponse
    }) as AxiosAdapter

    await api.get('/health', { skipDedup: true })
    await api.post('/mutation', {}, { skipDedup: true })

    expect(mutationToken).toBe('header-token')
  })
})
