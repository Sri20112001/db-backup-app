import { useState } from 'react'
import { DatabaseBackup, Loader2 } from 'lucide-react'

const BROWSE_BASE = import.meta.env.VITE_AGENT_BROWSE_URL || 'http://127.0.0.1:7546'

interface Props {
  value: string
  onChange: (name: string) => void
}

// Lists PostgreSQL databases via the agent on this machine. Connection
// details (including the password) go only to the loopback agent helper for
// one-shot discovery — they are never sent to the SaaS server or stored.
const PgDatabasePicker = ({ value, onChange }: Props) => {
  const [host, setHost] = useState('localhost')
  const [port, setPort] = useState('5432')
  const [user, setUser] = useState('postgres')
  const [password, setPassword] = useState('')
  const [dbs, setDbs] = useState<string[] | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')
  const [manual, setManual] = useState(false)

  const discover = async () => {
    setIsLoading(true)
    setError('')
    try {
      const res = await fetch(`${BROWSE_BASE}/pg-databases`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ host, port: parseInt(port) || 5432, user, password }),
      })
      if (!res.ok) throw new Error('bad response')
      const data: { databases?: { name: string }[]; error?: string } = await res.json()
      const names = (data.databases || []).map((d) => d.name).filter(Boolean)
      if (names.length === 0) {
        throw new Error(data.error || 'No databases returned.')
      }
      setDbs(names)
      setManual(false)
    } catch (e) {
      setError(
        e instanceof Error
          ? e.message
          : 'Could not reach the agent helper. It must run on this machine (127.0.0.1:7546).',
      )
      setDbs(null)
    } finally {
      setIsLoading(false)
    }
  }

  const showSelect = dbs !== null && !manual
  const inputCls =
    'w-full h-9 px-3 rounded-lg border border-[#e9edff] bg-[#f9f9ff] font-mono text-[13px] text-[#141b2b] focus:outline-none focus:ring-2 focus:ring-[#2563eb]/20 focus:border-[#2563eb] transition-all'
  const connCls =
    'h-9 px-3 rounded-lg border border-[#e9edff] bg-[#f9f9ff] font-mono text-[13px] text-[#141b2b] focus:outline-none focus:ring-2 focus:ring-[#2563eb]/20 focus:border-[#2563eb] transition-all'

  return (
    <div className="flex flex-col gap-2">
      <label className="block text-[13px] font-medium text-[#434655]">Database Name *</label>

      <div className="grid grid-cols-2 gap-2">
        <input
          type="text"
          value={host}
          onChange={(e) => setHost(e.target.value)}
          placeholder="host"
          title="PostgreSQL host"
          className={connCls}
        />
        <input
          type="text"
          value={port}
          onChange={(e) => setPort(e.target.value)}
          placeholder="port"
          title="PostgreSQL port"
          className={connCls}
        />
        <input
          type="text"
          value={user}
          onChange={(e) => setUser(e.target.value)}
          placeholder="user"
          title="PostgreSQL user"
          className={connCls}
        />
        <input
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder="password"
          title="PostgreSQL password (used once for discovery, never stored)"
          autoComplete="off"
          className={connCls}
        />
      </div>

      <button
        type="button"
        onClick={() => void discover()}
        disabled={isLoading}
        className="self-start flex items-center gap-1.5 text-[12px] font-medium text-[#004ac6] hover:underline disabled:opacity-50"
      >
        {isLoading ? <Loader2 size={13} className="animate-spin" /> : <DatabaseBackup size={13} />}
        {dbs === null ? 'Detect databases' : 'Re-detect'}
      </button>

      {showSelect ? (
        <select
          value={dbs.includes(value) ? value : value === '' ? '' : '__manual'}
          onChange={(e) => {
            if (e.target.value === '__manual') {
              setManual(true)
            } else {
              onChange(e.target.value)
            }
          }}
          className={`${inputCls} appearance-none cursor-pointer`}
        >
          <option value="">Select a database…</option>
          {dbs.map((d) => (
            <option key={d} value={d}>
              {d}
            </option>
          ))}
          <option value="__manual">Type manually…</option>
        </select>
      ) : (
        <input
          type="text"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder="mydb"
          className={inputCls}
        />
      )}

      {manual && dbs !== null && (
        <button
          type="button"
          onClick={() => setManual(false)}
          className="self-start text-[12px] font-medium text-[#004ac6] hover:underline"
        >
          ← Back to detected list
        </button>
      )}

      {error && <p className="text-[12px] text-[#737686]">{error}</p>}
      <p className="text-[11px] text-[#737686]">
        Credentials go only to the agent on this machine for discovery — never stored.
      </p>
    </div>
  )
}

export default PgDatabasePicker
