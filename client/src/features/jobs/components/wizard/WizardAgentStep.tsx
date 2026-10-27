import type { Agent } from '@/types'
import { Server } from 'lucide-react'
import type { JobWizardFormData } from './types'

interface WizardAgentStepProps {
  form: JobWizardFormData
  update: (key: string, value: unknown) => void
  agents: Agent[]
  onNavigateAgents: () => void
}

export default function WizardAgentStep({
  form,
  update,
  agents,
  onNavigateAgents,
}: WizardAgentStepProps) {
  return (
    <div className="flex flex-col gap-4">
      <h2 className="text-[16px] font-semibold text-on-surface">Select Agent</h2>
      {agents.length === 0 ? (
        <div className="text-center py-8 text-on-surface-variant text-[13px]">
          No agents registered.{' '}
          <button type="button" onClick={onNavigateAgents} className="text-primary hover:underline">
            Register an agent first.
          </button>
        </div>
      ) : (
        <div className="flex flex-col gap-2 max-h-[320px] overflow-y-auto pr-1">
          {agents.map((agent) => (
            <button
              key={agent.id}
              type="button"
              onClick={() => update('agent_id', agent.id)}
              className={`flex items-center gap-4 p-4 rounded-xl border-2 text-left transition-all ${
                form.agent_id === agent.id ? 'border-primary bg-primary-container/20' : 'border-surface-variant hover:border-outline'
              }`}
            >
              <div
                className={`w-10 h-10 rounded-lg flex items-center justify-center ${
                  agent.status === 'ONLINE' ? 'bg-primary-container/40 text-on-primary-container' : 'bg-surface-variant text-on-surface-variant'
                }`}
              >
                <Server size={20} />
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-[14px] font-semibold text-on-surface">{agent.name}</p>
                <p className="font-mono text-[11px] text-on-surface-variant">
                  v{agent.version} · {agent.status === 'ONLINE' ? 'Online' : 'Offline'}
                </p>
              </div>
              <span className={`w-2.5 h-2.5 rounded-full ${agent.status === 'ONLINE' ? 'bg-primary' : 'bg-[#737686]'}`} />
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
