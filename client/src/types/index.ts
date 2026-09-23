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

export interface Agent {
  id: string
  organization_id: string
  name: string
  status: AgentStatus
  version: string
  last_seen_at: string | null
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

export interface StorageTarget {
  id: string
  organization_id: string
  name: string
  type: StorageType
  bucket: string
  region: string
  endpoint: string
  path: string
  created_at: string
}

export type BackupSourceType = 'FILESYSTEM' | 'MSSQL_SERVER' | 'DBF' | 'POSTGRES'
export type BackupMode = 'NORMAL' | 'COMPRESSED'

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
  name: string
  source_type: BackupSourceType
  source_path: string
  source_database: string
  include_patterns: string
  exclude_patterns: string
  mode: BackupMode
  encrypted: boolean
  retention_days: number
  enabled: boolean
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
}

export interface JobHealth {
  job_id: string
  job_name: string
  status: 'HEALTHY' | 'WARNING' | 'FAILED' | 'UNKNOWN'
  last_success_at: string | null
  last_failure_at: string | null
  recovery_points: number
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
