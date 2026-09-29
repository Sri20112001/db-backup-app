import { ArrowRight, Play } from 'lucide-react'
import StatusBadge from '@/components/StatusBadge'
import { SkeletonRow } from '@/components/Skeleton'
import { formatDuration, formatRelative } from '@/utils/format'
import type { JobHealth } from '@/types'
import { fmtMinutes, slaColor, slaLabel } from '../utils/slaFormat'

const MAX_JOBS = 10

interface JobsPanelProps {
  health: JobHealth[]
  isLoading: boolean
  onRunNow: (jobId: string, jobName: string) => void
  onNewJob: () => void
  onSelectJob: (jobId: string) => void
  onViewAll: () => void
}

// Unified job list: health status, RPO/SLA posture, freshness, and run
// actions in one scrollable rail. Preview is capped; /jobs has the full
// filterable list.
const JobsPanel = ({ health, isLoading, onRunNow, onNewJob, onSelectJob, onViewAll }: JobsPanelProps) => (
  <div className="flex flex-col flex-1 min-h-0 rounded-lg bg-surface-container-lowest shadow-sm overflow-hidden border border-surface-variant">
    <div className="px-4 py-2.5 bg-surface-container-low/30 flex items-center justify-between shrink-0">
      <div className="flex items-center gap-2">
        <span className="relative flex h-2 w-2">
          <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-primary opacity-75" />
          <span className="relative inline-flex rounded-full h-2 w-2 bg-primary" />
        </span>
        <h2 className="text-[13px] font-semibold text-on-surface">Backup Jobs</h2>
      </div>
      <span className="text-[11px] font-mono text-outline">{health.length}</span>
    </div>
    <div className="flex-1 min-h-0 overflow-y-auto flex flex-col divide-y divide-[#e9edff]">
      {isLoading ? (
        [...Array(4)].map((_, i) => <SkeletonRow key={i} cols={1} />)
      ) : health.length === 0 ? (
        <div className="p-5 text-center text-[13px] text-outline">
          No jobs yet.{' '}
          <button type="button" onClick={onNewJob} className="text-primary hover:underline">
            Create one
          </button>
        </div>
      ) : (
        health.slice(0, MAX_JOBS).map((job) => (
          <div
            key={job.job_id}
            onClick={() => onSelectJob(job.job_id)}
            className="px-3.5 py-2.5 flex items-center gap-3 hover:bg-surface-container-low/40 transition-colors cursor-pointer shrink-0"
          >
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <p className="text-[13px] font-medium text-on-surface truncate">{job.job_name}</p>
                <StatusBadge status={job.status} size="sm" />
              </div>
              <p className="text-[11px] text-outline mt-0.5 truncate">
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
                  {job.rpo_target_minutes > 0 && job.sla_target_minutes > 0 && <span className="text-outline"> · </span>}
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
              className="p-1.5 rounded-lg text-on-surface-variant hover:bg-surface-container-high hover:text-on-surface transition-colors shrink-0"
              title="Run now (with pre-flight check)"
            >
              <Play size={14} />
            </button>
          </div>
        ))
      )}
    </div>
    {health.length > MAX_JOBS && (
      <button
        type="button"
        onClick={onViewAll}
        className="flex items-center justify-center gap-1.5 px-4 py-2 text-[12px] font-medium text-primary bg-surface-container-low/40 hover:bg-surface-container-low transition-colors shrink-0"
      >
        View all {health.length} jobs
        <ArrowRight size={13} />
      </button>
    )}
  </div>
)

export default JobsPanel
