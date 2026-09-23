import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { jobApi, agentApi, storageApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import type { Agent, StorageTarget, BackupSourceType, BackupMode } from '@/types'
import { ChevronRight, Loader2 } from 'lucide-react'

const EditJobPage = () => {
  const { id } = useParams<{ id: string }>()
  const { currentOrg } = useAuthStore()
  const { addToast } = useUIStore()
  const navigate = useNavigate()

  const [agents, setAgents] = useState<Agent[]>([])
  const [storageTargets, setStorageTargets] = useState<StorageTarget[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [isSubmitting, setIsSubmitting] = useState(false)

  const [form, setForm] = useState({
    name: '',
    source_type: 'FILESYSTEM' as BackupSourceType,
    source_path: '',
    source_database: '',
    include_patterns: '',
    exclude_patterns: '',
    agent_id: '',
    storage_target_id: '',
    cron_expr: '',
    timezone: 'UTC',
    mode: 'COMPRESSED' as BackupMode,
    encrypted: true,
    retention_days: 30,
  })

  useEffect(() => {
    if (!currentOrg || !id) return
    Promise.all([
      jobApi.get(currentOrg.id, id),
      agentApi.list(currentOrg.id),
      storageApi.list(currentOrg.id),
    ])
      .then(([job, a, s]) => {
        setAgents(a)
        setStorageTargets(s)
        setForm({
          name: job.name,
          source_type: job.source_type,
          source_path: job.source_path ?? '',
          source_database: job.source_database ?? '',
          include_patterns: job.include_patterns ?? '',
          exclude_patterns: job.exclude_patterns ?? '',
          agent_id: job.agent_id,
          storage_target_id: job.storage_target_id,
          cron_expr: job.schedule?.cron_expr ?? '',
          timezone: job.schedule?.timezone ?? 'UTC',
          mode: job.mode,
          encrypted: job.encrypted,
          retention_days: job.retention_days,
        })
      })
      .catch(() => addToast('error', 'Failed to load job'))
      .finally(() => setIsLoading(false))
  }, [currentOrg, id])

  const update = (key: string, value: unknown) => setForm((f) => ({ ...f, [key]: value }))

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!currentOrg || !id) return
    setIsSubmitting(true)
    try {
      await jobApi.update(currentOrg.id, id, form as Record<string, unknown>)
      addToast('success', `${form.name} updated`)
      navigate(`/jobs/${id}`)
    } catch (err: unknown) {
      addToast('error', err instanceof Error ? err.message : 'Failed to update job')
    } finally {
      setIsSubmitting(false)
    }
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 size={32} className="text-[#c3c6d7] animate-spin" />
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-6 max-w-2xl mx-auto">
      <div>
        <div className="flex items-center gap-2 text-[12px] text-[#434655] mb-1">
          <button type="button" onClick={() => navigate('/jobs')} className="hover:text-[#004ac6]">Backup Jobs</button>
          <ChevronRight size={14} />
          <button type="button" onClick={() => navigate(`/jobs/${id}`)} className="hover:text-[#004ac6]">{form.name}</button>
          <ChevronRight size={14} />
          <span className="text-[#004ac6] font-medium">Edit</span>
        </div>
        <h1 className="text-[20px] font-semibold text-[#141b2b]">Edit Backup Job</h1>
      </div>

      <form onSubmit={handleSubmit} className="flex flex-col gap-5 bg-[#ffffff] rounded-xl border border-[#e9edff] shadow-sm p-6">
        <Field label="Job Name *">
          <input
            type="text"
            value={form.name}
            onChange={(e) => update('name', e.target.value)}
            required
            className="w-full h-9 px-3 rounded-lg border border-[#e9edff] bg-[#f9f9ff] text-[14px] text-[#141b2b] focus:outline-none focus:ring-2 focus:ring-[#2563eb]/20 focus:border-[#2563eb] transition-all"
          />
        </Field>

        {(form.source_type === 'FILESYSTEM' || form.source_type === 'DBF') ? (
          <Field label="Source Path">
            <input
              type="text"
              value={form.source_path}
              onChange={(e) => update('source_path', e.target.value)}
              className="w-full h-9 px-3 rounded-lg border border-[#e9edff] bg-[#f9f9ff] font-mono text-[13px] text-[#141b2b] focus:outline-none focus:ring-2 focus:ring-[#2563eb]/20 focus:border-[#2563eb] transition-all"
            />
          </Field>
        ) : (
          <Field label="Database Name">
            <input
              type="text"
              value={form.source_database}
              onChange={(e) => update('source_database', e.target.value)}
              className="w-full h-9 px-3 rounded-lg border border-[#e9edff] bg-[#f9f9ff] font-mono text-[13px] text-[#141b2b] focus:outline-none focus:ring-2 focus:ring-[#2563eb]/20 focus:border-[#2563eb] transition-all"
            />
          </Field>
        )}

        {form.source_type === 'FILESYSTEM' && (
          <div className="grid grid-cols-2 gap-4">
            <Field label="Include Patterns">
              <textarea
                value={form.include_patterns}
                onChange={(e) => update('include_patterns', e.target.value)}
                rows={3}
                className="w-full px-3 py-2 rounded-lg border border-[#e9edff] bg-[#f9f9ff] font-mono text-[12px] text-[#141b2b] focus:outline-none focus:ring-2 focus:ring-[#2563eb]/20 focus:border-[#2563eb] transition-all resize-none"
              />
            </Field>
            <Field label="Exclude Patterns">
              <textarea
                value={form.exclude_patterns}
                onChange={(e) => update('exclude_patterns', e.target.value)}
                rows={3}
                className="w-full px-3 py-2 rounded-lg border border-[#e9edff] bg-[#f9f9ff] font-mono text-[12px] text-[#141b2b] focus:outline-none focus:ring-2 focus:ring-[#2563eb]/20 focus:border-[#2563eb] transition-all resize-none"
              />
            </Field>
          </div>
        )}

        <Field label="Agent">
          <select
            value={form.agent_id}
            onChange={(e) => update('agent_id', e.target.value)}
            className="w-full h-9 px-3 rounded-lg border border-[#e9edff] bg-[#f9f9ff] text-[14px] text-[#141b2b] focus:outline-none focus:ring-2 focus:ring-[#2563eb]/20 focus:border-[#2563eb] transition-all"
          >
            {agents.map((a) => <option key={a.id} value={a.id}>{a.name} ({a.status})</option>)}
          </select>
        </Field>

        <Field label="Storage Target">
          <select
            value={form.storage_target_id}
            onChange={(e) => update('storage_target_id', e.target.value)}
            className="w-full h-9 px-3 rounded-lg border border-[#e9edff] bg-[#f9f9ff] text-[14px] text-[#141b2b] focus:outline-none focus:ring-2 focus:ring-[#2563eb]/20 focus:border-[#2563eb] transition-all"
          >
            {storageTargets.map((s) => <option key={s.id} value={s.id}>{s.name} ({s.type})</option>)}
          </select>
        </Field>

        <div className="grid grid-cols-2 gap-4">
          <Field label="Cron Expression">
            <input
              type="text"
              value={form.cron_expr}
              onChange={(e) => update('cron_expr', e.target.value)}
              className="w-full h-9 px-3 rounded-lg border border-[#e9edff] bg-[#f9f9ff] font-mono text-[13px] text-[#141b2b] focus:outline-none focus:ring-2 focus:ring-[#2563eb]/20 focus:border-[#2563eb] transition-all"
            />
          </Field>
          <Field label="Timezone">
            <select
              value={form.timezone}
              onChange={(e) => update('timezone', e.target.value)}
              className="w-full h-9 px-3 rounded-lg border border-[#e9edff] bg-[#f9f9ff] text-[14px] text-[#141b2b] focus:outline-none focus:ring-2 focus:ring-[#2563eb]/20 focus:border-[#2563eb] transition-all"
            >
              {['UTC', 'America/New_York', 'America/Chicago', 'America/Los_Angeles', 'Europe/London', 'Europe/Berlin', 'Asia/Tokyo'].map((tz) => (
                <option key={tz} value={tz}>{tz}</option>
              ))}
            </select>
          </Field>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <Field label="Retention (days)">
            <input
              type="number"
              value={form.retention_days}
              onChange={(e) => update('retention_days', parseInt(e.target.value))}
              min={1}
              max={365}
              className="w-full h-9 px-3 rounded-lg border border-[#e9edff] bg-[#f9f9ff] text-[14px] text-[#141b2b] focus:outline-none focus:ring-2 focus:ring-[#2563eb]/20 focus:border-[#2563eb] transition-all"
            />
          </Field>
          <Field label="Mode">
            <select
              value={form.mode}
              onChange={(e) => update('mode', e.target.value)}
              className="w-full h-9 px-3 rounded-lg border border-[#e9edff] bg-[#f9f9ff] text-[14px] text-[#141b2b] focus:outline-none focus:ring-2 focus:ring-[#2563eb]/20 focus:border-[#2563eb] transition-all"
            >
              <option value="NORMAL">Normal</option>
              <option value="COMPRESSED">Compressed</option>
            </select>
          </Field>
        </div>

        <div className="flex items-center justify-between p-4 rounded-xl border border-[#e9edff] bg-[#f9f9ff]">
          <div>
            <p className="text-[14px] font-medium text-[#141b2b]">Encryption</p>
            <p className="text-[12px] text-[#737686]">AES-256-GCM — data encrypted before leaving your machine</p>
          </div>
          <label className="relative inline-flex items-center cursor-pointer">
            <input type="checkbox" checked={form.encrypted} onChange={(e) => update('encrypted', e.target.checked)} className="sr-only peer" />
            <div className="w-11 h-6 bg-[#dce2f7] peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-[#2563eb]" />
          </label>
        </div>

        <div className="flex items-center justify-between pt-2">
          <button
            type="button"
            onClick={() => navigate(`/jobs/${id}`)}
            className="px-4 h-9 rounded-lg bg-[#ffffff] border border-[#e9edff] text-[#141b2b] text-[13px] font-medium hover:bg-[#f1f3ff] transition-colors"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={isSubmitting}
            className="px-4 h-9 rounded-lg bg-[#2563eb] text-white text-[13px] font-medium hover:bg-[#1d4ed8] transition-colors disabled:opacity-60 flex items-center gap-2"
          >
            {isSubmitting && <Loader2 size={14} className="animate-spin" />}
            {isSubmitting ? 'Saving...' : 'Save Changes'}
          </button>
        </div>
      </form>
    </div>
  )
}

const Field = ({ label, children }: { label: string; children: React.ReactNode }) => (
  <div className="flex flex-col gap-1.5">
    <label className="text-[13px] font-medium text-[#434655]">{label}</label>
    {children}
  </div>
)

export default EditJobPage
