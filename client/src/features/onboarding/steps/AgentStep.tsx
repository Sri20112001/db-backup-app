import { useEffect, useState } from 'react'
import { agentApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import AgentEnrollmentPanel from '@/features/agents/components/AgentEnrollmentPanel'
import { ArrowRight, CheckCircle2, Loader2 } from 'lucide-react'
import StepShell from '../components/StepShell'

interface AgentStepProps {
  onDone: () => void
  onBack: () => void
}

const POLL_MS = 4000

const AgentStep = ({ onDone, onBack }: AgentStepProps) => {
  const { currentOrg } = useAuthStore()
  const [agentDetected, setAgentDetected] = useState(false)

  // The onboarding route renders outside AppLayout, so socket presence isn't
  // available here — poll the agent list instead to detect a fresh enrollment.
  useEffect(() => {
    if (!currentOrg) return
    let cancelled = false
    const check = async () => {
      try {
        const res = await agentApi.list(currentOrg.id, { limit: 1 })
        if (!cancelled && res.total > 0) setAgentDetected(true)
      } catch {
        // Keep polling; transient errors shouldn't break the wizard.
      }
    }
    void check()
    const timer = setInterval(check, POLL_MS)
    return () => {
      cancelled = true
      clearInterval(timer)
    }
  }, [currentOrg])

  return (
    <StepShell title="Register an agent" subtitle="Install the agent on a machine, then enroll it with a one-time token.">
      <AgentEnrollmentPanel />

      {agentDetected ? (
        <div className="flex items-center gap-2.5 px-3 py-2.5 rounded-lg bg-success-container border border-success/30 text-[13px] font-medium text-on-success-container">
          <CheckCircle2 size={16} className="shrink-0 text-success" />
          Agent connected — you're ready to create a backup job.
        </div>
      ) : (
        <div className="flex items-center gap-2.5 px-3 py-2.5 rounded-lg bg-surface-container-low border border-surface-variant text-[12.5px] text-on-surface-variant">
          <Loader2 size={14} className="animate-spin shrink-0 text-on-surface-variant" />
          Waiting for an agent to enroll — this page checks every few seconds.
        </div>
      )}

      <div className="flex items-center justify-between">
        <button type="button" onClick={onBack} className="text-[13px] text-on-surface-variant hover:text-on-surface transition-colors">
          ← Back
        </button>
        <div className="flex items-center gap-3">
          {!agentDetected && (
            <button type="button" onClick={onDone} className="text-[13px] text-on-surface-variant hover:text-on-surface transition-colors">
              Continue anyway
            </button>
          )}
          <button
            type="button"
            onClick={onDone}
            disabled={!agentDetected}
            className="flex items-center gap-2 px-5 h-10 rounded-lg bg-primary text-on-primary text-[13px] font-medium hover:bg-primary-container transition-colors disabled:opacity-50"
          >
            Continue
            <ArrowRight size={16} />
          </button>
        </div>
      </div>
    </StepShell>
  )
}

export default AgentStep
