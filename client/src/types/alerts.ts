export type AlertType =
  | 'BACKUP_FAILED'
  | 'BACKUP_MISSED'
  | 'AGENT_OFFLINE'
  | 'STORAGE_LOW'
  | 'RESTORE_FAILED'
  | 'VERIFY_FAILED'

export type AlertSeverity = 'info' | 'warning' | 'critical'

export type AlertStatus = 'firing' | 'acknowledged' | 'resolved'

export interface Alert {
  id: string
  organization_id: string
  type: AlertType
  severity: AlertSeverity
  status: AlertStatus
  title: string
  message: string
  read: boolean
  dedup_key?: string
  occurrence_count: number
  acknowledged_by?: string
  acknowledged_at?: string
  resolved_at?: string
  last_occurred_at?: string
  backup_job_id?: string
  agent_id?: string
  created_at: string
}

export interface NotificationSettings {
  organization_id: string
  webhook_url: string
  webhook_enabled: boolean
  quiet_hours_start: number
  quiet_hours_end: number
  min_email_severity: AlertSeverity
  offline_threshold_minutes: number
  missed_window_multiplier: number
  updated_at?: string
}

export interface UserNotificationPrefs {
  user_id: string
  email_enabled: boolean
  min_severity: AlertSeverity
  updated_at?: string
}
