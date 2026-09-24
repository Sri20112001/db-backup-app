import { useState } from 'react'
import { Loader2, RefreshCw } from 'lucide-react'
import { MongoDbIcon } from '@/components/ui/SourceIcons'

const BROWSE_BASE = import.meta.env.VITE_AGENT_BROWSE_URL || 'http://127.0.0.1:7546'

interface Props {
  value: string
  onChange: (name: string) => void
}

// Lists MongoDB databases via the agent on this machine. The connection
// string goes only to the loopback agent helper for one-shot discovery —
// it is never sent to the SaaS server or stored.
const MongoDatabasePicker = ({ value, onChange }: Props) => {
  const [uri, setUri] = useState('mongodb://localhost:27017')
  const [dbs, setDbs] = useState<string[] | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')
  const [manual, setManual] = useState(false)

  const discover = async () => {
    setIsLoading(true)
    setError('')
    try {
      const res = await fetch(`${BROWSE_BASE}/mongo-databases`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ uri }),
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

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center gap-2">
        <MongoDbIcon size={16} />
        <label className="block text-[13px] font-medium text-[#434655]">Database Name *</label>
      </div>

      <div className="flex gap-2">
        <input
          type="text"
          value={uri}
          onChange={(e) => setUri(e.target.value)}
          placeholder="mongodb://localhost:27017"
          title="MongoDB connection string (used once for discovery, never stored)"
          autoComplete="off"
          spellCheck={false}
          className={`${inputCls} flex-1`}
        />
        <button
          type="button"
          onClick={() => void discover()}
          disabled={isLoading || !uri.trim()}
          className="shrink-0 flex items-center gap-1.5 px-3 h-9 rounded-lg bg-[#f1f3ff] text-[#004ac6] text-[12px] font-medium hover:bg-[#e9edff] transition-colors disabled:opacity-50"
        >
          {isLoading ? <Loader2 size={13} className="animate-spin" /> : <RefreshCw size={13} />}
          {dbs === null ? 'Detect' : 'Re-detect'}
        </button>
      </div>

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
        Connection string goes only to the agent on this machine for discovery — never stored.
      </p>
    </div>
  )
}

export default MongoDatabasePicker
