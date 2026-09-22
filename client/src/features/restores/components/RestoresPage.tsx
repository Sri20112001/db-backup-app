import { useEffect, useState } from 'react'
import { restoreApi, runApi, agentApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import type { RestoreJob, BackupRun, Agent } from '@/types'
import StatusBadge from '@/components/StatusBadge'
import EmptyState from '@/components/EmptyState'
import { formatRelative, formatDate } from '@/utils/format'
import { RotateCcw, X, Check, Loader2 } from 'lucide-react'

const RestoresPage = () => {
  const { currentOrg } = useAuthStore()
  const { addToast } = useUIStore()
  const [restores, setRestores] = useState<RestoreJob[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [showWizard, setShowWizard] = useState(false)
  const [wizardStep, setWizardStep] = useState(0)
  const [recentRuns, setRecentRuns] = useState<BackupRun[]>([])
  const [agents, setAgents] = useState<Agent[]>([])
  const [form, setForm] = useState({ backup_run_id: '', agent_id: '', destination_path: '', target_database: '' })
  const [isSubmitting, setIsSubmitting] = useState(false)

  const loadRestores = () => {
    if (!currentOrg) return
    setIsLoading(true)
    restoreApi.list(currentOrg.id).then(setRestores).catch(() => addToast('error', 'Failed to load restores')).finally(() => setIsLoading(false))
  }

  useEffect(() => { loadRestores() }, [currentOrg])

  const openWizard = async () => {
    if (!currentOrg) return
    try {
      const [runs, ags] = await Promise.all([
        runApi.list(currentOrg.id, { status: 'COMPLETED', limit: 50 }),
        agentApi.list(currentOrg.id),
      ])
      setRecentRuns(runs.data)
      setAgents(ags)
      setShowWizard(true)
      setWizardStep(0)
    } catch { addToast('error', 'Failed to load data') }
  }

  const handleCreate = async () => {
    if (!currentOrg) return
    setIsSubmitting(true)
    try {
      await restoreApi.create(currentOrg.id, form as Record<string, unknown>)
      addToast('success', 'Restore job started')
      setShowWizard(false)
      setForm({ backup_run_id: '', agent_id: '', destination_path: '', target_database: '' })
      loadRestores()
    } catch { addToast('error', 'Failed to start restore') }
    finally { setIsSubmitting(false) }
  }

  const selectedRun = recentRuns.find((r) => r.id === form.backup_run_id)
  const inputCls = "w-full h-9 px-3 rounded-lg border border-[#e9edff] bg-[#f9f9ff] text-[14px] text-[#141b2b] focus:outline-none focus:ring-2 focus:ring-[#2563eb]/20 focus:border-[#2563eb] transition-all"

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-[20px] font-semibold text-[#141b2b] tracking-tight">Restores</h1>
          <p className="text-[12px] text-[#434655] mt-0.5">Restore data from backup recovery points</p>
        </div>
        <button type="button" onClick={openWizard} className="flex items-center gap-2 px-4 h-9 rounded-lg bg-[#2563eb] text-white text-[13px] font-medium hover:bg-[#1d4ed8] transition-colors shadow-sm">
          <RotateCcw size={16} />
          New Restore
        </button>
      </div>

      {isLoading ? (
        <div className="flex flex-col gap-2">{[...Array(4)].map((_, i) => <div key={i} className="animate-pulse h-16 rounded-xl bg-[#ffffff] border border-[#e9edff]" />)}</div>
      ) : restores.length === 0 ? (
        <EmptyState icon={RotateCcw} title="No restore jobs" description="Restore jobs will appear here once you initiate a restore from a recovery point."
          action={<button type="button" onClick={openWizard} className="px-4 h-9 rounded-lg bg-[#2563eb] text-white text-[13px] font-medium hover:bg-[#1d4ed8] transition-colors">Start First Restore</button>}
        />
      ) : (
        <div className="flex flex-col gap-2">
          {restores.map((r) => (
            <div key={r.id} className="flex items-center gap-4 p-4 rounded-xl bg-[#ffffff] border border-[#e9edff] shadow-sm hover:shadow-md transition-all">
              <StatusBadge status={r.status} />
              <div className="flex-1 min-w-0">
                <p className="text-[14px] font-medium text-[#141b2b] truncate">
                  {r.backup_run?.backup_job?.name ?? 'Unknown Job'}
                </p>
                <p className="font-mono text-[11px] text-[#737686] truncate">
                  → {r.destination_path || r.target_database || 'Original location'}
                </p>
              </div>
              <div className="text-right shrink-0">
                <p className="text-[12px] text-[#737686]">{formatRelative(r.created_at)}</p>
                {r.completed_at && <p className="text-[11px] text-[#737686]">Done: {formatDate(r.completed_at)}</p>}
              </div>
              {r.error_message && (
                <div className="px-2 py-1 rounded bg-[#ffdad6] text-[#93000a] text-[11px] font-mono max-w-[200px] truncate" title={r.error_message}>
                  {r.error_message}
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Restore wizard drawer */}
      {showWizard && (
        <div className="fixed inset-0 z-[100] flex justify-end">
          <div className="absolute inset-0 bg-[#141b2b]/30 backdrop-blur-[2px]" onClick={() => setShowWizard(false)} />
          <div className="relative w-full max-w-[520px] bg-[#ffffff] h-full overflow-y-auto shadow-2xl border-l border-[#e9edff] flex flex-col">
            <div className="flex items-center justify-between p-5 border-b border-[#e9edff] sticky top-0 bg-[#ffffff] z-10">
              <h2 className="text-[16px] font-semibold text-[#141b2b]">New Restore</h2>
              <button type="button" onClick={() => setShowWizard(false)} className="p-1.5 rounded-lg text-[#434655] hover:bg-[#e9edff]">
                <X size={18} />
              </button>
            </div>

            {/* Step indicator */}
            <div className="flex items-center gap-1 px-5 py-4 border-b border-[#e9edff]">
              {['Recovery Point', 'Destination', 'Confirm'].map((s, i) => (
                <div key={s} className="flex items-center flex-1">
                  <div className="flex flex-col items-center flex-1">
                    <div className={`w-7 h-7 rounded-full flex items-center justify-center text-[12px] font-semibold ${i < wizardStep ? 'bg-[#006591] text-white' : i === wizardStep ? 'bg-[#2563eb] text-white' : 'bg-[#e9edff] text-[#737686]'}`}>
                      {i < wizardStep ? <Check size={14} /> : i + 1}
                    </div>
                    <span className="text-[10px] text-[#737686] mt-1 text-center">{s}</span>
                  </div>
                  {i < 2 && <div className={`h-0.5 flex-1 mx-1 mb-4 ${i < wizardStep ? 'bg-[#006591]' : 'bg-[#e9edff]'}`} />}
                </div>
              ))}
            </div>

            <div className="flex flex-col gap-4 p-5 flex-1">
              {wizardStep === 0 && (
                <>
                  <h3 className="text-[14px] font-semibold text-[#141b2b]">Select Recovery Point</h3>
                  <div className="flex flex-col gap-2 max-h-96 overflow-y-auto">
                    {recentRuns.map((run) => (
                      <button key={run.id} type="button" onClick={() => setForm((f) => ({ ...f, backup_run_id: run.id }))}
                        className={`flex items-center gap-3 p-3.5 rounded-xl border-2 text-left transition-all ${form.backup_run_id === run.id ? 'border-[#2563eb] bg-[#dbe1ff]/20' : 'border-[#e9edff] hover:border-[#c3c6d7]'}`}>
                        <div className="flex-1 min-w-0">
                          <p className="text-[13px] font-semibold text-[#141b2b]">{run.backup_job?.name ?? '—'}</p>
                          <p className="text-[11px] text-[#737686]">{formatRelative(run.completed_at)} · {run.backup_job?.source_type}</p>
                        </div>
                        <StatusBadge status={run.status} size="sm" />
                      </button>
                    ))}
                  </div>
                </>
              )}

              {wizardStep === 1 && (
                <>
                  <h3 className="text-[14px] font-semibold text-[#141b2b]">Destination</h3>
                  <div>
                    <label className="block text-[13px] font-medium text-[#434655] mb-1.5">Target Agent</label>
                    <select value={form.agent_id} onChange={(e) => setForm((f) => ({ ...f, agent_id: e.target.value }))} className={inputCls}>
                      <option value="">Select agent...</option>
                      {agents.map((a) => <option key={a.id} value={a.id}>{a.name} ({a.status})</option>)}
                    </select>
                  </div>
                  <div>
                    <label className="block text-[13px] font-medium text-[#434655] mb-1.5">Destination Path (leave empty for original)</label>
                    <input type="text" value={form.destination_path} onChange={(e) => setForm((f) => ({ ...f, destination_path: e.target.value }))} placeholder="D:\Restored\" className={`${inputCls} font-mono text-[13px]`} />
                  </div>
                  {selectedRun?.backup_job?.source_type === 'SQL_SERVER' && (
                    <div>
                      <label className="block text-[13px] font-medium text-[#434655] mb-1.5">Target Database Name</label>
                      <input type="text" value={form.target_database} onChange={(e) => setForm((f) => ({ ...f, target_database: e.target.value }))} placeholder="ERP_Restored" className={`${inputCls} font-mono text-[13px]`} />
                    </div>
                  )}
                </>
              )}

              {wizardStep === 2 && (
                <>
                  <h3 className="text-[14px] font-semibold text-[#141b2b]">Confirm Restore</h3>
                  <div className="flex flex-col gap-2 text-[13px]">
                    {[
                      { label: 'Recovery Point', value: selectedRun?.backup_job?.name ?? '—' },
                      { label: 'Agent', value: agents.find((a) => a.id === form.agent_id)?.name ?? '—' },
                      { label: 'Destination', value: form.destination_path || 'Original location' },
                      ...(form.target_database ? [{ label: 'Target DB', value: form.target_database }] : []),
                    ].map((r) => (
                      <div key={r.label} className="flex items-start gap-3">
                        <span className="text-[#737686] w-32 shrink-0">{r.label}</span>
                        <span className="text-[#141b2b] font-medium">{r.value}</span>
                      </div>
                    ))}
                  </div>
                  <div className="p-3 rounded-lg bg-[#ffdad6]/40 border border-[#fca5a5] text-[#93000a] text-[12px]">
                    ⚠ This will overwrite existing data at the destination path.
                  </div>
                </>
              )}
            </div>

            <div className="flex items-center justify-between p-5 border-t border-[#e9edff]">
              <button type="button" onClick={() => wizardStep === 0 ? setShowWizard(false) : setWizardStep((s) => s - 1)}
                className="px-4 h-9 rounded-lg bg-[#ffffff] border border-[#e9edff] text-[#141b2b] text-[13px] font-medium hover:bg-[#f1f3ff] transition-colors">
                {wizardStep === 0 ? 'Cancel' : '← Back'}
              </button>
              {wizardStep < 2 ? (
                <button type="button" onClick={() => setWizardStep((s) => s + 1)}
                  disabled={wizardStep === 0 ? !form.backup_run_id : !form.agent_id}
                  className="px-4 h-9 rounded-lg bg-[#2563eb] text-white text-[13px] font-medium hover:bg-[#1d4ed8] transition-colors disabled:opacity-50">
                  Next →
                </button>
              ) : (
                <button type="button" onClick={handleCreate} disabled={isSubmitting}
                  className="px-4 h-9 rounded-lg bg-[#ba1a1a] text-white text-[13px] font-medium hover:bg-[#93000a] transition-colors disabled:opacity-60 flex items-center gap-2">
                  {isSubmitting && <Loader2 size={14} className="animate-spin" />}
                  Start Restore
                </button>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

export default RestoresPage
