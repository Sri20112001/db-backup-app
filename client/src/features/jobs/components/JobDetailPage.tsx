import { useEffect, useRef, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { jobApi, runApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import type { BackupJob, BackupRun, BackupRunStatus } from '@/types'
import SortableTh from '@/components/ui/SortableTh'
import type { SortDir } from '@/hooks/useSort'
import StatusBadge from '@/components/StatusBadge'
import { SkeletonRow } from '@/components/Skeleton'
import ConfirmDialog from '@/components/ConfirmDialog'
import RunDetailDrawer from '@/features/history/components/RunDetailDrawer'
import EditJobModal from './EditJobModal'
import Pagination from '@/components/Pagination'
import { useRealtimeStore, mergeRunPatch } from '@/stores/realtimeStore'
import { formatBytes, formatDuration, formatRelative } from '@/utils/format'
import { ChevronRight, Play, Loader2 } from 'lucide-react'
import normalizeWindowsPath from '@/utils/normalizeWindowsPath'
import Action3DButton from '@/components/ui/Action3DButton'
import { Page } from '@/components/Page'

const JobDetailPage = () => {
  const { id } = useParams<{ id: string }>()
  const { currentOrg } = useAuthStore()
  const { addToast, openRunDetail, runDetailId } = useUIStore()
  const navigate = useNavigate()
  const [job, setJob] = useState<BackupJob | null>(null)
  const [runs, setRuns] = useState<BackupRun[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [isLoading, setIsLoading] = useState(true)
  const [showDelete, setShowDelete] = useState(false)
  const [showEdit, setShowEdit] = useState(false)
  const [runStatusFilter, setRunStatusFilter] = useState<BackupRunStatus | ''>('')
  const [sortKey, setSortKey] = useState<RunSortKey>('started')
  const [sortDir, setSortDir] = useState<SortDir>('desc')
  const limit = 20

  const toggleSort = (key: string) => {
    if (!RUN_SORT_KEYS.includes(key as RunSortKey)) return
    if (key === sortKey) setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'))
    else { setSortKey(key as RunSortKey); setSortDir('desc') }
    setPage(1)
  }
  const lastRunEvent = useRealtimeStore((s) => s.lastRunEvent)

  // Live patches for this job's runs; unseen runs trigger a refetch.
  const runsRef = useRef(runs)
  useEffect(() => { runsRef.current = runs })
  useEffect(() => {
    if (!lastRunEvent) return
    if (lastRunEvent.backup_job_id && lastRunEvent.backup_job_id !== id) return
    if (!runsRef.current.some((r) => r.id === lastRunEvent.id)) {
      if (page === 1) loadData()
      return
    }
    setRuns((prev) => mergeRunPatch(prev, lastRunEvent))
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [lastRunEvent])

  const loadData = () => {
    if (!currentOrg || !id) return
    setIsLoading(true)
    Promise.all([
      jobApi.get(currentOrg.id, id),
      runApi.list(currentOrg.id, {
        job_id: id,
        status: runStatusFilter || undefined,
        sort: sortKey,
        order: sortDir,
        page,
        limit,
      }),
    ])
      .then(([j, r]) => { setJob(j); setRuns(r.data); setTotal(r.total) })
      .catch(() => addToast('error', 'Failed to load job'))
      .finally(() => setIsLoading(false))
  }

  useEffect(() => { loadData() }, [currentOrg, id, page, runStatusFilter, sortKey, sortDir])

  const handleRunNow = async () => {
    if (!currentOrg || !job) return
    try {
      await jobApi.runNow(currentOrg.id, job.id)
      addToast('success', `${job.name} started`)
      loadData()
    } catch { addToast('error', 'Failed to start backup') }
  }

  const handleToggle = async () => {
    if (!currentOrg || !job) return
    try {
      if (job.enabled) await jobApi.disable(currentOrg.id, job.id)
      else await jobApi.enable(currentOrg.id, job.id)
      addToast('success', `Job ${job.enabled ? 'disabled' : 'enabled'}`)
      loadData()
    } catch { addToast('error', 'Failed to update job') }
  }

  const handleDelete = async () => {
    if (!currentOrg || !job) return
    try {
      await jobApi.delete(currentOrg.id, job.id)
      addToast('success', 'Job deleted')
      navigate('/jobs')
    } catch { addToast('error', 'Failed to delete job') }
  }

  const totalPages = Math.ceil(total / limit)

  if (isLoading && !job) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 size={32} className="text-[#c3c6d7] animate-spin" />
      </div>
    )
  }

  if (!job) return null

  return (
    <Page scroll>
      {/* Breadcrumb */}
      <div className="flex items-center gap-2 text-[12px] text-on-surface-variant">
        <button type="button" onClick={() => navigate('/jobs')} className="hover:text-primary">Backup Jobs</button>
        <ChevronRight size={14} />
        <span className="text-primary font-medium">{job.name}</span>
      </div>

      {/* Header card */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 p-5 rounded-xl bg-surface-container-lowest border border-surface-variant shadow-sm">
        <div className="flex items-center gap-4">
          <StatusBadge status={job.enabled ? 'ONLINE' : 'OFFLINE'} />
            <div>
              <h1 className="text-[20px] font-semibold text-on-surface">{job.name}</h1>
              <div className="flex items-center gap-2 mt-1 flex-wrap">
                <span className="px-2 py-0.5 rounded bg-surface-container-high text-on-surface-variant text-[11px] font-medium uppercase">
                  {job.source_type.replace('_', ' ')}
                </span>
                {job.policy ? (
                  <span className="px-2 py-0.5 rounded bg-primary-container text-on-primary-container text-[11px] font-medium" title={`Schedule: ${job.policy.cron_expr || 'manual'} · Retention: ${job.policy.retention_days}d${job.policy.max_retries > 0 ? ` · Retries: ${job.policy.max_retries}` : ''}`}>
                    Policy: {job.policy.name}
                  </span>
                ) : (
                  <span className="text-[11px] text-outline">Job-specific settings</span>
                )}
                <span className="text-[12px] text-outline">{job.agent?.name}</span>
                <span className="text-[12px] text-outline">→ {job.storage_target?.name}</span>
              </div>
            </div>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <label className="relative inline-flex items-center cursor-pointer">
            <input type="checkbox" checked={job.enabled} onChange={handleToggle} className="sr-only peer" />
            <div className="w-9 h-5 bg-surface-variant peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-surface-container-lowest after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-primary" />
          </label>
          <Action3DButton onClick={handleRunNow}>
            <Play size={16} />
            Run Now
          </Action3DButton>
          <button
            type="button"
            onClick={() => setShowEdit(true)}
            className="px-3 h-9 rounded-lg bg-surface-container-lowest border border-surface-variant text-on-surface text-[13px] font-medium hover:bg-surface-container-low transition-colors"
          >
            Edit
          </button>
          <button
            type="button"
            onClick={() => setShowDelete(true)}
            className="px-3 h-9 rounded-lg bg-surface-container-lowest border border-error text-error text-[13px] font-medium hover:bg-error-container transition-colors"
          >
            Delete
          </button>
        </div>
      </div>

      {/* Two-column layout */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
        {/* Left: run history */}
        <div className="lg:col-span-8 flex flex-col gap-4">
          <div className="flex items-center justify-between gap-3 flex-wrap">
            <h2 className="text-[14px] font-semibold uppercase tracking-wider text-on-surface-variant">Run History</h2>
            <div className="flex items-center gap-1.5">
              {RUN_STATUS_FILTERS.map((f) => (
                <button
                  key={f.value}
                  type="button"
                  onClick={() => { setRunStatusFilter(f.value); setPage(1) }}
                  className={`px-2.5 h-7 rounded-lg text-[12px] font-medium whitespace-nowrap transition-colors ${
                    runStatusFilter === f.value
                      ? 'bg-primary text-on-primary shadow-sm'
                      : 'bg-surface-container-low text-on-surface-variant hover:bg-surface-container-high'
                  }`}
                >
                  {f.label}
                </button>
              ))}
            </div>
          </div>
          <div className="flex flex-col rounded-lg bg-surface-container-lowest shadow-sm overflow-hidden border border-surface-variant">
            <div className="overflow-auto max-h-[380px]">
              <table className="w-full text-left border-collapse">
                <thead className="sticky top-0 z-10">
                  <tr className="bg-surface-container-low text-[12px] font-semibold uppercase tracking-wider text-on-surface-variant">
                    <th className="py-2.5 px-4">Status</th>
                    <SortableTh label="Started" sortKey="started" activeKey={sortKey} dir={sortDir} onToggle={toggleSort} />
                    <SortableTh label="Duration" sortKey="duration" activeKey={sortKey} dir={sortDir} onToggle={toggleSort} />
                    <th className="py-2.5 px-4">Original</th>
                    <SortableTh label="Uploaded" sortKey="uploaded" activeKey={sortKey} dir={sortDir} onToggle={toggleSort} />
                    <th className="py-2.5 px-4">Checksum</th>
                    <th className="py-2.5 px-4" />
                  </tr>
                </thead>
                <tbody className="divide-y divide-[#e9edff] text-[14px] text-on-surface">
                  {isLoading ? (
                    [...Array(5)].map((_, i) => <SkeletonRow key={i} cols={7} />)
                  ) : runs.map((run) => (
                    <tr
                      key={run.id}
                      onClick={() => openRunDetail(run.id)}
                      className={`hover:bg-surface-container-low/50 transition-colors cursor-pointer ${run.status === 'FAILED' ? 'bg-error-container/10' : ''}`}
                    >
                      <td className="py-3 px-4"><StatusBadge status={run.status} size="sm" /></td>
                      <td className="py-3 px-4 text-[12px] text-outline">{formatRelative(run.started_at)}</td>
                      <td className="py-3 px-4 font-mono text-[12px]">{formatDuration(run.duration_seconds)}</td>
                      <td className="py-3 px-4 font-mono text-[12px]">{formatBytes(run.bytes_read)}</td>
                      <td className="py-3 px-4 font-mono text-[12px]">{formatBytes(run.bytes_uploaded)}</td>
                      <td className="py-3 px-4 font-mono text-[11px] text-outline">
                        {run.checksum ? `${run.checksum.slice(0, 8)}...` : '—'}
                      </td>
                      <td className="py-3 px-4 text-right">
                        <button type="button" className="px-2 py-1 rounded text-[12px] text-on-surface-variant hover:bg-surface-container-high">Details</button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {totalPages > 1 && (
              <div className="p-3 bg-surface-container-low/40 shrink-0">
                <Pagination page={page} totalPages={totalPages} total={total} perPage={limit} onPage={setPage} />
              </div>
            )}
          </div>
        </div>

        {/* Right: config cards */}
        <div className="lg:col-span-4 flex flex-col gap-4">
          <ConfigCard title="Configuration" items={[
            { label: 'Source', value: job.source_path || job.source_database || '—', mono: true },
            { label: 'Mode', value: job.mode },
            { label: 'Encrypted', value: job.encrypted ? 'AES-256-GCM' : 'No' },
            { label: 'Retention', value: `${job.retention_days} days` },
          ]} />
          {job.schedule && (
            <ConfigCard title="Schedule" items={[
              { label: 'Cron', value: job.schedule.cron_expr, mono: true },
              { label: 'Timezone', value: job.schedule.timezone },
            ]} />
          )}
          <ConfigCard title="Storage" items={[
            { label: 'Target', value: job.storage_target?.name ?? '—' },
            { label: 'Type', value: job.storage_target?.type ?? '—' },
            { label: 'Bucket', value: normalizeWindowsPath(job.storage_target?.bucket || job.storage_target?.path || '—'), mono: true },
          ]} />
        </div>
      </div>

      {showDelete && (
        <ConfirmDialog
          title={`Delete "${job.name}"?`}
          message="This will not delete existing backup data in storage. The job configuration will be permanently removed."
          confirmLabel="Delete Job"
          danger
          onConfirm={handleDelete}
          onCancel={() => setShowDelete(false)}
        />
      )}

      {showEdit && id && (
        <EditJobModal jobId={id} onClose={() => setShowEdit(false)} onSaved={loadData} />
      )}

      {runDetailId && <RunDetailDrawer runId={runDetailId} />}
    </Page>
  )
}

const RUN_STATUS_FILTERS: { label: string; value: BackupRunStatus | '' }[] = [
  { label: 'All', value: '' },
  { label: 'Completed', value: 'COMPLETED' },
  { label: 'Running', value: 'RUNNING' },
  { label: 'Failed', value: 'FAILED' },
  { label: 'Cancelled', value: 'CANCELLED' },
]

const RUN_SORT_KEYS = ['started', 'duration', 'uploaded'] as const
type RunSortKey = (typeof RUN_SORT_KEYS)[number]

const ConfigCard = ({ title, items }: { title: string; items: { label: string; value: string; mono?: boolean }[] }) => (
  <div className="p-4 rounded-xl bg-surface-container-lowest border border-surface-variant shadow-sm">
    <h3 className="text-[12px] font-semibold uppercase tracking-wider text-on-surface-variant mb-3">{title}</h3>
    <div className="flex flex-col gap-2">
      {items.map((item) => (
        <div key={item.label} className="flex items-start gap-3 text-[13px]">
          <span className="text-outline w-20 shrink-0">{item.label}</span>
          <span className={`text-on-surface break-all ${item.mono ? 'font-mono text-[11px]' : 'font-medium'}`}>{item.value}</span>
        </div>
      ))}
    </div>
  </div>
)

export default JobDetailPage
