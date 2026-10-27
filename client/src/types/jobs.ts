import type { Agent } from './agents'
import type { StorageTarget } from './storage'
import type { BackupPolicy, BackupMode } from './policies'
import type { BackupRun } from './runs'
import type { PaginatedResponse } from './common'

export type BackupSourceType = 'FILESYSTEM' | 'MSSQL_SERVER' | 'DBF' | 'POSTGRES' | 'MONGODB'

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
  policy_id?: string | null
  policy?: BackupPolicy | null
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

export interface SizeAnomaly {
  job_id: string
  job_name: string
  run_id: string
  actual_bytes: number
  avg_bytes: number
  pct_change: number
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

export interface ListCounts {
  total: number
  enabled: number
  disabled: number
  by_source: Record<string, number>
}

export interface JobListResponse extends PaginatedResponse<BackupJob> {
  counts: ListCounts
}
