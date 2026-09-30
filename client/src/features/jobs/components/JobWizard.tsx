import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { jobApi, agentApi, storageApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import type { Agent, StorageTarget, BackupSourceType, BackupMode } from '@/types'
import { ChevronRight, Check, Server, CloudUpload, FolderOpen, HardDrive, Loader2 } from 'lucide-react'
import SqlDatabasePicker from './SqlDatabasePicker'
import PgDatabasePicker from './PgDatabasePicker'
import MongoDatabasePicker from './MongoDatabasePicker'
import Action3DButton from '@/components/ui/Action3DButton'
import NeoToggle from '@/components/ui/NeoToggle'
import { FileSystemIcon, MssqlServerIcon, PostgresIcon, MongoDbIcon, DbfIcon } from '@/components/ui/SourceIcons'

const STEPS = ['Source', 'Agent', 'Schedule', 'Processing', 'Review']

const PRESETS = [
  { label: 'Every 1 min (test)', cron: '*/1 * * * *' },
  { label: 'Every hour', cron: '@hourly' },
  { label: 'Every 6 hours', cron: '0 */6 * * *' },
  { label: 'Daily at 11 PM', cron: '0 23 * * *' },
  { label: 'Weekly Sunday', cron: '0 0 * * 0' },
  { label: 'Monthly', cron: '0 0 1 * *' },
]

const storageTypeIcon = { S3: CloudUpload, SMB: FolderOpen, LOCAL: HardDrive }

const JobWizard = ({ onClose, onSaved }: { onClose?: () => void; onSaved?: () => void } = {}) => {
  const { currentOrg } = useAuthStore()
  const { addToast } = useUIStore()
  const navigate = useNavigate()
  const [step, setStep] = useState(0)
  const [agents, setAgents] = useState<Agent[]>([])
  const [storageTargets, setStorageTargets] = useState<StorageTarget[]>([])
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
    cron_expr: '0 23 * * *',
    timezone: 'UTC',
    mode: 'COMPRESSED' as BackupMode,
    encrypted: true,
    retention_days: 30,
    export_format: 'ARCHIVE',
  })

  useEffect(() => {
    if (!currentOrg) return
    Promise.all([agentApi.list(currentOrg.id), storageApi.list(currentOrg.id)])
      .then(([a, s]) => { setAgents(a); setStorageTargets(s) })
      .catch(() => addToast('error', 'Failed to load resources'))
  }, [currentOrg])

  const update = (key: string, value: unknown) => setForm((f) => ({ ...f, [key]: value }))

  const canAdvance = () => {
    if (step === 0) return !!form.name && !!(form.source_path || form.source_database)
    if (step === 1) return !!form.agent_id
    if (step === 2) return !!form.cron_expr
    if (step === 3) return !!form.storage_target_id
    return true
  }

  const handleSubmit = async () => {
    if (!currentOrg) return
    setIsSubmitting(true)
    try {
      // export_format is MongoDB-only server-side: strip it for every other
      // source type (the form keeps an ARCHIVE default for the Mongo picker).
      const payload: Record<string, unknown> = { ...form }
      if (payload.source_type !== 'MONGODB') delete payload.export_format
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

  const closeWizard = () => {
    if (onClose) onClose()
    else navigate('/jobs')
  }

  return (
    <div className={onClose ? 'flex flex-col gap-5' : 'h-full min-h-0 overflow-y-auto flex flex-col gap-5 max-w-3xl mx-auto pr-0.5 pb-1'}>
      {!onClose && (
      <div>
        <div className="flex items-center gap-2 text-[12px] text-on-surface-variant mb-1">
          <button type="button" onClick={() => navigate('/jobs')} className="hover:text-primary">Backup Jobs</button>
          <ChevronRight size={14} />
          <span className="text-primary font-medium">New Job</span>
        </div>
        <h1 className="text-[20px] font-semibold text-on-surface">Create Backup Job</h1>
      </div>
      )}

      {/* Step indicator */}
      <div className="flex items-center gap-1">
        {STEPS.map((s, i) => (
          <div key={s} className="flex items-center flex-1">
            <div className="flex flex-col items-center flex-1">
              <div className={`w-8 h-8 rounded-full flex items-center justify-center text-[13px] font-semibold transition-colors ${
                i < step ? 'bg-primary text-on-primary' : i === step ? 'bg-primary text-on-primary' : 'bg-surface-container-high text-outline'
              }`}>
                {i < step ? <Check size={16} /> : i + 1}
              </div>
              <span className="text-[11px] text-outline mt-1 text-center">{s}</span>
            </div>
            {i < STEPS.length - 1 && (
              <div className={`h-0.5 flex-1 mx-1 mb-4 ${i < step ? 'bg-primary' : 'bg-surface-container-high'}`} />
            )}
          </div>
        ))}
      </div>

      {/* Step content */}
      <div className="bg-surface-container-lowest rounded-xl border border-surface-variant shadow-sm p-6">
        {/* Step 0: Source */}
        {step === 0 && (
          <div className="flex flex-col gap-5">
            <h2 className="text-[16px] font-semibold text-on-surface">Source Configuration</h2>
            <div>
              <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Job Name *</label>
              <input
                type="text"
                value={form.name}
                onChange={(e) => update('name', e.target.value)}
                placeholder="e.g. Daily ERP Backup"
                className="w-full h-9 px-3 rounded-lg border border-surface-variant bg-surface-container-low text-[14px] text-on-surface focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all"
              />
            </div>
            <div>
              <label className="block text-[13px] font-medium text-on-surface-variant mb-2">Source Type *</label>
              <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3">
                {([
                  { type: 'FILESYSTEM', Icon: FileSystemIcon, label: 'Filesystem' },
                  { type: 'MSSQL_SERVER', Icon: MssqlServerIcon, label: 'SQL Server' },
                  { type: 'POSTGRES', Icon: PostgresIcon, label: 'PostgreSQL' },
                  { type: 'MONGODB', Icon: MongoDbIcon, label: 'MongoDB' },
                  { type: 'DBF', Icon: DbfIcon, label: 'DBF Dataset' },
                ] as const).map((s) => (
                  <button
                    key={s.type}
                    type="button"
                    onClick={() => update('source_type', s.type)}
                    className={`flex flex-col items-center gap-2 p-4 rounded-xl border-2 transition-all ${
                      form.source_type === s.type
                        ? 'border-primary bg-primary-container/30'
                        : 'border-surface-variant hover:border-outline'
                    }`}
                  >
                    <s.Icon size={28} className={form.source_type === s.type ? 'text-primary' : 'text-on-surface-variant'} />
                    <span className="text-[13px] font-medium text-on-surface">{s.label}</span>
                  </button>
                ))}
              </div>
            </div>
            {form.source_type === 'FILESYSTEM' || form.source_type === 'DBF' ? (
              <div>
                <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">
                  {form.source_type === 'DBF' ? 'Dataset Directory *' : 'Source Path *'}
                </label>
                <input
                  type="text"
                  value={form.source_path}
                  onChange={(e) => update('source_path', e.target.value)}
                  placeholder={form.source_type === 'DBF' ? 'C:\\LegacyApp\\Data\\' : 'D:\\CompanyData\\'}
                  className="w-full h-9 px-3 rounded-lg border border-surface-variant bg-surface-container-low font-mono text-[13px] text-on-surface focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all"
                />
              </div>
            ) : form.source_type === 'POSTGRES' ? (
              <PgDatabasePicker
                value={form.source_database}
                onChange={(v) => update('source_database', v)}
              />
            ) : form.source_type === 'MONGODB' ? (
              <div className="flex flex-col gap-4">
                <MongoDatabasePicker
                  value={form.source_database}
                  onChange={(v) => update('source_database', v)}
                />
                <div>
                  <label className="block text-[13px] font-medium text-on-surface-variant mb-2">Export Format</label>
                  <div className="grid grid-cols-3 gap-2">
                    {[
                      { v: 'ARCHIVE', t: 'Archive', d: 'Binary, full fidelity' },
                      { v: 'JSON', t: 'JSON', d: 'Readable, per-doc lines' },
                      { v: 'CSV', t: 'CSV', d: 'Spreadsheet friendly' },
                    ].map((o) => (
                      <button
                        key={o.v}
                        type="button"
                        onClick={() => update('export_format', o.v)}
                        className={`flex flex-col items-start gap-0.5 p-3 rounded-xl border-2 text-left transition-all ${
                          form.export_format === o.v
                            ? 'border-primary bg-primary-container/20'
                            : 'border-surface-variant hover:border-outline'
                        }`}
                      >
                        <span className="text-[13px] font-semibold text-on-surface">{o.t}</span>
                        <span className="text-[11px] text-outline">{o.d}</span>
                      </button>
                    ))}
                  </div>
                  {form.export_format === 'CSV' && (
                    <p className="text-[11px] text-outline mt-1.5">
                      CSV restores values as strings — use Archive or JSON for exact restores.
                    </p>
                  )}
                </div>
              </div>
            ) : (
              <SqlDatabasePicker
                value={form.source_database}
                onChange={(v) => update('source_database', v)}
              />
            )}
            {(form.source_type === 'FILESYSTEM') && (
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Include Patterns</label>
                  <textarea
                    value={form.include_patterns}
                    onChange={(e) => update('include_patterns', e.target.value)}
                    placeholder="*.xlsx&#10;*.pdf&#10;*.docx"
                    rows={3}
                    className="w-full px-3 py-2 rounded-lg border border-surface-variant bg-surface-container-low font-mono text-[12px] text-on-surface focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all resize-none"
                  />
                </div>
                <div>
                  <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Exclude Patterns</label>
                  <textarea
                    value={form.exclude_patterns}
                    onChange={(e) => update('exclude_patterns', e.target.value)}
                    placeholder="*.tmp&#10;*.log&#10;node_modules/"
                    rows={3}
                    className="w-full px-3 py-2 rounded-lg border border-surface-variant bg-surface-container-low font-mono text-[12px] text-on-surface focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all resize-none"
                  />
                </div>
              </div>
            )}
          </div>
        )}

        {/* Step 1: Agent */}
        {step === 1 && (
          <div className="flex flex-col gap-4">
            <h2 className="text-[16px] font-semibold text-on-surface">Select Agent</h2>
            {agents.length === 0 ? (
              <div className="text-center py-8 text-outline text-[13px]">
                No agents registered. <button type="button" onClick={() => navigate('/agents')} className="text-primary hover:underline">Register an agent first.</button>
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
                    <div className={`w-10 h-10 rounded-lg flex items-center justify-center ${agent.status === 'ONLINE' ? 'bg-primary-container/40 text-on-primary-container' : 'bg-surface-variant text-outline'}`}>
                      <Server size={20} />
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-[14px] font-semibold text-on-surface">{agent.name}</p>
                      <p className="font-mono text-[11px] text-outline">v{agent.version} · {agent.status === 'ONLINE' ? 'Online' : 'Offline'}</p>
                    </div>
                    <span className={`w-2.5 h-2.5 rounded-full ${agent.status === 'ONLINE' ? 'bg-primary' : 'bg-[#737686]'}`} />
                  </button>
                ))}
              </div>
            )}
          </div>
        )}

        {/* Step 2: Schedule */}
        {step === 2 && (
          <div className="flex flex-col gap-5">
            <h2 className="text-[16px] font-semibold text-on-surface">Schedule</h2>
            <div className="flex flex-wrap gap-2">
              {PRESETS.map((p) => (
                <button
                  key={p.cron}
                  type="button"
                  onClick={() => update('cron_expr', p.cron)}
                  className={`px-3 h-8 rounded-lg text-[12px] font-medium transition-colors ${
                    form.cron_expr === p.cron ? 'bg-primary text-on-primary' : 'bg-surface-container-low text-on-surface-variant hover:bg-surface-container-high'
                  }`}
                >
                  {p.label}
                </button>
              ))}
            </div>
            <div>
              <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Cron Expression</label>
              <input
                type="text"
                value={form.cron_expr}
                onChange={(e) => update('cron_expr', e.target.value)}
                className="w-full h-9 px-3 rounded-lg border border-surface-variant bg-surface-container-low font-mono text-[13px] text-on-surface focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all"
              />
            </div>
            <div>
              <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Timezone</label>
              <select
                value={form.timezone}
                onChange={(e) => update('timezone', e.target.value)}
                className="w-full h-9 px-3 rounded-lg border border-surface-variant bg-surface-container-low text-[14px] text-on-surface focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all"
              >
                {['UTC', 'America/New_York', 'America/Chicago', 'America/Los_Angeles', 'Europe/London', 'Europe/Berlin', 'Asia/Tokyo'].map((tz) => (
                  <option key={tz} value={tz}>{tz}</option>
                ))}
              </select>
            </div>
          </div>
        )}

        {/* Step 3: Processing & Storage */}
        {step === 3 && (
          <div className="flex flex-col gap-5">
            <h2 className="text-[16px] font-semibold text-on-surface">Processing & Storage</h2>

            <div className="flex flex-col gap-3">
              {[
                { key: 'mode', label: 'Compression', desc: 'Zstandard — reduces storage by up to 60%', value: form.mode === 'COMPRESSED', toggle: (v: boolean) => update('mode', v ? 'COMPRESSED' : 'NORMAL') },
                { key: 'encrypted', label: 'Encryption', desc: 'AES-256-GCM — data encrypted before leaving your machine', value: form.encrypted, toggle: (v: boolean) => update('encrypted', v) },
              ].map((t) => (
                <div key={t.key} className="flex items-center justify-between p-4 rounded-xl border border-surface-variant bg-surface-container-low">
                  <div>
                    <p className="text-[14px] font-medium text-on-surface">{t.label}</p>
                    <p className="text-[12px] text-outline">{t.desc}</p>
                  </div>
                  <NeoToggle
                    id={`toggle-${t.key}`}
                    checked={t.value}
                    onChange={(checked) => t.toggle(checked)}
                  />
                </div>
              ))}
            </div>

            <div>
              <label className="block text-[13px] font-medium text-on-surface-variant mb-2">Storage Target *</label>
              {storageTargets.length === 0 ? (
                <div className="text-center py-6 text-outline text-[13px]">
                  No storage targets. <button type="button" onClick={() => navigate('/storage')} className="text-primary hover:underline">Add one first.</button>
                </div>
              ) : (
                <div className="flex flex-col gap-2 max-h-[280px] overflow-y-auto pr-1">
                  {storageTargets.map((s) => {
                    const StorIcon = storageTypeIcon[s.type as keyof typeof storageTypeIcon] ?? HardDrive
                    return (
                      <button
                        key={s.id}
                        type="button"
                        onClick={() => update('storage_target_id', s.id)}
                        className={`flex items-center gap-3 p-3.5 rounded-xl border-2 text-left transition-all ${
                          form.storage_target_id === s.id ? 'border-primary bg-primary-container/20' : 'border-surface-variant hover:border-outline'
                        }`}
                      >
                        <StorIcon size={20} className="text-primary" />
                        <div>
                          <p className="text-[13px] font-semibold text-on-surface">{s.name}</p>
                          <p className="font-mono text-[11px] text-outline">{s.type} · {s.bucket || s.path || '—'}</p>
                        </div>
                      </button>
                    )
                  })}
                </div>
              )}
            </div>

            <div>
              <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Retention (days)</label>
              <input
                type="number"
                value={form.retention_days}
                onChange={(e) => update('retention_days', parseInt(e.target.value))}
                min={1}
                max={365}
                className="w-32 h-9 px-3 rounded-lg border border-surface-variant bg-surface-container-low text-[14px] text-on-surface focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all"
              />
            </div>
          </div>
        )}

        {/* Step 4: Review */}
        {step === 4 && (
          <div className="flex flex-col gap-5">
            <h2 className="text-[16px] font-semibold text-on-surface">Review & Create</h2>
            <div className="grid grid-cols-2 gap-x-8 gap-y-3 text-[13px]">
              {[
                { label: 'Job Name', value: form.name },
                { label: 'Source Type', value: form.source_type.replace('_', ' ') },
                { label: 'Source', value: form.source_path || form.source_database || '—', mono: true },
                ...(form.source_type === 'MONGODB'
                  ? [{ label: 'Export Format', value: form.export_format || 'ARCHIVE' }]
                  : []),
                { label: 'Agent', value: selectedAgent?.name ?? '—' },
                { label: 'Schedule', value: form.cron_expr, mono: true },
                { label: 'Timezone', value: form.timezone },
                { label: 'Compression', value: form.mode === 'COMPRESSED' ? 'Zstandard' : 'Disabled' },
                { label: 'Encryption', value: form.encrypted ? 'AES-256-GCM' : 'Disabled' },
                { label: 'Storage', value: selectedStorage?.name ?? '—' },
                { label: 'Retention', value: `${form.retention_days} days` },
              ].map((r) => (
                <div key={r.label} className="flex flex-col gap-0.5">
                  <span className="text-[11px] font-semibold uppercase tracking-wider text-outline">{r.label}</span>
                  <span className={`text-on-surface ${r.mono ? 'font-mono text-[12px]' : 'font-medium'}`}>{r.value}</span>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* Navigation */}
      <div className="flex items-center justify-between">
        <button
          type="button"
          onClick={() => step === 0 ? closeWizard() : setStep((s) => s - 1)}
          className="px-4 h-9 rounded-lg bg-surface-container-lowest border border-surface-variant text-on-surface text-[13px] font-medium hover:bg-surface-container-low transition-colors"
        >
          {step === 0 ? 'Cancel' : '← Back'}
        </button>
        {step < STEPS.length - 1 ? (
          <Action3DButton
            type="button"
            onClick={() => setStep((s) => s + 1)}
            disabled={!canAdvance()}
          >
            Next →
          </Action3DButton>
        ) : (
          <Action3DButton
            type="button"
            onClick={handleSubmit}
            disabled={isSubmitting}
          >
            {isSubmitting && <Loader2 size={14} className="animate-spin" />}
            {isSubmitting ? 'Creating...' : 'Create Backup Job'}
          </Action3DButton>
        )}
      </div>
    </div>
  )
}

export default JobWizard
