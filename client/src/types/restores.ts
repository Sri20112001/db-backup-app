import type { BackupRun } from './runs'

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
