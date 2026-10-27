export type AgentStatus = 'ONLINE' | 'OFFLINE'
export type AgentLifecycle = 'PENDING' | 'ONLINE' | 'OFFLINE' | 'REVOKED'

export interface Agent {
  id: string
  organization_id: string
  name: string
  platform: string
  architecture: string
  machine_name: string
  status: AgentStatus
  lifecycle?: AgentLifecycle
  version: string
  last_seen_at: string | null
  installed_at: string | null
  revoked_at: string | null
  created_at: string
}

export interface EnrollmentTokenResult {
  agent_id: string
  enrollment_token: string
  expires_at: string
}

// EnrollmentTokenInfo is list metadata only — hashes never reach the client.
export interface EnrollmentTokenInfo {
  id: string
  organization_id: string
  agent_id: string
  expires_at: string
  used_at: string | null
  revoked_at: string | null
  created_at: string
}

export interface Machine {
  id: string
  organization_id: string
  agent_id: string
  hostname: string
  os: string
  ip_address: string
  agent: Agent
}
