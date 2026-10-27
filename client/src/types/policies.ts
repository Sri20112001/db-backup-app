export type BackupMode = 'NORMAL' | 'COMPRESSED'

export interface BackupPolicy {
  id: string
  organization_id: string
  name: string
  strategy: string
  cron_expr: string
  timezone: string
  enabled: boolean
  mode: BackupMode
  encrypted: boolean
  retention_days: number
  verification_enabled: boolean
  max_retries: number
  retry_delay_seconds: number
  sla_target_minutes: number
  rpo_target_minutes: number
  rto_target_minutes: number
  created_at: string
  updated_at?: string
}
