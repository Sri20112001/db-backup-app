import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { agentApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import type { Agent, AgentStatus } from '@/types'
import StatusBadge from '@/components/StatusBadge'
import EmptyState from '@/components/EmptyState'
import ConfirmDialog from '@/components/ConfirmDialog'
import Pagination from '@/components/Pagination'
import { usePagination } from '@/hooks/usePagination'
import { useRealtimeStore } from '@/stores/realtimeStore'
import { formatRelative } from '@/utils/format'
import { Plus, Server, Trash2, Copy, Check, Loader2 } from 'lucide-react'
import SearchInput from '@/components/ui/SearchInput'
import SortSelect from '@/components/ui/SortSelect'

const AGENT_STATUS_FILTERS: { label: string; value: AgentStatus | '' }[] = [
  { label: 'All', value: '' },
  { label: 'Online', value: 'ONLINE' },
  { label: 'Offline', value: 'OFFLINE' },
]

const AgentsPage = () => {
  const { currentOrg } = useAuthStore()
  const { addToast } = useUIStore()
  const navigate = useNavigate()
  const [agents, setAgents] = useState<Agent[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [showRegister, setShowRegister] = useState(false)
  const [regToken, setRegToken] = useState<{ agent_id: string; registration_key: string } | null>(null)
  const [copied, setCopied] = useState(false)
  const [deleteId, setDeleteId] = useState<string | null>(null)
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<AgentStatus | ''>('')
  const [sort, setSort] = useState('name-asc')
  const filtered = agents.filter((a) => {
    const matchStatus = !statusFilter || a.status === statusFilter
    const matchSearch = !search || a.name.toLowerCase().includes(search.toLowerCase())
    return matchStatus && matchSearch
  })
  const sorted = [...filtered].sort((a, b) => {
    switch (sort) {
      case 'name-desc':
        return b.name.localeCompare(a.name)
      case 'recently-seen':
        return (b.last_seen_at ?? '').localeCompare(a.last_seen_at ?? '')
      default:
        return a.name.localeCompare(b.name)
    }
  })
  const paged = usePagination(sorted, 6, `${search}|${statusFilter}|${sort}|${currentOrg?.id ?? ''}`)
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

  useEffect(() => {
    loadAgents()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [agentsSeq])

  const loadAgents = () => {
    if (!currentOrg) return
    setIsLoading(true)
    agentApi.list(currentOrg.id)
      .then(setAgents)
      .catch(() => addToast('error', 'Failed to load agents'))
      .finally(() => setIsLoading(false))
  }

  useEffect(() => { loadAgents() }, [currentOrg])

  const handleGenerateToken = async () => {
    if (!currentOrg) return
    try {
      const token = await agentApi.generateToken(currentOrg.id)
      setRegToken(token)
      setShowRegister(true)
    } catch { addToast('error', 'Failed to generate token') }
  }

  const handleCopy = () => {
    if (!regToken) return
    navigator.clipboard.writeText(regToken.registration_key)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
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
    <div className="h-full min-h-0 flex flex-col gap-4">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-[20px] font-semibold text-on-surface tracking-tight">Agents & Machines</h1>
          <p className="text-[12px] text-on-surface-variant mt-0.5">Windows backup agents installed on your infrastructure</p>
        </div>
        <button
          type="button"
          onClick={handleGenerateToken}
          className="flex items-center gap-2 px-4 h-9 rounded-lg bg-primary text-on-primary text-[13px] font-medium hover:bg-primary-container transition-colors shadow-sm"
        >
          <Plus size={16} />
          Register Agent
        </button>
      </div>

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
      ) : filtered.length === 0 ? (
        <EmptyState
          icon={Server}
          title="No agents registered"
          description="Install the VaultGuard agent on your Windows machines and register them here."
          action={
            <button type="button" onClick={handleGenerateToken} className="px-4 h-9 rounded-lg bg-primary text-on-primary text-[13px] font-medium hover:bg-primary-container transition-colors">
              Register First Agent
            </button>
          }
        />
      ) : (
        <div className="flex-1 min-h-0 overflow-y-auto grid grid-cols-1 md:grid-cols-2 gap-4 content-start pr-0.5">
          {paged.pageItems.map((agent) => (
            <div
              key={agent.id}
              onClick={() => navigate(`/agents/${agent.id}`)}
              className="flex flex-col gap-3 p-5 rounded-xl bg-surface-container-lowest border border-surface-variant shadow-sm hover:shadow-md transition-all cursor-pointer"
            >
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-3">
                  <div className={`w-10 h-10 rounded-lg flex items-center justify-center ${agent.status === 'ONLINE' ? 'bg-primary-container/40 text-on-primary-container' : 'bg-surface-variant text-outline'}`}>
                    <Server size={20} />
                  </div>
                  <div>
                    <h3 className="text-[15px] font-semibold text-on-surface">{agent.name}</h3>
                    <p className="font-mono text-[11px] text-outline">v{agent.version || '—'}</p>
                  </div>
                </div>
                <StatusBadge status={agent.status} size="sm" />
              </div>

              <div className="flex items-center justify-between text-[12px] text-outline">
                <span>Last seen: {formatRelative(agent.last_seen_at)}</span>
                <button
                  type="button"
                  onClick={(e) => { e.stopPropagation(); setDeleteId(agent.id) }}
                  className="p-1.5 rounded-lg text-outline hover:text-error hover:bg-error-container/30 transition-colors"
                >
                  <Trash2 size={16} />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      <Pagination
        page={paged.page}
        totalPages={paged.totalPages}
        total={paged.total}
        perPage={paged.perPage}
        onPage={paged.setPage}
      />

      {/* Register modal */}
      {showRegister && (
        <div className="fixed inset-0 z-[200] flex items-center justify-center">
          <div className="absolute inset-0 bg-on-surface/30 backdrop-blur-[2px]" onClick={() => { setShowRegister(false); setRegToken(null); loadAgents() }} />
          <div className="relative bg-surface-container-lowest rounded-xl shadow-2xl p-6 w-full max-w-md mx-4 border border-surface-variant">
            <h2 className="text-[16px] font-semibold text-on-surface mb-2">Register New Agent</h2>
            <p className="text-[13px] text-on-surface-variant mb-4">
              Run the VaultGuard agent installer on your Windows machine and enter this registration key when prompted.
            </p>
            {regToken ? (
              <>
                <div className="flex items-center gap-2 p-3 rounded-lg bg-surface-container-low border border-surface-variant mb-4">
                  <code className="flex-1 font-mono text-[12px] text-on-surface break-all">{regToken.registration_key}</code>
                  <button
                    type="button"
                    onClick={handleCopy}
                    className="p-1.5 rounded text-on-surface-variant hover:bg-surface-container-high transition-colors shrink-0"
                  >
                    {copied ? <Check size={16} className="text-on-primary-container" /> : <Copy size={16} />}
                  </button>
                </div>
                <p className="text-[12px] text-outline mb-4">This key can only be used once. Keep it secure.</p>
              </>
            ) : (
              <div className="flex items-center justify-center py-8">
                <Loader2 size={32} className="text-[#c3c6d7] animate-spin" />
              </div>
            )}
            <button
              type="button"
              onClick={() => { setShowRegister(false); setRegToken(null); loadAgents() }}
              className="w-full h-9 rounded-lg bg-primary text-on-primary text-[13px] font-medium hover:bg-primary-container transition-colors"
            >
              Done
            </button>
          </div>
        </div>
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
    </div>
  )
}

export default AgentsPage
