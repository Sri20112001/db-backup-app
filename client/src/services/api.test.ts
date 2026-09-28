import { beforeEach, describe, expect, it, vi } from 'vitest'
import { authApi, orgApi } from './api'

type FetchResponder = { status: number; body?: unknown }

function mockFetch(queue: FetchResponder[]) {
  const calls: string[] = []
  const fn = vi.fn(async (url: string) => {
    calls.push(url)
    const next = queue.shift() ?? { status: 500, body: { error: 'no mock response' } }
    return {
      status: next.status,
      ok: next.status >= 200 && next.status < 300,
      json: async () => next.body ?? {},
    }
  })
  globalThis.fetch = fn as unknown as typeof fetch
  return calls
}

function stubBrowser() {
  const store: Record<string, string> = {}
  vi.stubGlobal('localStorage', {
    getItem: (k: string) => store[k] ?? null,
    setItem: (k: string, v: string) => { store[k] = v },
    removeItem: (k: string) => { delete store[k] },
    clear: () => { for (const k of Object.keys(store)) delete store[k] },
  })
  const location = { pathname: '/', href: '' }
  vi.stubGlobal('window', { location })
  return { store, location }
}

const flush = () => new Promise<void>((r) => setTimeout(r, 0))

describe('api auth retry', () => {
  beforeEach(() => {
    vi.unstubAllGlobals()
    stubBrowser()
  })

  it('failed login surfaces the server error without attempting refresh', async () => {
    const calls = mockFetch([{ status: 401, body: { error: 'invalid credentials' } }])
    await expect(authApi.login('a@x.io', 'wrong')).rejects.toThrow('invalid credentials')
    expect(calls).toHaveLength(1)
    expect(calls[0]).toContain('/auth/login')
  })

  it('expired access token refreshes once and retries the request', async () => {
    const { store } = stubBrowser()
    store['access_token'] = 'old-access'
    store['refresh_token'] = 'good-refresh'
    const calls = mockFetch([
      { status: 401, body: { error: 'invalid token' } },
      { status: 200, body: { access_token: 'new-access', refresh_token: 'new-refresh' } },
      { status: 200, body: [{ id: '1', name: 'Org' }] },
    ])
    const orgs = await orgApi.list()
    expect(orgs).toHaveLength(1)
    expect(calls.filter((u) => u.includes('/auth/refresh'))).toHaveLength(1)
    expect(store['access_token']).toBe('new-access')
    expect(store['refresh_token']).toBe('new-refresh')
  })

  it('missing refresh token redirects quietly instead of throwing', async () => {
    const { store, location } = stubBrowser()
    store['access_token'] = 'stale-access'
    const calls = mockFetch([{ status: 401, body: { error: 'invalid token' } }])
    // Do NOT await: the request intentionally never settles after redirect.
    void orgApi.list().then(
      () => { throw new Error('should not resolve') },
      () => { throw new Error('should not reject') },
    )
    await flush()
    await flush()
    expect(calls).toHaveLength(1)
    expect(location.href).toBe('/login')
  })

  it('rejected refresh clears storage, redirects, and throws session expired', async () => {
    const { store, location } = stubBrowser()
    store['access_token'] = 'old-access'
    store['refresh_token'] = 'dead-refresh'
    mockFetch([
      { status: 401, body: { error: 'invalid token' } },
      { status: 401, body: { error: 'refresh token not found or expired' } },
    ])
    await expect(orgApi.list()).rejects.toThrow('session expired')
    expect(store['access_token']).toBeUndefined()
    expect(store['refresh_token']).toBeUndefined()
    expect(location.href).toBe('/login')
  })

  it('concurrent 401s share a single refresh call', async () => {
    const { store } = stubBrowser()
    store['access_token'] = 'old-access'
    store['refresh_token'] = 'good-refresh'
    const calls = mockFetch([
      { status: 401, body: {} },
      { status: 401, body: {} },
      { status: 200, body: { access_token: 'n', refresh_token: 'm' } },
      { status: 200, body: [] },
      { status: 200, body: [] },
    ])
    const [a, b] = await Promise.all([orgApi.list(), orgApi.list()])
    expect(a).toEqual([])
    expect(b).toEqual([])
    expect(calls.filter((u) => u.includes('/auth/refresh'))).toHaveLength(1)
  })
})
