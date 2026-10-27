import { useCallback, useEffect, useState } from 'react'
import { alertApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import type { Alert, AlertType, AlertSeverity, AlertStatus } from '@/types'
import EmptyState from '@/components/EmptyState'
import Pagination from '@/components/Pagination'
import { useRealtimeStore } from '@/stores/realtimeStore'
import { formatRelative } from '@/utils/format'
import {
  XCircle, Clock, WifiOff, Database, RotateCcw, ShieldOff, Bell,
  ArrowDownWideNarrow, ArrowUpNarrowWide, Check, CheckCheck, CircleDot,
  CheckCircle2, Download, CheckSquare, Square
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import SearchInput from '@/components/ui/SearchInput'
import { Page, PageHeader } from '@/components/Page'
import { useDebouncedValue } from '@/hooks/useDebouncedValue'
import { exportToCsv } from '@/utils/exportCsv'

const TYPE_FILTERS: { label: string; value: AlertType | '' }[] = [
  { label: 'All types', value: '' },
  { label: 'Failed', value: 'BACKUP_FAILED' },
  { label: 'Missed', value: 'BACKUP_MISSED' },
  { label: 'Offline', value: 'AGENT_OFFLINE' },
  { label: 'Storage', value: 'STORAGE_LOW' },
  { label: 'Restore', value: 'RESTORE_FAILED' },
  { label: 'Verify', value: 'VERIFY_FAILED' },
]

const STATUS_TABS: { label: string; value: AlertStatus | '' }[] = [
  { label: 'All', value: '' },
  { label: 'Firing', value: 'firing' },
  { label: 'Acknowledged', value: 'acknowledged' },
  { label: 'Resolved', value: 'resolved' },
]

const SEVERITY_FILTERS: { label: string; value: AlertSeverity | '' }[] = [
  { label: 'Any severity', value: '' },
  { label: 'Critical', value: 'critical' },
  { label: 'Warning', value: 'warning' },
  { label: 'Info', value: 'info' },
]

const GROUP_ORDER = ['Today', 'Yesterday', 'This week', 'Older']

const alertIcon: Record<AlertType, LucideIcon> = {
  BACKUP_FAILED: XCircle,
  BACKUP_MISSED: Clock,
  AGENT_OFFLINE: WifiOff,
  STORAGE_LOW: Database,
  RESTORE_FAILED: RotateCcw,
  VERIFY_FAILED: ShieldOff,
}

const alertColor: Record<AlertType, { icon: string; bg: string }> = {
  BACKUP_FAILED:  { icon: 'text-error', bg: 'bg-error-container' },
  BACKUP_MISSED:  { icon: 'text-[#d97706]', bg: 'bg-[#fef3c7]' },
  AGENT_OFFLINE:  { icon: 'text-on-surface-variant', bg: 'bg-surface-variant' },
  STORAGE_LOW:    { icon: 'text-[#d97706]', bg: 'bg-[#fef3c7]' },
  RESTORE_FAILED: { icon: 'text-error', bg: 'bg-error-container' },
  VERIFY_FAILED:  { icon: 'text-error', bg: 'bg-error-container' },
}

const severityChip: Record<AlertSeverity, string> = {
  critical: 'bg-error-container text-error',
  warning:  'bg-[#fef3c7] text-[#d97706]',
  info:     'bg-surface-variant text-on-surface-variant',
}

const statusChip: Record<AlertStatus, { label: string; cls: string }> = {
  firing:       { label: 'Firing', cls: 'bg-error-container text-error' },
  acknowledged: { label: 'Acked', cls: 'bg-[#e0e7ff] text-[#3730a3]' },
  resolved:     { label: 'Resolved', cls: 'bg-[#d1fae5] text-[#065f46]' },
}

const AlertsPage = () => {
  const { currentOrg } = useAuthStore()
  const { addToast } = useUIStore()
  const [alerts, setAlerts] = useState<Alert[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [unreadOnly, setUnreadOnly] = useState(false)
  const [typeFilter, setTypeFilter] = useState<AlertType | ''>('')
  const [statusFilter, setStatusFilter] = useState<AlertStatus | ''>('')
  const [severityFilter, setSeverityFilter] = useState<AlertSeverity | ''>('')
  const [search, setSearch] = useState('')
  const [oldestFirst, setOldestFirst] = useState(false)
  const [isLoading, setIsLoading] = useState(true)
  const [selectedIds, setSelectedIds] = useState<string[]>([])
  const [isBatchProcessing, setIsBatchProcessing] = useState(false)
  const limit = 50
  const alertTick = useRealtimeStore((s) => s.alertTick)
  const debouncedSearch = useDebouncedValue(search)

  const loadAlerts = useCallback(() => {
    if (!currentOrg) return
    setIsLoading(true)
    alertApi.list(currentOrg.id, {
      unread: unreadOnly || undefined,
      search: debouncedSearch || undefined,
      type: typeFilter || undefined,
      status: statusFilter || undefined,
      severity: severityFilter || undefined,
      order: oldestFirst ? 'asc' : 'desc',
      page,
      limit,
    })
      .then((res) => { setAlerts(res.data); setTotal(res.total) })
      .catch(() => addToast('error', 'Failed to load alerts'))
      .finally(() => setIsLoading(false))
  }, [currentOrg, unreadOnly, page, debouncedSearch, oldestFirst, typeFilter, statusFilter, severityFilter, limit, addToast])

  useEffect(() => { loadAlerts() }, [loadAlerts])

  useEffect(() => {
    if (alertTick === 0) return
    loadAlerts()
  }, [alertTick, loadAlerts])

  const notifyAlertsChanged = useRealtimeStore((s) => s.notifyAlertsChanged)

  const handleMarkRead = async (id: string) => {
    if (!currentOrg) return
    try {
      await alertApi.markRead(currentOrg.id, id)
      setAlerts((prev) => prev.map((a) => a.id === id ? { ...a, read: true } : a))
      notifyAlertsChanged()
    } catch { addToast('error', 'Failed to mark as read') }
  }

  const handleMarkAllRead = async () => {
    if (!currentOrg) return
    try {
      const res = await alertApi.markAllRead(currentOrg.id)
      setAlerts((prev) => prev.map((a) => ({ ...a, read: true })))
      notifyAlertsChanged()
      addToast('success', res.count === 1 ? '1 alert marked as read' : `${res.count} alerts marked as read`)
    } catch { addToast('error', 'Failed to mark alerts as read') }
  }

  const handleAck = async (id: string) => {
    if (!currentOrg) return
    try {
      const updated = await alertApi.ack(currentOrg.id, id)
      setAlerts((prev) => prev.map((a) => a.id === id ? updated : a))
      notifyAlertsChanged()
      addToast('success', 'Alert acknowledged')
    } catch { addToast('error', 'Failed to acknowledge alert') }
  }

  const handleResolve = async (id: string) => {
    if (!currentOrg) return
    try {
      const updated = await alertApi.resolve(currentOrg.id, id)
      setAlerts((prev) => prev.map((a) => a.id === id ? updated : a))
      notifyAlertsChanged()
      addToast('success', 'Alert resolved')
    } catch { addToast('error', 'Failed to resolve alert') }
  }

  const toggleSelectAlert = (id: string) => {
    setSelectedIds((prev) =>
      prev.includes(id) ? prev.filter((item) => item !== id) : [...prev, id]
    )
  }

  const toggleSelectAll = () => {
    if (selectedIds.length === alerts.length) {
      setSelectedIds([])
    } else {
      setSelectedIds(alerts.map((a) => a.id))
    }
  }

  const handleBatchAck = async () => {
    if (!currentOrg || selectedIds.length === 0) return
    setIsBatchProcessing(true)
    try {
      await Promise.allSettled(selectedIds.map((id) => alertApi.ack(currentOrg.id, id)))
      addToast('success', `Acknowledged ${selectedIds.length} alerts`)
      setSelectedIds([])
      loadAlerts()
      notifyAlertsChanged()
    } catch {
      addToast('error', 'Failed to batch acknowledge alerts')
    } finally {
      setIsBatchProcessing(false)
    }
  }

  const handleBatchResolve = async () => {
    if (!currentOrg || selectedIds.length === 0) return
    setIsBatchProcessing(true)
    try {
      await Promise.allSettled(selectedIds.map((id) => alertApi.resolve(currentOrg.id, id)))
      addToast('success', `Resolved ${selectedIds.length} alerts`)
      setSelectedIds([])
      loadAlerts()
      notifyAlertsChanged()
    } catch {
      addToast('error', 'Failed to batch resolve alerts')
    } finally {
      setIsBatchProcessing(false)
    }
  }

  const handleBatchMarkRead = async () => {
    if (!currentOrg || selectedIds.length === 0) return
    setIsBatchProcessing(true)
    try {
      await Promise.allSettled(selectedIds.map((id) => alertApi.markRead(currentOrg.id, id)))
      addToast('success', `Marked ${selectedIds.length} alerts as read`)
      setSelectedIds([])
      loadAlerts()
      notifyAlertsChanged()
    } catch {
      addToast('error', 'Failed to batch mark alerts as read')
    } finally {
      setIsBatchProcessing(false)
    }
  }

  const handleExportCsv = () => {
    if (alerts.length === 0) {
      addToast('info', 'No alerts available to export')
      return
    }
    const headers = ['ID', 'Type', 'Title', 'Message', 'Severity', 'Status', 'Read', 'Created At']
    const rows = alerts.map((a) => [
      a.id,
      a.type,
      a.title,
      a.message,
      a.severity,
      a.status,
      a.read ? 'Yes' : 'No',
      a.created_at,
    ])
    exportToCsv(`vaultguard-alerts-${new Date().toISOString().slice(0, 10)}.csv`, headers, rows)
    addToast('success', `Exported ${alerts.length} alerts to CSV`)
  }

  const unreadCount = alerts.filter((a) => !a.read).length
  const firingCount = alerts.filter((a) => a.status === 'firing').length

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
  const groupKeys = (oldestFirst ? [...GROUP_ORDER].reverse() : GROUP_ORDER).filter((g) => grouped[g])

  return (
    <Page>
      <PageHeader
        title={
          <>
            Alerts
            {unreadCount > 0 && (
              <span className="px-2 py-0.5 rounded-full bg-error text-on-primary text-[12px] font-medium">{unreadCount}</span>
            )}
          </>
        }
        description="System alerts and notifications"
        actions={
          <>
            <button
              type="button"
              onClick={handleExportCsv}
              disabled={alerts.length === 0}
              className="flex items-center gap-1.5 px-3 h-8 rounded-lg text-[12px] font-medium transition-colors border border-outline-variant/60 bg-surface-container-low text-on-surface hover:bg-surface-container-high disabled:opacity-50"
              title="Download filtered alerts as CSV"
            >
              <Download size={14} />
              Export CSV
            </button>
            <button
              type="button"
              onClick={() => { setOldestFirst((v) => !v); setPage(1) }}
              title={oldestFirst ? 'Oldest first' : 'Newest first'}
              className="flex items-center gap-1.5 px-3 h-8 rounded-lg text-[12px] font-medium transition-colors bg-surface-container-low text-on-surface-variant hover:bg-surface-container-high"
            >
              {oldestFirst ? <ArrowUpNarrowWide size={14} /> : <ArrowDownWideNarrow size={14} />}
              {oldestFirst ? 'Oldest' : 'Newest'}
            </button>
            <button
              type="button"
              onClick={() => { setUnreadOnly(!unreadOnly); setPage(1) }}
              className={`px-3 h-8 rounded-lg text-[12px] font-medium transition-colors ${unreadOnly ? 'bg-primary text-on-primary' : 'bg-surface-container-low text-on-surface-variant hover:bg-surface-container-high'}`}
            >
              Unread only
            </button>
            {unreadCount > 0 && (
              <button
                type="button"
                onClick={handleMarkAllRead}
                className="flex items-center gap-1.5 px-3 h-8 rounded-lg bg-primary text-on-primary text-[12px] font-medium hover:bg-primary/90 transition-colors shadow-xs whitespace-nowrap"
              >
                <CheckCheck size={14} />
                Mark all read
              </button>
            )}
          </>
        }
      />

      {/* Status tabs */}
      <div className="flex items-center gap-1.5 overflow-x-auto">
        {STATUS_TABS.map((t) => (
          <button
            key={t.value}
            type="button"
            onClick={() => { setStatusFilter(t.value); setPage(1) }}
            className={`px-3 h-8 rounded-lg text-[12px] font-medium whitespace-nowrap transition-colors ${
              statusFilter === t.value
                ? 'bg-primary text-on-primary shadow-sm'
                : 'bg-surface-container-low text-on-surface-variant hover:bg-surface-container-high'
            }`}
          >
            {t.label}
            {t.value === 'firing' && firingCount > 0 && (
              <span className={`ml-1.5 px-1.5 py-0.5 rounded-full text-[10px] ${statusFilter === t.value ? 'bg-on-primary/20' : 'bg-error-container text-error'}`}>
                {firingCount}
              </span>
            )}
          </button>
        ))}
      </div>

      {/* Search + filters + Select All */}
      <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-3">
        <div className="flex flex-col lg:flex-row lg:items-center gap-3 flex-1">
          <SearchInput
            value={search}
            onChange={(e) => { setSearch(e.target.value); setPage(1) }}
            placeholder="Search alerts…"
            className="lg:max-w-xs"
          />
          <div className="flex items-center gap-1.5 overflow-x-auto">
            {TYPE_FILTERS.map((f) => (
              <button
                key={f.value}
                type="button"
                onClick={() => { setTypeFilter(f.value); setPage(1) }}
                className={`px-3 h-8 rounded-lg text-[12px] font-medium whitespace-nowrap transition-colors ${
                  typeFilter === f.value
                    ? 'bg-primary text-on-primary shadow-sm'
                    : 'bg-surface-container-low text-on-surface-variant hover:bg-surface-container-high'
                }`}
              >
                {f.label}
              </button>
            ))}
          </div>
          <div className="flex items-center gap-1.5 overflow-x-auto">
            {SEVERITY_FILTERS.map((f) => (
              <button
                key={f.value}
                type="button"
                onClick={() => { setSeverityFilter(f.value); setPage(1) }}
                className={`px-3 h-8 rounded-lg text-[12px] font-medium whitespace-nowrap transition-colors ${
                  severityFilter === f.value
                    ? 'bg-primary text-on-primary shadow-sm'
                    : 'bg-surface-container-low text-on-surface-variant hover:bg-surface-container-high'
                }`}
              >
                {f.label}
              </button>
            ))}
          </div>
        </div>

        {alerts.length > 0 && (
          <button
            type="button"
            onClick={toggleSelectAll}
            className="flex items-center gap-1.5 px-3 h-8 text-[12px] font-medium text-on-surface-variant hover:text-on-surface self-start lg:self-auto"
          >
            {selectedIds.length === alerts.length && alerts.length > 0 ? (
              <CheckSquare size={15} className="text-primary" />
            ) : (
              <Square size={15} />
            )}
            Select page ({selectedIds.length}/{alerts.length})
          </button>
        )}
      </div>

      {/* Floating Batch Actions Bar */}
      {selectedIds.length > 0 && (
        <div className="sticky top-0 z-20 flex items-center justify-between gap-3 p-3 rounded-xl bg-primary text-on-primary shadow-lg animate-in slide-in-from-top-2 duration-150">
          <div className="flex items-center gap-2 text-[13px] font-medium">
            <span>{selectedIds.length} alert{selectedIds.length > 1 ? 's' : ''} selected</span>
          </div>

          <div className="flex items-center gap-2">
            <button
              type="button"
              disabled={isBatchProcessing}
              onClick={handleBatchAck}
              className="px-3 py-1.5 rounded-lg bg-surface-container-lowest/20 hover:bg-surface-container-lowest/30 text-[12.5px] font-medium transition-colors disabled:opacity-50"
            >
              Acknowledge
            </button>
            <button
              type="button"
              disabled={isBatchProcessing}
              onClick={handleBatchResolve}
              className="px-3 py-1.5 rounded-lg bg-surface-container-lowest/20 hover:bg-surface-container-lowest/30 text-[12.5px] font-medium transition-colors disabled:opacity-50"
            >
              Resolve
            </button>
            <button
              type="button"
              disabled={isBatchProcessing}
              onClick={handleBatchMarkRead}
              className="px-3 py-1.5 rounded-lg bg-surface-container-lowest/20 hover:bg-surface-container-lowest/30 text-[12.5px] font-medium transition-colors disabled:opacity-50"
            >
              Mark Read
            </button>
            <button
              type="button"
              onClick={() => setSelectedIds([])}
              className="px-2.5 py-1.5 text-[12px] underline opacity-90 hover:opacity-100 transition-opacity"
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {isLoading ? (
        <div className="flex flex-col gap-2">{[...Array(6)].map((_, i) => <div key={i} className="animate-pulse h-16 rounded-xl bg-surface-container-lowest border border-surface-variant" />)}</div>
      ) : alerts.length === 0 ? (
        <EmptyState icon={Bell} title={unreadOnly ? 'All caught up' : 'No alerts'} description={unreadOnly ? 'No unread alerts.' : 'Alerts will appear here when issues are detected.'} />
      ) : (
        <div className="flex-1 min-h-0 overflow-y-auto flex flex-col gap-4 pr-0.5">
          {groupKeys.map((group) => (
            <div key={group} className="flex flex-col gap-2">
              <span className="text-[12px] font-semibold uppercase tracking-wider text-on-surface-variant px-1">{group}</span>
              {grouped[group].map((alert) => {
                const c = alertColor[alert.type]
                const AlertIcon = alertIcon[alert.type]
                const sc = statusChip[alert.status]
                const dimmed = alert.status === 'resolved'
                const isSelected = selectedIds.includes(alert.id)
                return (
                  <div
                    key={alert.id}
                    onClick={() => !alert.read && handleMarkRead(alert.id)}
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
                        toggleSelectAlert(alert.id)
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
                        <p className={`text-[14px] font-medium text-on-surface ${!alert.read && !dimmed ? 'font-semibold' : ''}`}>{alert.title}</p>
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
                        <span className="text-[12px] text-on-surface-variant shrink-0 ml-auto">{formatRelative(alert.last_occurred_at ?? alert.created_at)}</span>
                      </div>
                      <p className="text-[13px] text-on-surface-variant mt-0.5">{alert.message}</p>
                    </div>
                    <div className="flex items-center gap-1 shrink-0 mt-0.5">
                      {alert.status === 'firing' && (
                        <>
                          <button
                            type="button"
                            onClick={(e) => { e.stopPropagation(); handleAck(alert.id) }}
                            title="Acknowledge"
                            className="flex items-center gap-1 px-2 py-1 rounded-lg text-[12px] font-medium text-on-surface-variant hover:bg-surface-container-high transition-colors"
                          >
                            <Check size={13} />
                            Ack
                          </button>
                          <button
                            type="button"
                            onClick={(e) => { e.stopPropagation(); handleResolve(alert.id) }}
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
                          onClick={(e) => { e.stopPropagation(); handleResolve(alert.id) }}
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
                          onClick={(e) => { e.stopPropagation(); handleMarkRead(alert.id) }}
                          className="flex items-center gap-1 px-2 py-1 rounded-lg text-[12px] font-medium text-primary hover:bg-surface-container-high transition-colors"
                        >
                          <Check size={13} />
                          Mark read
                        </button>
                      )}
                    </div>
                  </div>
                )
              })}
            </div>
          ))}
        </div>
      )}

      <Pagination
        page={page}
        totalPages={Math.ceil(total / limit)}
        total={total}
        perPage={limit}
        onPage={setPage}
      />
    </Page>
  )
}

export default AlertsPage
