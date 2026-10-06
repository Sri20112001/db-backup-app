export interface User {
  id: string
  email: string
  name: string
}

export interface Organization {
  id: string
  name: string
  slug: string
  user_limit: number
}

export interface OrganizationMember {
  id: string
  organization_id: string
  user_id: string
  role: MemberRole
  user: User
}

export type MemberRole = 'OWNER' | 'ADMIN' | 'OPERATOR' | 'VIEWER'

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

export type StorageType = 'S3' | 'LOCAL' | 'SMB'

export interface S3Region {
  id: string
  code: string
  name: string
  provider: string
  endpoint?: string
  is_system: boolean
  active: boolean
}

export interface StorageTarget {
  id: string
  organization_id: string
  name: string
  type: StorageType
  bucket: string
  region: string
  endpoint: string
  use_path_style: boolean
  path: string
  has_credentials: boolean
  created_at: string
}

export type BackupSourceType = 'FILESYSTEM' | 'MSSQL_SERVER' | 'DBF' | 'POSTGRES' | 'MONGODB'
export type BackupMode = 'NORMAL' | 'COMPRESSED'

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

export interface BackupSchedule {
  id: string
  backup_job_id: string
  cron_expr: string
  timezone: string
}

export interface BackupJob {
  id: string
  organization_id: string
  agent_id: string
  storage_target_id: string
  connection_id?: string | null
  name: string
  source_type: BackupSourceType
  source_path: string
  source_database: string
  include_patterns: string
  exclude_patterns: string
  mode: BackupMode
  encrypted: boolean
  retention_days: number
  export_format: string
  enabled: boolean
  sla_target_minutes: number
  rpo_target_minutes: number
  rto_target_minutes: number
  schedule?: BackupSchedule
  agent: Agent
  storage_target: StorageTarget
  created_at: string
}

export type BackupRunStatus =
  | 'PENDING'
  | 'RUNNING'
  | 'UPLOADING'
  | 'VERIFYING'
  | 'COMPLETED'
  | 'FAILED'
  | 'CANCELLED'

export interface BackupRun {
  id: string
  organization_id: string
  backup_job_id: string
  agent_id: string
  storage_target_id: string
  status: BackupRunStatus
  started_at: string | null
  completed_at: string | null
  bytes_read: number
  bytes_compressed: number
  bytes_uploaded: number
  duration_seconds: number
  source_type: string
  error_message: string
  failure_category: string
  storage_path: string
  checksum: string
  backup_job: BackupJob
  created_at: string
}

export interface BackupArtifact {
  id: string
  backup_run_id: string
  name: string
  size: number
  checksum: string
  storage_path: string
}

export type RestoreStatus = 'PENDING' | 'RUNNING' | 'COMPLETED' | 'FAILED'

export interface RestoreJob {
  id: string
  organization_id: string
  backup_run_id: string
  agent_id: string
  connection_id?: string | null
  status: RestoreStatus
  destination_path: string
  target_database: string
  started_at: string | null
  completed_at: string | null
  error_message: string
  backup_run: BackupRun
  created_at: string
}

export type AlertType =
  | 'BACKUP_FAILED'
  | 'BACKUP_MISSED'
  | 'AGENT_OFFLINE'
  | 'STORAGE_LOW'
  | 'RESTORE_FAILED'
  | 'VERIFY_FAILED'

export interface Alert {
  id: string
  organization_id: string
  type: AlertType
  title: string
  message: string
  read: boolean
  backup_job_id?: string
  agent_id?: string
  created_at: string
}

export interface DashboardOverview {
  total_jobs: number
  enabled_jobs: number
  agents_online: number
  agents_offline: number
  successful_runs: number
  failed_runs: number
  total_bytes: number
  recent_runs: BackupRun[]
  sla_breaches_24h: number
  size_anomalies: SizeAnomaly[]
}

export interface SizeAnomaly {
  job_id: string
  job_name: string
  run_id: string
  actual_bytes: number
  avg_bytes: number
  pct_change: number
}

export interface PreflightCheck {
  name: string
  status: 'OK' | 'WARN' | 'FAIL'
  detail?: string
}

export interface PreflightResult {
  overall: 'OK' | 'WARN' | 'FAIL'
  checks: PreflightCheck[]
}

export interface JobHealth {
  job_id: string
  job_name: string
  status: 'HEALTHY' | 'WARNING' | 'FAILED' | 'UNKNOWN'
  last_success_at: string | null
  last_failure_at: string | null
  recovery_points: number
  next_run_at: string | null
  sla_target_minutes: number
  last_duration_seconds: number
  sla_status: 'OK' | 'BREACH' | 'UNKNOWN'
  rpo_target_minutes: number
  actual_rpo_minutes: number
  rpo_status: 'OK' | 'BREACH' | 'UNKNOWN'
  rto_target_minutes: number
  last_restore_seconds: number
  rto_status: 'OK' | 'BREACH' | 'UNKNOWN' | 'NO_DATA'
  size_anomaly_pct: number | null
}

export interface PaginatedResponse<T> {
  data: T[]
  total: number
  page: number
  limit: number
}

export interface AuthTokens {
  access_token: string
  refresh_token: string
  user: User
}
