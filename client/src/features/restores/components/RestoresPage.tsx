import { useEffect, useState } from 'react'
import { restoreApi, runApi, agentApi, connectionsApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import type { RestoreJob, BackupRun, Agent, RestoreStatus, DatabaseConnection } from '@/types'
import StatusBadge from '@/components/StatusBadge'
import EmptyState from '@/components/EmptyState'
import Pagination from '@/components/Pagination'
import { usePagination } from '@/hooks/usePagination'
import { useRealtimeStore } from '@/stores/realtimeStore'
import { formatRelative, formatDate } from '@/utils/format'
import { RotateCcw, X, Check, Loader2 } from 'lucide-react'
import SearchInput from '@/components/ui/SearchInput'
import SortSelect from '@/components/ui/SortSelect'
import { Page, PageHeader } from '@/components/Page'

const RESTORE_STATUS_FILTERS: { label: string; value: RestoreStatus | '' }[] = [
  { label: 'All', value: '' },
  { label: 'Pending', value: 'PENDING' },
  { label: 'Running', value: 'RUNNING' },
  { label: 'Completed', value: 'COMPLETED' },
  { label: 'Failed', value: 'FAILED' },
]

const RestoresPage = () => {
  const { currentOrg } = useAuthStore()
  const { addToast } = useUIStore()
  const [restores, setRestores] = useState<RestoreJob[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [showWizard, setShowWizard] = useState(false)
  const [wizardStep, setWizardStep] = useState(0)
  const [recentRuns, setRecentRuns] = useState<BackupRun[]>([])
  const [agents, setAgents] = useState<Agent[]>([])
  const [connections, setConnections] = useState<DatabaseConnection[]>([])
  const [form, setForm] = useState({ backup_run_id: '', agent_id: '', destination_path: '', target_database: '' })
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<RestoreStatus | ''>('')
  const [sort, setSort] = useState('newest')

  const loadRestores = () => {
    if (!currentOrg) return
    setIsLoading(true)
    restoreApi.list(currentOrg.id).then(setRestores).catch(() => addToast('error', 'Failed to load restores')).finally(() => setIsLoading(false))
  }

  useEffect(() => { loadRestores() }, [currentOrg])

  const restoresSeq = useRealtimeStore((s) => s.entitySeq.restores)
  const lastRunEvent = useRealtimeStore((s) => s.lastRunEvent)

  // Restore lifecycle + finished backup runs refresh the list.
  useEffect(() => {
    loadRestores()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [restoresSeq])

  useEffect(() => {
    if (!lastRunEvent) return
    const s = lastRunEvent.status ?? ''
    if (s === 'COMPLETED' || s === 'FAILED' || s === 'CANCELLED') loadRestores()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [lastRunEvent])

  const openWizard = async () => {
    if (!currentOrg) return
    try {
      const [runs, ags, conns] = await Promise.all([
        runApi.list(currentOrg.id, { status: 'COMPLETED', limit: 50 }),
        agentApi.list(currentOrg.id),
        connectionsApi.list(currentOrg.id),
      ])
      setRecentRuns(runs.data)
      setAgents(ags)
      setConnections(conns)
      setShowWizard(true)
      setWizardStep(0)
    } catch { addToast('error', 'Failed to load data') }
  }

  const handleCreate = async () => {
    if (!currentOrg) return
    setIsSubmitting(true)
    try {
      // confirmed:true acknowledges overwrite; connection_id is resolved
      // server-side from the source backup job (never sent from the UI).
      await restoreApi.create(currentOrg.id, { ...form, confirmed: true })
      addToast('success', 'Restore job started')
      setShowWizard(false)
      setForm({ backup_run_id: '', agent_id: '', destination_path: '', target_database: '' })
      loadRestores()
    } catch { addToast('error', 'Failed to start restore') }
    finally { setIsSubmitting(false) }
  }

  const selectedRun = recentRuns.find((r) => r.id === form.backup_run_id)
  const selectedConnection = connections.find((c) => c.id === selectedRun?.backup_job?.connection_id)

  // Database restores reuse the backup job's saved connection: pin the
  // wizard to the connection's agent so credentials resolve correctly.
  const selectRun = (run: BackupRun) => {
    const connAgent = run.backup_job?.agent_id
    setForm((f) => ({ ...f, backup_run_id: run.id, agent_id: connAgent || f.agent_id }))
  }
  const filtered = restores.filter((r) => {
    const matchStatus = !statusFilter || r.status === statusFilter
    const q = search.toLowerCase()
    const matchSearch =
      !search ||
      (r.backup_run?.backup_job?.name ?? '').toLowerCase().includes(q) ||
      (r.destination_path || '').toLowerCase().includes(q) ||
      (r.target_database || '').toLowerCase().includes(q)
    return matchStatus && matchSearch
  })
  const sorted = [...filtered].sort((a, b) =>
    sort === 'oldest'
      ? a.created_at.localeCompare(b.created_at)
      : b.created_at.localeCompare(a.created_at),
  )
  const paged = usePagination(sorted, 8, `${search}|${statusFilter}|${sort}|${currentOrg?.id ?? ''}`)
  const inputCls = "w-full h-9 px-3 rounded-lg border border-surface-variant bg-surface-container-low text-[14px] text-on-surface focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all"

  return (
    <Page>
      <PageHeader
        title="Restores"
        description="Restore data from backup recovery points"
        actions={
          <button type="button" onClick={openWizard} className="flex items-center gap-2 px-4 h-9 rounded-lg bg-primary text-on-primary text-[13px] font-medium hover:bg-primary-container transition-colors shadow-sm">
            <RotateCcw size={16} />
            New Restore
          </button>
        }
      />

      {/* Search + filters */}
      <div className="flex flex-col lg:flex-row lg:items-center gap-3">
        <SearchInput
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Search by job or destination…"
          className="lg:max-w-xs"
        />
        <div className="flex items-center gap-1.5 overflow-x-auto">
          <SortSelect
            value={sort}
            onChange={setSort}
            options={[
              { label: 'Newest first', value: 'newest' },
              { label: 'Oldest first', value: 'oldest' },
            ]}
          />
          {RESTORE_STATUS_FILTERS.map((f) => (
            <button
              key={f.value}
              type="button"
              onClick={() => setStatusFilter(f.value)}
              className={`px-3 h-8 rounded-lg text-[12px] font-medium whitespace-nowrap transition-colors ${
                statusFilter === f.value
                  ? 'bg-primary text-on-primary shadow-sm'
                  : 'bg-surface-container-low text-on-surface-variant hover:bg-surface-container-high'
              }`}
            >
              {f.label}
            </button>
          ))}
        </div>
      </div>

      {isLoading ? (
        <div className="flex flex-col gap-2">{[...Array(4)].map((_, i) => <div key={i} className="animate-pulse h-16 rounded-xl bg-surface-container-lowest border border-surface-variant" />)}</div>
      ) : filtered.length === 0 ? (
        <EmptyState icon={RotateCcw} title="No restore jobs" description="Restore jobs will appear here once you initiate a restore from a recovery point."
          action={<button type="button" onClick={openWizard} className="px-4 h-9 rounded-lg bg-primary text-on-primary text-[13px] font-medium hover:bg-primary-container transition-colors">Start First Restore</button>}
        />
      ) : (
        <div className="flex-1 min-h-0 overflow-y-auto flex flex-col gap-2 pr-0.5">
          {paged.pageItems.map((r) => (
            <div key={r.id} className="flex items-center gap-4 p-4 rounded-xl bg-surface-container-lowest border border-surface-variant shadow-sm hover:shadow-md transition-all">
              <StatusBadge status={r.status} />
              <div className="flex-1 min-w-0">
                <p className="text-[14px] font-medium text-on-surface truncate">
                  {r.backup_run?.backup_job?.name ?? 'Unknown Job'}
                </p>
                <p className="font-mono text-[11px] text-outline truncate">
                  → {r.destination_path || r.target_database || 'Original location'}
                </p>
              </div>
              <div className="text-right shrink-0">
                <p className="text-[12px] text-outline">{formatRelative(r.created_at)}</p>
                {r.completed_at && <p className="text-[11px] text-outline">Done: {formatDate(r.completed_at)}</p>}
              </div>
              {r.error_message && (
                <div className="px-2 py-1 rounded bg-error-container text-on-error-container text-[11px] font-mono max-w-[200px] truncate" title={r.error_message}>
                  {r.error_message}
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      <Pagination
        page={paged.page}
        totalPages={paged.totalPages}
        total={paged.total}
        perPage={paged.perPage}
        onPage={paged.setPage}
      />

      {/* Restore wizard drawer */}
      {showWizard && (
        <div className="fixed inset-0 z-[100] flex justify-end">
          <div className="absolute inset-0 bg-on-surface/30 backdrop-blur-[2px]" onClick={() => setShowWizard(false)} />
          <div className="relative w-full max-w-[520px] bg-surface-container-lowest h-full overflow-y-auto shadow-2xl border-l border-surface-variant flex flex-col">
            <div className="flex items-center justify-between p-5 border-b border-surface-variant sticky top-0 bg-surface-container-lowest z-10">
              <h2 className="text-[16px] font-semibold text-on-surface">New Restore</h2>
              <button type="button" onClick={() => setShowWizard(false)} className="p-1.5 rounded-lg text-on-surface-variant hover:bg-surface-container-high">
                <X size={18} />
              </button>
            </div>

            {/* Step indicator */}
            <div className="flex items-center gap-1 px-5 py-4 border-b border-surface-variant">
              {['Recovery Point', 'Destination', 'Confirm'].map((s, i) => (
                <div key={s} className="flex items-center flex-1">
                  <div className="flex flex-col items-center flex-1">
                    <div className={`w-7 h-7 rounded-full flex items-center justify-center text-[12px] font-semibold ${i < wizardStep ? 'bg-primary text-on-primary' : i === wizardStep ? 'bg-primary text-on-primary' : 'bg-surface-container-high text-outline'}`}>
                      {i < wizardStep ? <Check size={14} /> : i + 1}
                    </div>
                    <span className="text-[10px] text-outline mt-1 text-center">{s}</span>
                  </div>
                  {i < 2 && <div className={`h-0.5 flex-1 mx-1 mb-4 ${i < wizardStep ? 'bg-primary' : 'bg-surface-container-high'}`} />}
                </div>
              ))}
            </div>

            <div className="flex flex-col gap-4 p-5 flex-1">
              {wizardStep === 0 && (
                <>
                  <h3 className="text-[14px] font-semibold text-on-surface">Select Recovery Point</h3>
                  <div className="flex flex-col gap-2 max-h-96 overflow-y-auto">
                    {recentRuns.map((run) => (
                      <button key={run.id} type="button" onClick={() => selectRun(run)}
                        className={`flex items-center gap-3 p-3.5 rounded-xl border-2 text-left transition-all ${form.backup_run_id === run.id ? 'border-primary bg-primary-container/20' : 'border-surface-variant hover:border-outline'}`}>
                        <div className="flex-1 min-w-0">
                          <p className="text-[13px] font-semibold text-on-surface">{run.backup_job?.name ?? '—'}</p>
                          <p className="text-[11px] text-outline">{formatRelative(run.completed_at)} · {run.backup_job?.source_type}</p>
                        </div>
                        <StatusBadge status={run.status} size="sm" />
                      </button>
                    ))}
                  </div>
                </>
              )}

              {wizardStep === 1 && (
                <>
                  <h3 className="text-[14px] font-semibold text-on-surface">Destination</h3>
                  {selectedConnection && (
                    <div className="p-3 rounded-lg bg-primary-container/20 border border-primary/30 text-[12px] text-on-surface">
                      Restores using saved connection <span className="font-semibold">{selectedConnection.name}</span>
                      {' '}({selectedConnection.host}:{selectedConnection.port}) — agent is pinned to the connection's agent and credentials resolve securely.
                    </div>
                  )}
                  <div>
                    <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Target Agent{selectedConnection ? ' (pinned to connection)' : ''}</label>
                    <select value={form.agent_id} onChange={(e) => setForm((f) => ({ ...f, agent_id: e.target.value }))} disabled={!!selectedConnection} className={`${inputCls} ${selectedConnection ? 'opacity-60' : ''}`}>
                      <option value="">Select agent...</option>
                      {agents.map((a) => <option key={a.id} value={a.id}>{a.name} ({a.status})</option>)}
                    </select>
                  </div>
                  <div>
                    <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Destination Path (leave empty for original)</label>
                    <input type="text" value={form.destination_path} onChange={(e) => setForm((f) => ({ ...f, destination_path: e.target.value }))} placeholder="D:\Restored\" className={`${inputCls} font-mono text-[13px]`} />
                  </div>
                  {(selectedRun?.backup_job?.source_type === 'MSSQL_SERVER' ||
                    selectedRun?.backup_job?.source_type === 'POSTGRES' ||
                    selectedRun?.backup_job?.source_type === 'MONGODB') && (
                    <div>
                      <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Target Database Name</label>
                      <input type="text" value={form.target_database} onChange={(e) => setForm((f) => ({ ...f, target_database: e.target.value }))} placeholder="ERP_Restored" className={`${inputCls} font-mono text-[13px]`} />
                    </div>
                  )}
                </>
              )}

              {wizardStep === 2 && (
                <>
                  <h3 className="text-[14px] font-semibold text-on-surface">Confirm Restore</h3>
                  <div className="flex flex-col gap-2 text-[13px]">
                    {[
                      { label: 'Recovery Point', value: selectedRun?.backup_job?.name ?? '—' },
                      { label: 'Agent', value: agents.find((a) => a.id === form.agent_id)?.name ?? '—' },
                      { label: 'Destination', value: form.destination_path || 'Original location' },
                      ...(form.target_database ? [{ label: 'Target DB', value: form.target_database }] : []),
                    ].map((r) => (
                      <div key={r.label} className="flex items-start gap-3">
                        <span className="text-outline w-32 shrink-0">{r.label}</span>
                        <span className="text-on-surface font-medium">{r.value}</span>
                      </div>
                    ))}
                  </div>
                  <div className="p-3 rounded-lg bg-error-container/40 border border-error text-on-error-container text-[12px]">
                    ⚠ This will overwrite existing data at the destination path.
                  </div>
                </>
              )}
            </div>

            <div className="flex items-center justify-between p-5 border-t border-surface-variant">
              <button type="button" onClick={() => wizardStep === 0 ? setShowWizard(false) : setWizardStep((s) => s - 1)}
                className="px-4 h-9 rounded-lg bg-surface-container-lowest border border-surface-variant text-on-surface text-[13px] font-medium hover:bg-surface-container-low transition-colors">
                {wizardStep === 0 ? 'Cancel' : '← Back'}
              </button>
              {wizardStep < 2 ? (
                <button type="button" onClick={() => setWizardStep((s) => s + 1)}
                  disabled={wizardStep === 0 ? !form.backup_run_id : !form.agent_id}
                  className="px-4 h-9 rounded-lg bg-primary text-on-primary text-[13px] font-medium hover:bg-primary-container transition-colors disabled:opacity-50">
                  Next →
                </button>
              ) : (
                <button type="button" onClick={handleCreate} disabled={isSubmitting}
                  className="px-4 h-9 rounded-lg bg-error text-on-primary text-[13px] font-medium hover:bg-error transition-colors disabled:opacity-60 flex items-center gap-2">
                  {isSubmitting && <Loader2 size={14} className="animate-spin" />}
                  Start Restore
                </button>
              )}
            </div>
          </div>
        </div>
      )}
    </Page>
  )
}

export default RestoresPage
