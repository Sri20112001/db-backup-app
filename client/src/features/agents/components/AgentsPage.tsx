import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { agentApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import type { Agent } from '@/types'
import StatusBadge from '@/components/StatusBadge'
import EmptyState from '@/components/EmptyState'
import ConfirmDialog from '@/components/ConfirmDialog'
import { formatRelative } from '@/utils/format'
import { Plus, Server, Trash2, Copy, Check, Loader2 } from 'lucide-react'

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
    <div className="flex flex-col gap-6">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-[20px] font-semibold text-[#141b2b] tracking-tight">Agents & Machines</h1>
          <p className="text-[12px] text-[#434655] mt-0.5">Windows backup agents installed on your infrastructure</p>
        </div>
        <button
          type="button"
          onClick={handleGenerateToken}
          className="flex items-center gap-2 px-4 h-9 rounded-lg bg-[#2563eb] text-white text-[13px] font-medium hover:bg-[#1d4ed8] transition-colors shadow-sm"
        >
          <Plus size={16} />
          Register Agent
        </button>
      </div>

      {isLoading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {[...Array(4)].map((_, i) => (
            <div key={i} className="animate-pulse p-5 rounded-xl bg-[#ffffff] border border-[#e9edff] h-36" />
          ))}
        </div>
      ) : agents.length === 0 ? (
        <EmptyState
          icon={Server}
          title="No agents registered"
          description="Install the VaultGuard agent on your Windows machines and register them here."
          action={
            <button type="button" onClick={handleGenerateToken} className="px-4 h-9 rounded-lg bg-[#2563eb] text-white text-[13px] font-medium hover:bg-[#1d4ed8] transition-colors">
              Register First Agent
            </button>
          }
        />
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {agents.map((agent) => (
            <div
              key={agent.id}
              onClick={() => navigate(`/agents/${agent.id}`)}
              className="flex flex-col gap-3 p-5 rounded-xl bg-[#ffffff] border border-[#e9edff] shadow-sm hover:shadow-md transition-all cursor-pointer"
            >
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-3">
                  <div className={`w-10 h-10 rounded-lg flex items-center justify-center ${agent.status === 'ONLINE' ? 'bg-[#c9e6ff]/40 text-[#006591]' : 'bg-[#dce2f7] text-[#737686]'}`}>
                    <Server size={20} />
                  </div>
                  <div>
                    <h3 className="text-[15px] font-semibold text-[#141b2b]">{agent.name}</h3>
                    <p className="font-mono text-[11px] text-[#737686]">v{agent.version || '—'}</p>
                  </div>
                </div>
                <StatusBadge status={agent.status} size="sm" />
              </div>

              <div className="flex items-center justify-between text-[12px] text-[#737686]">
                <span>Last seen: {formatRelative(agent.last_seen_at)}</span>
                <button
                  type="button"
                  onClick={(e) => { e.stopPropagation(); setDeleteId(agent.id) }}
                  className="p-1.5 rounded-lg text-[#737686] hover:text-[#ba1a1a] hover:bg-[#ffdad6]/30 transition-colors"
                >
                  <Trash2 size={16} />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Register modal */}
      {showRegister && (
        <div className="fixed inset-0 z-[200] flex items-center justify-center">
          <div className="absolute inset-0 bg-[#141b2b]/30 backdrop-blur-[2px]" onClick={() => { setShowRegister(false); setRegToken(null); loadAgents() }} />
          <div className="relative bg-[#ffffff] rounded-xl shadow-2xl p-6 w-full max-w-md mx-4 border border-[#e9edff]">
            <h2 className="text-[16px] font-semibold text-[#141b2b] mb-2">Register New Agent</h2>
            <p className="text-[13px] text-[#434655] mb-4">
              Run the VaultGuard agent installer on your Windows machine and enter this registration key when prompted.
            </p>
            {regToken ? (
              <>
                <div className="flex items-center gap-2 p-3 rounded-lg bg-[#f1f3ff] border border-[#e9edff] mb-4">
                  <code className="flex-1 font-mono text-[12px] text-[#141b2b] break-all">{regToken.registration_key}</code>
                  <button
                    type="button"
                    onClick={handleCopy}
                    className="p-1.5 rounded text-[#434655] hover:bg-[#e9edff] transition-colors shrink-0"
                  >
                    {copied ? <Check size={16} className="text-[#006591]" /> : <Copy size={16} />}
                  </button>
                </div>
                <p className="text-[12px] text-[#737686] mb-4">This key can only be used once. Keep it secure.</p>
              </>
            ) : (
              <div className="flex items-center justify-center py-8">
                <Loader2 size={32} className="text-[#c3c6d7] animate-spin" />
              </div>
            )}
            <button
              type="button"
              onClick={() => { setShowRegister(false); setRegToken(null); loadAgents() }}
              className="w-full h-9 rounded-lg bg-[#2563eb] text-white text-[13px] font-medium hover:bg-[#1d4ed8] transition-colors"
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
