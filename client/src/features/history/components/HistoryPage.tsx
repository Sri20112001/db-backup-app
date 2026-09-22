import { useEffect, useState } from 'react'
import { runApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import type { BackupRun, BackupRunStatus } from '@/types'
import StatusBadge from '@/components/StatusBadge'
import EmptyState from '@/components/EmptyState'
import SkeletonRow from '@/components/Skeleton'
import RunDetailDrawer from './RunDetailDrawer'
import { formatBytes, formatDuration, formatRelative } from '@/utils/format'
import { ChevronLeft, ChevronRight, History } from 'lucide-react'

const STATUS_FILTERS: { label: string; value: BackupRunStatus | '' }[] = [
  { label: 'All', value: '' },
  { label: 'Completed', value: 'COMPLETED' },
  { label: 'Running', value: 'RUNNING' },
  { label: 'Failed', value: 'FAILED' },
  { label: 'Cancelled', value: 'CANCELLED' },
]

const HistoryPage = () => {
  const { currentOrg } = useAuthStore()
  const { addToast, openRunDetail, runDetailId } = useUIStore()
  const [runs, setRuns] = useState<BackupRun[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [statusFilter, setStatusFilter] = useState<BackupRunStatus | ''>('')
  const [isLoading, setIsLoading] = useState(true)
  const limit = 50

  useEffect(() => {
    if (!currentOrg) return
    setIsLoading(true)
    runApi.list(currentOrg.id, { status: statusFilter || undefined, page, limit })
      .then((res) => { setRuns(res.data); setTotal(res.total) })
      .catch(() => addToast('error', 'Failed to load history'))
      .finally(() => setIsLoading(false))
  }, [currentOrg, statusFilter, page])

  const totalPages = Math.ceil(total / limit)

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-[20px] font-semibold text-[#141b2b] tracking-tight">Backup History</h1>
          <p className="text-[12px] text-[#434655] mt-0.5">All backup run executions across all jobs</p>
        </div>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-1.5 overflow-x-auto">
        {STATUS_FILTERS.map((f) => (
          <button
            key={f.value}
            type="button"
            onClick={() => { setStatusFilter(f.value); setPage(1) }}
            className={`px-3 h-8 rounded-lg text-[12px] font-medium whitespace-nowrap transition-colors ${
              statusFilter === f.value
                ? 'bg-[#2563eb] text-white shadow-sm'
                : 'bg-[#f1f3ff] text-[#434655] hover:bg-[#e9edff]'
            }`}
          >
            {f.label}
          </button>
        ))}
      </div>

      {/* Table */}
      <div className="flex flex-col rounded-lg bg-[#ffffff] shadow-sm overflow-hidden border border-[#e9edff]">
        <div className="overflow-x-auto">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="bg-[#f1f3ff] text-[12px] font-semibold uppercase tracking-wider text-[#434655]">
                {['Status', 'Job Name', 'Source', 'Agent', 'Started', 'Duration', 'Original', 'Uploaded', ''].map((h) => (
                  <th key={h} className="py-2.5 px-4 whitespace-nowrap">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-[#e9edff] text-[14px] text-[#141b2b]">
              {isLoading ? (
                [...Array(8)].map((_, i) => <SkeletonRow key={i} cols={9} />)
              ) : runs.length === 0 ? (
                <tr>
                  <td colSpan={9}>
                    <EmptyState icon={History} title="No backup runs" description="Backup runs will appear here once jobs execute." />
                  </td>
                </tr>
              ) : runs.map((run) => (
                <tr
                  key={run.id}
                  onClick={() => openRunDetail(run.id)}
                  className={`hover:bg-[#f1f3ff]/50 transition-colors cursor-pointer ${run.status === 'FAILED' ? 'bg-[#ffdad6]/10' : ''}`}
                >
                  <td className="py-3 px-4"><StatusBadge status={run.status} size="sm" /></td>
                  <td className="py-3 px-4 font-medium">{run.backup_job?.name ?? '—'}</td>
                  <td className="py-3 px-4">
                    <span className="px-1.5 py-0.5 rounded bg-[#e9edff] text-[#434655] font-mono text-[11px]">
                      {run.source_type || run.backup_job?.source_type || '—'}
                    </span>
                  </td>
                  <td className="py-3 px-4 font-mono text-[12px] text-[#434655]">{run.backup_job?.agent?.name ?? '—'}</td>
                  <td className="py-3 px-4 text-[12px] text-[#737686]">{formatRelative(run.started_at)}</td>
                  <td className="py-3 px-4 font-mono text-[12px]">{formatDuration(run.duration_seconds)}</td>
                  <td className="py-3 px-4 font-mono text-[12px]">{formatBytes(run.bytes_read)}</td>
                  <td className="py-3 px-4 font-mono text-[12px]">{formatBytes(run.bytes_uploaded)}</td>
                  <td className="py-3 px-4 text-right">
                    <button type="button" className="px-2 py-1 rounded text-[12px] text-[#434655] hover:bg-[#e9edff] transition-colors">
                      Details
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        <div className="p-3 bg-[#f1f3ff]/40 flex items-center justify-between text-[12px] text-[#434655]">
          <span>Showing {Math.min((page - 1) * limit + 1, total)}–{Math.min(page * limit, total)} of {total} runs</span>
          <div className="flex items-center gap-1">
            <button
              type="button"
              disabled={page === 1}
              onClick={() => setPage((p) => p - 1)}
              className="p-1.5 rounded-lg bg-[#e9edff] text-[#434655] disabled:opacity-40 hover:bg-[#dce2f7] transition-colors"
            >
              <ChevronLeft size={16} />
            </button>
            <span className="px-3 py-1 rounded-lg bg-[#2563eb] text-white font-medium">{page}</span>
            <button
              type="button"
              disabled={page >= totalPages}
              onClick={() => setPage((p) => p + 1)}
              className="p-1.5 rounded-lg bg-[#e9edff] text-[#434655] disabled:opacity-40 hover:bg-[#dce2f7] transition-colors"
            >
              <ChevronRight size={16} />
            </button>
          </div>
        </div>
      </div>

      {runDetailId && <RunDetailDrawer runId={runDetailId} />}
    </div>
  )
}

export default HistoryPage
