import { ArrowRight, Play } from 'lucide-react'
import StatusBadge from '@/components/StatusBadge'
import { SkeletonListRow } from '@/components/Skeleton'
import { formatDuration, formatRelative } from '@/utils/format'
import type { JobHealth } from '@/types'
import { fmtMinutes, slaColor, slaLabel } from '../utils/slaFormat'

const MAX_JOBS = 5

interface JobsPanelProps {
  health: JobHealth[]
  isLoading: boolean
  onRunNow: (jobId: string, jobName: string) => void
  onNewJob: () => void
  onSelectJob: (jobId: string) => void
  onViewAll: () => void
}

// Most-recently-active first, so the preview shows what changed last.
// Jobs that never ran sort to the bottom (empty date = "oldest").
const byRecency = (a: JobHealth, b: JobHealth) =>
  (b.last_success_at ?? b.next_run_at ?? '').localeCompare(a.last_success_at ?? a.next_run_at ?? '')

// Unified job list: health status, RPO/SLA posture, freshness, and run
// actions in one rail. Preview is deliberately short — the most recent
// workloads only; /jobs has the full filterable list.
const JobsPanel = ({ health, isLoading, onRunNow, onNewJob, onSelectJob, onViewAll }: JobsPanelProps) => (
  <div className="flex flex-col flex-1 min-h-0 rounded-lg bg-surface-container-lowest shadow-sm overflow-hidden border border-surface-variant">
    <div className="px-4 py-2.5 bg-surface-container-low/30 flex items-center justify-between shrink-0">
      <div className="flex items-center gap-2">
        <span className="relative flex h-2 w-2">
          <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-primary opacity-75" />
          <span className="relative inline-flex rounded-full h-2 w-2 bg-primary" />
        </span>
        <h2 className="text-[13px] font-semibold text-on-surface">Recent Workloads</h2>
      </div>
      <span className="text-[11px] font-mono text-on-surface-variant" title={`${health.length} workloads in total — open the jobs page for all of them`}>
        {isLoading || health.length === 0 ? '' : `top ${Math.min(MAX_JOBS, health.length)} of ${health.length}`}
      </span>
    </div>
    <div className="flex-1 min-h-0 overflow-y-auto flex flex-col divide-y divide-[#e9edff]">
      {isLoading ? (
        [...Array(4)].map((_, i) => <SkeletonListRow key={i} />)
      ) : health.length === 0 ? (
        <div className="p-5 text-center text-[13px] text-on-surface-variant">
          No jobs yet.{' '}
          <button type="button" onClick={onNewJob} className="text-primary hover:underline">
            Create one
          </button>
        </div>
      ) : (
        [...health]
          .sort(byRecency)
          .slice(0, MAX_JOBS)
          .map((job) => (
          <div
            key={job.job_id}
            role="button"
            tabIndex={0}
            onClick={() => onSelectJob(job.job_id)}
            onKeyDown={(e) => {
              if (e.target !== e.currentTarget) return
              if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault()
                onSelectJob(job.job_id)
              }
            }}
            className="px-3.5 py-2.5 flex items-center gap-3 hover:bg-surface-container-low/40 transition-colors cursor-pointer shrink-0 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary/40"
          >
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <p className="text-[13px] font-medium text-on-surface truncate">{job.job_name}</p>
                <StatusBadge status={job.status} size="sm" />
              </div>
              <p className="text-[11px] text-on-surface-variant mt-0.5 truncate">
                {job.last_success_at ? formatRelative(job.last_success_at) : 'Never run'}
                {job.next_run_at ? ` · next ${formatRelative(job.next_run_at)}` : ''}
                {' · '}{job.recovery_points} pts
              </p>
              {(job.rpo_target_minutes > 0 || job.sla_target_minutes > 0) && (
                <p className="text-[11px] mt-0.5 truncate">
                  {job.rpo_target_minutes > 0 && (
                    <span className={slaColor(job.rpo_status)}>
                      RPO {slaLabel(job.rpo_status)} {fmtMinutes(job.actual_rpo_minutes)}/{fmtMinutes(job.rpo_target_minutes)}
                    </span>
                  )}
                  {job.rpo_target_minutes > 0 && job.sla_target_minutes > 0 && <span className="text-on-surface-variant"> · </span>}
                  {job.sla_target_minutes > 0 && (
                    <span className={slaColor(job.sla_status)}>
                      SLA {slaLabel(job.sla_status)} {formatDuration(job.last_duration_seconds)}/{fmtMinutes(job.sla_target_minutes)}
                    </span>
                  )}
                </p>
              )}
            </div>
            <button
              type="button"
              onClick={(e) => { e.stopPropagation(); onRunNow(job.job_id, job.job_name) }}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-primary/10 text-primary hover:bg-primary hover:text-on-primary transition-all duration-200 shrink-0 text-[12px] font-medium"
              title="Run now (with pre-flight check)"
            >
              <Play size={12} className="fill-current" />
              Run
            </button>
          </div>
        ))
      )}
    </div>
    {health.length > 0 && (
      <button
        type="button"
        onClick={onViewAll}
        className="flex items-center justify-center gap-1.5 px-4 py-2 text-[12px] font-medium text-primary bg-surface-container-low/40 hover:bg-surface-container-low transition-colors shrink-0"
      >
        View all {health.length} workloads
        <ArrowRight size={13} />
      </button>
    )}
  </div>
)

export default JobsPanel
