import { beforeEach, describe, expect, it, vi } from 'vitest'
import { authApi, orgApi } from './api'
import { useAuthStore } from '../store/authStore'
import { LOGIN_URL } from '../CONSTANTS'

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

// API calls excluding the fire-and-forget /auth/logout from store.logout().
function apiCalls(calls: string[]) {
  return calls.filter((u) => !u.includes('/auth/logout'))
}

function stubBrowser(pathname = '/') {
  const store: Record<string, string> = {}
  vi.stubGlobal('localStorage', {
    getItem: (k: string) => store[k] ?? null,
    setItem: (k: string, v: string) => { store[k] = v },
    removeItem: (k: string) => { delete store[k] },
    clear: () => { for (const k of Object.keys(store)) delete store[k] },
  })
  const location = { pathname, href: '' }
  vi.stubGlobal('window', { location })
  return { store, location }
}

function seedAuth(access: string | null, refresh: string | null) {
  useAuthStore.setState({
    user: access ? { id: 'u1', email: 'a@x.io', name: 'A' } : null,
    accessToken: access,
    refreshToken: refresh,
    currentOrg: null,
  })
}

const authTokens = () => ({
  accessToken: useAuthStore.getState().accessToken,
  refreshToken: useAuthStore.getState().refreshToken,
  user: useAuthStore.getState().user,
})

const flush = () => new Promise<void>((r) => setTimeout(r, 0))

describe('api auth retry', () => {
  beforeEach(() => {
    vi.unstubAllGlobals()
    stubBrowser()
    seedAuth(null, null)
  })

  it('failed login surfaces the server error without attempting refresh', async () => {
    const calls = mockFetch([{ status: 401, body: { error: 'invalid credentials' } }])
    await expect(authApi.login('a@x.io', 'wrong')).rejects.toThrow('invalid credentials')
    expect(apiCalls(calls)).toHaveLength(1)
    expect(apiCalls(calls)[0]).toContain('/auth/login')
    expect(authTokens().user).toBeNull()
  })

  it('expired access token refreshes once and retries the request', async () => {
    seedAuth('old-access', 'good-refresh')
    const calls = mockFetch([
      { status: 401, body: { error: 'invalid token' } },
      { status: 200, body: { access_token: 'new-access', refresh_token: 'new-refresh' } },
      { status: 200, body: [{ id: '1', name: 'Org' }] },
    ])
    const orgs = await orgApi.list()
    expect(orgs).toHaveLength(1)
    expect(apiCalls(calls).filter((u) => u.includes('/auth/refresh'))).toHaveLength(1)
    expect(authTokens().accessToken).toBe('new-access')
    expect(authTokens().refreshToken).toBe('new-refresh')
  })

  it('missing refresh token logs out and redirects quietly instead of throwing', async () => {
    seedAuth('stale-access', null)
    const { location } = stubBrowser()
    const calls = mockFetch([{ status: 401, body: { error: 'invalid token' } }])
    // Do NOT await: the request intentionally never settles after logout.
    void orgApi.list().then(
      () => { throw new Error('should not resolve') },
      () => { throw new Error('should not reject') },
    )
    await flush()
    await flush()
    expect(apiCalls(calls)).toHaveLength(1)
    expect(location.href).toBe(LOGIN_URL)
    expect(authTokens().accessToken).toBeNull()
    expect(authTokens().user).toBeNull()
  })

  it('rejected refresh logs out, redirects, and throws session expired', async () => {
    seedAuth('old-access', 'dead-refresh')
    const { location } = stubBrowser()
    mockFetch([
      { status: 401, body: { error: 'invalid token' } },
      { status: 401, body: { error: 'refresh token not found or expired' } },
    ])
    await expect(orgApi.list()).rejects.toThrow('session expired')
    expect(authTokens().accessToken).toBeNull()
    expect(authTokens().refreshToken).toBeNull()
    expect(location.href).toBe(LOGIN_URL)
  })

  it('retried request rejected again logs out and redirects quietly', async () => {
    seedAuth('old-access', 'good-refresh')
    const { location } = stubBrowser()
    const calls = mockFetch([
      { status: 401, body: { error: 'invalid token' } },
      { status: 200, body: { access_token: 'new-access', refresh_token: 'new-refresh' } },
      { status: 401, body: { error: 'invalid token' } },
    ])
    // Do NOT await: the request intentionally never settles after logout.
    void orgApi.list().then(
      () => { throw new Error('should not resolve') },
      () => { throw new Error('should not reject') },
    )
    await flush()
    await flush()
    await flush()
    expect(apiCalls(calls).filter((u) => u.includes('/auth/refresh'))).toHaveLength(1)
    expect(location.href).toBe(LOGIN_URL)
    expect(authTokens().accessToken).toBeNull()
    expect(authTokens().user).toBeNull()
  })

  it('concurrent 401s share a single refresh call', async () => {
    seedAuth('old-access', 'good-refresh')
    stubBrowser()
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
    expect(apiCalls(calls).filter((u) => u.includes('/auth/refresh'))).toHaveLength(1)
  })
})
