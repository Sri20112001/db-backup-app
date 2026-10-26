import { useState } from 'react'
import { Navigate, useNavigate } from 'react-router-dom'
import { useAuthStore } from '@/store/authStore'
import { useOnboardingStore } from '@/store/onboardingStore'
import { useOnboardingStatus } from './hooks/useOnboardingStatus'
import { Database, Server, Archive, Check } from 'lucide-react'
import { Logo } from '@/components/ui/Logo'
import ToastContainer from '@/components/ToastContainer'
import WelcomeStep from './steps/WelcomeStep'
import StorageStep from './steps/StorageStep'
import AgentStep from './steps/AgentStep'
import JobStep from './steps/JobStep'
import DoneStep from './steps/DoneStep'

const RAIL = [
  { label: 'Add storage', icon: Database },
  { label: 'Register agent', icon: Server },
  { label: 'Create backup job', icon: Archive },
]

// Steps: 0 welcome · 1 storage · 2 agent · 3 job · 4 done
const OnboardingPage = () => {
  const navigate = useNavigate()
  const { currentOrg } = useAuthStore()
  const { hasStorage, hasAgent, hasJob, allDone, isLoading, reload } = useOnboardingStatus()
  const undismiss = useOnboardingStore((s) => s.undismiss)
  const [step, setStep] = useState(0)
  // Set the moment a job is created so the Done screen can confirm success
  // without waiting for the status refetch (and never over-claim on skip).
  const [jobCreated, setJobCreated] = useState(false)

  const effectiveHasJob = hasJob || jobCreated
  const doneFlags = [hasStorage, hasAgent, effectiveHasJob]
  const allComplete = hasStorage && hasAgent && effectiveHasJob
  const missing = [
    !hasStorage && 'a storage target',
    !hasAgent && 'a registered agent',
    !effectiveHasJob && 'a backup job',
  ].filter((v): v is string => Boolean(v))

  const finish = () => navigate('/', { replace: true })
  const skip = () => {
    // Bring the dashboard checklist back so setup can be resumed later.
    if (currentOrg) undismiss(currentOrg.id)
    navigate('/', { replace: true })
  }

  // Already-configured orgs (or users who finished elsewhere) skip straight in.
  if (!isLoading && allDone && step === 0) return <Navigate to="/" replace />

  return (
    <div className="min-h-screen bg-surface-container-low flex">
      <ToastContainer />

      <aside className="hidden lg:flex w-[280px] shrink-0 flex-col gap-8 p-8 bg-surface-container-lowest border-r border-surface-variant">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-xl bg-surface-container-highest flex items-center justify-center border border-outline-variant/30 shadow-inner">
            <Logo className="w-6 h-6" />
          </div>
          <div>
            <h1 className="text-[18px] font-semibold text-on-surface tracking-tight">VaultGuard</h1>
            <p className="text-[12px] text-on-surface-variant">Initial setup</p>
          </div>
        </div>
        <nav className="flex flex-col gap-1">
          {RAIL.map((s, i) => {
            const isDone = doneFlags[i]
            const isActive = step === i + 1
            const highlighted = isDone || isActive
            return (
              <div key={s.label} className={`flex items-center gap-3 px-3 py-2.5 rounded-lg ${isActive ? 'bg-primary-container/40' : ''}`}>
                <div className={`w-8 h-8 rounded-full flex items-center justify-center shrink-0 ${highlighted ? 'bg-primary text-on-primary' : 'bg-surface-container-high text-on-surface-variant'}`}>
                  {isDone ? <Check size={15} /> : <s.icon size={15} />}
                </div>
                <span className={`text-[13px] ${highlighted ? 'text-on-surface font-medium' : 'text-on-surface-variant'}`}>{s.label}</span>
              </div>
            )
          })}
        </nav>
        <div className="mt-auto">
          <button type="button" onClick={skip} className="text-[13px] text-on-surface-variant hover:text-on-surface transition-colors">
            Skip setup
          </button>
        </div>
      </aside>

      <main className="flex-1 min-h-screen flex flex-col">
        <div className="flex items-center justify-end p-6">
          <button type="button" onClick={skip} className="text-[13px] text-on-surface-variant hover:text-on-surface transition-colors">
            Skip for now
          </button>
        </div>
        <div className="flex-1 flex items-center justify-center px-6 pb-16">
          {step === 0 && <WelcomeStep onStart={() => setStep(1)} onSkip={skip} />}
          {step === 1 && <StorageStep onDone={() => { reload(); setStep(2) }} onBack={() => setStep(0)} />}
          {step === 2 && <AgentStep onDone={() => { reload(); setStep(3) }} onBack={() => setStep(1)} />}
          {step === 3 && (
            <JobStep
              onDone={() => { setJobCreated(true); reload(); setStep(4) }}
              onSkip={() => { reload(); setStep(4) }}
            />
          )}
          {step === 4 && <DoneStep onFinish={finish} done={allComplete} missing={missing} />}
        </div>
      </main>
    </div>
  )
}

export default OnboardingPage
