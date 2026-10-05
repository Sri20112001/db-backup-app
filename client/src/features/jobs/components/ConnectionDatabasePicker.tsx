import { useState } from 'react'
import { connectionsApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import type { Agent, DatabaseConnection } from '@/types'
import { formatBytes } from '@/utils/format'
import { Loader2, RefreshCw } from 'lucide-react'

interface Props {
  connections: DatabaseConnection[]
  connectionId: string
  database: string
  agent: Agent | undefined
  onConnectionChange: (id: string) => void
  onDatabaseChange: (name: string) => void
  onRefreshConnections: () => void
}

// Connection-first database picker: the job references a saved connection
// (never credentials). Database options come from the cached list; Refresh
// re-queries the server without exposing any secret.
const ConnectionDatabasePicker = ({
  connections, connectionId, database, agent, onConnectionChange, onDatabaseChange, onRefreshConnections,
}: Props) => {
  const { currentOrg } = useAuthStore()
  const { addToast } = useUIStore()
  const [refreshing, setRefreshing] = useState(false)
  const [manual, setManual] = useState(false)

  const selected = connections.find((c) => c.id === connectionId)
  const dbs = selected?.databases ?? []
  const dbNames = dbs.map((d) => d.name)

  const refresh = async () => {
    if (!currentOrg || !selected) return
    setRefreshing(true)
    try {
      const res = await connectionsApi.databases(currentOrg.id, selected.id)
      onRefreshConnections()
      if (res.databases.length === 0) {
        addToast('error', 'No databases returned. You can type the name manually.')
        setManual(true)
      }
    } catch {
      addToast('error', 'Could not refresh databases — type the name manually if needed.')
      setManual(true)
    } finally {
      setRefreshing(false)
    }
  }

  const inputCls = 'w-full h-9 px-3 rounded-lg border border-surface-variant bg-surface-container-low text-[13px] text-on-surface focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all'

  if (connections.length === 0) {
    return (
      <div className="flex flex-col gap-2">
        <label className="block text-[13px] font-medium text-on-surface-variant">Database Name *</label>
        <input type="text" value={database} onChange={(e) => onDatabaseChange(e.target.value)} placeholder="mydb" className={`${inputCls} font-mono`} />
        <p className="text-[11px] text-outline">
          No saved connections of this type yet — create one under Connections for reusable, credential-free jobs.
          This legacy path uses the agent's environment credentials.
        </p>
      </div>
    )
  }

  const agentOffline = agent?.status !== 'ONLINE'

  return (
    <div className="flex flex-col gap-3">
      <div>
        <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Connection *</label>
        <select value={connectionId} onChange={(e) => { onConnectionChange(e.target.value); setManual(false) }} className={inputCls}>
          <option value="">Select a connection…</option>
          {connections.map((c) => (
            <option key={c.id} value={c.id}>{c.name} — {c.host}:{c.port}</option>
          ))}
        </select>
        {selected && (
          <p className="text-[11px] text-outline mt-1">
            {selected.username}@{selected.host}:{selected.port}
            {agent ? ` · Agent: ${agent.name} (${agent.status})` : ''}
            {selected.status === 'ERROR' ? ' · Last check failed' : ''}
          </p>
        )}
      </div>
      <div>
        <div className="flex items-center justify-between mb-1.5">
          <label className="block text-[13px] font-medium text-on-surface-variant">Database *</label>
          {selected && (
            <button type="button" onClick={() => void refresh()} disabled={refreshing}
              className="flex items-center gap-1 text-[12px] font-medium text-primary hover:underline disabled:opacity-50">
              {refreshing ? <Loader2 size={12} className="animate-spin" /> : <RefreshCw size={12} />} Refresh
            </button>
          )}
        </div>
        {!selected ? (
          <p className="text-[12px] text-outline">Select a connection to list databases.</p>
        ) : agentOffline ? (
          <div className="flex flex-col gap-2">
            <input type="text" value={database} onChange={(e) => onDatabaseChange(e.target.value)} placeholder="mydb" disabled
              className={`${inputCls} font-mono opacity-50`} />
            <p className="text-[12px] text-error">Agent {agent?.name ?? ''} is offline — database selection is disabled until it reconnects.</p>
          </div>
        ) : !manual && dbs.length > 0 ? (
          <div className="flex flex-col gap-2">
            <select
              value={dbNames.includes(database) ? database : database === '' ? '' : '__manual'}
              onChange={(e) => {
                if (e.target.value === '__manual') setManual(true)
                else onDatabaseChange(e.target.value)
              }}
              className={`${inputCls} font-mono appearance-none cursor-pointer`}
            >
              <option value="">Select a database…</option>
              {dbs.map((d) => (
                <option key={d.name} value={d.name}>
                  {d.name}{d.size_bytes >= 0 ? ` (${formatBytes(d.size_bytes)})` : ''}{d.table_count >= 0 ? ` · ${d.table_count} tables` : ''}
                </option>
              ))}
              <option value="__manual">Type manually…</option>
            </select>
          </div>
        ) : (
          <div className="flex flex-col gap-2">
            <input type="text" value={database} onChange={(e) => onDatabaseChange(e.target.value)} placeholder="mydb" className={`${inputCls} font-mono`} />
            {dbs.length > 0 && (
              <button type="button" onClick={() => setManual(false)} className="self-start text-[12px] font-medium text-primary hover:underline">
                ← Back to discovered list
              </button>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

export default ConnectionDatabasePicker
