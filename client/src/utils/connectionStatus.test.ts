import { describe, expect, it } from 'vitest'
import {
  agentLifecycleOf,
  badgeDotClass,
  badgeToneClass,
  connectionBadge,
} from './connectionStatus'
import type { Agent, AgentLifecycle, DatabaseConnection } from '@/types'

const agent = (over: Partial<Agent> = {}): Agent => ({
  id: 'agent-1',
  organization_id: 'org-1',
  name: 'db-host-01',
  platform: 'linux',
  architecture: 'amd64',
  machine_name: 'db-host-01',
  status: 'ONLINE',
  version: '1.0.0',
  last_seen_at: '2026-10-26T10:00:00Z',
  installed_at: '2026-10-01T10:00:00Z',
  revoked_at: null,
  created_at: '2026-10-01T10:00:00Z',
  ...over,
})

const connection = (over: Partial<DatabaseConnection> = {}): DatabaseConnection => ({
  id: 'conn-1',
  organization_id: 'org-1',
  agent_id: 'agent-1',
  name: 'prod-pg',
  type: 'POSTGRES',
  host: 'localhost',
  port: 5432,
  username: 'postgres',
  status: 'CONNECTED',
  last_checked_at: '2026-10-26T10:00:00Z',
  databases: [],
  database_count: 0,
  created_at: '2026-10-01T10:00:00Z',
  ...over,
})

describe('agentLifecycleOf', () => {
  it('returns undefined when there is no agent', () => {
    expect(agentLifecycleOf(undefined)).toBeUndefined()
    expect(agentLifecycleOf(null)).toBeUndefined()
  })

  it('prefers an explicit lifecycle over raw fields', () => {
    // Installed and online in the raw fields, but the server said PENDING.
    expect(agentLifecycleOf(agent({ lifecycle: 'PENDING', installed_at: '2026-10-01T10:00:00Z', status: 'ONLINE' }))).toBe('PENDING')
  })

  it('derives REVOKED, PENDING, ONLINE and OFFLINE from the raw agent record', () => {
    expect(agentLifecycleOf(agent({ revoked_at: '2026-10-20T10:00:00Z' }))).toBe('REVOKED')
    expect(agentLifecycleOf(agent({ installed_at: null }))).toBe('PENDING')
    expect(agentLifecycleOf(agent({ status: 'ONLINE' }))).toBe('ONLINE')
    expect(agentLifecycleOf(agent({ status: 'OFFLINE' }))).toBe('OFFLINE')
  })
})

describe('connectionBadge', () => {
  it('trusts the cached status when the agent is online', () => {
    const badge = connectionBadge(connection({ agent_lifecycle: 'ONLINE', status: 'CONNECTED' }))
    expect(badge).toMatchObject({ label: 'Connected', tone: 'ok', trusted: true })
  })

  it('maps each cached status to its own badge when the agent is online', () => {
    expect(connectionBadge(connection({ agent_lifecycle: 'ONLINE', status: 'ERROR' })))
      .toMatchObject({ label: 'Error', tone: 'error', trusted: true })
    expect(connectionBadge(connection({ agent_lifecycle: 'ONLINE', status: 'DISCONNECTED' })))
      .toMatchObject({ label: 'Disconnected', tone: 'muted', trusted: true })
    expect(connectionBadge(connection({ agent_lifecycle: 'ONLINE', status: 'UNKNOWN' })))
      .toMatchObject({ label: 'Not checked yet', tone: 'muted', trusted: true })
  })

  it('does not trust a cached CONNECTED while the agent is not online', () => {
    const expectations: Array<[AgentLifecycle, string]> = [
      ['PENDING', 'Agent pending'],
      ['OFFLINE', 'Agent offline'],
      ['REVOKED', 'Agent revoked'],
    ]
    for (const [lifecycle, label] of expectations) {
      const badge = connectionBadge(connection({ agent_lifecycle: lifecycle, status: 'CONNECTED' }))
      expect(badge.label).toBe(label)
      expect(badge.tone).toBe('muted')
      expect(badge.trusted).toBe(false)
      expect(badge.hint).toContain('cached result')
    }
  })

  it('falls back to the embedded agent when agent_lifecycle is absent', () => {
    const badge = connectionBadge(connection({ agent: agent({ status: 'OFFLINE' }) }))
    expect(badge).toMatchObject({ label: 'Agent offline', trusted: false })
  })

  it('falls back to the cached status when no agent data exists at all', () => {
    const badge = connectionBadge(connection({ status: 'CONNECTED' }))
    expect(badge).toMatchObject({ label: 'Connected', trusted: true })
  })
})

describe('badge classes', () => {
  it('defines token classes for every tone', () => {
    for (const tone of ['ok', 'error', 'muted'] as const) {
      expect(badgeToneClass[tone]).toBeTruthy()
      expect(badgeDotClass[tone]).toBeTruthy()
      // All M3 tokens — no hardcoded hex sneaking back in.
      expect(badgeToneClass[tone]).not.toMatch(/#[0-9a-fA-F]{3,8}\b/)
      expect(badgeDotClass[tone]).not.toMatch(/#[0-9a-fA-F]{3,8}\b/)
    }
  })
})
