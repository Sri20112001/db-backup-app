import { useEffect, useState } from 'react'
import { connectionsApi, agentApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import type { Agent, ConnectionType, DatabaseConnection } from '@/types'
import EmptyState from '@/components/EmptyState'
import ConfirmDialog from '@/components/ConfirmDialog'
import SearchInput from '@/components/ui/SearchInput'
import { Page, PageHeader } from '@/components/Page'
import { Database, Plus, Pencil, Trash2, RefreshCw, Loader2, Server, CheckCircle2, XCircle, HelpCircle } from 'lucide-react'
import ConnectionModal from './ConnectionModal'
import ConnectionViewModal from './ConnectionViewModal'

const TYPE_FILTERS: { label: string; value: ConnectionType | '' }[] = [
  { label: 'All', value: '' },
  { label: 'PostgreSQL', value: 'POSTGRES' },
  { label: 'MongoDB', value: 'MONGODB' },
  { label: 'SQL Server', value: 'MSSQL' },
]

const statusIcon = (s: string) => {
  if (s === 'CONNECTED') return <CheckCircle2 size={14} className="text-primary" />
  if (s === 'ERROR') return <XCircle size={14} className="text-error" />
  return <HelpCircle size={14} className="text-outline" />
}

const ConnectionsPage = () => {
  const { currentOrg } = useAuthStore()
  const { addToast } = useUIStore()
  const [connections, setConnections] = useState<DatabaseConnection[]>([])
  const [agents, setAgents] = useState<Agent[]>([])
  const [search, setSearch] = useState('')
  const [typeFilter, setTypeFilter] = useState<ConnectionType | ''>('')
  const [isLoading, setIsLoading] = useState(true)
  const [showModal, setShowModal] = useState(false)
  const [editing, setEditing] = useState<DatabaseConnection | null>(null)
  const [viewing, setViewing] = useState<DatabaseConnection | null>(null)
  const [deleteId, setDeleteId] = useState<string | null>(null)
  const [testingId, setTestingId] = useState<string | null>(null)
  const [refreshingId, setRefreshingId] = useState<string | null>(null)

  const load = () => {
    if (!currentOrg) return
    setIsLoading(true)
    Promise.all([connectionsApi.list(currentOrg.id), agentApi.list(currentOrg.id)])
      .then(([c, a]) => { setConnections(c); setAgents(a) })
      .catch(() => addToast('error', 'Failed to load connections'))
      .finally(() => setIsLoading(false))
  }

  useEffect(() => { load() }, [currentOrg])

  const agentName = (id: string) => agents.find((a) => a.id === id)?.name ?? '—'
  const agentOnline = (id: string) => agents.find((a) => a.id === id)?.status === 'ONLINE'

  const handleTest = async (id: string) => {
    if (!currentOrg) return
    setTestingId(id)
    try {
      await connectionsApi.test(currentOrg.id, id)
      addToast('success', 'Connection verified')
      load()
    } catch (e) {
      addToast('error', e instanceof Error ? e.message : 'Connection test failed')
      load()
    } finally {
      setTestingId(null)
    }
  }

  const handleRefreshDbs = async (id: string) => {
    if (!currentOrg) return
    setRefreshingId(id)
    try {
      const res = await connectionsApi.databases(currentOrg.id, id)
      addToast('success', res.cached ? `Using cached list (${res.databases.length} databases)` : `Found ${res.databases.length} databases`)
      load()
    } catch {
      addToast('error', 'Failed to refresh databases')
    } finally {
      setRefreshingId(null)
    }
  }

  const handleDelete = async () => {
    if (!currentOrg || !deleteId) return
    try {
      await connectionsApi.delete(currentOrg.id, deleteId)
      addToast('success', 'Connection deleted')
      setDeleteId(null)
      load()
    } catch (e) {
      addToast('error', e instanceof Error ? e.message : 'Delete failed (it may be used by a job)')
    }
  }

  // Refresh scoped to the open detail view: updates the modal instantly,
  // then reloads the list in the background.
  const handleViewRefresh = async () => {
    if (!currentOrg || !viewing) return
    setRefreshingId(viewing.id)
    try {
      const res = await connectionsApi.databases(currentOrg.id, viewing.id)
      setViewing({ ...viewing, databases: res.databases, database_count: res.databases.length })
      load()
    } catch {
      addToast('error', 'Failed to refresh databases')
    } finally {
      setRefreshingId(null)
    }
  }

  const filtered = connections.filter((c) => {
    const matchType = !typeFilter || c.type === typeFilter
    const q = search.toLowerCase()
    return matchType && (!search || c.name.toLowerCase().includes(q) || c.host.toLowerCase().includes(q))
  })

  return (
    <Page>
      <PageHeader
        title="Connections"
        description="Configure database access once, then reuse it across backup jobs."
        actions={
          <button
            type="button"
            onClick={() => { setEditing(null); setShowModal(true) }}
            className="flex items-center gap-1.5 px-4 h-9 rounded-lg bg-primary text-on-primary text-[13px] font-semibold hover:opacity-90"
          >
            <Plus size={15} /> Add Connection
          </button>
        }
      />

      <div className="flex items-center gap-2 flex-wrap">
        <SearchInput value={search} onChange={(e) => setSearch(e.target.value)} placeholder="Search connections…" />
        <div className="flex gap-1.5">
          {TYPE_FILTERS.map((f) => (
            <button
              key={f.label}
              type="button"
              onClick={() => setTypeFilter(f.value)}
              className={`px-3 h-8 rounded-lg text-[12px] font-medium ${typeFilter === f.value ? 'bg-primary text-on-primary' : 'bg-surface-container-low text-on-surface-variant hover:bg-surface-container-high'}`}
            >
              {f.label}
            </button>
          ))}
        </div>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-16 text-outline"><Loader2 size={20} className="animate-spin" /></div>
      ) : filtered.length === 0 ? (
        <EmptyState
          icon={Database}
          title="No connections yet"
          description="Add a database connection once — backup jobs will reference it instead of asking for passwords."
          action={
            <button
              type="button"
              onClick={() => { setEditing(null); setShowModal(true) }}
              className="px-4 h-9 rounded-lg bg-primary text-on-primary text-[13px] font-semibold hover:opacity-90"
            >
              Add Connection
            </button>
          }
        />
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          {filtered.map((c) => (
            <div key={c.id} onClick={() => setViewing(c)}
              className="bg-surface-container-lowest border border-surface-variant rounded-xl p-4 flex flex-col gap-2.5 shadow-sm cursor-pointer hover:shadow-md transition-shadow">
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0">
                  <p className="text-[14px] font-semibold text-on-surface truncate">{c.name}</p>
                  <p className="font-mono text-[11px] text-outline">{c.type} · {c.host}:{c.port}</p>
                </div>
                <span className="flex items-center gap-1 text-[11px] font-medium text-on-surface-variant shrink-0">
                  {statusIcon(c.status)}{c.status === 'CONNECTED' ? 'Connected' : c.status === 'ERROR' ? 'Error' : 'Unknown'}
                </span>
              </div>
              <div className="flex items-center gap-1.5 text-[12px] text-on-surface-variant">
                <Server size={13} className="text-outline" />
                <span className="truncate">Agent: {c.agent?.name ?? agentName(c.agent_id)}</span>
                {!agentOnline(c.agent_id) && <span className="text-error font-medium">· Offline</span>}
                <span className="text-outline">· {c.database_count} databases</span>
              </div>
              {c.last_checked_at && (
                <p className="text-[11px] text-outline">Last checked: {new Date(c.last_checked_at).toLocaleString()}</p>
              )}
              <div className="flex items-center gap-1.5 pt-1 flex-wrap" onClick={(e) => e.stopPropagation()}>
                <button type="button" onClick={() => void handleTest(c.id)} disabled={testingId === c.id}
                  className="px-2.5 h-7 rounded-lg text-[12px] font-medium bg-surface-container-low border border-surface-variant hover:bg-surface-container-high disabled:opacity-50 flex items-center gap-1">
                  {testingId === c.id ? <Loader2 size={12} className="animate-spin" /> : null} Test
                </button>
                <button type="button" onClick={() => void handleRefreshDbs(c.id)} disabled={refreshingId === c.id}
                  className="px-2.5 h-7 rounded-lg text-[12px] font-medium bg-surface-container-low border border-surface-variant hover:bg-surface-container-high disabled:opacity-50 flex items-center gap-1">
                  {refreshingId === c.id ? <Loader2 size={12} className="animate-spin" /> : <RefreshCw size={12} />} Databases
                </button>
                <button type="button" onClick={() => { setEditing(c); setShowModal(true) }}
                  className="px-2.5 h-7 rounded-lg text-[12px] font-medium bg-surface-container-low border border-surface-variant hover:bg-surface-container-high flex items-center gap-1">
                  <Pencil size={12} /> Edit
                </button>
                <button type="button" onClick={() => setDeleteId(c.id)}
                  className="px-2.5 h-7 rounded-lg text-[12px] font-medium text-error bg-surface-container-low border border-surface-variant hover:bg-error-container/30 flex items-center gap-1">
                  <Trash2 size={12} /> Delete
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {viewing && (
        <ConnectionViewModal
          connection={viewing}
          agentName={viewing.agent?.name ?? agentName(viewing.agent_id)}
          agentOnline={agentOnline(viewing.agent_id)}
          refreshing={refreshingId === viewing.id}
          onRefresh={() => void handleViewRefresh()}
          onEdit={() => { const v = viewing; setViewing(null); setEditing(v); setShowModal(true) }}
          onClose={() => setViewing(null)}
        />
      )}
      {showModal && (
        <ConnectionModal
          agents={agents}
          editing={editing}
          onClose={() => { setShowModal(false); setEditing(null) }}
          onSaved={() => { setShowModal(false); setEditing(null); load() }}
        />
      )}
      {deleteId !== null && (
        <ConfirmDialog
          title="Delete connection?"
          message="The saved credential is removed. Jobs using this connection must be reassigned first."
          onCancel={() => setDeleteId(null)}
          onConfirm={() => void handleDelete()}
        />
      )}
    </Page>
  )
}

export default ConnectionsPage
