import { useEffect, useState } from 'react'
import { dashboardApi, jobApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import { useRealtimeStore } from '@/stores/realtimeStore'
import type { DashboardOverview, JobHealth, PreflightResult } from '@/types'

export interface PreflightState {
  jobId: string
  result: PreflightResult
}

// Owns dashboard fetching, realtime refresh triggers, and the
// preflight-gated "run now" flow. Pure data — no JSX.
export function useDashboardData() {
  const { currentOrg } = useAuthStore()
  const { addToast } = useUIStore()
  const [overview, setOverview] = useState<DashboardOverview | null>(null)
  const [health, setHealth] = useState<JobHealth[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [preflight, setPreflight] = useState<PreflightState | null>(null)

  const jobsSeq = useRealtimeStore((s) => s.entitySeq.jobs)
  const presenceSeq = useRealtimeStore((s) => s.presenceSeq)
  const alertTick = useRealtimeStore((s) => s.alertTick)
  const lastRunEvent = useRealtimeStore((s) => s.lastRunEvent)

  const loadDashboard = () => {
    if (!currentOrg) return
    setIsLoading(true)
    Promise.all([
      dashboardApi.overview(currentOrg.id),
      dashboardApi.health(currentOrg.id),
    ])
      .then(([ov, h]) => { setOverview(ov); setHealth(h) })
      .catch(() => addToast('error', 'Failed to load dashboard'))
      .finally(() => setIsLoading(false))
  }

  useEffect(() => { loadDashboard() }, [currentOrg]) // eslint-disable-line
  useEffect(() => { loadDashboard() }, [jobsSeq, presenceSeq, alertTick]) // eslint-disable-line
  useEffect(() => {
    if (!lastRunEvent) return
    const s = lastRunEvent.status ?? ''
    // eslint-disable-next-line react-hooks/set-state-in-effect -- realtime event fans out to a refetch by design
    if (s === 'PENDING' || s === 'COMPLETED' || s === 'FAILED' || s === 'CANCELLED') loadDashboard()
  }, [lastRunEvent]) // eslint-disable-line

  const doRunNow = async (jobId: string, jobName: string) => {
    if (!currentOrg) return
    setPreflight(null)
    try {
      await jobApi.runNow(currentOrg.id, jobId)
      addToast('success', `${jobName} started`)
    } catch {
      addToast('error', 'Failed to start backup')
    }
  }

  const handleRunNow = async (jobId: string, jobName: string) => {
    if (!currentOrg) return
    // Run preflight first
    try {
      const result = await jobApi.preflight(currentOrg.id, jobId)
      if (result.overall === 'FAIL' || result.overall === 'WARN') {
        setPreflight({ jobId, result })
        return
      }
    } catch {
      // preflight unavailable — proceed anyway
    }
    await doRunNow(jobId, jobName)
  }

  return { overview, health, isLoading, preflight, setPreflight, loadDashboard, handleRunNow, doRunNow }
}
