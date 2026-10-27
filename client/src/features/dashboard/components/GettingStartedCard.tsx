import { useNavigate } from 'react-router-dom'
import { useAuthStore } from '@/store/authStore'
import { useOnboardingStore } from '@/store/onboardingStore'
import { useOnboardingStatus } from '@/features/onboarding/hooks/useOnboardingStatus'
import { Database, Server, Archive, Check, X, ArrowRight } from 'lucide-react'
import { Card, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'

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
    <Card padding="sm" className="shrink-0">
      <CardHeader
        action={
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
              <X size={14} />
            </button>
          </div>
        }
      >
        <CardTitle as="h2" className="text-[14px]">Getting started</CardTitle>
        <CardDescription className="text-[12px]">
          {doneCount} of 3 complete — finish setting up to run your first backup.
        </CardDescription>
      </CardHeader>
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 pt-1">
        {ITEMS.map((item) => {
          const done = flags[item.key]
          const Icon = item.icon
          return (
            <button
              key={item.key}
              type="button"
              onClick={() => navigate(item.to)}
              className={`flex items-center gap-3 p-3 rounded-lg border text-left transition-all ${
                done
                  ? 'border-surface-variant bg-surface-container-low/50 opacity-60'
                  : 'border-surface-variant bg-surface-container-low hover:border-primary/50 hover:bg-primary-container/10'
              }`}
            >
              <div
                className={`w-7 h-7 rounded-md flex items-center justify-center shrink-0 ${
                  done
                    ? 'bg-emerald-500/10 text-emerald-600'
                    : 'bg-primary-container/30 text-primary'
                }`}
              >
                {done ? <Check size={14} /> : <Icon size={14} />}
              </div>
              <div className="min-w-0 flex-1">
                <p className="text-[13px] font-medium text-on-surface truncate">{item.label}</p>
                <p className="text-[11px] text-on-surface-variant">
                  {done ? 'Complete' : 'Not configured'}
                </p>
              </div>
            </button>
          )
        })}
      </div>
    </Card>
  )
}

export default GettingStartedCard
