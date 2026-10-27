import { useCallback, useEffect, useRef, useState } from 'react'
import { dashboardApi, jobApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import { useRealtimeStore } from '@/store/realtimeStore'
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
  const [isRefreshing, setIsRefreshing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [preflight, setPreflight] = useState<PreflightState | null>(null)

  const jobsSeq = useRealtimeStore((s) => s.entitySeq.jobs)
  const presenceSeq = useRealtimeStore((s) => s.presenceSeq)
  const lastRunEvent = useRealtimeStore((s) => s.lastRunEvent)

  // reqId: only the latest load may write state, so a slow response for the
  // previous org can never overwrite the current one. pending: callers firing
  // in the same tick (mount runs several effects) coalesce into one fetch.
  const reqIdRef = useRef(0)
  const pendingRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const loadedRef = useRef(false)

  const loadDashboard = useCallback(() => {
    if (!currentOrg) return
    if (pendingRef.current) return
    pendingRef.current = setTimeout(() => {
      pendingRef.current = null
      const reqId = ++reqIdRef.current
      if (loadedRef.current) setIsRefreshing(true)
      else setIsLoading(true)
      Promise.all([
        dashboardApi.overview(currentOrg.id),
        dashboardApi.health(currentOrg.id),
      ])
        .then(([ov, h]) => {
          if (reqId !== reqIdRef.current) return
          setOverview(ov)
          setHealth(h)
          setError(null)
          loadedRef.current = true
        })
        .catch(() => {
          if (reqId !== reqIdRef.current) return
          setError('Failed to load dashboard')
          addToast('error', 'Failed to load dashboard')
        })
        .finally(() => {
          if (reqId !== reqIdRef.current) return
          setIsLoading(false)
          setIsRefreshing(false)
        })
    }, 0)
  }, [currentOrg, addToast])

  // Org switch: drop the previous org's data immediately (no cross-org flash)
  // and invalidate in-flight responses from the old org.
  useEffect(() => {
    if (pendingRef.current) {
      clearTimeout(pendingRef.current)
      pendingRef.current = null
    }
    reqIdRef.current++
    loadedRef.current = false
    setOverview(null)
    setHealth([])
    setError(null)
    setIsLoading(true)
    setIsRefreshing(false)
  }, [currentOrg])

  // Unmount: cancel scheduled work and ignore late responses.
  useEffect(() => () => {
    reqIdRef.current++
    if (pendingRef.current) {
      clearTimeout(pendingRef.current)
      pendingRef.current = null
    }
  }, [])

  useEffect(() => { loadDashboard() }, [loadDashboard])
  // seq deps intentionally drive refetches without appearing in the body.
  useEffect(() => { loadDashboard() }, [jobsSeq, presenceSeq, loadDashboard])
  useEffect(() => {
    if (!lastRunEvent) return
    const s = lastRunEvent.status ?? ''
    if (s === 'PENDING' || s === 'COMPLETED' || s === 'FAILED' || s === 'CANCELLED') loadDashboard()
  }, [lastRunEvent, loadDashboard])

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

  return { overview, health, isLoading, isRefreshing, error, preflight, setPreflight, loadDashboard, handleRunNow, doRunNow }
}
