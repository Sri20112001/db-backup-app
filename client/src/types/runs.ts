import type { BackupJob } from './jobs'

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
  retry_of_run_id?: string | null
  retry_attempt: number
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
  verification_status: string
  verified_at?: string | null
}
