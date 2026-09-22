import type { BackupRunStatus, RestoreStatus, AgentStatus } from '../types'

type Status = BackupRunStatus | RestoreStatus | AgentStatus | 'HEALTHY' | 'WARNING' | 'UNKNOWN' | 'PAUSED'

const config: Record<string, { bg: string; text: string; dot?: string; pulse?: boolean }> = {
  HEALTHY:   { bg: 'bg-[#c9e6ff]', text: 'text-[#004c6e]', dot: 'bg-[#006591]' },
  COMPLETED: { bg: 'bg-[#c9e6ff]', text: 'text-[#004c6e]', dot: 'bg-[#006591]' },
  ONLINE:    { bg: 'bg-[#c9e6ff]', text: 'text-[#004c6e]', dot: 'bg-[#006591]', pulse: true },
  RUNNING:   { bg: 'bg-[#dbe1ff]', text: 'text-[#003ea8]', dot: 'bg-[#004ac6]', pulse: true },
  UPLOADING: { bg: 'bg-[#dbe1ff]', text: 'text-[#003ea8]', dot: 'bg-[#004ac6]', pulse: true },
  VERIFYING: { bg: 'bg-[#e1e0ff]', text: 'text-[#2f2ebe]', dot: 'bg-[#3e3fcc]', pulse: true },
  PENDING:   { bg: 'bg-[#e1e0ff]', text: 'text-[#2f2ebe]', dot: 'bg-[#3e3fcc]' },
  WARNING:   { bg: 'bg-[#ffdad6]/60', text: 'text-[#93000a]', dot: 'bg-[#ba1a1a]' },
  FAILED:    { bg: 'bg-[#ffdad6]', text: 'text-[#93000a]', dot: 'bg-[#ba1a1a]' },
  CANCELLED: { bg: 'bg-[#dce2f7]', text: 'text-[#434655]', dot: 'bg-[#737686]' },
  OFFLINE:   { bg: 'bg-[#dce2f7]', text: 'text-[#434655]', dot: 'bg-[#737686]' },
  UNKNOWN:   { bg: 'bg-[#dce2f7]', text: 'text-[#434655]', dot: 'bg-[#737686]' },
  PAUSED:    { bg: 'bg-[#dce2f7]', text: 'text-[#434655]', dot: 'bg-[#737686]' },
}

interface Props {
  status: Status
  size?: 'sm' | 'md'
}

const StatusBadge = ({ status, size = 'md' }: Props) => {
  const c = config[status] ?? config.UNKNOWN
  const padding = size === 'sm' ? 'px-1.5 py-0.5 text-[11px]' : 'px-2 py-0.5 text-[12px]'

  return (
    <span className={`inline-flex items-center gap-1.5 rounded-full font-medium ${padding} ${c.bg} ${c.text}`}>
      {c.dot && (
        <span className={`w-1.5 h-1.5 rounded-full shrink-0 ${c.dot} ${c.pulse ? 'animate-pulse' : ''}`} />
      )}
      {status}
    </span>
  )
}

export default StatusBadge
