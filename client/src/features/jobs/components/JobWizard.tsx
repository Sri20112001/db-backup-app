import { useCallback, useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { jobApi, agentApi, storageApi, connectionsApi, policyApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import type { Agent, StorageTarget, BackupPolicy, DatabaseConnection } from '@/types'
import { ChevronRight, Check, Loader2 } from 'lucide-react'
import Action3DButton from '@/components/ui/Action3DButton'
import FolderBrowser from '@/components/FolderBrowser'

import {
  INITIAL_JOB_WIZARD_FORM,
  SOURCE_TO_CONNECTION,
  WIZARD_STEPS,
  type JobWizardFormData,
} from './wizard/types'
import WizardSourceStep from './wizard/WizardSourceStep'
import WizardAgentStep from './wizard/WizardAgentStep'
import WizardScheduleStep from './wizard/WizardScheduleStep'
import WizardProcessingStep from './wizard/WizardProcessingStep'
import WizardReviewStep from './wizard/WizardReviewStep'

interface JobWizardProps {
  onClose?: () => void
  onSaved?: () => void
  onCancel?: () => void
}

const JobWizard = ({ onClose, onSaved, onCancel }: JobWizardProps = {}) => {
  const { currentOrg } = useAuthStore()
  const { addToast } = useUIStore()
  const navigate = useNavigate()
  const [step, setStep] = useState(0)
  const [agents, setAgents] = useState<Agent[]>([])
  const [storageTargets, setStorageTargets] = useState<StorageTarget[]>([])
  const [connections, setConnections] = useState<DatabaseConnection[]>([])
  const [policies, setPolicies] = useState<BackupPolicy[]>([])
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [showBrowser, setShowBrowser] = useState(false)

  const [form, setForm] = useState<JobWizardFormData>(INITIAL_JOB_WIZARD_FORM)

  // Wrapped so the fetch effect can depend on it; loads once per org.
  const loadResources = useCallback(() => {
    if (!currentOrg) return
    Promise.all([
      agentApi.list(currentOrg.id, { limit: 200 }),
      storageApi.list(currentOrg.id),
      connectionsApi.list(currentOrg.id, { limit: 200 }),
      policyApi.list(currentOrg.id),
    ])
      .then(([a, s, c, p]) => {
        setAgents(a.data)
        setStorageTargets(s)
        setConnections(c.data)
        setPolicies(p)
      })
      .catch(() => addToast('error', 'Failed to load resources'))
  }, [currentOrg, addToast])

  useEffect(() => {
    loadResources()
  }, [loadResources])

  const update = (key: string, value: unknown) => setForm((f) => ({ ...f, [key]: value }))

  const connectionType = SOURCE_TO_CONNECTION[form.source_type]
  const visibleConnections = connectionType ? connections.filter((c) => c.type === connectionType) : []

  // Picking a connection pins the job to the connection's agent so the host
  // with database access runs the backup.
  const handleConnectionChange = (id: string) => {
    const conn = connections.find((c) => c.id === id)
    setForm((f) => ({
      ...f,
      connection_id: id,
      source_database: '',
      agent_id: conn ? conn.agent_id : f.agent_id,
    }))
  }

  const isDatabaseSource = connectionType !== undefined
  const selectedPolicy = policies.find((p) => p.id === form.policy_id)
  const policyGoverns = (field: 'schedule' | 'processing') => {
    if (!selectedPolicy) return false
    if (field === 'schedule') return selectedPolicy.cron_expr !== ''
    return true
  }

  const canAdvance = () => {
    if (step === 0) {
      if (!form.name) return false
      if (isDatabaseSource) {
        if (visibleConnections.length > 0 && !form.connection_id) return false
        return !!form.source_database
      }
      return !!form.source_path
    }
    if (step === 1) return !!form.agent_id
    if (step === 2) return !!form.cron_expr
    if (step === 3) return !!form.storage_target_id
    return true
  }

  const handleSubmit = async () => {
    if (!currentOrg) return
    setIsSubmitting(true)
    try {
      const payload: Record<string, unknown> = { ...form }
      if (payload.source_type !== 'MONGODB') delete payload.export_format
      if (!payload.connection_id) delete payload.connection_id
      if (!payload.policy_id) delete payload.policy_id
      if (payload.source_type === 'FILESYSTEM' || payload.source_type === 'DBF') delete payload.connection_id
      await jobApi.create(currentOrg.id, payload)
      addToast('success', `${form.name} created successfully`)
      if (onSaved) onSaved()
      if (onClose) onClose()
      else navigate('/jobs')
    } catch (err: unknown) {
      addToast('error', err instanceof Error ? err.message : 'Failed to create job')
    } finally {
      setIsSubmitting(false)
    }
  }

  const selectedAgent = agents.find((a) => a.id === form.agent_id)
  const selectedStorage = storageTargets.find((s) => s.id === form.storage_target_id)
  const selectedConnection = connections.find((c) => c.id === form.connection_id)

  const closeWizard = () => {
    if (onClose) onClose()
    else navigate('/jobs')
  }

  return (
    <>
      <div className={onClose ? 'flex flex-col gap-5' : 'h-full min-h-0 overflow-y-auto flex flex-col gap-4 pr-0.5 pb-1'}>
        {!onClose && (
          <div>
            <div className="flex items-center gap-2 text-[12px] text-on-surface-variant mb-1">
              <button type="button" onClick={() => navigate('/jobs')} className="hover:text-primary">
                Backup Jobs
              </button>
              <ChevronRight size={14} />
              <span className="text-primary font-medium">New Job</span>
            </div>
            <h1 className="text-[20px] font-semibold text-on-surface">Create Backup Job</h1>
          </div>
        )}

        {/* Step indicator */}
        <div className="flex items-center gap-1">
          {WIZARD_STEPS.map((s, i) => (
            <div key={s} className="flex items-center flex-1">
              <div className="flex flex-col items-center flex-1">
                <div
                  className={`w-8 h-8 rounded-full flex items-center justify-center text-[13px] font-semibold transition-colors ${
                    i < step
                      ? 'bg-primary text-on-primary'
                      : i === step
                      ? 'bg-primary text-on-primary'
                      : 'bg-surface-container-high text-on-surface-variant'
                  }`}
                >
                  {i < step ? <Check size={16} /> : i + 1}
                </div>
                <span className="text-[11px] text-on-surface-variant mt-1 text-center">{s}</span>
              </div>
              {i < WIZARD_STEPS.length - 1 && (
                <div className={`h-0.5 flex-1 mx-1 mb-4 ${i < step ? 'bg-primary' : 'bg-surface-container-high'}`} />
              )}
            </div>
          ))}
        </div>

        {/* Step content */}
        <div className="bg-surface-container-lowest rounded-xl border border-surface-variant shadow-sm p-6">
          {step === 0 && (
            <WizardSourceStep
              form={form}
              update={update}
              visibleConnections={visibleConnections}
              agents={agents}
              handleConnectionChange={handleConnectionChange}
              loadResources={loadResources}
              onOpenBrowser={() => setShowBrowser(true)}
            />
          )}

          {step === 1 && (
            <WizardAgentStep
              form={form}
              update={update}
              agents={agents}
              onNavigateAgents={() => navigate('/agents')}
            />
          )}

          {step === 2 && (
            <WizardScheduleStep
              form={form}
              update={update}
              selectedPolicy={selectedPolicy}
              policyGovernsSchedule={policyGoverns('schedule')}
            />
          )}

          {step === 3 && (
            <WizardProcessingStep
              form={form}
              update={update}
              policies={policies}
              selectedPolicy={selectedPolicy}
              policyGovernsProcessing={policyGoverns('processing')}
              storageTargets={storageTargets}
              onNavigateStorage={() => navigate('/storage')}
            />
          )}

          {step === 4 && (
            <WizardReviewStep
              form={form}
              selectedConnection={selectedConnection}
              selectedAgent={selectedAgent}
              selectedPolicy={selectedPolicy}
              selectedStorage={selectedStorage}
              policyGoverns={policyGoverns}
            />
          )}
        </div>

        {/* Navigation */}
        <div className="flex items-center justify-between">
          <button
            type="button"
            onClick={() => (step === 0 ? (onCancel ? onCancel() : closeWizard()) : setStep((s) => s - 1))}
            className="px-4 h-9 rounded-lg bg-surface-container-lowest border border-surface-variant text-on-surface text-[13px] font-medium hover:bg-surface-container-low transition-colors"
          >
            {step === 0 ? 'Cancel' : '← Back'}
          </button>
          {step < WIZARD_STEPS.length - 1 ? (
            <Action3DButton type="button" onClick={() => setStep((s) => s + 1)} disabled={!canAdvance()}>
              Next →
            </Action3DButton>
          ) : (
            <Action3DButton type="button" onClick={handleSubmit} disabled={isSubmitting}>
              {isSubmitting && <Loader2 size={14} className="animate-spin" />}
              {isSubmitting ? 'Creating...' : 'Create Backup Job'}
            </Action3DButton>
          )}
        </div>
      </div>

      {showBrowser && (
        <FolderBrowser
          initialPath={form.source_path}
          onSelect={(p) => {
            update('source_path', p)
            setShowBrowser(false)
          }}
          onClose={() => setShowBrowser(false)}
        />
      )}
    </>
  )
}

export default JobWizard
