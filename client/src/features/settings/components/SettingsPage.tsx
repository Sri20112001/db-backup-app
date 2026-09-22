import { useEffect, useState } from 'react'
import { memberApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import type { OrganizationMember, MemberRole } from '@/types'
import { UserPlus, UserMinus, Loader2 } from 'lucide-react'

const TABS = ['Organization', 'Team Members', 'Security']

const roleColor: Record<MemberRole, string> = {
  OWNER: 'bg-[#e1e0ff] text-[#2f2ebe]',
  ADMIN: 'bg-[#dbe1ff] text-[#003ea8]',
  OPERATOR: 'bg-[#c9e6ff] text-[#004c6e]',
  VIEWER: 'bg-[#dce2f7] text-[#434655]',
}

const SettingsPage = () => {
  const { currentOrg, user } = useAuthStore()
  const { addToast } = useUIStore()
  const [tab, setTab] = useState(0)
  const [members, setMembers] = useState<OrganizationMember[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [showInvite, setShowInvite] = useState(false)
  const [inviteForm, setInviteForm] = useState({ email: '', name: '', role: 'VIEWER' as MemberRole })
  const [isInviting, setIsInviting] = useState(false)

  const loadMembers = () => {
    if (!currentOrg) return
    setIsLoading(true)
    memberApi.list(currentOrg.id).then(setMembers).catch(() => addToast('error', 'Failed to load members')).finally(() => setIsLoading(false))
  }

  useEffect(() => { if (tab === 1) loadMembers() }, [tab, currentOrg])

  const handleInvite = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!currentOrg) return
    setIsInviting(true)
    try {
      await memberApi.invite(currentOrg.id, inviteForm.email, inviteForm.name, inviteForm.role)
      addToast('success', `${inviteForm.email} invited`)
      setShowInvite(false)
      setInviteForm({ email: '', name: '', role: 'VIEWER' })
      loadMembers()
    } catch { addToast('error', 'Failed to invite member') }
    finally { setIsInviting(false) }
  }

  const handleRoleChange = async (userId: string, role: MemberRole) => {
    if (!currentOrg) return
    try {
      await memberApi.updateRole(currentOrg.id, userId, role)
      addToast('success', 'Role updated')
      loadMembers()
    } catch { addToast('error', 'Failed to update role') }
  }

  const handleRemove = async (userId: string) => {
    if (!currentOrg) return
    try {
      await memberApi.remove(currentOrg.id, userId)
      addToast('success', 'Member removed')
      loadMembers()
    } catch { addToast('error', 'Failed to remove member') }
  }

  const inputCls = "w-full h-9 px-3 rounded-lg border border-[#e9edff] bg-[#f9f9ff] text-[14px] text-[#141b2b] focus:outline-none focus:ring-2 focus:ring-[#2563eb]/20 focus:border-[#2563eb] transition-all"

  return (
    <div className="flex flex-col gap-6 max-w-3xl">
      <div>
        <h1 className="text-[20px] font-semibold text-[#141b2b] tracking-tight">Settings</h1>
        <p className="text-[12px] text-[#434655] mt-0.5">Manage your organization and team</p>
      </div>

      {/* Tabs */}
      <div className="flex items-center gap-1 border-b border-[#e9edff]">
        {TABS.map((t, i) => (
          <button
            key={t}
            type="button"
            onClick={() => setTab(i)}
            className={`px-4 py-2.5 text-[13px] font-medium transition-colors border-b-2 -mb-px ${
              tab === i ? 'border-primary-container text-primary-container' : 'border-transparent text-[#434655] hover:text-[#141b2b]'
            }`}
          >
            {t}
          </button>
        ))}
      </div>

      {/* Organization tab */}
      {tab === 0 && (
        <div className="flex flex-col gap-4 p-5 rounded-xl bg-[#ffffff] border border-[#e9edff] shadow-sm">
          <h2 className="text-[14px] font-semibold text-[#141b2b]">Organization Details</h2>
          <div>
            <label className="block text-[13px] font-medium text-[#434655] mb-1.5">Organization Name</label>
            <input type="text" defaultValue={currentOrg?.name} readOnly className={`${inputCls} bg-[#f9f9ff]`} />
          </div>
          <div>
            <label className="block text-[13px] font-medium text-[#434655] mb-1.5">Slug</label>
            <input type="text" defaultValue={currentOrg?.slug} readOnly className={`${inputCls} font-mono text-[13px] bg-[#f9f9ff] text-[#737686]`} />
          </div>
          <div>
            <label className="block text-[13px] font-medium text-[#434655] mb-1.5">Organization ID</label>
            <input type="text" defaultValue={currentOrg?.id} readOnly className={`${inputCls} font-mono text-[12px] bg-[#f9f9ff] text-[#737686]`} />
          </div>
        </div>
      )}

      {/* Members tab */}
      {tab === 1 && (
        <div className="flex flex-col gap-4">
          <div className="flex items-center justify-between">
            <h2 className="text-[14px] font-semibold text-[#141b2b]">Team Members ({members.length})</h2>
            <button type="button" onClick={() => setShowInvite(true)} className="flex items-center gap-1.5 px-3 h-8 rounded-lg bg-[#2563eb] text-white text-[12px] font-medium hover:bg-[#1d4ed8] transition-colors">
              <UserPlus size={14} />
              Invite Member
            </button>
          </div>

          <div className="flex flex-col rounded-xl bg-[#ffffff] border border-[#e9edff] shadow-sm overflow-hidden">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="bg-[#f1f3ff] text-[12px] font-semibold uppercase tracking-wider text-[#434655]">
                  {['Member', 'Email', 'Role', 'Actions'].map((h) => (
                    <th key={h} className="py-2.5 px-4">{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y divide-[#e9edff]">
                {isLoading ? (
                  [...Array(3)].map((_, i) => (
                    <tr key={i} className="animate-pulse">
                      {[...Array(4)].map((_, j) => <td key={j} className="py-3 px-4"><div className="h-4 bg-[#e9edff] rounded w-3/4" /></td>)}
                    </tr>
                  ))
                ) : members.map((m) => (
                  <tr key={m.id} className="hover:bg-[#f9f9ff] transition-colors">
                    <td className="py-3 px-4">
                      <div className="flex items-center gap-2.5">
                        <div className="w-7 h-7 rounded-full bg-[#2563eb] flex items-center justify-center text-white text-[11px] font-bold">
                          {m.user?.name?.charAt(0)?.toUpperCase() ?? '?'}
                        </div>
                        <span className="text-[14px] font-medium text-[#141b2b]">{m.user?.name ?? '—'}</span>
                      </div>
                    </td>
                    <td className="py-3 px-4 text-[13px] text-[#434655]">{m.user?.email ?? '—'}</td>
                    <td className="py-3 px-4">
                      <span className={`px-2 py-0.5 rounded-full text-[11px] font-semibold uppercase ${roleColor[m.role]}`}>{m.role}</span>
                    </td>
                    <td className="py-3 px-4">
                      {m.user?.id !== user?.id && (
                        <div className="flex items-center gap-2">
                          <select
                            value={m.role}
                            onChange={(e) => handleRoleChange(m.user_id, e.target.value as MemberRole)}
                            className="h-7 px-2 rounded border border-[#e9edff] bg-[#f9f9ff] text-[12px] text-[#141b2b] focus:outline-none"
                          >
                            {(['OWNER', 'ADMIN', 'OPERATOR', 'VIEWER'] as MemberRole[]).map((r) => (
                              <option key={r} value={r}>{r}</option>
                            ))}
                          </select>
                          <button type="button" onClick={() => handleRemove(m.user_id)} className="p-1 rounded text-[#737686] hover:text-[#ba1a1a] hover:bg-[#ffdad6]/30 transition-colors">
                            <UserMinus size={14} />
                          </button>
                        </div>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Invite modal */}
          {showInvite && (
            <div className="fixed inset-0 z-[200] flex items-center justify-center">
              <div className="absolute inset-0 bg-[#141b2b]/30 backdrop-blur-[2px]" onClick={() => setShowInvite(false)} />
              <div className="relative bg-[#ffffff] rounded-xl shadow-2xl p-6 w-full max-w-md mx-4 border border-[#e9edff]">
                <h2 className="text-[16px] font-semibold text-[#141b2b] mb-4">Invite Team Member</h2>
                <form onSubmit={handleInvite} className="flex flex-col gap-4">
                  <div><label className="block text-[13px] font-medium text-[#434655] mb-1.5">Email *</label><input type="email" value={inviteForm.email} onChange={(e) => setInviteForm((f) => ({ ...f, email: e.target.value }))} required className={inputCls} /></div>
                  <div><label className="block text-[13px] font-medium text-[#434655] mb-1.5">Name</label><input type="text" value={inviteForm.name} onChange={(e) => setInviteForm((f) => ({ ...f, name: e.target.value }))} className={inputCls} /></div>
                  <div>
                    <label className="block text-[13px] font-medium text-[#434655] mb-1.5">Role *</label>
                    <select value={inviteForm.role} onChange={(e) => setInviteForm((f) => ({ ...f, role: e.target.value as MemberRole }))} className={inputCls}>
                      {(['OWNER', 'ADMIN', 'OPERATOR', 'VIEWER'] as MemberRole[]).map((r) => <option key={r} value={r}>{r}</option>)}
                    </select>
                  </div>
                  <div className="flex items-center justify-end gap-3 mt-2">
                    <button type="button" onClick={() => setShowInvite(false)} className="px-4 h-9 rounded-lg bg-[#ffffff] border border-[#e9edff] text-[#141b2b] text-[13px] font-medium hover:bg-[#f1f3ff] transition-colors">Cancel</button>
                    <button type="submit" disabled={isInviting} className="px-4 h-9 rounded-lg bg-[#2563eb] text-white text-[13px] font-medium hover:bg-[#1d4ed8] transition-colors disabled:opacity-60 flex items-center gap-2">
                      {isInviting && <Loader2 size={14} className="animate-spin" />}
                      Send Invite
                    </button>
                  </div>
                </form>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Security tab */}
      {tab === 2 && (
        <div className="flex flex-col gap-4 p-5 rounded-xl bg-[#ffffff] border border-[#e9edff] shadow-sm">
          <h2 className="text-[14px] font-semibold text-[#141b2b]">Security</h2>
          <div className="flex flex-col gap-4">
            <div>
              <label className="block text-[13px] font-medium text-[#434655] mb-1.5">Current Password</label>
              <input type="password" className={inputCls} />
            </div>
            <div>
              <label className="block text-[13px] font-medium text-[#434655] mb-1.5">New Password</label>
              <input type="password" className={inputCls} />
            </div>
            <div>
              <label className="block text-[13px] font-medium text-[#434655] mb-1.5">Confirm New Password</label>
              <input type="password" className={inputCls} />
            </div>
            <button type="button" className="w-fit px-4 h-9 rounded-lg bg-[#2563eb] text-white text-[13px] font-medium hover:bg-[#1d4ed8] transition-colors">
              Update Password
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

export default SettingsPage
