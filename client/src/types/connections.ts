import type { Agent, AgentLifecycle } from './agents'

export type ConnectionType = 'POSTGRES' | 'MONGODB' | 'MSSQL'
export type ConnectionStatus = 'CONNECTED' | 'DISCONNECTED' | 'ERROR' | 'UNKNOWN'

// DatabaseConnection is connection metadata only — the API never returns a
// password. `databases` carries cached details (sizes, table counts);
// -1 sentinels mean "unknown for this engine" and render as "—".
export interface DatabaseInfo {
  name: string
  size_bytes: number
  table_count: number
}

export interface TableInfo {
  schema?: string
  name: string
  rows: number
  size_bytes: number
}

export interface DatabaseConnection {
  id: string
  organization_id: string
  agent_id: string
  name: string
  type: ConnectionType
  host: string
  port: number
  username: string
  status: ConnectionStatus
  last_checked_at: string | null
  databases: DatabaseInfo[]
  database_count: number
  agent?: Agent
  /** Enrollment-aware agent state (PENDING/ONLINE/OFFLINE/REVOKED) — the
   *  cached `status` above only reflects the last server-side check. */
  agent_lifecycle?: AgentLifecycle
  created_at: string
  updated_at?: string
}

export interface CreateConnectionRequest {
  name: string
  type: ConnectionType
  agent_id: string
  host: string
  port: number
  username: string
  password: string
  databases?: string[]
}

export interface TestConnectionResult {
  status: string
  latency_ms?: number
  databases?: DatabaseInfo[]
  agent_status?: string
  error?: string
  stage?: string
}

export interface DatabaseListResult {
  databases: DatabaseInfo[]
  cached: boolean
  agent_status?: string
}

export interface TableListResult {
  database: string
  tables: TableInfo[]
}
