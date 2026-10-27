import { Check, CheckCircle2, CircleDot, CheckSquare, Square } from 'lucide-react'
import type { Alert } from '@/types'
import { formatRelative } from '@/utils/format'
import {
  alertIcon, alertColor, severityChip, statusChip
} from './alertConstants'

interface AlertCardProps {
  alert: Alert
  isSelected: boolean
  onToggleSelect: (id: string) => void
  onMarkRead: (id: string) => void
  onAck: (id: string) => void
  onResolve: (id: string) => void
}

export const AlertCard = ({
  alert,
  isSelected,
  onToggleSelect,
  onMarkRead,
  onAck,
  onResolve,
}: AlertCardProps) => {
  const c = alertColor[alert.type]
  const AlertIcon = alertIcon[alert.type]
  const sc = statusChip[alert.status]
  const dimmed = alert.status === 'resolved'

  return (
    <div
      onClick={() => !alert.read && onMarkRead(alert.id)}
      className={`flex items-start gap-3 p-4 rounded-xl border transition-all cursor-pointer ${
        isSelected ? 'ring-2 ring-primary border-primary bg-primary/5' : ''
      } ${
        alert.read || dimmed
          ? 'bg-surface-container-lowest border-surface-variant opacity-75'
          : 'bg-surface-container-lowest border-surface-variant border-l-4 border-l-[#ba1a1a]'
      } hover:shadow-sm`}
    >
      {/* Selection Checkbox */}
      <button
        type="button"
        onClick={(e) => {
          e.stopPropagation()
          onToggleSelect(alert.id)
        }}
        className="mt-1 text-on-surface-variant hover:text-primary transition-colors"
        aria-label={isSelected ? 'Deselect alert' : 'Select alert'}
      >
        {isSelected ? (
          <CheckSquare size={17} className="text-primary" />
        ) : (
          <Square size={17} />
        )}
      </button>

      <div className={`w-8 h-8 rounded-lg ${c.bg} ${c.icon} flex items-center justify-center shrink-0 mt-0.5`}>
        <AlertIcon size={16} />
      </div>

      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 flex-wrap">
          <p className={`text-[14px] font-medium text-on-surface ${!alert.read && !dimmed ? 'font-semibold' : ''}`}>
            {alert.title}
          </p>
          <span className={`px-1.5 py-0.5 rounded-full text-[10px] font-medium ${severityChip[alert.severity]}`}>
            {alert.severity}
          </span>
          <span className={`px-1.5 py-0.5 rounded-full text-[10px] font-medium ${sc.cls}`}>
            {sc.label}
          </span>
          {alert.occurrence_count > 1 && (
            <span
              className="px-1.5 py-0.5 rounded-full text-[10px] font-medium bg-surface-variant text-on-surface-variant flex items-center gap-0.5"
              title={`Seen ${alert.occurrence_count} times, last ${alert.last_occurred_at ? formatRelative(alert.last_occurred_at) : formatRelative(alert.created_at)}`}
            >
              <CircleDot size={10} />
              ×{alert.occurrence_count}
            </span>
          )}
          <span className="text-[12px] text-on-surface-variant shrink-0 ml-auto">
            {formatRelative(alert.last_occurred_at ?? alert.created_at)}
          </span>
        </div>
        <p className="text-[13px] text-on-surface-variant mt-0.5">{alert.message}</p>
      </div>

      <div className="flex items-center gap-1 shrink-0 mt-0.5">
        {alert.status === 'firing' && (
          <>
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation()
                onAck(alert.id)
              }}
              title="Acknowledge"
              className="flex items-center gap-1 px-2 py-1 rounded-lg text-[12px] font-medium text-on-surface-variant hover:bg-surface-container-high transition-colors"
            >
              <Check size={13} />
              Ack
            </button>
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation()
                onResolve(alert.id)
              }}
              title="Resolve"
              className="flex items-center gap-1 px-2 py-1 rounded-lg text-[12px] font-medium text-primary hover:bg-surface-container-high transition-colors"
            >
              <CheckCircle2 size={13} />
              Resolve
            </button>
          </>
        )}
        {alert.status === 'acknowledged' && (
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation()
              onResolve(alert.id)
            }}
            title="Resolve"
            className="flex items-center gap-1 px-2 py-1 rounded-lg text-[12px] font-medium text-primary hover:bg-surface-container-high transition-colors"
          >
            <CheckCircle2 size={13} />
            Resolve
          </button>
        )}
        {!alert.read && alert.status === 'resolved' && (
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation()
              onMarkRead(alert.id)
            }}
            className="flex items-center gap-1 px-2 py-1 rounded-lg text-[12px] font-medium text-primary hover:bg-surface-container-high transition-colors"
          >
            <Check size={13} />
            Mark read
          </button>
        )}
      </div>
    </div>
  )
}

export default AlertCard
