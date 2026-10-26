import type { Agent, AgentLifecycle, ConnectionStatus, DatabaseConnection } from '@/types'

// The badge shown next to a connection. A connection's own `status` is only a
// cached snapshot of the last server-side Test/Refresh — the server dials the
// database directly, so that value stays "CONNECTED" forever even after the
// agent that actually runs backups has gone pending, offline, or revoked.
// The badge therefore leads with the agent lifecycle: without an ONLINE agent
// nothing can reach the database, so the cached status can't be trusted.
export interface ConnectionBadge {
  label: string
  tone: 'ok' | 'error' | 'muted'
  /** Plain-language explanation, used as the badge's tooltip. */
  hint: string
  /** False when the cached CONNECTED claim is stale (agent not online). */
  trusted: boolean
}

// Same derivation the server uses for the transient `lifecycle` field, for
// older payloads that only carry the raw agent record.
export function agentLifecycleOf(agent?: Partial<Agent> | null): AgentLifecycle | undefined {
  if (!agent) return undefined
  if (agent.lifecycle) return agent.lifecycle
  if (agent.revoked_at) return 'REVOKED'
  if (!agent.installed_at) return 'PENDING'
  return agent.status === 'ONLINE' ? 'ONLINE' : 'OFFLINE'
}

const lastChecked = (c: DatabaseConnection): string =>
  c.last_checked_at ? `last verified ${new Date(c.last_checked_at).toLocaleString()}` : 'never verified'

const lifecycleBadge = (lifecycle: AgentLifecycle, c: DatabaseConnection): ConnectionBadge => {
  const stale = `This is the cached result of the last connection check (${lastChecked(c)}), taken while the agent was reachable.`
  switch (lifecycle) {
    case 'ONLINE':
      return statusBadge(c)
    case 'PENDING':
      return {
        label: 'Agent pending',
        tone: 'muted',
        hint: `The agent has been created but never enrolled, so no backup can run yet. ${stale}`,
        trusted: false,
      }
    case 'REVOKED':
      return {
        label: 'Agent revoked',
        tone: 'muted',
        hint: `The agent's credentials were revoked, so it can no longer authenticate. ${stale}`,
        trusted: false,
      }
    default:
      return {
        label: 'Agent offline',
        tone: 'muted',
        hint: `The agent hasn't reported in recently, so backups and connection checks can't run. ${stale}`,
        trusted: false,
      }
  }
}

function statusBadge(c: DatabaseConnection): ConnectionBadge {
  switch (c.status as ConnectionStatus) {
    case 'CONNECTED':
      return {
        label: 'Connected',
        tone: 'ok',
        hint: `The last connection check succeeded (${lastChecked(c)}).`,
        trusted: true,
      }
    case 'ERROR':
      return {
        label: 'Error',
        tone: 'error',
        hint: `The last connection check failed (${lastChecked(c)}).`,
        trusted: true,
      }
    case 'DISCONNECTED':
      return {
        label: 'Disconnected',
        tone: 'muted',
        hint: `The database refused the last connection attempt (${lastChecked(c)}).`,
        trusted: true,
      }
    default:
      return {
        label: 'Not checked yet',
        tone: 'muted',
        hint: 'Run a connection test to see whether the database is reachable.',
        trusted: true,
      }
  }
}

// Prefers the server-computed `agent_lifecycle`, falling back to the embedded
// agent record (and finally to the raw cached status when no agent is known).
export function connectionBadge(c: DatabaseConnection): ConnectionBadge {
  const lifecycle = c.agent_lifecycle ?? agentLifecycleOf(c.agent)
  if (lifecycle && lifecycle !== 'ONLINE') return lifecycleBadge(lifecycle, c)
  return statusBadge(c)
}

// Tailwind classes for a badge tone — all M3 tokens, dark-mode safe.
export const badgeToneClass: Record<ConnectionBadge['tone'], string> = {
  ok: 'bg-success-container text-on-success-container',
  error: 'bg-error-container text-on-error-container',
  muted: 'bg-surface-container-high text-on-surface-variant border border-surface-variant',
}

export const badgeDotClass: Record<ConnectionBadge['tone'], string> = {
  ok: 'bg-success',
  error: 'bg-error',
  muted: 'bg-surface-variant',
}
