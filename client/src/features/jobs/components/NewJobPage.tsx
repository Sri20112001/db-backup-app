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

const NewJobPage = () => {
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
      await jobApi.create(currentOrg.id, form as Record<string, unknown>)
      addToast('success', `${form.name} created successfully`)
      navigate('/jobs')
    } catch (err: unknown) {
      addToast('error', err instanceof Error ? err.message : 'Failed to create job')
    } finally {
      setIsSubmitting(false)
    }
  }

  const selectedAgent = agents.find((a) => a.id === form.agent_id)
  const selectedStorage = storageTargets.find((s) => s.id === form.storage_target_id)

  return (
    <div className="flex flex-col gap-6 max-w-3xl mx-auto">
      <div>
        <div className="flex items-center gap-2 text-[12px] text-[#434655] mb-1">
          <button type="button" onClick={() => navigate('/jobs')} className="hover:text-[#004ac6]">Backup Jobs</button>
          <ChevronRight size={14} />
          <span className="text-[#004ac6] font-medium">New Job</span>
        </div>
        <h1 className="text-[20px] font-semibold text-[#141b2b]">Create Backup Job</h1>
      </div>

      {/* Step indicator */}
      <div className="flex items-center gap-1">
        {STEPS.map((s, i) => (
          <div key={s} className="flex items-center flex-1">
            <div className="flex flex-col items-center flex-1">
              <div className={`w-8 h-8 rounded-full flex items-center justify-center text-[13px] font-semibold transition-colors ${
                i < step ? 'bg-[#006591] text-white' : i === step ? 'bg-[#2563eb] text-white' : 'bg-[#e9edff] text-[#737686]'
              }`}>
                {i < step ? <Check size={16} /> : i + 1}
              </div>
              <span className="text-[11px] text-[#737686] mt-1 text-center">{s}</span>
            </div>
            {i < STEPS.length - 1 && (
              <div className={`h-0.5 flex-1 mx-1 mb-4 ${i < step ? 'bg-[#006591]' : 'bg-[#e9edff]'}`} />
            )}
          </div>
        ))}
      </div>

      {/* Step content */}
      <div className="bg-[#ffffff] rounded-xl border border-[#e9edff] shadow-sm p-6">
        {/* Step 0: Source */}
        {step === 0 && (
          <div className="flex flex-col gap-5">
            <h2 className="text-[16px] font-semibold text-[#141b2b]">Source Configuration</h2>
            <div>
              <label className="block text-[13px] font-medium text-[#434655] mb-1.5">Job Name *</label>
              <input
                type="text"
                value={form.name}
                onChange={(e) => update('name', e.target.value)}
                placeholder="e.g. Daily ERP Backup"
                className="w-full h-9 px-3 rounded-lg border border-[#e9edff] bg-[#f9f9ff] text-[14px] text-[#141b2b] focus:outline-none focus:ring-2 focus:ring-[#2563eb]/20 focus:border-[#2563eb] transition-all"
              />
            </div>
            <div>
              <label className="block text-[13px] font-medium text-[#434655] mb-2">Source Type *</label>
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
                        ? 'border-[#2563eb] bg-[#dbe1ff]/30'
                        : 'border-[#e9edff] hover:border-[#c3c6d7]'
                    }`}
                  >
                    <s.Icon size={28} className={form.source_type === s.type ? 'text-[#2563eb]' : 'text-[#434655]'} />
                    <span className="text-[13px] font-medium text-[#141b2b]">{s.label}</span>
                  </button>
                ))}
              </div>
            </div>
            {form.source_type === 'FILESYSTEM' || form.source_type === 'DBF' ? (
              <div>
                <label className="block text-[13px] font-medium text-[#434655] mb-1.5">
                  {form.source_type === 'DBF' ? 'Dataset Directory *' : 'Source Path *'}
                </label>
                <input
                  type="text"
                  value={form.source_path}
                  onChange={(e) => update('source_path', e.target.value)}
                  placeholder={form.source_type === 'DBF' ? 'C:\\LegacyApp\\Data\\' : 'D:\\CompanyData\\'}
                  className="w-full h-9 px-3 rounded-lg border border-[#e9edff] bg-[#f9f9ff] font-mono text-[13px] text-[#141b2b] focus:outline-none focus:ring-2 focus:ring-[#2563eb]/20 focus:border-[#2563eb] transition-all"
                />
              </div>
            ) : form.source_type === 'POSTGRES' ? (
              <PgDatabasePicker
                value={form.source_database}
                onChange={(v) => update('source_database', v)}
              />
            ) : form.source_type === 'MONGODB' ? (
              <MongoDatabasePicker
                value={form.source_database}
                onChange={(v) => update('source_database', v)}
              />
            ) : (
              <SqlDatabasePicker
                value={form.source_database}
                onChange={(v) => update('source_database', v)}
              />
            )}
            {(form.source_type === 'FILESYSTEM') && (
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-[13px] font-medium text-[#434655] mb-1.5">Include Patterns</label>
                  <textarea
                    value={form.include_patterns}
                    onChange={(e) => update('include_patterns', e.target.value)}
                    placeholder="*.xlsx&#10;*.pdf&#10;*.docx"
                    rows={3}
                    className="w-full px-3 py-2 rounded-lg border border-[#e9edff] bg-[#f9f9ff] font-mono text-[12px] text-[#141b2b] focus:outline-none focus:ring-2 focus:ring-[#2563eb]/20 focus:border-[#2563eb] transition-all resize-none"
                  />
                </div>
                <div>
                  <label className="block text-[13px] font-medium text-[#434655] mb-1.5">Exclude Patterns</label>
                  <textarea
                    value={form.exclude_patterns}
                    onChange={(e) => update('exclude_patterns', e.target.value)}
                    placeholder="*.tmp&#10;*.log&#10;node_modules/"
                    rows={3}
                    className="w-full px-3 py-2 rounded-lg border border-[#e9edff] bg-[#f9f9ff] font-mono text-[12px] text-[#141b2b] focus:outline-none focus:ring-2 focus:ring-[#2563eb]/20 focus:border-[#2563eb] transition-all resize-none"
                  />
                </div>
              </div>
            )}
          </div>
        )}

        {/* Step 1: Agent */}
        {step === 1 && (
          <div className="flex flex-col gap-4">
            <h2 className="text-[16px] font-semibold text-[#141b2b]">Select Agent</h2>
            {agents.length === 0 ? (
              <div className="text-center py-8 text-[#737686] text-[13px]">
                No agents registered. <button type="button" onClick={() => navigate('/agents')} className="text-[#004ac6] hover:underline">Register an agent first.</button>
              </div>
            ) : (
              <div className="flex flex-col gap-2">
                {agents.map((agent) => (
                  <button
                    key={agent.id}
                    type="button"
                    onClick={() => update('agent_id', agent.id)}
                    className={`flex items-center gap-4 p-4 rounded-xl border-2 text-left transition-all ${
                      form.agent_id === agent.id ? 'border-[#2563eb] bg-[#dbe1ff]/20' : 'border-[#e9edff] hover:border-[#c3c6d7]'
                    }`}
                  >
                    <div className={`w-10 h-10 rounded-lg flex items-center justify-center ${agent.status === 'ONLINE' ? 'bg-[#c9e6ff]/40 text-[#006591]' : 'bg-[#dce2f7] text-[#737686]'}`}>
                      <Server size={20} />
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-[14px] font-semibold text-[#141b2b]">{agent.name}</p>
                      <p className="font-mono text-[11px] text-[#737686]">v{agent.version} · {agent.status === 'ONLINE' ? 'Online' : 'Offline'}</p>
                    </div>
                    <span className={`w-2.5 h-2.5 rounded-full ${agent.status === 'ONLINE' ? 'bg-[#006591]' : 'bg-[#737686]'}`} />
                  </button>
                ))}
              </div>
            )}
          </div>
        )}

        {/* Step 2: Schedule */}
        {step === 2 && (
          <div className="flex flex-col gap-5">
            <h2 className="text-[16px] font-semibold text-[#141b2b]">Schedule</h2>
            <div className="flex flex-wrap gap-2">
              {PRESETS.map((p) => (
                <button
                  key={p.cron}
                  type="button"
                  onClick={() => update('cron_expr', p.cron)}
                  className={`px-3 h-8 rounded-lg text-[12px] font-medium transition-colors ${
                    form.cron_expr === p.cron ? 'bg-[#2563eb] text-white' : 'bg-[#f1f3ff] text-[#434655] hover:bg-[#e9edff]'
                  }`}
                >
                  {p.label}
                </button>
              ))}
            </div>
            <div>
              <label className="block text-[13px] font-medium text-[#434655] mb-1.5">Cron Expression</label>
              <input
                type="text"
                value={form.cron_expr}
                onChange={(e) => update('cron_expr', e.target.value)}
                className="w-full h-9 px-3 rounded-lg border border-[#e9edff] bg-[#f9f9ff] font-mono text-[13px] text-[#141b2b] focus:outline-none focus:ring-2 focus:ring-[#2563eb]/20 focus:border-[#2563eb] transition-all"
              />
            </div>
            <div>
              <label className="block text-[13px] font-medium text-[#434655] mb-1.5">Timezone</label>
              <select
                value={form.timezone}
                onChange={(e) => update('timezone', e.target.value)}
                className="w-full h-9 px-3 rounded-lg border border-[#e9edff] bg-[#f9f9ff] text-[14px] text-[#141b2b] focus:outline-none focus:ring-2 focus:ring-[#2563eb]/20 focus:border-[#2563eb] transition-all"
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
            <h2 className="text-[16px] font-semibold text-[#141b2b]">Processing & Storage</h2>

            <div className="flex flex-col gap-3">
              {[
                { key: 'mode', label: 'Compression', desc: 'Zstandard — reduces storage by up to 60%', value: form.mode === 'COMPRESSED', toggle: (v: boolean) => update('mode', v ? 'COMPRESSED' : 'NORMAL') },
                { key: 'encrypted', label: 'Encryption', desc: 'AES-256-GCM — data encrypted before leaving your machine', value: form.encrypted, toggle: (v: boolean) => update('encrypted', v) },
              ].map((t) => (
                <div key={t.key} className="flex items-center justify-between p-4 rounded-xl border border-[#e9edff] bg-[#f9f9ff]">
                  <div>
                    <p className="text-[14px] font-medium text-[#141b2b]">{t.label}</p>
                    <p className="text-[12px] text-[#737686]">{t.desc}</p>
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
              <label className="block text-[13px] font-medium text-[#434655] mb-2">Storage Target *</label>
              {storageTargets.length === 0 ? (
                <div className="text-center py-6 text-[#737686] text-[13px]">
                  No storage targets. <button type="button" onClick={() => navigate('/storage')} className="text-[#004ac6] hover:underline">Add one first.</button>
                </div>
              ) : (
                <div className="flex flex-col gap-2">
                  {storageTargets.map((s) => {
                    const StorIcon = storageTypeIcon[s.type as keyof typeof storageTypeIcon] ?? HardDrive
                    return (
                      <button
                        key={s.id}
                        type="button"
                        onClick={() => update('storage_target_id', s.id)}
                        className={`flex items-center gap-3 p-3.5 rounded-xl border-2 text-left transition-all ${
                          form.storage_target_id === s.id ? 'border-[#2563eb] bg-[#dbe1ff]/20' : 'border-[#e9edff] hover:border-[#c3c6d7]'
                        }`}
                      >
                        <StorIcon size={20} className="text-[#004ac6]" />
                        <div>
                          <p className="text-[13px] font-semibold text-[#141b2b]">{s.name}</p>
                          <p className="font-mono text-[11px] text-[#737686]">{s.type} · {s.bucket || s.path || '—'}</p>
                        </div>
                      </button>
                    )
                  })}
                </div>
              )}
            </div>

            <div>
              <label className="block text-[13px] font-medium text-[#434655] mb-1.5">Retention (days)</label>
              <input
                type="number"
                value={form.retention_days}
                onChange={(e) => update('retention_days', parseInt(e.target.value))}
                min={1}
                max={365}
                className="w-32 h-9 px-3 rounded-lg border border-[#e9edff] bg-[#f9f9ff] text-[14px] text-[#141b2b] focus:outline-none focus:ring-2 focus:ring-[#2563eb]/20 focus:border-[#2563eb] transition-all"
              />
            </div>
          </div>
        )}

        {/* Step 4: Review */}
        {step === 4 && (
          <div className="flex flex-col gap-5">
            <h2 className="text-[16px] font-semibold text-[#141b2b]">Review & Create</h2>
            <div className="grid grid-cols-2 gap-x-8 gap-y-3 text-[13px]">
              {[
                { label: 'Job Name', value: form.name },
                { label: 'Source Type', value: form.source_type.replace('_', ' ') },
                { label: 'Source', value: form.source_path || form.source_database || '—', mono: true },
                { label: 'Agent', value: selectedAgent?.name ?? '—' },
                { label: 'Schedule', value: form.cron_expr, mono: true },
                { label: 'Timezone', value: form.timezone },
                { label: 'Compression', value: form.mode === 'COMPRESSED' ? 'Zstandard' : 'Disabled' },
                { label: 'Encryption', value: form.encrypted ? 'AES-256-GCM' : 'Disabled' },
                { label: 'Storage', value: selectedStorage?.name ?? '—' },
                { label: 'Retention', value: `${form.retention_days} days` },
              ].map((r) => (
                <div key={r.label} className="flex flex-col gap-0.5">
                  <span className="text-[11px] font-semibold uppercase tracking-wider text-[#737686]">{r.label}</span>
                  <span className={`text-[#141b2b] ${r.mono ? 'font-mono text-[12px]' : 'font-medium'}`}>{r.value}</span>
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
          onClick={() => step === 0 ? navigate('/jobs') : setStep((s) => s - 1)}
          className="px-4 h-9 rounded-lg bg-[#ffffff] border border-[#e9edff] text-[#141b2b] text-[13px] font-medium hover:bg-[#f1f3ff] transition-colors"
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

export default NewJobPage
