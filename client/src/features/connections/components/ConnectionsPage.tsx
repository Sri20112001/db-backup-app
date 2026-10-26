import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { connectionsApi, agentApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import type { Agent, ConnectionType, DatabaseConnection } from '@/types'
import EmptyState from '@/components/EmptyState'
import ConfirmDialog from '@/components/ConfirmDialog'
import Pagination from '@/components/Pagination'
import SearchInput from '@/components/ui/SearchInput'
import SortSelect from '@/components/ui/SortSelect'
import { useDebouncedValue } from '@/hooks/useDebouncedValue'
import { Page, PageHeader } from '@/components/Page'
import { DbTypeIcon } from '@/components/ui/SourceIcons'
import { typeLabel } from '@/components/ui/typeLabel'
import { Database, Plus, Pencil, Trash2, RefreshCw, Loader2, Server, CheckCircle2, XCircle, HelpCircle, Clock3 } from 'lucide-react'
import { badgeToneClass, connectionBadge } from '@/utils/connectionStatus'
import ConnectionModal from './ConnectionModal'

const TYPE_FILTERS: { label: string; value: ConnectionType | '' }[] = [
  { label: 'All', value: '' },
  { label: 'PostgreSQL', value: 'POSTGRES' },
  { label: 'MongoDB', value: 'MONGODB' },
  { label: 'SQL Server', value: 'MSSQL' },
]

const statusIcon = (badge: ReturnType<typeof connectionBadge>) => {
  if (badge.tone === 'ok') return <CheckCircle2 size={14} className="text-primary" />
  if (badge.tone === 'error') return <XCircle size={14} className="text-error" />
  // Muted = the agent can't reach the database right now (pending/offline/
  // revoked) or nothing has been checked yet — a clock, never a green check.
  return badge.label === 'Not checked yet'
    ? <HelpCircle size={14} className="text-on-surface-variant" />
    : <Clock3 size={14} className="text-on-surface-variant" />
}

const PER_PAGE = 8

// Maps the UI sort options onto the server's allowlisted column + direction.
const SORT_PARAMS: Record<string, { sort: string; order: 'asc' | 'desc' }> = {
  'name-asc': { sort: 'name', order: 'asc' },
  'name-desc': { sort: 'name', order: 'desc' },
  'status': { sort: 'status', order: 'asc' },
}

const ConnectionsPage = () => {
  const { currentOrg } = useAuthStore()
  const { addToast } = useUIStore()
  const navigate = useNavigate()
  const [connections, setConnections] = useState<DatabaseConnection[]>([])
  const [agents, setAgents] = useState<Agent[]>([])
  const [search, setSearch] = useState('')
  const [typeFilter, setTypeFilter] = useState<ConnectionType | ''>('')
  const [sort, setSort] = useState('name-asc')
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [isLoading, setIsLoading] = useState(true)
  const [showModal, setShowModal] = useState(false)
  const [editing, setEditing] = useState<DatabaseConnection | null>(null)
  const [deleteId, setDeleteId] = useState<string | null>(null)
  const [testingId, setTestingId] = useState<string | null>(null)
  const [refreshingId, setRefreshingId] = useState<string | null>(null)

  const debouncedSearch = useDebouncedValue(search, 300)

  // Reloads the current page of connections using the active filters.
  const load = () => {
    if (!currentOrg) return
    setIsLoading(true)
    const sortParams = SORT_PARAMS[sort] ?? SORT_PARAMS['name-asc']
    connectionsApi.list(currentOrg.id, {
      page,
      limit: PER_PAGE,
      search: debouncedSearch || undefined,
      type: typeFilter || undefined,
      sort: sortParams.sort,
      order: sortParams.order,
    })
      .then((res) => { setConnections(res.data); setTotal(res.total) })
      .catch(() => addToast('error', 'Failed to load connections'))
      .finally(() => setIsLoading(false))
  }

  // The agent roster feeds the create/edit modal; loaded once per org.
  const loadAgents = () => {
    if (!currentOrg) return
    agentApi.list(currentOrg.id, { limit: 200 })
      .then((res) => setAgents(res.data))
      .catch(() => { /* the modal still works without the roster */ })
  }

  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => { load() }, [currentOrg, page, debouncedSearch, typeFilter, sort])
  useEffect(() => { setPage(1) }, [debouncedSearch, typeFilter, sort])
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => { loadAgents() }, [currentOrg])

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
        <SortSelect
          value={sort}
          onChange={setSort}
          options={[
            { label: 'Name A–Z', value: 'name-asc' },
            { label: 'Name Z–A', value: 'name-desc' },
            { label: 'Status', value: 'status' },
          ]}
        />
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
        <div className="flex items-center justify-center py-16 text-on-surface-variant"><Loader2 size={20} className="animate-spin" /></div>
      ) : total === 0 ? (
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
          {connections.map((c) => {
            const badge = connectionBadge(c)
            return (
            <div key={c.id} onClick={() => navigate(`/connections/${c.id}`)}
              className="bg-surface-container-lowest border border-surface-variant rounded-xl p-4 flex flex-col gap-2.5 shadow-sm cursor-pointer hover:shadow-md transition-shadow">
              <div className="flex items-start justify-between gap-2">
                <div className="flex items-center gap-3 min-w-0">
                  <span className="shrink-0 w-9 h-9 rounded-lg bg-surface-container-high/60 border border-surface-variant flex items-center justify-center">
                    <DbTypeIcon type={c.type} size={18} />
                  </span>
                  <div className="min-w-0">
                    <p className="text-[14px] font-semibold text-on-surface truncate">{c.name}</p>
                    <p className="font-mono text-[11px] text-on-surface-variant">{typeLabel(c.type)} · {c.host}:{c.port}</p>
                  </div>
                </div>
                <span
                  title={badge.hint}
                  className={`flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-medium shrink-0 ${badgeToneClass[badge.tone]}`}
                >
                  {statusIcon(badge)}{badge.label}
                </span>
              </div>
              <div className="flex items-center gap-1.5 text-[12px] text-on-surface-variant">
                <Server size={13} className="text-on-surface-variant" />
                <span className="truncate">Agent: {c.agent?.name ?? agentName(c.agent_id)}</span>
                {/* Roster cross-check: the chip can only reflect the payload
                    we fetched, so still flag an offline agent if the roster
                    disagrees with a "trusted" (agent-online) badge. */}
                {badge.trusted && !agentOnline(c.agent_id) && (
                  <span className="text-error font-medium">· Offline</span>
                )}
                <span className="text-on-surface-variant">· {c.database_count} databases</span>
              </div>
              {c.last_checked_at && (
                <p className="text-[11px] text-on-surface-variant">Last checked: {new Date(c.last_checked_at).toLocaleString()}</p>
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
            )
          })}
        </div>
      )}

      <Pagination
        page={page}
        totalPages={Math.max(1, Math.ceil(total / PER_PAGE))}
        total={total}
        perPage={PER_PAGE}
        onPage={setPage}
      />

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
