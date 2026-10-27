import type { ComponentType } from 'react'
import type { BackupSourceType, BackupMode, ConnectionType } from '@/types'
import { CloudUpload, FolderOpen, HardDrive } from 'lucide-react'

export interface JobWizardFormData {
  name: string
  source_type: BackupSourceType
  connection_id: string
  policy_id: string
  source_path: string
  source_database: string
  include_patterns: string
  exclude_patterns: string
  agent_id: string
  storage_target_id: string
  cron_expr: string
  timezone: string
  mode: BackupMode
  encrypted: boolean
  retention_days: number
  export_format: string
}

export const INITIAL_JOB_WIZARD_FORM: JobWizardFormData = {
  name: '',
  source_type: 'FILESYSTEM',
  connection_id: '',
  policy_id: '',
  source_path: '',
  source_database: '',
  include_patterns: '',
  exclude_patterns: '',
  agent_id: '',
  storage_target_id: '',
  cron_expr: '0 23 * * *',
  timezone: 'UTC',
  mode: 'COMPRESSED',
  encrypted: true,
  retention_days: 30,
  export_format: 'ARCHIVE',
}

export const SOURCE_TO_CONNECTION: Partial<Record<BackupSourceType, ConnectionType>> = {
  POSTGRES: 'POSTGRES',
  MONGODB: 'MONGODB',
  MSSQL_SERVER: 'MSSQL',
}

export const WIZARD_STEPS = ['Source', 'Agent', 'Schedule', 'Processing', 'Review'] as const

export const STORAGE_TYPE_ICON: Record<string, ComponentType<{ size?: number; className?: string }>> = {
  S3: CloudUpload,
  SMB: FolderOpen,
  LOCAL: HardDrive,
}
