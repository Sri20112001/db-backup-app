import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { jobApi, runApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import type { BackupJob, BackupRun } from '@/types'
import StatusBadge from '@/components/StatusBadge'
import { SkeletonRow } from '@/components/Skeleton'
import ConfirmDialog from '@/components/ConfirmDialog'
import RunDetailDrawer from '@/features/history/components/RunDetailDrawer'
import { formatBytes, formatDuration, formatRelative } from '@/utils/format'
import { ChevronRight, Play, ChevronLeft, Loader2 } from 'lucide-react'
import normalizeWindowsPath from '@/utils/normalizeWindowsPath'
import Action3DButton from '@/components/ui/Action3DButton'

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
  const limit = 20

  const loadData = () => {
    if (!currentOrg || !id) return
    setIsLoading(true)
    Promise.all([
      jobApi.get(currentOrg.id, id),
      runApi.list(currentOrg.id, { job_id: id, page, limit }),
    ])
      .then(([j, r]) => { setJob(j); setRuns(r.data); setTotal(r.total) })
      .catch(() => addToast('error', 'Failed to load job'))
      .finally(() => setIsLoading(false))
  }

  useEffect(() => { loadData() }, [currentOrg, id, page])

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
    <div className="flex flex-col gap-6">
      {/* Breadcrumb */}
      <div className="flex items-center gap-2 text-[12px] text-[#434655]">
        <button type="button" onClick={() => navigate('/jobs')} className="hover:text-[#004ac6]">Backup Jobs</button>
        <ChevronRight size={14} />
        <span className="text-[#004ac6] font-medium">{job.name}</span>
      </div>

      {/* Header card */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 p-5 rounded-xl bg-[#ffffff] border border-[#e9edff] shadow-sm">
        <div className="flex items-center gap-4">
          <StatusBadge status={job.enabled ? 'ONLINE' : 'OFFLINE'} />
          <div>
            <h1 className="text-[20px] font-semibold text-[#141b2b]">{job.name}</h1>
            <div className="flex items-center gap-2 mt-1">
              <span className="px-2 py-0.5 rounded bg-[#e9edff] text-[#434655] text-[11px] font-medium uppercase">
                {job.source_type.replace('_', ' ')}
              </span>
              <span className="text-[12px] text-[#737686]">{job.agent?.name}</span>
              <span className="text-[12px] text-[#737686]">→ {job.storage_target?.name}</span>
            </div>
          </div>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <label className="relative inline-flex items-center cursor-pointer">
            <input type="checkbox" checked={job.enabled} onChange={handleToggle} className="sr-only peer" />
            <div className="w-9 h-5 bg-[#dce2f7] peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-[#2563eb]" />
          </label>
          <Action3DButton onClick={handleRunNow}>
            <Play size={16} />
            Run Now
          </Action3DButton>
          <button
            type="button"
            onClick={() => navigate(`/jobs/${job.id}/edit`)}
            className="px-3 h-9 rounded-lg bg-[#ffffff] border border-[#e9edff] text-[#141b2b] text-[13px] font-medium hover:bg-[#f1f3ff] transition-colors"
          >
            Edit
          </button>
          <button
            type="button"
            onClick={() => setShowDelete(true)}
            className="px-3 h-9 rounded-lg bg-[#ffffff] border border-[#fca5a5] text-[#ba1a1a] text-[13px] font-medium hover:bg-[#ffdad6] transition-colors"
          >
            Delete
          </button>
        </div>
      </div>

      {/* Two-column layout */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
        {/* Left: run history */}
        <div className="lg:col-span-8 flex flex-col gap-4">
          <h2 className="text-[14px] font-semibold uppercase tracking-wider text-[#434655]">Run History</h2>
          <div className="flex flex-col rounded-lg bg-[#ffffff] shadow-sm overflow-hidden border border-[#e9edff]">
            <div className="overflow-x-auto">
              <table className="w-full text-left border-collapse">
                <thead>
                  <tr className="bg-[#f1f3ff] text-[12px] font-semibold uppercase tracking-wider text-[#434655]">
                    {['Status', 'Started', 'Duration', 'Original', 'Uploaded', 'Checksum', ''].map((h) => (
                      <th key={h} className="py-2.5 px-4">{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y divide-[#e9edff] text-[14px] text-[#141b2b]">
                  {isLoading ? (
                    [...Array(5)].map((_, i) => <SkeletonRow key={i} cols={7} />)
                  ) : runs.map((run) => (
                    <tr
                      key={run.id}
                      onClick={() => openRunDetail(run.id)}
                      className={`hover:bg-[#f1f3ff]/50 transition-colors cursor-pointer ${run.status === 'FAILED' ? 'bg-[#ffdad6]/10' : ''}`}
                    >
                      <td className="py-3 px-4"><StatusBadge status={run.status} size="sm" /></td>
                      <td className="py-3 px-4 text-[12px] text-[#737686]">{formatRelative(run.started_at)}</td>
                      <td className="py-3 px-4 font-mono text-[12px]">{formatDuration(run.duration_seconds)}</td>
                      <td className="py-3 px-4 font-mono text-[12px]">{formatBytes(run.bytes_read)}</td>
                      <td className="py-3 px-4 font-mono text-[12px]">{formatBytes(run.bytes_uploaded)}</td>
                      <td className="py-3 px-4 font-mono text-[11px] text-[#737686]">
                        {run.checksum ? `${run.checksum.slice(0, 8)}...` : '—'}
                      </td>
                      <td className="py-3 px-4 text-right">
                        <button type="button" className="px-2 py-1 rounded text-[12px] text-[#434655] hover:bg-[#e9edff]">Details</button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {totalPages > 1 && (
              <div className="p-3 bg-[#f1f3ff]/40 flex items-center justify-between text-[12px] text-[#434655]">
                <span>{total} total runs</span>
                <div className="flex items-center gap-1">
                  <button type="button" disabled={page === 1} onClick={() => setPage((p) => p - 1)} className="p-1.5 rounded bg-[#e9edff] disabled:opacity-40">
                    <ChevronLeft size={16} />
                  </button>
                  <span className="px-3 py-1 rounded bg-[#2563eb] text-white font-medium">{page}</span>
                  <button type="button" disabled={page >= totalPages} onClick={() => setPage((p) => p + 1)} className="p-1.5 rounded bg-[#e9edff] disabled:opacity-40">
                    <ChevronRight size={16} />
                  </button>
                </div>
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

      {runDetailId && <RunDetailDrawer runId={runDetailId} />}
    </div>
  )
}

const ConfigCard = ({ title, items }: { title: string; items: { label: string; value: string; mono?: boolean }[] }) => (
  <div className="p-4 rounded-xl bg-[#ffffff] border border-[#e9edff] shadow-sm">
    <h3 className="text-[12px] font-semibold uppercase tracking-wider text-[#434655] mb-3">{title}</h3>
    <div className="flex flex-col gap-2">
      {items.map((item) => (
        <div key={item.label} className="flex items-start gap-3 text-[13px]">
          <span className="text-[#737686] w-20 shrink-0">{item.label}</span>
          <span className={`text-[#141b2b] break-all ${item.mono ? 'font-mono text-[11px]' : 'font-medium'}`}>{item.value}</span>
        </div>
      ))}
    </div>
  </div>
)

export default JobDetailPage
