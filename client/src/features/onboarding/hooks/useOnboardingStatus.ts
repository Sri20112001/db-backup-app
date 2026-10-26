import { useCallback, useEffect, useState } from 'react'
import { agentApi, jobApi, storageApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'

export interface OnboardingStatus {
  hasStorage: boolean
  hasAgent: boolean
  hasJob: boolean
  allDone: boolean
  isLoading: boolean
  reload: () => void
}

// Derives setup progress from live resources rather than a stored flag, so the
// wizard and checklist stay accurate for returning users (and after a refresh).
export const useOnboardingStatus = (): OnboardingStatus => {
  const { currentOrg } = useAuthStore()
  const orgId = currentOrg?.id
  const [counts, setCounts] = useState({ hasStorage: false, hasAgent: false, hasJob: false })
  const [isLoading, setIsLoading] = useState(true)
  const [tick, setTick] = useState(0)

  useEffect(() => {
    if (!orgId) {
      setCounts({ hasStorage: false, hasAgent: false, hasJob: false })
      setIsLoading(false)
      return
    }
    let cancelled = false
    setIsLoading(true)
    Promise.all([
      storageApi.list(orgId).then((s) => s.length).catch(() => 0),
      agentApi.list(orgId, { limit: 1 }).then((a) => a.total).catch(() => 0),
      jobApi.list(orgId, { limit: 1 }).then((j) => j.total).catch(() => 0),
    ])
      .then(([storage, agents, jobs]) => {
        if (cancelled) return
        setCounts({ hasStorage: storage > 0, hasAgent: agents > 0, hasJob: jobs > 0 })
      })
      .finally(() => {
        if (!cancelled) setIsLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [orgId, tick])

  const reload = useCallback(() => setTick((t) => t + 1), [])

  return {
    ...counts,
    allDone: counts.hasStorage && counts.hasAgent && counts.hasJob,
    isLoading,
    reload,
  }
}
