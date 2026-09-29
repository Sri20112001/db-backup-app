import StatusBadge from '@/components/StatusBadge'
import { formatBytes, formatDuration, formatRelative } from '@/utils/format'
import type { BackupRun } from '@/types'

const MAX_RUNS = 10

interface RecentRunsTableProps {
  runs: BackupRun[]
  isLoading: boolean
  onOpenRun: (runId: string) => void
  onViewAll: () => void
  onNewJob: () => void
}

const RecentRunsTable = ({ runs, isLoading, onOpenRun, onViewAll, onNewJob }: RecentRunsTableProps) => (
  // Dashboard preview: latest 10 runs only. Full filterable history lives on /history.

  <div className="flex flex-col flex-1 min-h-0 rounded-lg bg-surface-container-lowest shadow-sm overflow-hidden border border-surface-variant">
    <div className="px-4 py-2.5 flex items-center justify-between bg-surface-container-low/30 shrink-0">
      <div>
        <h2 className="text-[13px] font-semibold text-on-surface">Recent Backup Executions</h2>
        <p className="text-[11px] text-on-surface-variant">Real-time pipeline run log</p>
      </div>
      <button type="button" onClick={onViewAll} className="text-[12px] text-primary hover:underline shrink-0">
        View all →
      </button>
    </div>
    <div className="flex-1 min-h-0 overflow-y-auto">
      <table className="w-full text-left border-collapse">
        <thead className="sticky top-0 z-10">
          <tr className="bg-surface-container-low text-[11px] font-semibold uppercase tracking-wider text-on-surface-variant">
            {['Status', 'Job', 'Type', 'Started', 'Duration', 'Uploaded', 'Category', ''].map((h) => (
              <th key={h} className="py-2 px-3 bg-surface-container-low">{h}</th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-[#e9edff] text-[13px] text-on-surface">
          {isLoading ? (
            [...Array(5)].map((_, i) => (
              <tr key={i} className="animate-pulse">
                {[...Array(8)].map((_, j) => (
                  <td key={j} className="py-2 px-3"><div className="h-4 bg-surface-container-high rounded w-3/4" /></td>
                ))}
              </tr>
            ))
          ) : runs.length === 0 ? (
            <tr>
              <td colSpan={8} className="py-10 text-center text-[13px] text-outline">
                No backup runs yet. <button type="button" onClick={onNewJob} className="text-primary hover:underline">Create a backup job</button> to get started.
              </td>
            </tr>
          ) : runs.slice(0, MAX_RUNS).map((run) => (
            <tr
              key={run.id}
              onClick={() => onOpenRun(run.id)}
              className={`hover:bg-surface-container-low/50 transition-colors cursor-pointer ${run.status === 'FAILED' ? 'bg-error-container/20' : run.status === 'RUNNING' || run.status === 'UPLOADING' ? 'bg-primary-container/10' : ''}`}
            >
              <td className="py-2 px-3"><StatusBadge status={run.status} size="sm" /></td>
              <td className="py-2 px-3 font-medium text-on-surface whitespace-nowrap">{run.backup_job?.name ?? '—'}</td>
              <td className="py-2 px-3">
                <span className="px-1.5 py-0.5 rounded bg-surface-container-high text-on-surface-variant font-mono text-[11px]">
                  {run.source_type || run.backup_job?.source_type}
                </span>
              </td>
              <td className="py-2 px-3 text-[12px] text-outline whitespace-nowrap">{formatRelative(run.started_at)}</td>
              <td className="py-2 px-3 font-mono text-[12px] whitespace-nowrap">{formatDuration(run.duration_seconds)}</td>
              <td className="py-2 px-3 font-mono text-[12px] whitespace-nowrap">{formatBytes(run.bytes_uploaded)}</td>
              <td className="py-2 px-3">
                {run.failure_category ? (
                  <span className="px-1.5 py-0.5 rounded bg-red-50 text-red-700 font-mono text-[11px]">
                    {run.failure_category}
                  </span>
                ) : '—'}
              </td>
              <td className="py-2 px-3 text-right">
                <button type="button" className="px-2 py-0.5 rounded text-[12px] text-on-surface-variant hover:bg-surface-container-high transition-colors">
                  Details
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  </div>
)

export default RecentRunsTable
