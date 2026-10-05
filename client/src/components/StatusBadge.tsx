import type { BackupRunStatus, RestoreStatus, AgentStatus } from '../types'

type Status = BackupRunStatus | RestoreStatus | AgentStatus | 'HEALTHY' | 'WARNING' | 'UNKNOWN' | 'PAUSED' | 'REVOKED'

const config: Record<string, { bg: string; text: string; dot?: string; pulse?: boolean }> = {
  // Success / Active States (Green)
  HEALTHY:   { bg: 'bg-success-container', text: 'text-on-success-container', dot: 'bg-success' },
  COMPLETED: { bg: 'bg-success-container', text: 'text-on-success-container', dot: 'bg-success' },
  ONLINE:    { bg: 'bg-success-container', text: 'text-on-success-container', dot: 'bg-success', pulse: true },

  // Active / Processing States (Blue)
  RUNNING:   { bg: 'bg-info-container', text: 'text-on-info-container', dot: 'bg-info', pulse: true },
  UPLOADING: { bg: 'bg-info-container', text: 'text-on-info-container', dot: 'bg-info', pulse: true },
  VERIFYING: { bg: 'bg-info-container', text: 'text-on-info-container', dot: 'bg-info', pulse: true },

  // Attention / Waiting States (Amber)
  WARNING:   { bg: 'bg-warning-container', text: 'text-on-warning-container', dot: 'bg-warning' },
  PENDING:   { bg: 'bg-warning-container', text: 'text-on-warning-container', dot: 'bg-warning' },
  PAUSED:    { bg: 'bg-warning-container/80', text: 'text-on-warning-container', dot: 'bg-warning' },

  // Critical / Destructive States (Red)
  FAILED:    { bg: 'bg-error-container', text: 'text-on-error-container', dot: 'bg-error' },
  REVOKED:   { bg: 'bg-error-container', text: 'text-on-error-container', dot: 'bg-error' },

  // Inactive / Neutral States (Gray)
  CANCELLED: { bg: 'bg-inactive-container', text: 'text-on-inactive-container', dot: 'bg-inactive' },
  OFFLINE:   { bg: 'bg-inactive-container', text: 'text-on-inactive-container', dot: 'bg-inactive' },
  UNKNOWN:   { bg: 'bg-inactive-container', text: 'text-on-inactive-container', dot: 'bg-inactive' },
}

interface Props {
  status: Status
  size?: 'sm' | 'md'
}

const StatusBadge = ({ status, size = 'md' }: Props) => {
  const c = config[status] ?? config.UNKNOWN
  const padding = size === 'sm' ? 'px-1.5 py-0.5 text-[11px]' : 'px-2 py-0.5 text-[12px]'

  return (
    <span className={`inline-flex w-fit items-center gap-1.5 rounded-full font-medium ${padding} ${c.bg} ${c.text}`}>
      {c.dot && (
        <span className={`w-1.5 h-1.5 rounded-full shrink-0 ${c.dot} ${c.pulse ? 'animate-pulse' : ''}`} />
      )}
      {status}
    </span>
  )
}

export default StatusBadge
