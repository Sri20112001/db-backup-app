import { useEffect, useState } from 'react'
import { alertApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import type { Alert, AlertType } from '@/types'
import EmptyState from '@/components/EmptyState'
import { formatRelative } from '@/utils/format'
import { XCircle, Clock, WifiOff, Database, RotateCcw, ShieldOff, Bell, ChevronLeft, ChevronRight } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

const alertIcon: Record<AlertType, LucideIcon> = {
  BACKUP_FAILED: XCircle,
  BACKUP_MISSED: Clock,
  AGENT_OFFLINE: WifiOff,
  STORAGE_LOW: Database,
  RESTORE_FAILED: RotateCcw,
  VERIFY_FAILED: ShieldOff,
}

const alertColor: Record<AlertType, { icon: string; bg: string }> = {
  BACKUP_FAILED:  { icon: 'text-[#ba1a1a]', bg: 'bg-[#ffdad6]' },
  BACKUP_MISSED:  { icon: 'text-[#d97706]', bg: 'bg-[#fef3c7]' },
  AGENT_OFFLINE:  { icon: 'text-[#737686]', bg: 'bg-[#dce2f7]' },
  STORAGE_LOW:    { icon: 'text-[#d97706]', bg: 'bg-[#fef3c7]' },
  RESTORE_FAILED: { icon: 'text-[#ba1a1a]', bg: 'bg-[#ffdad6]' },
  VERIFY_FAILED:  { icon: 'text-[#ba1a1a]', bg: 'bg-[#ffdad6]' },
}

const AlertsPage = () => {
  const { currentOrg } = useAuthStore()
  const { addToast } = useUIStore()
  const [alerts, setAlerts] = useState<Alert[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [unreadOnly, setUnreadOnly] = useState(false)
  const [isLoading, setIsLoading] = useState(true)
  const limit = 50

  const loadAlerts = () => {
    if (!currentOrg) return
    setIsLoading(true)
    alertApi.list(currentOrg.id, { unread: unreadOnly || undefined, page, limit })
      .then((res) => { setAlerts(res.data); setTotal(res.total) })
      .catch(() => addToast('error', 'Failed to load alerts'))
      .finally(() => setIsLoading(false))
  }

  useEffect(() => { loadAlerts() }, [currentOrg, unreadOnly, page])

  const handleMarkRead = async (id: string) => {
    if (!currentOrg) return
    try {
      await alertApi.markRead(currentOrg.id, id)
      setAlerts((prev) => prev.map((a) => a.id === id ? { ...a, read: true } : a))
    } catch { addToast('error', 'Failed to mark as read') }
  }

  const unreadCount = alerts.filter((a) => !a.read).length

  const grouped: Record<string, Alert[]> = {}
  alerts.forEach((a) => {
    const d = new Date(a.created_at)
    const now = new Date()
    let key = 'Older'
    if (d.toDateString() === now.toDateString()) key = 'Today'
    else if (d.toDateString() === new Date(now.getTime() - 86400000).toDateString()) key = 'Yesterday'
    else if (now.getTime() - d.getTime() < 7 * 86400000) key = 'This week'
    grouped[key] = [...(grouped[key] ?? []), a]
  })

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-[20px] font-semibold text-[#141b2b] tracking-tight flex items-center gap-2.5">
            Alerts
            {unreadCount > 0 && (
              <span className="px-2 py-0.5 rounded-full bg-[#ba1a1a] text-white text-[12px] font-medium">{unreadCount}</span>
            )}
          </h1>
          <p className="text-[12px] text-[#434655] mt-0.5">System alerts and notifications</p>
        </div>
        <button
          type="button"
          onClick={() => { setUnreadOnly(!unreadOnly); setPage(1) }}
          className={`px-3 h-8 rounded-lg text-[12px] font-medium transition-colors ${unreadOnly ? 'bg-[#2563eb] text-white' : 'bg-[#f1f3ff] text-[#434655] hover:bg-[#e9edff]'}`}
        >
          Unread only
        </button>
      </div>

      {isLoading ? (
        <div className="flex flex-col gap-2">{[...Array(6)].map((_, i) => <div key={i} className="animate-pulse h-16 rounded-xl bg-[#ffffff] border border-[#e9edff]" />)}</div>
      ) : alerts.length === 0 ? (
        <EmptyState icon={Bell} title={unreadOnly ? 'All caught up' : 'No alerts'} description={unreadOnly ? 'No unread alerts.' : 'Alerts will appear here when issues are detected.'} />
      ) : (
        <div className="flex flex-col gap-6">
          {Object.entries(grouped).map(([group, items]) => (
            <div key={group} className="flex flex-col gap-2">
              <span className="text-[12px] font-semibold uppercase tracking-wider text-[#737686] px-1">{group}</span>
              {items.map((alert) => {
                const c = alertColor[alert.type]
                const AlertIcon = alertIcon[alert.type]
                return (
                  <div
                    key={alert.id}
                    onClick={() => !alert.read && handleMarkRead(alert.id)}
                    className={`flex items-start gap-3 p-4 rounded-xl border transition-all cursor-pointer ${
                      alert.read ? 'bg-[#ffffff] border-[#e9edff]' : 'bg-[#ffffff] border-[#e9edff] border-l-4 border-l-[#ba1a1a]'
                    } hover:shadow-sm`}
                  >
                    <div className={`w-8 h-8 rounded-lg ${c.bg} ${c.icon} flex items-center justify-center shrink-0 mt-0.5`}>
                      <AlertIcon size={16} />
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center justify-between gap-2">
                        <p className={`text-[14px] font-medium text-[#141b2b] ${!alert.read ? 'font-semibold' : ''}`}>{alert.title}</p>
                        <span className="text-[12px] text-[#737686] shrink-0">{formatRelative(alert.created_at)}</span>
                      </div>
                      <p className="text-[13px] text-[#434655] mt-0.5">{alert.message}</p>
                    </div>
                    {!alert.read && <span className="w-2 h-2 rounded-full bg-[#2563eb] shrink-0 mt-2" />}
                  </div>
                )
              })}
            </div>
          ))}
        </div>
      )}

      {Math.ceil(total / limit) > 1 && (
        <div className="flex items-center justify-center gap-2">
          <button type="button" disabled={page === 1} onClick={() => setPage((p) => p - 1)} className="p-1.5 rounded-lg bg-[#e9edff] disabled:opacity-40">
            <ChevronLeft size={16} />
          </button>
          <span className="text-[13px] text-[#434655]">Page {page}</span>
          <button type="button" disabled={page >= Math.ceil(total / limit)} onClick={() => setPage((p) => p + 1)} className="p-1.5 rounded-lg bg-[#e9edff] disabled:opacity-40">
            <ChevronRight size={16} />
          </button>
        </div>
      )}
    </div>
  )
}

export default AlertsPage
