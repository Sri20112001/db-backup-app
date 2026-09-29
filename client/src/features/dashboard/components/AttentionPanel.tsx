import { AlertTriangle, CheckCircle2, ChevronRight, Play, TrendingDown } from 'lucide-react'
import StatusBadge from '@/components/StatusBadge'
import { formatRelative } from '@/utils/format'
import type { JobHealth, SizeAnomaly } from '@/types'

interface AttentionPanelProps {
  failedJobs: JobHealth[]
  anomalies: SizeAnomaly[]
  onRunNow: (jobId: string, jobName: string) => void
  onSelectJob: (jobId: string) => void
  onReviewAlerts: () => void
}

// Actionable issues lane: failed/warning jobs (with instant run + details)
// and backup-size anomalies. Collapses to a slim all-clear strip when empty.
const AttentionPanel = ({ failedJobs, anomalies, onRunNow, onSelectJob, onReviewAlerts }: AttentionPanelProps) => {
  const empty = failedJobs.length === 0 && anomalies.length === 0

  if (empty) {
    return (
      <div className="flex items-center gap-2.5 px-4 py-2 rounded-lg bg-emerald-50 border border-emerald-200 shrink-0">
        <CheckCircle2 size={15} className="text-emerald-600 shrink-0" />
        <p className="text-[13px] font-medium text-emerald-800">
          All clear — every backup job is healthy.
        </p>
      </div>
    )
  }

  return (
    <div className="flex flex-col rounded-lg bg-surface-container-lowest shadow-sm overflow-hidden border border-surface-variant shrink-0">
      <div className="px-4 py-2 flex items-center justify-between bg-error-container/30 shrink-0">
        <div className="flex items-center gap-2">
          <AlertTriangle size={14} className="text-error" />
          <h2 className="text-[13px] font-semibold text-on-surface">
            Needs attention
            <span className="ml-1.5 px-1.5 py-0.5 rounded-full bg-error text-on-primary text-[11px] font-medium">
              {failedJobs.length + anomalies.length}
            </span>
          </h2>
        </div>
        {failedJobs.length > 0 && (
          <button type="button" onClick={onReviewAlerts} className="text-[12px] text-primary hover:underline shrink-0">
            Review alerts →
          </button>
        )}
      </div>
      <div className="max-h-36 overflow-y-auto divide-y divide-[#e9edff]">
        {failedJobs.map((job) => (
          <div key={job.job_id} className="px-4 py-2 flex items-center gap-3 hover:bg-surface-container-low/40 transition-colors">
            <StatusBadge status={job.status} size="sm" />
            <div className="flex-1 min-w-0">
              <p className="text-[13px] font-medium text-on-surface truncate">{job.job_name}</p>
              <p className="text-[11px] text-outline truncate">
                {job.status === 'FAILED' && job.last_failure_at
                  ? `Failed ${formatRelative(job.last_failure_at)}`
                  : job.last_success_at
                    ? `Last success ${formatRelative(job.last_success_at)}`
                    : 'Never completed successfully'}
              </p>
            </div>
            <div className="flex items-center gap-1 shrink-0">
              <button
                type="button"
                onClick={() => onRunNow(job.job_id, job.job_name)}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-primary/10 text-primary hover:bg-primary hover:text-on-primary transition-all duration-200 shrink-0 text-[12px] font-medium"
                title="Run now (with pre-flight check)"
              >
                <Play size={12} className="fill-current" />
                Retry
              </button>
              <button
                type="button"
                onClick={() => onSelectJob(job.job_id)}
                className="p-1 rounded-lg text-on-surface-variant hover:bg-surface-container-high transition-colors"
                title="View job details"
              >
                <ChevronRight size={15} />
              </button>
            </div>
          </div>
        ))}
        {anomalies.map((a) => (
          <div key={a.job_id} className="px-4 py-2 flex items-center gap-3">
            <TrendingDown size={14} className="text-amber-600 shrink-0" />
            <p className="text-[12px] text-on-surface-variant truncate">
              <span className="font-medium text-on-surface">{a.job_name}</span>
              {' — size '}
              <span className="font-mono font-medium text-amber-700">
                {a.pct_change > 0 ? '+' : ''}{a.pct_change}%
              </span>
              {' vs baseline'}
            </p>
          </div>
        ))}
      </div>
    </div>
  )
}

export default AttentionPanel
