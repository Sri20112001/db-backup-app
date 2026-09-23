import type { BackupRunStatus, RestoreStatus, AgentStatus } from '../types'

type Status = BackupRunStatus | RestoreStatus | AgentStatus | 'HEALTHY' | 'WARNING' | 'UNKNOWN' | 'PAUSED'

const config: Record<string, { bg: string; text: string; dot?: string; pulse?: boolean }> = {
  // Success / Active States (Emerald)
  HEALTHY:   { bg: 'bg-[#d1fae5]', text: 'text-[#065f46]', dot: 'bg-[#059669]' },
  COMPLETED: { bg: 'bg-[#d1fae5]', text: 'text-[#065f46]', dot: 'bg-[#059669]' },
  ONLINE:    { bg: 'bg-[#d1fae5]', text: 'text-[#065f46]', dot: 'bg-[#059669]', pulse: true },

  // Active / Processing States (Teal / Cyan)
  RUNNING:   { bg: 'bg-[#ccfbf1]', text: 'text-[#115e59]', dot: 'bg-[#0f766e]', pulse: true },
  UPLOADING: { bg: 'bg-[#ccfbf1]', text: 'text-[#115e59]', dot: 'bg-[#0f766e]', pulse: true },
  VERIFYING: { bg: 'bg-[#ccfbf1]', text: 'text-[#115e59]', dot: 'bg-[#0f766e]', pulse: true },

  // Attention / Waiting States (Amber)
  WARNING:   { bg: 'bg-[#fef3c7]', text: 'text-[#92400e]', dot: 'bg-[#d97706]' },
  PENDING:   { bg: 'bg-[#fef3c7]', text: 'text-[#92400e]', dot: 'bg-[#d97706]' },
  PAUSED:    { bg: 'bg-[#fef3c7]/80', text: 'text-[#92400e]', dot: 'bg-[#d97706]' },

  // Critical / Destructive States (Rose / Crimson)
  FAILED:    { bg: 'bg-[#ffdad6]', text: 'text-[#93000a]', dot: 'bg-[#ba1a1a]' },

  // Inactive / Neutral States (Muted Sage-Slate)
  CANCELLED: { bg: 'bg-[#e6ebe8]', text: 'text-[#3e4943]', dot: 'bg-[#6d7a73]' },
  OFFLINE:   { bg: 'bg-[#e6ebe8]', text: 'text-[#3e4943]', dot: 'bg-[#6d7a73]' },
  UNKNOWN:   { bg: 'bg-[#e6ebe8]', text: 'text-[#3e4943]', dot: 'bg-[#6d7a73]' },
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
