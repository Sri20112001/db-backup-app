import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { agentApi, machineApi, jobApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import type { Agent, Machine, BackupJob } from '@/types'
import StatusBadge from '@/components/StatusBadge'
import { Page } from '@/components/Page'
import Pagination from '@/components/Pagination'
import { usePagination } from '@/hooks/usePagination'
import NewJobModal from '@/features/jobs/components/NewJobModal'
import { formatRelative } from '@/utils/format'
import { ChevronRight, Server, Loader2, Monitor, Archive, Pencil, Check, X } from 'lucide-react'

const AgentDetailPage = () => {
  const { id } = useParams<{ id: string }>()
  const { currentOrg } = useAuthStore()
  const { addToast } = useUIStore()
  const navigate = useNavigate()
  const [agent, setAgent] = useState<Agent | null>(null)
  const [machines, setMachines] = useState<Machine[]>([])
  const [jobs, setJobs] = useState<BackupJob[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [showNew, setShowNew] = useState(false)
  const [reloadKey, setReloadKey] = useState(0)
  const [editingName, setEditingName] = useState(false)
  const [draftName, setDraftName] = useState('')
  const [isRenaming, setIsRenaming] = useState(false)
  const pagedMachines = usePagination(machines, 4, id)
  const pagedJobs = usePagination(jobs, 4, id)

  useEffect(() => {
    if (!currentOrg || !id) return
    setIsLoading(true)
    Promise.all([
      agentApi.get(currentOrg.id, id),
      machineApi.list(currentOrg.id),
      jobApi.list(currentOrg.id),
    ])
      .then(([a, m, j]) => {
        setAgent(a)
        setMachines(m.filter((machine) => machine.agent_id === id))
        setJobs(j.filter((job) => job.agent_id === id))
      })
      .catch(() => addToast('error', 'Failed to load agent'))
      .finally(() => setIsLoading(false))
  }, [currentOrg, id, reloadKey])

  const handleRename = async () => {
    if (!currentOrg || !id || !draftName.trim()) return
    setIsRenaming(true)
    try {
      const updated = await agentApi.rename(currentOrg.id, id, draftName.trim())
      setAgent(updated)
      setEditingName(false)
      addToast('success', 'Agent renamed')
    } catch {
      addToast('error', 'Failed to rename agent')
    } finally {
      setIsRenaming(false)
    }
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 size={32} className="text-[#c3c6d7] animate-spin" />
      </div>
    )
  }

  if (!agent) return null

  return (
    <Page scroll>
      {/* Breadcrumb */}
      <div className="flex items-center gap-2 text-[12px] text-on-surface-variant">
        <button type="button" onClick={() => navigate('/agents')} className="hover:text-primary">
          Agents & Machines
        </button>
        <ChevronRight size={14} />
        <span className="text-primary font-medium">{agent.name}</span>
      </div>

      {/* Header card */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 p-5 rounded-xl bg-surface-container-lowest border border-surface-variant shadow-sm">
        <div className="flex items-center gap-4">
          <div className={`w-12 h-12 rounded-xl flex items-center justify-center ${agent.status === 'ONLINE' ? 'bg-primary-container/40 text-on-primary-container' : 'bg-surface-variant text-outline'}`}>
            <Server size={24} />
          </div>
          <div>
            <h1 className="text-[20px] font-semibold text-on-surface flex items-center gap-2">
              {editingName ? (
                <>
                  <input
                    value={draftName}
                    onChange={(e) => setDraftName(e.target.value)}
                    maxLength={100}
                    autoFocus
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') void handleRename()
                      if (e.key === 'Escape') setEditingName(false)
                    }}
                    className="h-8 px-2 rounded-lg border border-surface-variant bg-surface-container-low text-[16px] text-on-surface focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
                  />
                  <button type="button" disabled={isRenaming} onClick={() => void handleRename()} aria-label="Save name"
                    className="p-1 rounded text-primary hover:bg-surface-container-high disabled:opacity-50">
                    {isRenaming ? <Loader2 size={15} className="animate-spin" /> : <Check size={15} />}
                  </button>
                  <button type="button" onClick={() => setEditingName(false)} aria-label="Cancel rename"
                    className="p-1 rounded text-on-surface-variant hover:bg-surface-container-high">
                    <X size={15} />
                  </button>
                </>
              ) : (
                <>
                  {agent.name}
                  <button type="button" aria-label="Rename agent"
                    onClick={() => { setDraftName(agent.name); setEditingName(true) }}
                    className="p-1 rounded text-outline hover:text-primary hover:bg-surface-container-high transition-colors">
                    <Pencil size={14} />
                  </button>
                </>
              )}
            </h1>
            <div className="flex items-center gap-2 mt-1">
              <StatusBadge status={agent.status} size="sm" />
              <span className="font-mono text-[12px] text-outline">v{agent.version || '—'}</span>
              <span className="text-[12px] text-outline">Last seen: {formatRelative(agent.last_seen_at)}</span>
            </div>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Machines */}
        <div className="flex flex-col gap-3">
          <h2 className="text-[13px] font-semibold uppercase tracking-wider text-on-surface-variant flex items-center gap-2">
            <Monitor size={14} />
            Registered Machines ({machines.length})
          </h2>
          {machines.length === 0 ? (
            <div className="p-5 rounded-xl bg-surface-container-lowest border border-surface-variant text-center text-[13px] text-outline">
              No machines registered for this agent yet.
            </div>
          ) : (
            <div className="flex flex-col gap-2">
              {pagedMachines.pageItems.map((m) => (
                <div key={m.id} className="p-4 rounded-xl bg-surface-container-lowest border border-surface-variant shadow-sm">
                  <div className="flex items-center justify-between">
                    <p className="text-[14px] font-semibold text-on-surface">{m.hostname}</p>
                    <span className="font-mono text-[11px] text-outline">{m.ip_address}</span>
                  </div>
                  <p className="text-[12px] text-outline mt-0.5">{m.os || '—'}</p>
                </div>
              ))}
              <Pagination
                page={pagedMachines.page}
                totalPages={pagedMachines.totalPages}
                total={pagedMachines.total}
                perPage={pagedMachines.perPage}
                onPage={pagedMachines.setPage}
              />
            </div>
          )}
        </div>

        {/* Backup jobs */}
        <div className="flex flex-col gap-3">
          <h2 className="text-[13px] font-semibold uppercase tracking-wider text-on-surface-variant flex items-center gap-2">
            <Archive size={14} />
            Assigned Backup Jobs ({jobs.length})
          </h2>
          {jobs.length === 0 ? (
            <div className="p-5 rounded-xl bg-surface-container-lowest border border-surface-variant text-center text-[13px] text-outline">
              No backup jobs assigned to this agent.{' '}
              <button type="button" onClick={() => setShowNew(true)} className="text-primary hover:underline">
                Create one
              </button>
            </div>
          ) : (
            <div className="flex flex-col gap-2">
              {pagedJobs.pageItems.map((job) => (
                <button
                  key={job.id}
                  type="button"
                  onClick={() => navigate(`/jobs/${job.id}`)}
                  className="flex items-center justify-between p-4 rounded-xl bg-surface-container-lowest border border-surface-variant shadow-sm hover:shadow-md transition-all text-left"
                >
                  <div>
                    <p className="text-[14px] font-semibold text-on-surface">{job.name}</p>
                    <p className="text-[12px] text-outline mt-0.5">
                      {job.source_type.replace('_', ' ')} · {job.schedule?.cron_expr ?? 'No schedule'}
                    </p>
                  </div>
                  <StatusBadge status={job.enabled ? 'ONLINE' : 'OFFLINE'} size="sm" />
                </button>
              ))}
              <Pagination
                page={pagedJobs.page}
                totalPages={pagedJobs.totalPages}
                total={pagedJobs.total}
                perPage={pagedJobs.perPage}
                onPage={pagedJobs.setPage}
              />
            </div>
          )}
        </div>
      </div>

      {/* Agent details */}
      <div className="p-5 rounded-xl bg-surface-container-lowest border border-surface-variant shadow-sm">
        <h3 className="text-[12px] font-semibold uppercase tracking-wider text-on-surface-variant mb-3">Agent Details</h3>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-[13px]">
          {[
            { label: 'Agent ID', value: agent.id, mono: true },
            { label: 'Version', value: agent.version || '—', mono: true },
            { label: 'Status', value: agent.status },
            { label: 'Registered', value: formatRelative(agent.created_at) },
          ].map((item) => (
            <div key={item.label}>
              <p className="text-[11px] font-semibold uppercase tracking-wider text-outline mb-0.5">{item.label}</p>
              <p className={`text-on-surface break-all ${item.mono ? 'font-mono text-[11px]' : 'font-medium'}`}>
                {item.value}
              </p>
            </div>
          ))}
        </div>
      </div>

      {showNew && (
        <NewJobModal onClose={() => setShowNew(false)} onSaved={() => setReloadKey((k) => k + 1)} />
      )}
    </Page>
  )
}

export default AgentDetailPage
