import { useCallback, useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { agentApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import type { Agent, AgentLifecycle } from '@/types'
import StatusBadge from '@/components/StatusBadge'
import EmptyState from '@/components/EmptyState'
import ConfirmDialog from '@/components/ConfirmDialog'
import Pagination from '@/components/Pagination'
import { useDebouncedValue } from '@/hooks/useDebouncedValue'
import { useRealtimeStore } from '@/store/realtimeStore'
import { formatRelative, formatDate } from '@/utils/format'
import { Plus, Server, Trash2, Ban } from 'lucide-react'
import SearchInput from '@/components/ui/SearchInput'
import SortSelect from '@/components/ui/SortSelect'
import { Page, PageHeader } from '@/components/Page'
import AgentEnrollmentPanel from './AgentEnrollmentPanel'

const AGENT_STATUS_FILTERS: { label: string; value: AgentLifecycle | '' }[] = [
  { label: 'All', value: '' },
  { label: 'Online', value: 'ONLINE' },
  { label: 'Offline', value: 'OFFLINE' },
  { label: 'Pending', value: 'PENDING' },
  { label: 'Revoked', value: 'REVOKED' },
]

// lifecycleOf prefers the server-computed lifecycle, falling back to the raw
// ONLINE/OFFLINE flag for older servers.
const lifecycleOf = (a: Agent): AgentLifecycle =>
  a.lifecycle ?? (a.status === 'ONLINE' ? 'ONLINE' : 'OFFLINE')

const PER_PAGE = 6

// Maps the UI sort options onto the server's allowlisted column + direction.
const SORT_PARAMS: Record<string, { sort: string; order: 'asc' | 'desc' }> = {
  'name-asc': { sort: 'name', order: 'asc' },
  'name-desc': { sort: 'name', order: 'desc' },
  'recently-seen': { sort: 'last_seen', order: 'desc' },
}

const AgentsPage = () => {
  const { currentOrg } = useAuthStore()
  const { addToast } = useUIStore()
  const navigate = useNavigate()
  const [agents, setAgents] = useState<Agent[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [showRegister, setShowRegister] = useState(false)
  const [deleteId, setDeleteId] = useState<string | null>(null)
  const [revokeId, setRevokeId] = useState<string | null>(null)
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<AgentLifecycle | ''>('')
  const [sort, setSort] = useState('name-asc')
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)

  const debouncedSearch = useDebouncedValue(search, 300)
  const presence = useRealtimeStore((s) => s.presence)
  const agentsSeq = useRealtimeStore((s) => s.entitySeq.agents)

  // Instant presence flips; full reload on register/remove.
  useEffect(() => {
    if (!presence?.id) return
    setAgents((prev) =>
      prev.map((a) =>
        a.id === presence.id ? { ...a, status: (presence.status as Agent['status']) ?? a.status, name: presence.name ?? a.name } : a,
      ),
    )
  }, [presence])

  // Wrapped so the fetch effects can depend on it; identity changes when any
  // query input changes.
  const loadAgents = useCallback(() => {
    if (!currentOrg) return
    setIsLoading(true)
    const sortParams = SORT_PARAMS[sort] ?? SORT_PARAMS['name-asc']
    agentApi.list(currentOrg.id, {
      page,
      limit: PER_PAGE,
      search: debouncedSearch || undefined,
      lifecycle: statusFilter || undefined,
      sort: sortParams.sort,
      order: sortParams.order,
    })
      .then((res) => {
        setAgents(res.data)
        setTotal(res.total)
      })
      .catch(() => addToast('error', 'Failed to load agents'))
      .finally(() => setIsLoading(false))
  }, [currentOrg, page, debouncedSearch, statusFilter, sort, addToast])

  useEffect(() => {
    loadAgents()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [agentsSeq])

  useEffect(() => { loadAgents() }, [loadAgents])

  // Any filter change resets to the first page.
  useEffect(() => { setPage(1) }, [debouncedSearch, statusFilter, sort])

  const openAddAgent = () => {
    setShowRegister(true)
  }

  const handleRevoke = async () => {
    if (!currentOrg || !revokeId) return
    try {
      await agentApi.revoke(currentOrg.id, revokeId)
      addToast('success', 'Agent revoked — its credential no longer works')
      setRevokeId(null)
      loadAgents()
    } catch { addToast('error', 'Failed to revoke agent') }
  }

  const closeRegister = () => {
    setShowRegister(false)
    loadAgents()
  }

  const handleDelete = async () => {
    if (!currentOrg || !deleteId) return
    try {
      await agentApi.delete(currentOrg.id, deleteId)
      addToast('success', 'Agent removed')
      setDeleteId(null)
      loadAgents()
    } catch { addToast('error', 'Failed to delete agent') }
  }

  return (
    <Page>
      <PageHeader
        title="Agents & Machines"
        description="Windows backup agents installed on your infrastructure"
        actions={
          <button
            type="button"
            onClick={openAddAgent}
            className="flex items-center gap-2 px-4 h-9 rounded-lg bg-primary text-on-primary text-[13px] font-medium hover:bg-primary-container transition-colors shadow-sm"
          >
            <Plus size={16} />
            Add Agent
          </button>
        }
      />

      {/* Search + filters */}
      <div className="flex flex-col lg:flex-row lg:items-center gap-3">
        <SearchInput
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Search agents by name…"
          className="lg:max-w-xs"
        />
        <div className="flex items-center gap-1.5 overflow-x-auto">
          <SortSelect
            value={sort}
            onChange={setSort}
            options={[
              { label: 'Name A–Z', value: 'name-asc' },
              { label: 'Name Z–A', value: 'name-desc' },
              { label: 'Recently seen', value: 'recently-seen' },
            ]}
          />
          {AGENT_STATUS_FILTERS.map((f) => (
            <button
              key={f.value}
              type="button"
              onClick={() => setStatusFilter(f.value)}
              className={`px-3 h-8 rounded-lg text-[12px] font-medium whitespace-nowrap transition-colors ${
                statusFilter === f.value
                  ? 'bg-primary text-on-primary shadow-sm'
                  : 'bg-surface-container-low text-on-surface-variant hover:bg-surface-container-high'
              }`}
            >
              {f.label}
            </button>
          ))}
        </div>
      </div>

      {isLoading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {[...Array(4)].map((_, i) => (
            <div key={i} className="animate-pulse p-5 rounded-xl bg-surface-container-lowest border border-surface-variant h-36" />
          ))}
        </div>
      ) : total === 0 ? (
        <EmptyState
          icon={Server}
          title="No agents registered"
          description="Install the VaultGuard agent on your Windows machines and register them here."
          action={
            <button type="button" onClick={openAddAgent} className="px-4 h-9 rounded-lg bg-primary text-on-primary text-[13px] font-medium hover:bg-primary-container transition-colors">
              Register First Agent
            </button>
          }
        />
      ) : (
        <div className="flex-1 min-h-0 overflow-y-auto grid grid-cols-1 md:grid-cols-2 gap-4 content-start pr-0.5">
          {agents.map((agent) => (
            <div
              key={agent.id}
              onClick={() => navigate(`/agents/${agent.id}`)}
              className="flex flex-col gap-3 p-5 rounded-xl bg-surface-container-lowest border border-surface-variant shadow-sm hover:shadow-md transition-all cursor-pointer"
            >
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-3">
                  <div className={`w-10 h-10 rounded-lg flex items-center justify-center ${lifecycleOf(agent) === 'ONLINE' ? 'bg-primary-container/40 text-on-primary-container' : 'bg-surface-variant text-on-surface-variant'}`}>
                    <Server size={20} />
                  </div>
                  <div>
                    <h3 className="text-[15px] font-semibold text-on-surface">{agent.name}</h3>
                    <p className="font-mono text-[11px] text-on-surface-variant">
                      {[agent.machine_name, [agent.platform, agent.architecture].filter(Boolean).join('/'), agent.version ? `v${agent.version}` : ''].filter(Boolean).join(' · ') || '—'}
                    </p>
                  </div>
                </div>
                <StatusBadge status={lifecycleOf(agent)} size="sm" />
              </div>

              <div className="flex items-center justify-between text-[12px] text-on-surface-variant">
                <span>
                  Last seen: {formatRelative(agent.last_seen_at)}
                  {agent.installed_at ? ` · Installed: ${formatDate(agent.installed_at)}` : ''}
                </span>
                <span className="flex items-center gap-1">
                  {lifecycleOf(agent) !== 'REVOKED' && lifecycleOf(agent) !== 'PENDING' && (
                    <button
                      type="button"
                      title="Revoke agent"
                      onClick={(e) => { e.stopPropagation(); setRevokeId(agent.id) }}
                      className="p-1.5 rounded-lg text-on-surface-variant hover:text-error hover:bg-error-container/30 transition-colors"
                    >
                      <Ban size={16} />
                    </button>
                  )}
                  <button
                    type="button"
                    onClick={(e) => { e.stopPropagation(); setDeleteId(agent.id) }}
                    className="p-1.5 rounded-lg text-on-surface-variant hover:text-error hover:bg-error-container/30 transition-colors"
                  >
                    <Trash2 size={16} />
                  </button>
                </span>
              </div>
            </div>
          ))}
        </div>
      )}

      <Pagination
        page={page}
        totalPages={Math.max(1, Math.ceil(total / PER_PAGE))}
        total={total}
        perPage={PER_PAGE}
        onPage={setPage}
      />

      {/* Add Agent / enrollment token modal */}
      {showRegister && (
        <div className="fixed inset-0 z-[200] flex items-center justify-center">
          <div className="absolute inset-0 bg-on-surface/30 backdrop-blur-[2px]" onClick={closeRegister} />
          <div className="relative bg-surface-container-lowest rounded-xl shadow-2xl p-6 w-full max-w-md mx-4 border border-surface-variant">
            <h2 className="text-[16px] font-semibold text-on-surface mb-2">Add Agent</h2>
            <AgentEnrollmentPanel />
            <button
              type="button"
              onClick={closeRegister}
              className="mt-4 w-full h-9 rounded-lg bg-surface-container-lowest border border-surface-variant text-on-surface text-[13px] font-medium hover:bg-surface-container-low transition-colors"
            >
               Done
            </button>
          </div>
        </div>
      )}

      {revokeId && (
        <ConfirmDialog
          title="Revoke Agent?"
          message="The agent's credential stops working immediately: heartbeat, job claims and restore claims are rejected. The record is kept for audit. Use this for lost, retired, or compromised machines."
          confirmLabel="Revoke Agent"
          danger
          onConfirm={() => void handleRevoke()}
          onCancel={() => setRevokeId(null)}
        />
      )}

      {deleteId && (
        <ConfirmDialog
          title="Remove Agent?"
          message="This will remove the agent from your organization. Existing backup jobs assigned to this agent will be affected."
          confirmLabel="Remove Agent"
          danger
          onConfirm={handleDelete}
          onCancel={() => setDeleteId(null)}
        />
      )}
    </Page>
  )
}

export default AgentsPage
