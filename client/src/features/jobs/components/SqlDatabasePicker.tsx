import { useCallback, useEffect, useState } from 'react'
import { Database, Loader2, RefreshCw } from 'lucide-react'

const BROWSE_BASE = import.meta.env.VITE_AGENT_BROWSE_URL || 'http://127.0.0.1:7546'

interface Props {
  value: string
  onChange: (name: string) => void
}

// Lists SQL Server databases discovered by the agent on this machine.
// Falls back to manual entry when the agent is unreachable (remote agents)
// or when no local SQL Server is found.
const SqlDatabasePicker = ({ value, onChange }: Props) => {
  const [dbs, setDbs] = useState<string[] | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')
  const [manual, setManual] = useState(false)

  const discover = useCallback(async () => {
    setIsLoading(true)
    setError('')
    try {
      const res = await fetch(`${BROWSE_BASE}/databases`)
      if (!res.ok) throw new Error('bad response')
      const data: { databases?: { name: string }[]; error?: string } = await res.json()
      const names = (data.databases || []).map((d) => d.name).filter(Boolean)
      if (names.length === 0) {
        throw new Error(
          data.error ||
            'No databases found on this machine. The agent queries localhost and localhost\\SQLEXPRESS via sqlcmd.',
        )
      }
      setDbs(names)
      setManual(false)
    } catch (e) {
      setError(
        e instanceof Error ? e.message : 'Could not reach the agent database discovery.',
      )
      setDbs(null)
    } finally {
      setIsLoading(false)
    }
  }, [])

  useEffect(() => {
    void discover()
  }, [discover])

  const showSelect = dbs !== null && !manual
  const inputCls =
    'w-full h-9 px-3 rounded-lg border border-[#e9edff] bg-[#f9f9ff] font-mono text-[13px] text-[#141b2b] focus:outline-none focus:ring-2 focus:ring-[#2563eb]/20 focus:border-[#2563eb] transition-all'

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-between">
        <label className="block text-[13px] font-medium text-[#434655]">Database Name *</label>
        <button
          type="button"
          onClick={() => void discover()}
          disabled={isLoading}
          className="flex items-center gap-1.5 text-[12px] font-medium text-[#004ac6] hover:underline disabled:opacity-50"
        >
          {isLoading ? <Loader2 size={13} className="animate-spin" /> : <RefreshCw size={13} />}
          {dbs === null ? 'Detect databases' : 'Re-detect'}
        </button>
      </div>

      {showSelect ? (
        <div className="relative">
          <Database
            size={15}
            className="absolute left-3 top-1/2 -translate-y-1/2 text-[#737686] pointer-events-none"
          />
          <select
            value={dbs.includes(value) ? value : value === '' ? '' : '__manual'}
            onChange={(e) => {
              if (e.target.value === '__manual') {
                setManual(true)
              } else {
                onChange(e.target.value)
              }
            }}
            className={`${inputCls} pl-9 appearance-none cursor-pointer`}
          >
            <option value="">Select a database…</option>
            {dbs.map((d) => (
              <option key={d} value={d}>
                {d}
              </option>
            ))}
            <option value="__manual">Type manually…</option>
          </select>
        </div>
      ) : (
        <input
          type="text"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder="ERP_Production"
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

      {isLoading && dbs === null && !error && (
        <p className="text-[12px] text-[#737686]">Detecting databases on this machine…</p>
      )}
      {error && (
        <p className="text-[12px] text-[#737686]">
          {error} You can still type the name manually — required for remote agents.
        </p>
      )}
    </div>
  )
}

export default SqlDatabasePicker
