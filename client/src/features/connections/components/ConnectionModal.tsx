import { useState } from 'react'
import { connectionsApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import type { Agent, ConnectionType, DatabaseConnection, DatabaseInfo } from '@/types'
import { formatBytes } from '@/utils/format'
import { AGENT_BROWSE_BASE_URL as BROWSE_BASE } from '@/CONSTANTS'
import { DbTypeIcon } from '@/components/ui/SourceIcons'
import { Loader2, X, DatabaseBackup, Radar } from 'lucide-react'

const DEFAULT_PORTS: Record<ConnectionType, number> = { POSTGRES: 5432, MONGODB: 27017, MSSQL: 1433 }

// Default login per engine — matches what the type switch already does.
const DEFAULT_USERS: Record<ConnectionType, string> = { POSTGRES: 'postgres', MONGODB: '', MSSQL: 'sa' }

// One engine found by the agent's /detect scan of this machine.
interface DetectedInstance {
  type: ConnectionType
  product: string
  host: string
  port: number
  source: string
}

interface Props {
  agents: Agent[]
  editing: DatabaseConnection | null
  onClose: () => void
  onSaved: () => void
}

// Add/Edit connection. Test + discovery go to the loopback agent helper
// (credentials never touch the server until Save), then the password is sent
// once over HTTPS and stored server-side as AES-256-GCM ciphertext.
const ConnectionModal = ({ agents, editing, onClose, onSaved }: Props) => {
  const { currentOrg } = useAuthStore()
  const { addToast } = useUIStore()
  const [form, setForm] = useState({
    name: editing?.name ?? '',
    type: (editing?.type ?? 'POSTGRES') as ConnectionType,
    agent_id: editing?.agent_id ?? '',
    host: editing?.host ?? 'localhost',
    port: editing?.port ?? 5432,
    username: editing?.username ?? 'postgres',
    password: '',
  })
  const [dbs, setDbs] = useState<DatabaseInfo[]>(editing?.databases ?? [])
  const [testing, setTesting] = useState(false)
  const [testOk, setTestOk] = useState<boolean | null>(null)
  const [testError, setTestError] = useState('')
  const [saving, setSaving] = useState(false)
  const [detecting, setDetecting] = useState(false)
  const [detected, setDetected] = useState<DetectedInstance[]>([])
  const [detectNote, setDetectNote] = useState<{ text: string; isError: boolean } | null>(null)

  type FormState = typeof form

  const update = (k: string, v: unknown) => setForm((f) => ({ ...f, [k]: v }))
  const changeType = (t: ConnectionType) => {
    setForm((f) => ({
      ...f,
      type: t,
      port: DEFAULT_PORTS[t],
      username: t === 'POSTGRES' ? 'postgres' : t === 'MSSQL' ? 'sa' : f.username,
    }))
    setDbs([])
    setTestOk(null)
    setTestError('')
  }

  const buildMongoUri = (f: FormState) => {
    // No-auth servers (like a local MongoDB without access control) get a
    // bare URI — never "user:@", which drivers reject.
    if (!f.username.trim()) {
      return `mongodb://${f.host}:${f.port}/?directConnection=true`
    }
    const u = encodeURIComponent(f.username)
    const p = encodeURIComponent(f.password)
    return `mongodb://${u}:${p}@${f.host}:${f.port}/?authSource=admin`
  }

  // Agent-side validation via loopback helper. Never sends creds to the server.
  const testViaAgent = async (f: FormState): Promise<string[]> => {
    if (f.type === 'POSTGRES') {
      const res = await fetch(`${BROWSE_BASE}/pg-databases`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ host: f.host, port: Number(f.port), user: f.username, password: f.password || undefined }),
      })
      if (!res.ok) throw new Error('Agent rejected the connection (bad response).')
      const data: { databases?: { name: string }[]; error?: string } = await res.json()
      const names = (data.databases || []).map((d) => d.name).filter(Boolean)
      if (data.error) throw new Error(data.error)
      if (names.length === 0) throw new Error('Connected, but no databases returned.')
      return names
    }
    if (f.type === 'MONGODB') {
      const res = await fetch(`${BROWSE_BASE}/mongo-databases`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ uri: buildMongoUri(f) }),
      })
      if (!res.ok) throw new Error('Agent rejected the connection (bad response).')
      const data: { databases?: { name: string }[]; error?: string } = await res.json()
      const names = (data.databases || []).map((d) => d.name).filter(Boolean)
      if (data.error) throw new Error(data.error)
      if (names.length === 0) throw new Error('Connected, but no databases returned.')
      return names
    }
    // MSSQL: loopback helper auto-detects localhost instances over Windows auth.
    const res = await fetch(`${BROWSE_BASE}/databases`)
    if (!res.ok) throw new Error('Agent database discovery unreachable.')
    const data: { databases?: { name: string }[]; error?: string } = await res.json()
    const names = (data.databases || []).map((d) => d.name).filter(Boolean)
    if (data.error) throw new Error(data.error)
    return names
  }

  // override lets "detect → use this instance" test the values it just set
  // without waiting for a state round-trip.
  const handleTest = async (override?: Partial<FormState>) => {
    const f: FormState = { ...form, ...(override ?? {}) }
    setTesting(true)
    setTestError('')
    setTestOk(null)
    try {
      // Editing without re-entering the password: verify via the server's
      // stored credential instead of the loopback helper.
      if (editing && !f.password) {
        if (!currentOrg) throw new Error('No organization.')
        const res = await connectionsApi.test(currentOrg.id, editing.id)
        if (res.status !== 'ok') throw new Error(res.error || 'Connection failed.')
        setDbs(res.databases ?? editing.databases ?? [])
        setTestOk(true)
        return
      }
      const names = await testViaAgent(f)
      // Loopback discovery yields names only; the server enriches details
      // on save/test via its own credentialed inspection.
      setDbs(names.map((n) => ({ name: n, size_bytes: -1, table_count: -1 })))
      setTestOk(true)
    } catch (e) {
      setTestOk(false)
      // Never echo credentials: generic message only.
      setTestError(e instanceof Error ? e.message : 'Connection failed. Check host, port, and credentials.')
    } finally {
      setTesting(false)
    }
  }

  // Asks the agent on this machine which database engines are listening, then
  // fills the form and immediately tries to connect.
  const handleDetect = async () => {
    setDetecting(true)
    setDetectNote(null)
    try {
      const res = await fetch(`${BROWSE_BASE}/detect`)
      if (!res.ok) throw new Error('bad response')
      const data: { instances?: DetectedInstance[]; error?: string } = await res.json()
      if (data.error) throw new Error(data.error)
      const found = data.instances ?? []
      setDetected(found)
      if (found.length === 0) {
        setDetectNote({
          text: 'Nothing listening on the usual local ports (5432, 5433, 1433, 27017, 27018). Enter the details by hand.',
          isError: false,
        })
      }
    } catch {
      setDetectNote({
        text: 'Could not reach the agent on this machine. Make sure it is running here, or enter the details by hand.',
        isError: true,
      })
    } finally {
      setDetecting(false)
    }
  }

  // Prefills the form from a detected instance and runs the connection test.
  const applyInstance = (i: DetectedInstance) => {
    const next: FormState = {
      ...form,
      type: i.type,
      host: i.host,
      port: i.port,
      username: DEFAULT_USERS[i.type] ?? form.username,
      name: form.name.trim() || `${i.product} ${i.host}`,
    }
    setForm(next)
    setDbs([])
    setTestOk(null)
    setTestError('')
    void handleTest(next)
  }

  const handleSave = async () => {
    if (!currentOrg) return
    if (!form.name.trim() || !form.agent_id || !form.host.trim()) {
      addToast('error', 'Name, agent, and host are required')
      return
    }
    // Username/password are optional: no-auth MongoDB and Windows-integrated
    // MSSQL legitimately have none. Blank means "connect without credentials".
    setSaving(true)
    try {
      const payload: Record<string, unknown> = {
        name: form.name.trim(),
        type: form.type,
        agent_id: form.agent_id,
        host: form.host.trim(),
        port: Number(form.port),
        username: form.username.trim(),
        databases: dbs.map((d) => d.name),
      }
      if (form.password) payload.password = form.password
      if (editing) {
        await connectionsApi.update(currentOrg.id, editing.id, payload)
        addToast('success', `${form.name} updated`)
      } else {
        await connectionsApi.create(currentOrg.id, payload as never)
        addToast('success', `${form.name} saved`)
      }
      onSaved()
    } catch (e) {
      addToast('error', e instanceof Error ? e.message : 'Save failed')
    } finally {
      setSaving(false)
    }
  }

  const inputCls = 'w-full h-9 px-3 rounded-lg border border-surface-variant bg-surface-container-low text-[13px] text-on-surface focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all'

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/40 p-4" onClick={onClose}>
      <div className="w-full max-w-lg bg-surface-container-lowest rounded-2xl border border-surface-variant shadow-xl p-6 flex flex-col gap-4 max-h-[90vh] overflow-y-auto" onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center justify-between">
          <h2 className="text-[16px] font-semibold text-on-surface">{editing ? 'Edit Connection' : 'Add Connection'}</h2>
          <button type="button" onClick={onClose} className="p-1.5 rounded-lg hover:bg-surface-container-high text-on-surface-variant"><X size={16} /></button>
        </div>

        <div>
          <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Connection Name *</label>
          <input type="text" value={form.name} onChange={(e) => update('name', e.target.value)} placeholder="e.g. PostgreSQL Production" className={inputCls} />
        </div>

        <div className="grid grid-cols-2 gap-3">
          <div>
            <label htmlFor="vg-conn-type" className="flex items-center gap-1.5 text-[13px] font-medium text-on-surface-variant mb-1.5">
              <DbTypeIcon type={form.type} size={14} />
              Database Type *
            </label>
            <select id="vg-conn-type" value={form.type} onChange={(e) => changeType(e.target.value as ConnectionType)} className={inputCls}>
              <option value="POSTGRES">PostgreSQL</option>
              <option value="MONGODB">MongoDB</option>
              <option value="MSSQL">SQL Server</option>
            </select>
          </div>
          <div>
            <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Agent *</label>
            <select value={form.agent_id} onChange={(e) => update('agent_id', e.target.value)} className={inputCls}>
              <option value="">Select agent…</option>
              {agents.map((a) => (
                <option key={a.id} value={a.id}>{a.name} ({a.status})</option>
              ))}
            </select>
          </div>
        </div>

        <div className="grid grid-cols-3 gap-3">
          <div className="col-span-2">
            <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Host *</label>
            <input type="text" value={form.host} onChange={(e) => update('host', e.target.value)} className={`${inputCls} font-mono`} />
          </div>
          <div>
            <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Port *</label>
            <input type="number" value={form.port} onChange={(e) => update('port', Number(e.target.value))} className={`${inputCls} font-mono`} />
          </div>
        </div>

        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Username (blank = no auth)</label>
            <input type="text" value={form.username} onChange={(e) => update('username', e.target.value)} autoComplete="off" className={`${inputCls} font-mono`} />
          </div>
          <div>
            <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Password {editing ? '(blank = keep)' : '(blank = no auth)'}</label>
            <input type="password" value={form.password} onChange={(e) => update('password', e.target.value)} autoComplete="new-password" className={inputCls} />
          </div>
        </div>

        <div className="rounded-xl border border-surface-variant bg-surface-container-low p-3">
          <div className="flex items-center justify-between gap-3">
            <div className="min-w-0">
              <p className="text-[12.5px] font-semibold text-on-surface">Installed on this machine</p>
              <p className="text-[11px] text-on-surface-variant">Scan for a running database engine and fill the form with it.</p>
            </div>
            <button
              type="button"
              onClick={() => void handleDetect()}
              disabled={detecting}
              className="shrink-0 flex items-center gap-1.5 px-3 h-8 rounded-lg border border-surface-variant bg-surface-container-lowest text-[12px] font-medium text-on-surface hover:bg-surface-container-high disabled:opacity-50 transition-colors"
            >
              {detecting ? <Loader2 size={13} className="animate-spin" /> : <Radar size={13} />}
              {detecting ? 'Scanning…' : 'Detect'}
            </button>
          </div>

          {detected.length > 0 && (
            <div className="mt-2.5 flex flex-col gap-1.5">
              {detected.map((i) => (
                <button
                  key={`${i.type}-${i.port}`}
                  type="button"
                  onClick={() => applyInstance(i)}
                  title={`Use ${i.product} at ${i.host}:${i.port}`}
                  className="flex items-center gap-2.5 px-2.5 py-2 rounded-lg border border-surface-variant bg-surface-container-lowest text-left hover:border-primary/50 hover:bg-primary-container/20 transition-colors"
                >
                  <DbTypeIcon type={i.type} size={18} />
                  <span className="text-[12.5px] font-medium text-on-surface">{i.product}</span>
                  <span className="font-mono text-[11.5px] text-on-surface-variant">{i.host}:{i.port}</span>
                  <span className="ml-auto text-[11.5px] font-medium text-primary">Use</span>
                </button>
              ))}
            </div>
          )}
          {detectNote && (
            <p className={`mt-2 text-[11.5px] ${detectNote.isError ? 'text-error' : 'text-on-surface-variant'}`}>
              {detectNote.text}
            </p>
          )}
        </div>

        <button type="button" onClick={() => void handleTest()} disabled={testing}
          className="self-start flex items-center gap-1.5 text-[12px] font-medium text-primary hover:underline disabled:opacity-50">
          {testing ? <Loader2 size={13} className="animate-spin" /> : <DatabaseBackup size={13} />}
          Test Connection
        </button>
        {testOk === true && <p className="text-[12px] text-primary">Connected{dbs.length > 0 ? ` — ${dbs.length} databases found.` : '.'}</p>}
        {testOk === false && <p className="text-[12px] text-error">{testError}</p>}

        {dbs.length > 0 && (
          <div className="rounded-xl border border-surface-variant bg-surface-container-low p-3">
            <p className="text-[12px] font-semibold text-on-surface mb-1.5">Available databases ({dbs.length})</p>
            <div className="flex flex-wrap gap-1.5 max-h-28 overflow-y-auto">
              {dbs.map((d) => (
                <span key={d.name} title={d.size_bytes >= 0 ? formatBytes(d.size_bytes) : undefined}
                  className="px-2 py-0.5 rounded-md bg-surface-container-high font-mono text-[11px] text-on-surface">
                  {d.name}{d.size_bytes >= 0 ? ` · ${formatBytes(d.size_bytes)}` : ''}
                </span>
              ))}
            </div>
          </div>
        )}
        <p className="text-[11px] text-on-surface-variant">Test credentials go only to the agent on this machine — the saved password is encrypted server-side and never shown again.</p>

        <div className="flex items-center justify-end gap-2 pt-1">
          <button type="button" onClick={onClose} className="px-4 h-9 rounded-lg border border-surface-variant text-[13px] font-medium text-on-surface hover:bg-surface-container-low">Cancel</button>
          <button type="button" onClick={() => void handleSave()} disabled={saving}
            className="px-4 h-9 rounded-lg bg-primary text-on-primary text-[13px] font-semibold hover:opacity-90 disabled:opacity-50 flex items-center gap-1.5">
            {saving && <Loader2 size={14} className="animate-spin" />} Save Connection
          </button>
        </div>
      </div>
    </div>
  )
}

export default ConnectionModal
