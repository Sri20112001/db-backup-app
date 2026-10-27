import { useRealtimeStore } from '@/store/realtimeStore'
import { WifiOff, Loader2 } from 'lucide-react'

export const ConnectionStatusBadge = () => {
  const status = useRealtimeStore((s) => s.status)

  if (status === 'open') return null

  return (
    <div
      role="status"
      aria-live="polite"
      className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-[11px] font-medium transition-colors ${
        status === 'connecting'
          ? 'bg-warning-container text-on-warning-container border border-warning/30'
          : 'bg-error-container text-error border border-error/30'
      }`}
      title={status === 'connecting' ? 'Reconnecting to live server…' : 'Live updates disconnected'}
    >
      {status === 'connecting' ? (
        <Loader2 size={12} className="animate-spin shrink-0" />
      ) : (
        <WifiOff size={12} className="shrink-0" />
      )}
      <span className="hidden sm:inline">
        {status === 'connecting' ? 'Reconnecting…' : 'Live sync offline'}
      </span>
    </div>
  )
}

export default ConnectionStatusBadge
