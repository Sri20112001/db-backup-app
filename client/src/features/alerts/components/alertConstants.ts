import {
  XCircle, Clock, WifiOff, Database, RotateCcw, ShieldOff,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import type { AlertType, AlertSeverity, AlertStatus } from '@/types'

export const TYPE_FILTERS: { label: string; value: AlertType | '' }[] = [
  { label: 'All types', value: '' },
  { label: 'Failed', value: 'BACKUP_FAILED' },
  { label: 'Missed', value: 'BACKUP_MISSED' },
  { label: 'Offline', value: 'AGENT_OFFLINE' },
  { label: 'Storage', value: 'STORAGE_LOW' },
  { label: 'Restore', value: 'RESTORE_FAILED' },
  { label: 'Verify', value: 'VERIFY_FAILED' },
]

export const STATUS_TABS: { label: string; value: AlertStatus | '' }[] = [
  { label: 'All', value: '' },
  { label: 'Firing', value: 'firing' },
  { label: 'Acknowledged', value: 'acknowledged' },
  { label: 'Resolved', value: 'resolved' },
]

export const SEVERITY_FILTERS: { label: string; value: AlertSeverity | '' }[] = [
  { label: 'Any severity', value: '' },
  { label: 'Critical', value: 'critical' },
  { label: 'Warning', value: 'warning' },
  { label: 'Info', value: 'info' },
]

export const GROUP_ORDER = ['Today', 'Yesterday', 'This week', 'Older']

export const alertIcon: Record<AlertType, LucideIcon> = {
  BACKUP_FAILED: XCircle,
  BACKUP_MISSED: Clock,
  AGENT_OFFLINE: WifiOff,
  STORAGE_LOW: Database,
  RESTORE_FAILED: RotateCcw,
  VERIFY_FAILED: ShieldOff,
}

export const alertColor: Record<AlertType, { icon: string; bg: string }> = {
  BACKUP_FAILED:  { icon: 'text-error', bg: 'bg-error-container' },
  BACKUP_MISSED:  { icon: 'text-[#d97706]', bg: 'bg-[#fef3c7]' },
  AGENT_OFFLINE:  { icon: 'text-on-surface-variant', bg: 'bg-surface-variant' },
  STORAGE_LOW:    { icon: 'text-[#d97706]', bg: 'bg-[#fef3c7]' },
  RESTORE_FAILED: { icon: 'text-error', bg: 'bg-error-container' },
  VERIFY_FAILED:  { icon: 'text-error', bg: 'bg-error-container' },
}

export const severityChip: Record<AlertSeverity, string> = {
  critical: 'bg-error-container text-error',
  warning:  'bg-[#fef3c7] text-[#d97706]',
  info:     'bg-surface-variant text-on-surface-variant',
}

export const statusChip: Record<AlertStatus, { label: string; cls: string }> = {
  firing:       { label: 'Firing', cls: 'bg-error-container text-error' },
  acknowledged: { label: 'Acked', cls: 'bg-[#e0e7ff] text-[#3730a3]' },
  resolved:     { label: 'Resolved', cls: 'bg-[#d1fae5] text-[#065f46]' },
}
