import { useState, useRef, useEffect } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { Bell, CheckCheck, ArrowRight } from 'lucide-react'
import { useAuthStore } from '../store/authStore'
import { alertApi } from '../services/api'
import { useRealtimeStore } from '../stores/realtimeStore'
import type { Alert, AlertSeverity } from '../types'
import { formatRelative } from '../utils/format'

const severityDot: Record<AlertSeverity, string> = {
  critical: 'bg-error',
  warning: 'bg-warning',
  info: 'bg-info',
}

const NotificationBell = () => {
  const { currentOrg } = useAuthStore()
  const navigate = useNavigate()
  const location = useLocation()
  const [open, setOpen] = useState(false)
  const [items, setItems] = useState<Alert[]>([])
  const [unreadCount, setUnreadCount] = useState(0)
  const [isLoading, setIsLoading] = useState(false)
  const wrapRef = useRef<HTMLDivElement>(null)
  const alertTick = useRealtimeStore((s) => s.alertTick)
  const notifyAlertsChanged = useRealtimeStore((s) => s.notifyAlertsChanged)

  // Cheap unread total (limit 1, read `total`). Refreshes on org change,
  // route change, and instantly on socket-pushed alerts.
  useEffect(() => {
    if (!currentOrg) {
      setUnreadCount(0)
      return
    }
    let cancelled = false
    alertApi
      .list(currentOrg.id, { unread: true, limit: 1 })
      .then((res) => {
        if (!cancelled) setUnreadCount(res.total)
      })
      .catch(() => {})
    return () => {
      cancelled = true
    }
  }, [currentOrg, location.pathname, alertTick])

  // Latest alerts for the panel — fetched on open, kept live via alertTick.
  useEffect(() => {
    if (!open || !currentOrg) return
    let cancelled = false
    setIsLoading(true)
    alertApi
      .list(currentOrg.id, { limit: 10, order: 'desc' })
      .then((res) => {
        if (!cancelled) setItems(res.data)
      })
      .catch(() => {})
      .finally(() => {
        if (!cancelled) setIsLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [open, currentOrg, alertTick])

  // Close on outside click or Escape; also close on navigation.
  useEffect(() => {
    if (!open) return
    const onOutside = (e: MouseEvent) => {
      if (wrapRef.current && !wrapRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onOutside)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onOutside)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  useEffect(() => {
    setOpen(false)
  }, [location.pathname])

  const handleOpenAlert = async (alert: Alert) => {
    if (!currentOrg) return
    try {
      if (!alert.read) await alertApi.markRead(currentOrg.id, alert.id)
    } catch {
      // Navigation still proceeds — read state is cosmetic.
    }
    notifyAlertsChanged()
    navigate('/alerts')
  }

  const handleMarkAllRead = async () => {
    if (!currentOrg) return
    try {
      await alertApi.markAllRead(currentOrg.id)
      setItems((prev) => prev.map((a) => ({ ...a, read: true })))
      notifyAlertsChanged()
    } catch {
      // Badge refreshes from the next tick regardless.
    }
  }

  return (
    <div ref={wrapRef} className="relative">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-label={`Notifications${unreadCount > 0 ? ` (${unreadCount} unread)` : ''}`}
        className={`relative flex items-center justify-center w-9 h-9 rounded-lg transition-colors focus:outline-none focus:ring-2 focus:ring-primary/20 ${
          open
            ? 'bg-surface-container-high text-on-surface'
            : 'text-on-surface-variant hover:bg-surface-container-high hover:text-on-surface'
        }`}
      >
        <Bell size={17} />
        {unreadCount > 0 && (
          <span className="absolute -top-0.5 -right-0.5 min-w-[18px] h-[18px] px-1 rounded-full bg-error text-on-primary text-[10px] font-bold flex items-center justify-center shadow-sm">
            {unreadCount > 99 ? '99+' : unreadCount}
          </span>
        )}
      </button>

      {open && (
        <div className="absolute right-0 top-[calc(100%+8px)] w-[360px] max-w-[calc(100vw-2rem)] bg-surface-container-lowest border border-outline-variant/40 rounded-xl shadow-[0_8px_24px_rgba(20,27,43,0.14)] z-50 overflow-hidden animate-in fade-in zoom-in-95 duration-100 flex flex-col">
          {/* Header */}
          <div className="flex items-center justify-between px-3.5 py-2.5 border-b border-outline-variant/30 bg-surface-container-low">
            <span className="text-[13px] font-semibold text-on-surface flex items-center gap-1.5">
              Notifications
              {unreadCount > 0 && (
                <span className="min-w-[18px] h-[18px] px-1 rounded-full bg-error text-on-primary text-[10px] font-bold flex items-center justify-center">
                  {unreadCount > 99 ? '99+' : unreadCount}
                </span>
              )}
            </span>
            {unreadCount > 0 && (
              <button
                type="button"
                onClick={handleMarkAllRead}
                className="flex items-center gap-1 px-2 py-1 rounded-md text-[11.5px] font-medium text-primary hover:bg-surface-container-high transition-colors"
              >
                <CheckCheck size={13} />
                Mark all read
              </button>
            )}
          </div>

          {/* List */}
          <div className="max-h-[380px] overflow-y-auto">
            {isLoading ? (
              <div className="flex flex-col gap-2 p-3">
                {[...Array(3)].map((_, i) => (
                  <div key={i} className="animate-pulse h-12 rounded-lg bg-surface-container-high" />
                ))}
              </div>
            ) : items.length === 0 ? (
              <div className="flex flex-col items-center gap-1.5 py-8 px-4 text-center">
                <Bell size={20} className="text-on-surface-variant" />
                <p className="text-[13px] font-medium text-on-surface-variant">No notifications</p>
                <p className="text-[12px] text-on-surface-variant">Alerts will appear here when issues are detected.</p>
              </div>
            ) : (
              items.map((alert) => (
                <button
                  key={alert.id}
                  type="button"
                  onClick={() => handleOpenAlert(alert)}
                  className={`w-full text-left px-3.5 py-2.5 flex gap-2.5 border-b border-outline-variant/20 last:border-0 transition-colors ${
                    alert.read
                      ? 'hover:bg-surface-container-low'
                      : 'bg-surface-container-low hover:bg-surface-container-high'
                  }`}
                >
                  <span className={`mt-1.5 w-2 h-2 rounded-full shrink-0 ${severityDot[alert.severity]}`} />
                  <span className="min-w-0 flex-1">
                    <span className="flex items-baseline gap-2">
                      <span
                        className={`text-[13px] truncate ${
                          alert.read ? 'font-normal text-on-surface-variant' : 'font-semibold text-on-surface'
                        }`}
                      >
                        {alert.title}
                      </span>
                      <span className="text-[11px] text-on-surface-variant shrink-0 ml-auto">
                        {formatRelative(alert.last_occurred_at ?? alert.created_at)}
                      </span>
                    </span>
                    <span className="block text-[12px] text-on-surface-variant line-clamp-2 mt-0.5">
                      {alert.message || alert.type.replaceAll('_', ' ')}
                    </span>
                  </span>
                </button>
              ))
            )}
          </div>

          {/* Footer */}
          <button
            type="button"
            onClick={() => navigate('/alerts')}
            className="flex items-center justify-center gap-1.5 px-3 py-2.5 border-t border-outline-variant/30 text-[12.5px] font-medium text-primary hover:bg-surface-container-low transition-colors"
          >
            View all alerts
            <ArrowRight size={13} />
          </button>
        </div>
      )}
    </div>
  )
}

export default NotificationBell
