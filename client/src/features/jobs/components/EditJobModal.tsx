import { useEffect, useState } from 'react'
import { jobApi, agentApi, storageApi, policyApi, connectionsApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import type { Agent, StorageTarget, BackupSourceType, BackupMode, BackupPolicy, DatabaseConnection, ConnectionType } from '@/types'
import { X, Loader2, FolderSearch } from 'lucide-react'
import FolderBrowser from '@/components/FolderBrowser'
import ScheduleBuilder from '@/components/ScheduleBuilder'
import ConnectionDatabasePicker from './ConnectionDatabasePicker'

interface Props {
  jobId: string
  onClose: () => void
  onSaved: () => void
}

// Job source types map onto the connection type that can serve them — used to
// filter the connection list shown in the database picker (same map as JobWizard).
const SOURCE_TO_CONNECTION: Partial<Record<BackupSourceType, ConnectionType>> = {
  POSTGRES: 'POSTGRES',
  MONGODB: 'MONGODB',
  MSSQL_SERVER: 'MSSQL',
}

const EditJobModal = ({ jobId, onClose, onSaved }: Props) => {
  const { currentOrg } = useAuthStore()
  const { addToast } = useUIStore()

  const [agents, setAgents] = useState<Agent[]>([])
  const [connections, setConnections] = useState<DatabaseConnection[]>([])
  const [storageTargets, setStorageTargets] = useState<StorageTarget[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [showBrowser, setShowBrowser] = useState(false)

  const [form, setForm] = useState({
    name: '',
    source_type: 'FILESYSTEM' as BackupSourceType,
    source_path: '',
    source_database: '',
    connection_id: '',
    include_patterns: '',
    exclude_patterns: '',
    agent_id: '',
    storage_target_id: '',
    policy_id: '',
    cron_expr: '',
    timezone: 'UTC',
    mode: 'COMPRESSED' as BackupMode,
    encrypted: true,
    retention_days: 30,
    export_format: 'ARCHIVE',
  })
  const [policies, setPolicies] = useState<BackupPolicy[]>([])

  useEffect(() => {
    if (!currentOrg || !jobId) return
    Promise.all([
      jobApi.get(currentOrg.id, jobId),
      agentApi.list(currentOrg.id, { limit: 200 }),
      storageApi.list(currentOrg.id),
      policyApi.list(currentOrg.id),
      connectionsApi.list(currentOrg.id, { limit: 200 }),
    ])
      .then(([job, a, s, p, c]) => {
        setAgents(a.data)
        setStorageTargets(s)
        setPolicies(p)
        setConnections(c.data)
        setForm({
          name: job.name,
          source_type: job.source_type,
          source_path: job.source_path ?? '',
          source_database: job.source_database ?? '',
          connection_id: job.connection_id ?? '',
          include_patterns: job.include_patterns ?? '',
          exclude_patterns: job.exclude_patterns ?? '',
          agent_id: job.agent_id,
          storage_target_id: job.storage_target_id,
          policy_id: job.policy_id ?? '',
          cron_expr: job.schedule?.cron_expr ?? '',
          timezone: job.schedule?.timezone ?? 'UTC',
          mode: job.mode,
          encrypted: job.encrypted,
          retention_days: job.retention_days,
          export_format: job.export_format || 'ARCHIVE',
        })
      })
      .catch(() => addToast('error', 'Failed to load job'))
      .finally(() => setIsLoading(false))
  }, [currentOrg, jobId])

  const update = (key: string, value: unknown) => setForm((f) => ({ ...f, [key]: value }))

  // Re-fetch only the connections (used by the picker's Refresh so the newly
  // discovered database list lands in state without reloading the whole job).
  const refreshConnections = () => {
    if (!currentOrg) return
    connectionsApi.list(currentOrg.id, { limit: 200 })
      .then((res) => setConnections(res.data))
      .catch(() => addToast('error', 'Failed to refresh connections'))
  }

  const connectionType = SOURCE_TO_CONNECTION[form.source_type]
  const visibleConnections = connectionType ? connections.filter((c) => c.type === connectionType) : []

  // Picking a connection pins the job to the connection's agent so the host
  // with database access runs the backup. The database choice resets because
  // it belongs to the previous server.
  const handleConnectionChange = (id: string) => {
    const conn = connections.find((c) => c.id === id)
    setForm((f) => ({
      ...f,
      connection_id: id,
      source_database: '',
      agent_id: conn ? conn.agent_id : f.agent_id,
    }))
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!currentOrg || !jobId) return
    setIsSubmitting(true)
    try {
      // export_format is MongoDB-only server-side: strip it for every other
      // source type (the form keeps an ARCHIVE fallback). Empty policy_id
      // means "leave attached policy unchanged". connection_id is omitted when
      // empty so the server leaves the job's connection untouched (filesystem /
      // legacy jobs stay backward compatible).
      const payload: Record<string, unknown> = { ...form }
      if (payload.source_type !== 'MONGODB') delete payload.export_format
      if (!payload.policy_id) delete payload.policy_id
      if (!payload.connection_id) delete payload.connection_id
      if (payload.source_type === 'FILESYSTEM' || payload.source_type === 'DBF') delete payload.connection_id
      await jobApi.update(currentOrg.id, jobId, payload)
      addToast('success', `${form.name} updated`)
      onSaved()
      onClose()
    } catch (err: unknown) {
      addToast('error', err instanceof Error ? err.message : 'Failed to update job')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <>
    <div className="fixed inset-0 z-[200] flex items-center justify-center">
      <div className="absolute inset-0 bg-on-surface/30 backdrop-blur-[2px]" onClick={onClose} />
      <div className="relative bg-surface-container-lowest rounded-xl shadow-2xl w-full max-w-2xl mx-4 border border-surface-variant max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between px-6 py-4 border-b border-surface-variant sticky top-0 bg-surface-container-lowest z-10 rounded-t-xl">
          <h2 className="text-[16px] font-semibold text-on-surface">Edit Backup Job</h2>
          <button
            type="button"
            onClick={onClose}
            className="p-1.5 rounded-lg text-on-surface-variant hover:bg-surface-container-high transition-colors"
          >
            <X size={18} />
          </button>
        </div>

        {isLoading ? (
          <div className="flex items-center justify-center py-16">
            <Loader2 size={32} className="text-on-surface-variant animate-spin" />
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="flex flex-col gap-4 p-6">
            <Field label="Job Name *">
              <input
                type="text"
                value={form.name}
                onChange={(e) => update('name', e.target.value)}
                required
                className="w-full h-9 px-3 rounded-lg border border-surface-variant bg-surface-container-low text-[14px] text-on-surface focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all"
              />
            </Field>

            {(form.source_type === 'FILESYSTEM' || form.source_type === 'DBF') ? (
              <Field label="Source Path">
                <div className="flex items-center gap-2">
                  <input
                    type="text"
                    value={form.source_path}
                    onChange={(e) => update('source_path', e.target.value)}
                    className="flex-1 h-9 px-3 rounded-lg border border-surface-variant bg-surface-container-low font-mono text-[13px] text-on-surface focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all"
                  />
                  <button
                    type="button"
                    onClick={() => setShowBrowser(true)}
                    title="Browse folders on this machine"
                    className="shrink-0 flex items-center gap-2 px-3 h-9 rounded-lg bg-surface-container-lowest border border-surface-variant text-on-surface text-[13px] font-medium hover:bg-surface-container-high transition-colors"
                  >
                    <FolderSearch size={16} />
                    Browse
                  </button>
                </div>
              </Field>
            ) : (
              <ConnectionDatabasePicker
                connections={visibleConnections}
                connectionId={form.connection_id}
                database={form.source_database}
                agent={agents.find((a) => a.id === form.agent_id)}
                onConnectionChange={handleConnectionChange}
                onDatabaseChange={(v) => update('source_database', v)}
                onRefreshConnections={refreshConnections}
              />
            )}

            {form.source_type === 'FILESYSTEM' && (
              <div className="grid grid-cols-2 gap-4">
                <Field label="Include Patterns">
                  <textarea
                    value={form.include_patterns}
                    onChange={(e) => update('include_patterns', e.target.value)}
                    rows={3}
                    className="w-full px-3 py-2 rounded-lg border border-surface-variant bg-surface-container-low font-mono text-[12px] text-on-surface focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all resize-none"
                  />
                </Field>
                <Field label="Exclude Patterns">
                  <textarea
                    value={form.exclude_patterns}
                    onChange={(e) => update('exclude_patterns', e.target.value)}
                    rows={3}
                    className="w-full px-3 py-2 rounded-lg border border-surface-variant bg-surface-container-low font-mono text-[12px] text-on-surface focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all resize-none"
                  />
                </Field>
              </div>
            )}

            <div className="grid grid-cols-2 gap-4">
              <Field label="Agent">
                <select
                  value={form.agent_id}
                  onChange={(e) => update('agent_id', e.target.value)}
                  className="w-full h-9 px-3 rounded-lg border border-surface-variant bg-surface-container-low text-[14px] text-on-surface focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all"
                >
                  {agents.map((a) => <option key={a.id} value={a.id}>{a.name} ({a.status})</option>)}
                </select>
              </Field>

              <Field label="Storage Target">
                <select
                  value={form.storage_target_id}
                  onChange={(e) => update('storage_target_id', e.target.value)}
                  className="w-full h-9 px-3 rounded-lg border border-surface-variant bg-surface-container-low text-[14px] text-on-surface focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all"
                >
                  {storageTargets.map((s) => <option key={s.id} value={s.id}>{s.name} ({s.type})</option>)}
                </select>
              </Field>
            </div>

            <ScheduleBuilder
              cron={form.cron_expr}
              timezone={form.timezone}
              onChange={(c) => update('cron_expr', c)}
              idPrefix="vg-editjob"
            />

            <Field label="Timezone">
              <select
                value={form.timezone}
                onChange={(e) => update('timezone', e.target.value)}
                className="w-full h-9 px-3 rounded-lg border border-surface-variant bg-surface-container-low text-[14px] text-on-surface focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all"
              >
                {['UTC', 'America/New_York', 'America/Chicago', 'America/Los_Angeles', 'Europe/London', 'Europe/Berlin', 'Asia/Tokyo'].map((tz) => (
                  <option key={tz} value={tz}>{tz}</option>
                ))}
              </select>
            </Field>

            <Field label="Backup Policy (optional — governs schedule, processing, retention, retry)">
              <select
                value={form.policy_id}
                onChange={(e) => update('policy_id', e.target.value)}
                className="w-full h-9 px-3 rounded-lg border border-surface-variant bg-surface-container-low text-[14px] text-on-surface focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all"
              >
                <option value="">Job-specific settings (no policy)</option>
                {policies.filter((p) => p.enabled).map((p) => (
                  <option key={p.id} value={p.id}>{p.name} · {p.cron_expr || 'manual'}</option>
                ))}
              </select>
            </Field>

            <div className="grid grid-cols-3 gap-4">
              <Field label="Retention (days)">
                <input
                  type="number"
                  value={form.retention_days}
                  onChange={(e) => update('retention_days', parseInt(e.target.value))}
                  min={1}
                  max={365}
                  className="w-full h-9 px-3 rounded-lg border border-surface-variant bg-surface-container-low text-[14px] text-on-surface focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all"
                />
              </Field>
              <Field label="Mode">
                <select
                  value={form.mode}
                  onChange={(e) => update('mode', e.target.value)}
                  className="w-full h-9 px-3 rounded-lg border border-surface-variant bg-surface-container-low text-[14px] text-on-surface focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all"
                >
                  <option value="NORMAL">Normal</option>
                  <option value="COMPRESSED">Compressed</option>
                </select>
              </Field>
              {form.source_type === 'MONGODB' && (
                <Field label="Export Format">
                  <select
                    value={form.export_format}
                    onChange={(e) => update('export_format', e.target.value)}
                    className="w-full h-9 px-3 rounded-lg border border-surface-variant bg-surface-container-low text-[14px] text-on-surface focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all"
                  >
                    <option value="ARCHIVE">Archive</option>
                    <option value="JSON">JSON</option>
                    <option value="CSV">CSV</option>
                  </select>
                </Field>
              )}
            </div>

            <div className="flex items-center justify-between p-4 rounded-xl border border-surface-variant bg-surface-container-low">
              <div>
                <p className="text-[14px] font-medium text-on-surface">Encryption</p>
                <p className="text-[12px] text-on-surface-variant">AES-256-GCM — data encrypted before leaving your machine</p>
              </div>
              <label className="relative inline-flex items-center cursor-pointer">
                <input type="checkbox" checked={form.encrypted} onChange={(e) => update('encrypted', e.target.checked)} className="sr-only peer" />
                <div className="w-11 h-6 bg-surface-variant peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-surface-container-lowest after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary" />
              </label>
            </div>

            <div className="flex items-center justify-end gap-2 pt-1">
              <button
                type="button"
                onClick={onClose}
                className="px-4 h-9 rounded-lg bg-surface-container-lowest border border-surface-variant text-on-surface text-[13px] font-medium hover:bg-surface-container-low transition-colors"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={isSubmitting}
                className="px-4 h-9 rounded-lg bg-primary text-on-primary text-[13px] font-medium hover:bg-primary-container transition-colors disabled:opacity-60 flex items-center gap-2"
              >
                {isSubmitting && <Loader2 size={14} className="animate-spin" />}
                {isSubmitting ? 'Saving...' : 'Save Changes'}
              </button>
            </div>
          </form>
        )}
      </div>
    </div>

    {showBrowser && (
      <FolderBrowser
        initialPath={form.source_path}
        onSelect={(p) => { update('source_path', p); setShowBrowser(false) }}
        onClose={() => setShowBrowser(false)}
      />
    )}
    </>
  )
}

const Field = ({ label, children }: { label: string; children: React.ReactNode }) => (
  <div className="flex flex-col gap-1.5">
    <label className="text-[13px] font-medium text-on-surface-variant">{label}</label>
    {children}
  </div>
)

export default EditJobModal
