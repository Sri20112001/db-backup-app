import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { dashboardApi, jobApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import type { DashboardOverview, JobHealth } from '@/types'
import StatusBadge from '@/components/StatusBadge'
import { SkeletonCard } from '@/components/Skeleton'
import { formatBytes, formatRelative, formatDuration } from '@/utils/format'
import RunDetailDrawer from '@/features/history/components/RunDetailDrawer'
import {
  AlertTriangle, X, Plus, Radio, FolderArchive, BadgeCheck,
  LayoutGrid, Play,
} from 'lucide-react'

const DashboardPage = () => {
  const { currentOrg } = useAuthStore()
  const { addToast, openRunDetail, runDetailId } = useUIStore()
  const navigate = useNavigate()
  const [overview, setOverview] = useState<DashboardOverview | null>(null)
  const [health, setHealth] = useState<JobHealth[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [alertDismissed, setAlertDismissed] = useState(false)

  useEffect(() => {
    if (!currentOrg) return
    setIsLoading(true)
    Promise.all([
      dashboardApi.overview(currentOrg.id),
      dashboardApi.health(currentOrg.id),
    ])
      .then(([ov, h]) => { setOverview(ov); setHealth(h) })
      .catch(() => addToast('error', 'Failed to load dashboard'))
      .finally(() => setIsLoading(false))
  }, [currentOrg])

  const handleRunNow = async (jobId: string, jobName: string) => {
    if (!currentOrg) return
    try {
      await jobApi.runNow(currentOrg.id, jobId)
      addToast('success', `${jobName} started`)
    } catch {
      addToast('error', 'Failed to start backup')
    }
  }

  const failedAlerts = health.filter((h) => h.status === 'FAILED' || h.status === 'WARNING')

  return (
    <div className="flex flex-col gap-6">
      {/* Alert banner */}
      {!alertDismissed && failedAlerts.length > 0 && (
        <div className="flex items-center justify-between px-4 py-2.5 rounded-lg bg-[#ffdad6]/60 text-[#141b2b] shadow-sm">
          <div className="flex items-center gap-3 min-w-0">
            <span className="flex items-center justify-center w-6 h-6 rounded-full bg-[#ba1a1a] text-white shrink-0">
              <AlertTriangle size={14} />
            </span>
            <p className="text-[14px] font-medium truncate">
              <strong>{failedAlerts.length} job{failedAlerts.length > 1 ? 's' : ''} require attention</strong>
              {' — '}{failedAlerts.map((f) => f.job_name).join(', ')}
            </p>
          </div>
          <div className="flex items-center gap-2 shrink-0">
            <button
              type="button"
              onClick={() => navigate('/alerts')}
              className="px-3 py-1 rounded text-[13px] text-[#004ac6] hover:bg-[#ffffff]/60 transition-colors"
            >
              Review alerts →
            </button>
            <button
              type="button"
              onClick={() => setAlertDismissed(true)}
              className="w-7 h-7 flex items-center justify-center rounded text-[#434655] hover:bg-[#ffffff]/50 transition-colors"
            >
              <X size={16} />
            </button>
          </div>
        </div>
      )}

      {/* Header */}
      <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-4">
        <div>
          <h1 className="text-[20px] font-semibold text-[#141b2b] tracking-tight">Command Center</h1>
          <p className="text-[12px] text-[#434655] mt-0.5">
            Live status of backup infrastructure, agent nodes, and replication pipelines.
          </p>
        </div>
        <div className="flex items-center gap-3">
          {overview && (
            <div className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-[#ffffff] shadow-sm border border-[#e9edff]">
              <span className="relative flex h-2.5 w-2.5">
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-[#39b8fd] opacity-75" />
                <span className="relative inline-flex rounded-full h-2.5 w-2.5 bg-[#006591]" />
              </span>
              <span className="text-[12px] font-medium text-[#141b2b]">
                Agents: <strong className="text-[#006591]">{overview.agents_online}/{overview.agents_online + overview.agents_offline} Online</strong>
              </span>
            </div>
          )}
          <button
            type="button"
            onClick={() => navigate('/jobs/new')}
            className="flex items-center gap-2 px-4 py-2 rounded-lg bg-[#2563eb] text-white text-[13px] font-medium hover:bg-[#1d4ed8] transition-colors shadow-sm"
          >
            <Plus size={16} />
            New Backup Job
          </button>
        </div>
      </div>

      {/* Health strip */}
      <div className="flex flex-col gap-2">
        <div className="flex items-center justify-between">
          <span className="text-[12px] font-semibold uppercase tracking-wider text-[#434655]">System Backup Health</span>
          <span className="text-[12px] font-mono text-[#737686]">{health.length} active tracks</span>
        </div>
        {isLoading ? (
          <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-5 gap-3">
            {[...Array(5)].map((_, i) => <SkeletonCard key={i} />)}
          </div>
        ) : health.length === 0 ? (
          <div className="p-6 rounded-lg bg-[#ffffff] border border-[#e9edff] text-center text-[13px] text-[#737686]">
            No backup jobs configured yet.{' '}
            <button type="button" onClick={() => navigate('/jobs/new')} className="text-[#004ac6] hover:underline">
              Create your first job
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-5 gap-3">
            {health.map((job) => {
              const borderColor = job.status === 'HEALTHY' ? 'bg-[#006591]' : job.status === 'FAILED' ? 'bg-[#ba1a1a]' : job.status === 'WARNING' ? 'bg-[#39b8fd]' : 'bg-[#737686]'
              return (
                <div
                  key={job.job_id}
                  onClick={() => navigate(`/jobs/${job.job_id}`)}
                  className="relative flex flex-col justify-between p-3.5 rounded-lg bg-[#ffffff] shadow-sm hover:shadow-md transition-shadow overflow-hidden cursor-pointer"
                >
                  <div className={`absolute left-0 top-0 bottom-0 w-1.5 ${borderColor}`} />
                  <div className="flex items-start justify-between gap-1 pl-1">
                    <div className="min-w-0">
                      <h2 className="text-[13px] font-medium text-[#141b2b] truncate">{job.job_name}</h2>
                    </div>
                    <StatusBadge status={job.status} size="sm" />
                  </div>
                  <div className="flex items-center justify-between text-[#434655] text-[11px] mt-3 pt-2 bg-[#f1f3ff]/40 rounded px-1.5">
                    <span>{job.last_success_at ? formatRelative(job.last_success_at) : 'Never run'}</span>
                    <span className="text-[#141b2b] font-medium">{job.recovery_points} pts</span>
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>

      {/* Two-column layout */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
        {/* Left: metrics + recent runs */}
        <div className="lg:col-span-8 flex flex-col gap-6">
          {/* Metric tiles */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            {isLoading ? (
              [...Array(4)].map((_, i) => <SkeletonCard key={i} />)
            ) : (
              <>
                <MetricTile Icon={LayoutGrid} iconBg="bg-[#dbe1ff]" iconColor="text-[#004ac6]"
                  label="Total Backup Jobs" value={`${overview?.total_jobs ?? 0}`}
                  sub={`${overview?.enabled_jobs ?? 0} active`} />
                <MetricTile Icon={Radio} iconBg="bg-[#c9e6ff]" iconColor="text-[#006591]"
                  label="Agent Fleet"
                  value={overview ? `${overview.agents_online} / ${overview.agents_online + overview.agents_offline} Online` : '0 Online'}
                  sub={overview && overview.agents_offline > 0 ? `${overview.agents_offline} offline` : 'No agents yet'} />
                <MetricTile Icon={FolderArchive} iconBg="bg-[#c9e6ff]/50" iconColor="text-[#006591]"
                  label="Storage Consumed" value={formatBytes(overview?.total_bytes ?? 0)}
                  sub="Total uploaded" />
                <MetricTile Icon={BadgeCheck} iconBg="bg-[#dbe1ff]" iconColor="text-[#004ac6]"
                  label="24h Execution" value={`${overview?.successful_runs ?? 0} ✓  ${overview?.failed_runs ?? 0} ✗`}
                  sub="Successful / Failed" />
              </>
            )}
          </div>

          {/* Recent runs table */}
          <div className="flex flex-col rounded-lg bg-[#ffffff] shadow-sm overflow-hidden border border-[#e9edff]">
            <div className="p-4 flex items-center justify-between bg-[#f1f3ff]/30">
              <div>
                <h2 className="text-[14px] font-semibold text-[#141b2b]">Recent Backup Executions</h2>
                <p className="text-[12px] text-[#434655]">Real-time pipeline run log</p>
              </div>
              <button type="button" onClick={() => navigate('/history')} className="text-[13px] text-[#004ac6] hover:underline">
                View all →
              </button>
            </div>
            <div className="overflow-x-auto">
              <table className="w-full text-left border-collapse">
                <thead>
                  <tr className="bg-[#f1f3ff] text-[12px] font-semibold uppercase tracking-wider text-[#434655]">
                    {['Status', 'Job', 'Type', 'Started', 'Duration', 'Uploaded', ''].map((h) => (
                      <th key={h} className="py-2.5 px-4">{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y divide-[#e9edff] text-[14px] text-[#141b2b]">
                  {isLoading ? (
                    [...Array(5)].map((_, i) => (
                      <tr key={i} className="animate-pulse">
                        {[...Array(7)].map((_, j) => (
                          <td key={j} className="py-3 px-4"><div className="h-4 bg-[#e9edff] rounded w-3/4" /></td>
                        ))}
                      </tr>
                    ))
                  ) : !overview?.recent_runs?.length ? (
                    <tr>
                      <td colSpan={7} className="py-12 text-center text-[13px] text-[#737686]">
                        No backup runs yet. <button type="button" onClick={() => navigate('/jobs/new')} className="text-[#004ac6] hover:underline">Create a backup job</button> to get started.
                      </td>
                    </tr>
                  ) : overview.recent_runs.map((run) => (
                    <tr
                      key={run.id}
                      onClick={() => openRunDetail(run.id)}
                      className={`hover:bg-[#f1f3ff]/50 transition-colors cursor-pointer ${run.status === 'FAILED' ? 'bg-[#ffdad6]/20' : run.status === 'RUNNING' || run.status === 'UPLOADING' ? 'bg-[#dbe1ff]/10' : ''}`}
                    >
                      <td className="py-3 px-4"><StatusBadge status={run.status} size="sm" /></td>
                      <td className="py-3 px-4 font-medium text-[#141b2b]">{run.backup_job?.name ?? '—'}</td>
                      <td className="py-3 px-4">
                        <span className="px-1.5 py-0.5 rounded bg-[#e9edff] text-[#434655] font-mono text-[11px]">
                          {run.source_type || run.backup_job?.source_type}
                        </span>
                      </td>
                      <td className="py-3 px-4 text-[12px] text-[#737686]">{formatRelative(run.started_at)}</td>
                      <td className="py-3 px-4 font-mono text-[12px]">{formatDuration(run.duration_seconds)}</td>
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
          </div>
        </div>

        {/* Right: activity feed + storage */}
        <div className="lg:col-span-4 flex flex-col gap-6">
          {/* Health jobs quick actions */}
          <div className="flex flex-col rounded-lg bg-[#ffffff] shadow-sm overflow-hidden border border-[#e9edff]">
            <div className="p-4 bg-[#f1f3ff]/30 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <span className="relative flex h-2 w-2">
                  <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-[#006591] opacity-75" />
                  <span className="relative inline-flex rounded-full h-2 w-2 bg-[#006591]" />
                </span>
                <h2 className="text-[14px] font-semibold text-[#141b2b]">Job Quick Actions</h2>
              </div>
            </div>
            <div className="flex flex-col divide-y divide-[#e9edff]">
              {health.length === 0 ? (
                <div className="p-5 text-center text-[13px] text-[#737686]">
                  No jobs yet.{' '}
                  <button type="button" onClick={() => navigate('/jobs/new')} className="text-[#004ac6] hover:underline">
                    Create one
                  </button>
                </div>
              ) : health.slice(0, 6).map((job) => (
                <div key={job.job_id} className="p-3.5 flex items-center justify-between gap-3 hover:bg-[#f1f3ff]/40 transition-colors">
                  <div className="flex-1 min-w-0">
                    <p className="text-[13px] font-medium text-[#141b2b] truncate">{job.job_name}</p>
                    <p className="text-[11px] text-[#737686] mt-0.5">
                      {job.last_success_at ? formatRelative(job.last_success_at) : 'Never run'} · {job.recovery_points} pts
                    </p>
                  </div>
                  <div className="flex items-center gap-2 shrink-0">
                    <StatusBadge status={job.status} size="sm" />
                    <button
                      type="button"
                      onClick={() => handleRunNow(job.job_id, job.job_name)}
                      className="p-1.5 rounded-lg text-[#434655] hover:bg-[#e9edff] hover:text-[#141b2b] transition-colors"
                      title="Run now"
                    >
                      <Play size={14} />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>

      {runDetailId && <RunDetailDrawer runId={runDetailId} />}
    </div>
  )
}

const MetricTile = ({ Icon, iconBg, iconColor, label, value, sub }: {
  Icon: React.ElementType; iconBg: string; iconColor: string; label: string; value: string; sub: string
}) => (
  <div className="flex flex-col justify-between p-4 rounded-lg bg-[#ffffff] shadow-sm border border-[#e9edff]">
    <div className="flex items-start justify-between">
      <div>
        <span className="text-[12px] font-semibold uppercase tracking-wider text-[#434655]">{label}</span>
        <div className="flex items-baseline gap-2 mt-1">
          <span className="text-[22px] font-bold text-[#141b2b] leading-tight">{value}</span>
        </div>
      </div>
      <div className={`w-9 h-9 rounded-lg ${iconBg} flex items-center justify-center ${iconColor} shrink-0`}>
        <Icon size={18} />
      </div>
    </div>
    <p className="text-[11px] text-[#737686] mt-2">{sub}</p>
  </div>
)

export default DashboardPage
