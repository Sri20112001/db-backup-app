import { useNavigate } from 'react-router-dom'
import { useAuthStore } from '@/store/authStore'
import { useOnboardingStore } from '@/store/onboardingStore'
import { useOnboardingStatus } from '@/features/onboarding/hooks/useOnboardingStatus'
import { Database, Server, Archive, Check, X, ArrowRight } from 'lucide-react'

const ITEMS = [
  { key: 'hasStorage', label: 'Add storage', to: '/storage', icon: Database },
  { key: 'hasAgent', label: 'Register an agent', to: '/agents', icon: Server },
  { key: 'hasJob', label: 'Create a backup job', to: '/jobs', icon: Archive },
] as const

// Dismissible first-run checklist. Hidden once setup is complete or the user
// dismisses it for the current org; offers a shortcut back into the wizard.
const GettingStartedCard = () => {
  const navigate = useNavigate()
  const { currentOrg } = useAuthStore()
  const { hasStorage, hasAgent, hasJob, allDone, isLoading } = useOnboardingStatus()
  const isDismissed = useOnboardingStore((s) => s.isDismissed)
  const dismiss = useOnboardingStore((s) => s.dismiss)

  if (isLoading || allDone || isDismissed(currentOrg?.id)) return null

  const flags: Record<string, boolean> = { hasStorage, hasAgent, hasJob }
  const doneCount = [hasStorage, hasAgent, hasJob].filter(Boolean).length

  return (
    <div className="shrink-0 rounded-lg bg-surface-container-lowest border border-surface-variant shadow-sm p-4">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h2 className="text-[14px] font-semibold text-on-surface">Getting started</h2>
          <p className="text-[12px] text-on-surface-variant mt-0.5">
            {doneCount} of 3 complete — finish setting up to run your first backup.
          </p>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <button
            type="button"
            onClick={() => navigate('/onboarding')}
            className="flex items-center gap-1.5 px-3 h-8 rounded-lg bg-primary text-on-primary text-[12px] font-medium hover:bg-primary-container transition-colors"
          >
            Resume setup
            <ArrowRight size={13} />
          </button>
          <button
            type="button"
            onClick={() => currentOrg && dismiss(currentOrg.id)}
            title="Dismiss"
            className="p-1.5 rounded-lg text-on-surface-variant hover:bg-surface-container-high transition-colors"
          >
            <X size={16} />
          </button>
        </div>
      </div>
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-2 mt-3">
        {ITEMS.map((it) => {
          const done = flags[it.key]
          return (
            <button
              key={it.key}
              type="button"
              onClick={() => { if (!done) navigate(it.to) }}
              className={`flex items-center gap-3 p-3 rounded-lg border text-left transition-colors ${done ? 'border-primary/30 bg-primary-container/10 cursor-default' : 'border-surface-variant hover:bg-surface-container-low'}`}
            >
              <div className={`w-8 h-8 rounded-lg flex items-center justify-center shrink-0 ${done ? 'bg-primary text-on-primary' : 'bg-surface-container-high text-on-surface-variant'}`}>
                {done ? <Check size={16} /> : <it.icon size={16} />}
              </div>
              <span className={`text-[13px] font-medium ${done ? 'text-on-surface' : 'text-on-surface-variant'}`}>{it.label}</span>
            </button>
          )
        })}
      </div>
    </div>
  )
}

export default GettingStartedCard
