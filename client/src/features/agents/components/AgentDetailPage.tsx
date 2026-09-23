import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { agentApi, machineApi, jobApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import type { Agent, Machine, BackupJob } from '@/types'
import StatusBadge from '@/components/StatusBadge'
import { formatRelative } from '@/utils/format'
import { ChevronRight, Server, Loader2, Monitor, Archive } from 'lucide-react'

const AgentDetailPage = () => {
  const { id } = useParams<{ id: string }>()
  const { currentOrg } = useAuthStore()
  const { addToast } = useUIStore()
  const navigate = useNavigate()
  const [agent, setAgent] = useState<Agent | null>(null)
  const [machines, setMachines] = useState<Machine[]>([])
  const [jobs, setJobs] = useState<BackupJob[]>([])
  const [isLoading, setIsLoading] = useState(true)

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
  }, [currentOrg, id])

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 size={32} className="text-[#c3c6d7] animate-spin" />
      </div>
    )
  }

  if (!agent) return null

  return (
    <div className="flex flex-col gap-6">
      {/* Breadcrumb */}
      <div className="flex items-center gap-2 text-[12px] text-[#434655]">
        <button type="button" onClick={() => navigate('/agents')} className="hover:text-[#004ac6]">
          Agents & Machines
        </button>
        <ChevronRight size={14} />
        <span className="text-[#004ac6] font-medium">{agent.name}</span>
      </div>

      {/* Header card */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 p-5 rounded-xl bg-[#ffffff] border border-[#e9edff] shadow-sm">
        <div className="flex items-center gap-4">
          <div className={`w-12 h-12 rounded-xl flex items-center justify-center ${agent.status === 'ONLINE' ? 'bg-[#c9e6ff]/40 text-[#006591]' : 'bg-[#dce2f7] text-[#737686]'}`}>
            <Server size={24} />
          </div>
          <div>
            <h1 className="text-[20px] font-semibold text-[#141b2b]">{agent.name}</h1>
            <div className="flex items-center gap-2 mt-1">
              <StatusBadge status={agent.status} size="sm" />
              <span className="font-mono text-[12px] text-[#737686]">v{agent.version || '—'}</span>
              <span className="text-[12px] text-[#737686]">Last seen: {formatRelative(agent.last_seen_at)}</span>
            </div>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Machines */}
        <div className="flex flex-col gap-3">
          <h2 className="text-[13px] font-semibold uppercase tracking-wider text-[#434655] flex items-center gap-2">
            <Monitor size={14} />
            Registered Machines ({machines.length})
          </h2>
          {machines.length === 0 ? (
            <div className="p-5 rounded-xl bg-[#ffffff] border border-[#e9edff] text-center text-[13px] text-[#737686]">
              No machines registered for this agent yet.
            </div>
          ) : (
            <div className="flex flex-col gap-2">
              {machines.map((m) => (
                <div key={m.id} className="p-4 rounded-xl bg-[#ffffff] border border-[#e9edff] shadow-sm">
                  <div className="flex items-center justify-between">
                    <p className="text-[14px] font-semibold text-[#141b2b]">{m.hostname}</p>
                    <span className="font-mono text-[11px] text-[#737686]">{m.ip_address}</span>
                  </div>
                  <p className="text-[12px] text-[#737686] mt-0.5">{m.os || '—'}</p>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Backup jobs */}
        <div className="flex flex-col gap-3">
          <h2 className="text-[13px] font-semibold uppercase tracking-wider text-[#434655] flex items-center gap-2">
            <Archive size={14} />
            Assigned Backup Jobs ({jobs.length})
          </h2>
          {jobs.length === 0 ? (
            <div className="p-5 rounded-xl bg-[#ffffff] border border-[#e9edff] text-center text-[13px] text-[#737686]">
              No backup jobs assigned to this agent.{' '}
              <button type="button" onClick={() => navigate('/jobs/new')} className="text-[#004ac6] hover:underline">
                Create one
              </button>
            </div>
          ) : (
            <div className="flex flex-col gap-2">
              {jobs.map((job) => (
                <button
                  key={job.id}
                  type="button"
                  onClick={() => navigate(`/jobs/${job.id}`)}
                  className="flex items-center justify-between p-4 rounded-xl bg-[#ffffff] border border-[#e9edff] shadow-sm hover:shadow-md transition-all text-left"
                >
                  <div>
                    <p className="text-[14px] font-semibold text-[#141b2b]">{job.name}</p>
                    <p className="text-[12px] text-[#737686] mt-0.5">
                      {job.source_type.replace('_', ' ')} · {job.schedule?.cron_expr ?? 'No schedule'}
                    </p>
                  </div>
                  <StatusBadge status={job.enabled ? 'ONLINE' : 'OFFLINE'} size="sm" />
                </button>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Agent details */}
      <div className="p-5 rounded-xl bg-[#ffffff] border border-[#e9edff] shadow-sm">
        <h3 className="text-[12px] font-semibold uppercase tracking-wider text-[#434655] mb-3">Agent Details</h3>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-[13px]">
          {[
            { label: 'Agent ID', value: agent.id, mono: true },
            { label: 'Version', value: agent.version || '—', mono: true },
            { label: 'Status', value: agent.status },
            { label: 'Registered', value: formatRelative(agent.created_at) },
          ].map((item) => (
            <div key={item.label}>
              <p className="text-[11px] font-semibold uppercase tracking-wider text-[#737686] mb-0.5">{item.label}</p>
              <p className={`text-[#141b2b] break-all ${item.mono ? 'font-mono text-[11px]' : 'font-medium'}`}>
                {item.value}
              </p>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

export default AgentDetailPage
