import { useState } from 'react'
import { connectionsApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import type { Agent, ConnectionType, DatabaseConnection, DatabaseInfo } from '@/types'
import { formatBytes } from '@/utils/format'
import { AGENT_BROWSE_BASE_URL as BROWSE_BASE } from '@/CONSTANTS'
import { Loader2, X, DatabaseBackup } from 'lucide-react'

const DEFAULT_PORTS: Record<ConnectionType, number> = { POSTGRES: 5432, MONGODB: 27017, MSSQL: 1433 }

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

  const buildMongoUri = () => {
    // No-auth servers (like a local MongoDB without access control) get a
    // bare URI — never "user:@", which drivers reject.
    if (!form.username.trim()) {
      return `mongodb://${form.host}:${form.port}/?directConnection=true`
    }
    const u = encodeURIComponent(form.username)
    const p = encodeURIComponent(form.password)
    return `mongodb://${u}:${p}@${form.host}:${form.port}/?authSource=admin`
  }

  // Agent-side validation via loopback helper. Never sends creds to the server.
  const testViaAgent = async (): Promise<string[]> => {
    if (form.type === 'POSTGRES') {
      const res = await fetch(`${BROWSE_BASE}/pg-databases`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ host: form.host, port: Number(form.port), user: form.username, password: form.password || undefined }),
      })
      if (!res.ok) throw new Error('Agent rejected the connection (bad response).')
      const data: { databases?: { name: string }[]; error?: string } = await res.json()
      const names = (data.databases || []).map((d) => d.name).filter(Boolean)
      if (data.error) throw new Error(data.error)
      if (names.length === 0) throw new Error('Connected, but no databases returned.')
      return names
    }
    if (form.type === 'MONGODB') {
      const res = await fetch(`${BROWSE_BASE}/mongo-databases`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ uri: buildMongoUri() }),
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

  const handleTest = async () => {
    setTesting(true)
    setTestError('')
    setTestOk(null)
    try {
      // Editing without re-entering the password: verify via the server's
      // stored credential instead of the loopback helper.
      if (editing && !form.password) {
        if (!currentOrg) throw new Error('No organization.')
        const res = await connectionsApi.test(currentOrg.id, editing.id)
        if (res.status !== 'ok') throw new Error(res.error || 'Connection failed.')
        setDbs(res.databases ?? editing.databases ?? [])
        setTestOk(true)
        return
      }
      const names = await testViaAgent()
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
            <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Database Type *</label>
            <select value={form.type} onChange={(e) => changeType(e.target.value as ConnectionType)} className={inputCls}>
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
        <p className="text-[11px] text-outline">Test credentials go only to the agent on this machine — the saved password is encrypted server-side and never shown again.</p>

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
