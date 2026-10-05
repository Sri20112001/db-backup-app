import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useUIStore } from '@/store/uiStore'
import RunDetailDrawer from '@/features/history/components/RunDetailDrawer'
import NewJobModal from '@/features/jobs/components/NewJobModal'
import { useDashboardData } from '../hooks/useDashboardData'
import AttentionPanel from './AttentionPanel'
import DashboardHeader from './DashboardHeader'
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
    overview, health, isLoading,
    preflight, setPreflight,
    loadDashboard, handleRunNow, doRunNow,
  } = useDashboardData()

  const failedJobs = health.filter((h) => h.status === 'FAILED' || h.status === 'WARNING')
  const anomalies = overview?.size_anomalies ?? []
  const preflightJob = preflight ? health.find((h) => h.job_id === preflight.jobId) : undefined

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

      <MetricsRow overview={overview} isLoading={isLoading} />

      <div className="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-12 gap-4">
        {/* Left: issues first, then the live run log */}
        <div className="lg:col-span-8 min-h-0 flex flex-col gap-4">
          <AttentionPanel
            failedJobs={failedJobs}
            anomalies={anomalies}
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

      {runDetailId && <RunDetailDrawer runId={runDetailId} />}
      {showNew && <NewJobModal onClose={() => setShowNew(false)} onSaved={loadDashboard} />}
    </div>
  )
}

export default DashboardPage
