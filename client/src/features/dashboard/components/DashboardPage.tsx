import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { AlertTriangle } from 'lucide-react'
import { useUIStore } from '@/store/uiStore'
import RunDetailDrawer from '@/features/history/components/RunDetailDrawer'
import NewJobModal from '@/features/jobs/components/NewJobModal'
import { useDashboardData } from '../hooks/useDashboardData'
import AttentionPanel from './AttentionPanel'
import DashboardHeader from './DashboardHeader'
import GettingStartedCard from './GettingStartedCard'
import JobsPanel from './JobsPanel'
import MetricsRow from './MetricsRow'
import PreflightModal from './PreflightModal'
import RecentRunsTable from './RecentRunsTable'

// Command Center layout (top → bottom):
//   1. slim header bar (title, agent presence, actions)
//   2. KPI hero band (fleet totals at a glance)
//   3. split view — left: attention lane + recent executions;
//      right: unified job list rail.
// Fixed viewport: bottom panels share leftover height and scroll inside.
const DashboardPage = () => {
  const navigate = useNavigate()
  const { openRunDetail, runDetailId } = useUIStore()
  const [showNew, setShowNew] = useState(false)

  const {
    overview, health, isLoading, isRefreshing, error,
    preflight, setPreflight,
    loadDashboard, handleRunNow, doRunNow,
  } = useDashboardData()

  const failedJobs = health.filter((h) => h.status === 'FAILED' || h.status === 'WARNING')
  const anomalies = overview?.size_anomalies ?? []
  const preflightJob = preflight ? health.find((h) => h.job_id === preflight.jobId) : undefined
  // Nothing loaded and the load failed: replace the widgets with an error
  // state instead of rendering zeros and a false "All clear".
  const loadFailed = error !== null && !isLoading && overview === null

  return (
    <div className="h-full min-h-0 overflow-hidden flex flex-col gap-4 pr-0.5">

      {preflight && (
        <PreflightModal
          result={preflight.result}
          onClose={() => setPreflight(null)}
          onRunAnyway={() => doRunNow(preflight.jobId, preflightJob?.job_name ?? preflight.jobId)}
        />
      )}

      <DashboardHeader overview={overview} onNewJob={() => setShowNew(true)} />

      {loadFailed ? (
        <div role="alert" className="flex-1 flex flex-col items-center justify-center gap-4 text-center px-4">
          <div className="w-12 h-12 rounded-full bg-error-container flex items-center justify-center">
            <AlertTriangle size={24} className="text-error" />
          </div>
          <div className="flex flex-col gap-1">
            <p className="text-[15px] font-semibold text-on-surface">Couldn't load the dashboard</p>
            <p className="text-[13px] text-on-surface-variant">{error}</p>
          </div>
          <button
            type="button"
            onClick={loadDashboard}
            className="px-4 h-9 rounded-lg bg-primary text-on-primary text-[13px] font-medium hover:bg-primary-container transition-colors"
          >
            Retry
          </button>
        </div>
      ) : (
        <>
          {/* Refresh failed but earlier data exists: keep showing it, flagged. */}
          {error !== null && overview !== null && (
            <div role="alert" className="shrink-0 flex items-center gap-3 px-3 py-2 rounded-lg bg-error-container/40 border border-error/30">
              <p className="text-[12.5px] text-on-surface flex-1">
                Couldn't refresh — showing the last loaded data.
              </p>
              <button type="button" onClick={loadDashboard} className="text-[12px] text-primary hover:underline shrink-0">
                Retry
              </button>
            </div>
          )}

          {/* Subtle refetch signal: data stays on screen while it refreshes. */}
          {isRefreshing && (
            <div className="shrink-0 h-0.5 rounded-full bg-primary/40 animate-pulse" aria-hidden="true" />
          )}

          <GettingStartedCard />

          <MetricsRow overview={overview} isLoading={isLoading} />

          <div className="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-12 gap-4">
            {/* Left: issues first, then the live run log */}
            <div className="lg:col-span-8 min-h-0 flex flex-col gap-4">
              <AttentionPanel
                failedJobs={failedJobs}
                anomalies={anomalies}
                isLoading={isLoading}
                onRunNow={handleRunNow}
                onSelectJob={(jobId) => navigate(`/jobs/${jobId}`)}
                onReviewAlerts={() => navigate('/alerts')}
              />
              <RecentRunsTable
                runs={overview?.recent_runs ?? []}
                isLoading={isLoading}
                onOpenRun={openRunDetail}
                onViewAll={() => navigate('/history')}
                onNewJob={() => setShowNew(true)}
              />
            </div>

            {/* Right: every job — health, posture, and actions in one rail */}
            <div className="lg:col-span-4 min-h-0 flex flex-col">
              <JobsPanel
                health={health}
                isLoading={isLoading}
                onRunNow={handleRunNow}
                onNewJob={() => setShowNew(true)}
                onSelectJob={(jobId) => navigate(`/jobs/${jobId}`)}
                onViewAll={() => navigate('/jobs')}
              />
            </div>
          </div>
        </>
      )}

      {runDetailId && <RunDetailDrawer runId={runDetailId} />}
      {showNew && <NewJobModal onClose={() => setShowNew(false)} onSaved={loadDashboard} />}
    </div>
  )
}

export default DashboardPage
