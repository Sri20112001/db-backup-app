import { useEffect, useRef, useState } from 'react'
import { runApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import type { BackupRun, BackupRunStatus } from '@/types'
import StatusBadge from '@/components/StatusBadge'
import EmptyState from '@/components/EmptyState'
import { SkeletonRow } from '@/components/Skeleton'
import RunDetailDrawer from './RunDetailDrawer'
import Pagination from '@/components/Pagination'
import { useRealtimeStore, mergeRunPatch } from '@/stores/realtimeStore'
import { formatBytes, formatDuration, formatRelative } from '@/utils/format'
import { History } from 'lucide-react'
import SearchInput from '@/components/ui/SearchInput'
import SortableTh from '@/components/ui/SortableTh'
import { Page, PageHeader } from '@/components/Page'
import { useDebouncedValue } from '@/hooks/useDebouncedValue'
import type { SortDir } from '@/hooks/useSort'

const STATUS_FILTERS: { label: string; value: BackupRunStatus | '' }[] = [
  { label: 'All', value: '' },
  { label: 'Completed', value: 'COMPLETED' },
  { label: 'Running', value: 'RUNNING' },
  { label: 'Failed', value: 'FAILED' },
  { label: 'Cancelled', value: 'CANCELLED' },
]

const RUN_SORT_KEYS = ['job', 'started', 'duration', 'uploaded'] as const
type RunSortKey = (typeof RUN_SORT_KEYS)[number]

const HistoryPage = () => {
  const { currentOrg } = useAuthStore()
  const { addToast, openRunDetail, runDetailId } = useUIStore()
  const [runs, setRuns] = useState<BackupRun[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [statusFilter, setStatusFilter] = useState<BackupRunStatus | ''>('')
  const [search, setSearch] = useState('')
  const [sortKey, setSortKey] = useState<RunSortKey>('started')
  const [sortDir, setSortDir] = useState<SortDir>('desc')
  const [isLoading, setIsLoading] = useState(true)
  const limit = 50
  const lastRunEvent = useRealtimeStore((s) => s.lastRunEvent)
  const debouncedSearch = useDebouncedValue(search)

  const toggleSort = (key: string) => {
    if (!RUN_SORT_KEYS.includes(key as RunSortKey)) return
    if (key === sortKey) setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'))
    else { setSortKey(key as RunSortKey); setSortDir('desc') }
    setPage(1)
  }

  const loadHistory = () => {
    if (!currentOrg) return
    setIsLoading(true)
    runApi.list(currentOrg.id, {
      status: statusFilter || undefined,
      search: debouncedSearch || undefined,
      sort: sortKey,
      order: sortDir,
      page,
      limit,
    })
      .then((res) => { setRuns(res.data); setTotal(res.total) })
      .catch(() => addToast('error', 'Failed to load history'))
      .finally(() => setIsLoading(false))
  }

  useEffect(() => { loadHistory() }, [currentOrg, statusFilter, page, debouncedSearch, sortKey, sortDir])

  // Live patches: merge known rows in place, refetch when an unseen run
  // appears (new scheduled/manual run on page 1).
  const runsRef = useRef(runs)
  useEffect(() => { runsRef.current = runs })
  useEffect(() => {
    if (!lastRunEvent) return
    if (!runsRef.current.some((r) => r.id === lastRunEvent.id)) {
      if (page === 1) loadHistory()
      return
    }
    setRuns((prev) => mergeRunPatch(prev, lastRunEvent))
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [lastRunEvent])

  const totalPages = Math.ceil(total / limit)

  return (
    <Page>
      <PageHeader
        title="Backup History"
        description="All backup run executions across all jobs"
      />

      {/* Filters */}
      <div className="flex flex-col lg:flex-row lg:items-center gap-3">
        <SearchInput
          value={search}
          onChange={(e) => { setSearch(e.target.value); setPage(1) }}
          placeholder="Search by job name…"
          className="lg:max-w-xs"
        />
        <div className="flex items-center gap-1.5 overflow-x-auto">
          {STATUS_FILTERS.map((f) => (
            <button
              key={f.value}
              type="button"
              onClick={() => { setStatusFilter(f.value); setPage(1) }}
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

      {/* Table */}
      <div className="flex-1 min-h-0 flex flex-col rounded-lg bg-surface-container-lowest shadow-sm overflow-hidden border border-surface-variant">
        <div className="flex-1 min-h-0 overflow-auto">
          <table className="w-full text-left border-collapse">
            <thead className="sticky top-0 z-10">
              <tr className="bg-surface-container-low text-[12px] font-semibold uppercase tracking-wider text-on-surface-variant">
                <th className="py-2.5 px-4 whitespace-nowrap">Status</th>
                <SortableTh label="Job Name" sortKey="job" activeKey={sortKey} dir={sortDir} onToggle={toggleSort} />
                <th className="py-2.5 px-4 whitespace-nowrap">Source</th>
                <th className="py-2.5 px-4 whitespace-nowrap">Agent</th>
                <SortableTh label="Started" sortKey="started" activeKey={sortKey} dir={sortDir} onToggle={toggleSort} />
                <SortableTh label="Duration" sortKey="duration" activeKey={sortKey} dir={sortDir} onToggle={toggleSort} />
                <th className="py-2.5 px-4 whitespace-nowrap">Original</th>
                <SortableTh label="Uploaded" sortKey="uploaded" activeKey={sortKey} dir={sortDir} onToggle={toggleSort} />
                <th className="py-2.5 px-4 whitespace-nowrap" />
              </tr>
            </thead>
            <tbody className="divide-y divide-[#e9edff] text-[14px] text-on-surface">
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
                  className={`hover:bg-surface-container-low/50 transition-colors cursor-pointer ${run.status === 'FAILED' ? 'bg-error-container/10' : ''}`}
                >
                  <td className="py-3 px-4"><StatusBadge status={run.status} size="sm" /></td>
                  <td className="py-3 px-4 font-medium">{run.backup_job?.name ?? '—'}</td>
                  <td className="py-3 px-4">
                    <span className="px-1.5 py-0.5 rounded bg-surface-container-high text-on-surface-variant font-mono text-[11px]">
                      {run.source_type || run.backup_job?.source_type || '—'}
                    </span>
                  </td>
                  <td className="py-3 px-4 font-mono text-[12px] text-on-surface-variant">{run.backup_job?.agent?.name ?? '—'}</td>
                  <td className="py-3 px-4 text-[12px] text-outline">{formatRelative(run.started_at)}</td>
                  <td className="py-3 px-4 font-mono text-[12px]">{formatDuration(run.duration_seconds)}</td>
                  <td className="py-3 px-4 font-mono text-[12px]">{formatBytes(run.bytes_read)}</td>
                  <td className="py-3 px-4 font-mono text-[12px]">{formatBytes(run.bytes_uploaded)}</td>
                  <td className="py-3 px-4 text-right">
                    <button type="button" className="px-2 py-1 rounded text-[12px] text-on-surface-variant hover:bg-surface-container-high transition-colors">
                      Details
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        <div className="p-3 bg-surface-container-low/40 shrink-0">
          <Pagination page={page} totalPages={totalPages} total={total} perPage={limit} onPage={setPage} />
        </div>
      </div>

      {runDetailId && <RunDetailDrawer runId={runDetailId} />}
    </Page>
  )
}

export default HistoryPage
