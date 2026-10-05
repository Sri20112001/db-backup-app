import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { agentApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import type { Agent, AgentLifecycle, EnrollmentTokenResult } from '@/types'
import StatusBadge from '@/components/StatusBadge'
import EmptyState from '@/components/EmptyState'
import ConfirmDialog from '@/components/ConfirmDialog'
import Pagination from '@/components/Pagination'
import { usePagination } from '@/hooks/usePagination'
import { useRealtimeStore } from '@/stores/realtimeStore'
import { formatRelative, formatDate } from '@/utils/format'
import { Plus, Server, Trash2, Copy, Check, Loader2, Ban } from 'lucide-react'
import SearchInput from '@/components/ui/SearchInput'
import SortSelect from '@/components/ui/SortSelect'
import { Page, PageHeader } from '@/components/Page'

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

const AgentsPage = () => {
  const { currentOrg } = useAuthStore()
  const { addToast } = useUIStore()
  const navigate = useNavigate()
  const [agents, setAgents] = useState<Agent[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [showRegister, setShowRegister] = useState(false)
  const [agentName, setAgentName] = useState('')
  const [enroll, setEnroll] = useState<EnrollmentTokenResult | null>(null)
  const [isGenerating, setIsGenerating] = useState(false)
  const [copied, setCopied] = useState(false)
  const [deleteId, setDeleteId] = useState<string | null>(null)
  const [revokeId, setRevokeId] = useState<string | null>(null)
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<AgentLifecycle | ''>('')
  const [sort, setSort] = useState('name-asc')
  const filtered = agents.filter((a) => {
    const matchStatus = !statusFilter || lifecycleOf(a) === statusFilter
    const matchSearch = !search || a.name.toLowerCase().includes(search.toLowerCase()) ||
      (a.machine_name || '').toLowerCase().includes(search.toLowerCase())
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

  const openAddAgent = () => {
    setAgentName('')
    setEnroll(null)
    setShowRegister(true)
  }

  const handleGenerateToken = async () => {
    if (!currentOrg) return
    setIsGenerating(true)
    try {
      const token = await agentApi.enrollmentToken(currentOrg.id, { name: agentName.trim() || undefined })
      setEnroll(token)
    } catch { addToast('error', 'Failed to generate enrollment token') }
    finally { setIsGenerating(false) }
  }

  const handleCopy = () => {
    if (!enroll) return
    const text = enroll.enrollment_token
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text)
    } else {
      // Fallback for insecure contexts (like exposed IPs without HTTPS)
      const textArea = document.createElement('textarea')
      textArea.value = text
      textArea.style.position = 'fixed'
      textArea.style.left = '-9999px'
      document.body.appendChild(textArea)
      textArea.focus()
      textArea.select()
      try {
        document.execCommand('copy')
      } catch (err) {
        console.error('Fallback copy failed', err)
      }
      document.body.removeChild(textArea)
    }
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
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
    setEnroll(null)
    setAgentName('')
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
      ) : filtered.length === 0 ? (
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
          {paged.pageItems.map((agent) => (
            <div
              key={agent.id}
              onClick={() => navigate(`/agents/${agent.id}`)}
              className="flex flex-col gap-3 p-5 rounded-xl bg-surface-container-lowest border border-surface-variant shadow-sm hover:shadow-md transition-all cursor-pointer"
            >
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-3">
                  <div className={`w-10 h-10 rounded-lg flex items-center justify-center ${lifecycleOf(agent) === 'ONLINE' ? 'bg-primary-container/40 text-on-primary-container' : 'bg-surface-variant text-outline'}`}>
                    <Server size={20} />
                  </div>
                  <div>
                    <h3 className="text-[15px] font-semibold text-on-surface">{agent.name}</h3>
                    <p className="font-mono text-[11px] text-outline">
                      {[agent.machine_name, [agent.platform, agent.architecture].filter(Boolean).join('/'), agent.version ? `v${agent.version}` : ''].filter(Boolean).join(' · ') || '—'}
                    </p>
                  </div>
                </div>
                <StatusBadge status={lifecycleOf(agent)} size="sm" />
              </div>

              <div className="flex items-center justify-between text-[12px] text-outline">
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
                      className="p-1.5 rounded-lg text-outline hover:text-error hover:bg-error-container/30 transition-colors"
                    >
                      <Ban size={16} />
                    </button>
                  )}
                  <button
                    type="button"
                    onClick={(e) => { e.stopPropagation(); setDeleteId(agent.id) }}
                    className="p-1.5 rounded-lg text-outline hover:text-error hover:bg-error-container/30 transition-colors"
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
        page={paged.page}
        totalPages={paged.totalPages}
        total={paged.total}
        perPage={paged.perPage}
        onPage={paged.setPage}
      />

      {/* Add Agent / enrollment token modal */}
      {showRegister && (
        <div className="fixed inset-0 z-[200] flex items-center justify-center">
          <div className="absolute inset-0 bg-on-surface/30 backdrop-blur-[2px]" onClick={closeRegister} />
          <div className="relative bg-surface-container-lowest rounded-xl shadow-2xl p-6 w-full max-w-md mx-4 border border-surface-variant">
            <h2 className="text-[16px] font-semibold text-on-surface mb-2">Add Agent</h2>
            <p className="text-[13px] text-on-surface-variant mb-4">
              Generate a one-time enrollment token, then run VaultGuard-Agent-Setup on the
              machine and paste it there. The token is for initial enrollment only —
              the agent receives its permanent credential automatically.
            </p>
            {enroll ? (
              <>
                <div className="flex items-center gap-2 p-3 rounded-lg bg-surface-container-low border border-surface-variant mb-2">
                  <code className="flex-1 font-mono text-[12px] text-on-surface break-all">{enroll.enrollment_token}</code>
                  <button
                    type="button"
                    onClick={handleCopy}
                    className="p-1.5 rounded text-on-surface-variant hover:bg-surface-container-high transition-colors shrink-0"
                  >
                    {copied ? <Check size={16} className="text-on-primary-container" /> : <Copy size={16} />}
                  </button>
                </div>
                <p className="text-[12px] text-outline mb-1">
                  Expires: {new Date(enroll.expires_at).toLocaleString()} · Single-use.
                </p>
                <p className="text-[12px] text-outline mb-4">This token will never be shown again. Permanent agent credentials are never displayed.</p>
              </>
            ) : (
              <>
                <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Agent name (optional)</label>
                <input
                  type="text"
                  value={agentName}
                  onChange={(e) => setAgentName(e.target.value)}
                  placeholder="e.g. Office-PC"
                  className="w-full h-9 px-3 rounded-lg border border-surface-variant bg-surface-container-low text-[13px] text-on-surface focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all mb-4"
                />
                <button
                  type="button"
                  onClick={() => void handleGenerateToken()}
                  disabled={isGenerating}
                  className="w-full h-9 rounded-lg bg-primary text-on-primary text-[13px] font-medium hover:bg-primary-container transition-colors disabled:opacity-50 flex items-center justify-center gap-2 mb-2"
                >
                  {isGenerating && <Loader2 size={14} className="animate-spin" />}
                  Generate enrollment token
                </button>
              </>
            )}
            <button
              type="button"
              onClick={closeRegister}
              className="w-full h-9 rounded-lg bg-surface-container-lowest border border-surface-variant text-on-surface text-[13px] font-medium hover:bg-surface-container-low transition-colors"
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
