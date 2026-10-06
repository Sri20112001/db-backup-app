import { describe, expect, it, vi, beforeEach } from 'vitest'
import { agentApi } from './api'
import { useAuthStore } from '../store/authStore'

function mockFetch() {
  const calls: { url: string; method: string; body?: unknown }[] = []
  const responses: { status: number; body?: unknown }[] = []
  const fn = vi.fn(async (url: string, init?: RequestInit) => {
    let body: unknown
    try {
      body = init?.body ? JSON.parse(init.body as string) : undefined
    } catch {
      body = undefined
    }
    calls.push({ url, method: init?.method ?? 'GET', body })
    const next = responses.shift() ?? { status: 500, body: { error: 'no mock' } }
    return {
      status: next.status,
      ok: next.status >= 200 && next.status < 300,
      json: async () => next.body ?? {},
    }
  })
  globalThis.fetch = fn as unknown as typeof fetch
  return { calls, responses }
}

beforeEach(() => {
  vi.unstubAllGlobals()
  vi.stubGlobal('window', { location: { pathname: '/', href: '' } })
  useAuthStore.setState({
    user: { id: 'u1', email: 'a@x.io', name: 'A' },
    accessToken: 'test-access',
    refreshToken: 'test-refresh',
    currentOrg: null,
  })
})

describe('agentApi enrollment management', () => {
  it('lists tokens with GET on the collection', async () => {
    const { calls, responses } = mockFetch()
    responses.push({ status: 200, body: [] })
    await agentApi.listTokens('org-1')
    expect(calls).toHaveLength(1)
    expect(calls[0].method).toBe('GET')
    expect(calls[0].url).toContain('/organizations/org-1/agents/enrollment-tokens')
  })

  it('revokes a token with DELETE on the member route', async () => {
    const { calls, responses } = mockFetch()
    responses.push({ status: 204, body: undefined })
    await agentApi.revokeToken('org-1', 'tok-1')
    expect(calls).toHaveLength(1)
    expect(calls[0].method).toBe('DELETE')
    expect(calls[0].url).toContain('/organizations/org-1/agents/enrollment-tokens/tok-1')
  })

  it('renames with PUT carrying only the name', async () => {
    const { calls, responses } = mockFetch()
    responses.push({ status: 200, body: { id: 'a1', name: 'New' } })
    await agentApi.rename('org-1', 'a1', 'New')
    expect(calls).toHaveLength(1)
    expect(calls[0].method).toBe('PUT')
    expect(calls[0].url).toContain('/organizations/org-1/agents/a1')
    expect(calls[0].body).toEqual({ name: 'New' })
  })
})
