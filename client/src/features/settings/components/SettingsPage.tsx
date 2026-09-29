import { useEffect, useState } from 'react'
import { memberApi, authApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import type { OrganizationMember, MemberRole } from '@/types'
import { UserPlus, UserMinus, Loader2 } from 'lucide-react'
import Pagination from '@/components/Pagination'
import { usePagination } from '@/hooks/usePagination'

const TABS = ['Organization', 'Team Members', 'Security']

const roleColor: Record<MemberRole, string> = {
  OWNER: 'bg-primary-container text-primary',
  ADMIN: 'bg-primary-container text-primary',
  OPERATOR: 'bg-primary-container text-primary',
  VIEWER: 'bg-surface-variant text-on-surface-variant',
}

const SettingsPage = () => {
  const { currentOrg, user } = useAuthStore()
  const { addToast } = useUIStore()
  const [tab, setTab] = useState(0)
  const [members, setMembers] = useState<OrganizationMember[]>([])
  const pagedMembers = usePagination(members, 8, currentOrg?.id ?? '')
  const [isLoading, setIsLoading] = useState(false)
  const [showInvite, setShowInvite] = useState(false)
  const [inviteForm, setInviteForm] = useState({ email: '', name: '', password: '', role: 'VIEWER' as MemberRole })
  const [isInviting, setIsInviting] = useState(false)
  const [pwForm, setPwForm] = useState({ current: '', next: '', confirm: '' })
  const [isPwLoading, setIsPwLoading] = useState(false)

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
      await memberApi.invite(currentOrg.id, inviteForm.email, inviteForm.name, inviteForm.role, inviteForm.password)
      addToast('success', `${inviteForm.email} added to organization`)
      setShowInvite(false)
      setInviteForm({ email: '', name: '', password: '', role: 'VIEWER' })
      loadMembers()
    } catch (err: unknown) { addToast('error', err instanceof Error ? err.message : 'Failed to add member') }
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

  const handlePasswordChange = async (e: React.FormEvent) => {
    e.preventDefault()
    if (pwForm.next !== pwForm.confirm) {
      addToast('error', 'New passwords do not match')
      return
    }
    if (pwForm.next.length < 8) {
      addToast('error', 'Password must be at least 8 characters')
      return
    }
    setIsPwLoading(true)
    try {
      await authApi.changePassword(pwForm.current, pwForm.next)
      addToast('success', 'Password updated')
      setPwForm({ current: '', next: '', confirm: '' })
    } catch (err: unknown) {
      addToast('error', err instanceof Error ? err.message : 'Failed to update password')
    } finally {
      setIsPwLoading(false)
    }
  }

  const inputCls = "w-full h-9 px-3 rounded-lg border border-surface-variant bg-surface-container-low text-[14px] text-on-surface focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all"

  return (
    <div className="h-full min-h-0 flex flex-col gap-4 max-w-3xl overflow-y-auto pr-0.5">
      <div>
        <h1 className="text-[20px] font-semibold text-on-surface tracking-tight">Settings</h1>
        <p className="text-[12px] text-on-surface-variant mt-0.5">Manage your organization and team</p>
      </div>

      {/* Tabs */}
      <div className="flex items-center gap-1 border-b border-surface-variant shrink-0">
        {TABS.map((t, i) => (
          <button
            key={t}
            type="button"
            onClick={() => setTab(i)}
            className={`px-4 py-2.5 text-[13px] font-medium transition-colors border-b-2 -mb-px ${
              tab === i ? 'border-primary text-primary' : 'border-transparent text-on-surface-variant hover:text-on-surface'
            }`}
          >
            {t}
          </button>
        ))}
      </div>

      {/* Organization tab */}
      {tab === 0 && (
        <div className="flex flex-col gap-4 p-5 rounded-xl bg-surface-container-lowest border border-surface-variant shadow-sm">
          <h2 className="text-[14px] font-semibold text-on-surface">Organization Details</h2>
          <div>
            <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Organization Name</label>
            <input type="text" defaultValue={currentOrg?.name} readOnly className={`${inputCls} bg-surface-container-low`} />
          </div>
          <div>
            <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Slug</label>
            <input type="text" defaultValue={currentOrg?.slug} readOnly className={`${inputCls} font-mono text-[13px] bg-surface-container-low text-outline`} />
          </div>
          <div>
            <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Organization ID</label>
            <input type="text" defaultValue={currentOrg?.id} readOnly className={`${inputCls} font-mono text-[12px] bg-surface-container-low text-outline`} />
          </div>
          <div>
            <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">User Limit</label>
            <input type="text" defaultValue={currentOrg?.user_limit} readOnly className={`${inputCls} bg-surface-container-low text-outline`} />
          </div>
        </div>
      )}

      {/* Members tab */}
      {tab === 1 && (
        <div className="flex flex-col gap-4">
          <div className="flex items-center justify-between">
            <h2 className="text-[14px] font-semibold text-on-surface">Team Members ({members.length})</h2>
            <button type="button" onClick={() => setShowInvite(true)} className="flex items-center gap-1.5 px-3 h-8 rounded-lg bg-primary text-on-primary text-[12px] font-medium hover:bg-primary-container transition-colors">
              <UserPlus size={14} />
              Add Member
            </button>
          </div>

          <div className="flex flex-col rounded-xl bg-surface-container-lowest border border-surface-variant shadow-sm overflow-hidden">
            <div className="overflow-y-auto max-h-[420px]">
            <table className="w-full text-left border-collapse">
              <thead className="sticky top-0 z-10">
                <tr className="bg-surface-container-low text-[12px] font-semibold uppercase tracking-wider text-on-surface-variant">
                  {['Member', 'Email', 'Role', 'Actions'].map((h) => (
                    <th key={h} className="py-2.5 px-4">{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y divide-[#e9edff]">
                {isLoading ? (
                  [...Array(3)].map((_, i) => (
                    <tr key={i} className="animate-pulse">
                      {[...Array(4)].map((_, j) => <td key={j} className="py-3 px-4"><div className="h-4 bg-surface-container-high rounded w-3/4" /></td>)}
                    </tr>
                  ))
                ) : members.length === 0 ? (
                  <tr>
                    <td colSpan={4} className="py-10 text-center text-[13px] text-outline">
                      No team members yet.
                    </td>
                  </tr>
                ) : pagedMembers.pageItems.map((m) => (
                  <tr key={m.id} className="hover:bg-surface-container-low transition-colors">
                    <td className="py-3 px-4">
                      <div className="flex items-center gap-2.5">
                        <div className="w-7 h-7 rounded-full bg-primary flex items-center justify-center text-on-primary text-[11px] font-bold">
                          {m.user?.name?.charAt(0)?.toUpperCase() ?? '?'}
                        </div>
                        <span className="text-[14px] font-medium text-on-surface">{m.user?.name ?? '—'}</span>
                      </div>
                    </td>
                    <td className="py-3 px-4 text-[13px] text-on-surface-variant">{m.user?.email ?? '—'}</td>
                    <td className="py-3 px-4">
                      <span className={`px-2 py-0.5 rounded-full text-[11px] font-semibold uppercase ${roleColor[m.role]}`}>{m.role}</span>
                    </td>
                    <td className="py-3 px-4">
                      {m.user?.id !== user?.id && (
                        <div className="flex items-center gap-2">
                          <select
                            value={m.role}
                            onChange={(e) => handleRoleChange(m.user_id, e.target.value as MemberRole)}
                            className="h-7 px-2 rounded border border-surface-variant bg-surface-container-low text-[12px] text-on-surface focus:outline-none"
                          >
                            {(['OWNER', 'ADMIN', 'OPERATOR', 'VIEWER'] as MemberRole[]).map((r) => (
                              <option key={r} value={r}>{r}</option>
                            ))}
                          </select>
                          <button type="button" onClick={() => handleRemove(m.user_id)} className="p-1 rounded text-outline hover:text-error hover:bg-error-container/30 transition-colors">
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
          </div>

          <Pagination
            page={pagedMembers.page}
            totalPages={pagedMembers.totalPages}
            total={pagedMembers.total}
            perPage={pagedMembers.perPage}
            onPage={pagedMembers.setPage}
          />

          {/* Invite modal */}
          {showInvite && (
            <div className="fixed inset-0 z-[200] flex items-center justify-center">
              <div className="absolute inset-0 bg-on-surface/30 backdrop-blur-[2px]" onClick={() => setShowInvite(false)} />
              <div className="relative bg-surface-container-lowest rounded-xl shadow-2xl p-6 w-full max-w-md mx-4 border border-surface-variant">
                <h2 className="text-[16px] font-semibold text-on-surface mb-4">Add Team Member</h2>
                <form onSubmit={handleInvite} className="flex flex-col gap-4">
                  <div><label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Email *</label><input type="email" value={inviteForm.email} onChange={(e) => setInviteForm((f) => ({ ...f, email: e.target.value }))} required className={inputCls} /></div>
                  <div><label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Name</label><input type="text" value={inviteForm.name} onChange={(e) => setInviteForm((f) => ({ ...f, name: e.target.value }))} className={inputCls} /></div>
                  <div><label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Initial Password *</label><input type="password" value={inviteForm.password} onChange={(e) => setInviteForm((f) => ({ ...f, password: e.target.value }))} required minLength={8} className={inputCls} /></div>
                  <div>
                    <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Role *</label>
                    <select value={inviteForm.role} onChange={(e) => setInviteForm((f) => ({ ...f, role: e.target.value as MemberRole }))} className={inputCls}>
                      {(['OWNER', 'ADMIN', 'OPERATOR', 'VIEWER'] as MemberRole[]).map((r) => <option key={r} value={r}>{r}</option>)}
                    </select>
                  </div>
                  <div className="flex items-center justify-end gap-3 mt-2">
                    <button type="button" onClick={() => setShowInvite(false)} className="px-4 h-9 rounded-lg bg-surface-container-lowest border border-surface-variant text-on-surface text-[13px] font-medium hover:bg-surface-container-low transition-colors">Cancel</button>
                    <button type="submit" disabled={isInviting} className="px-4 h-9 rounded-lg bg-primary text-on-primary text-[13px] font-medium hover:bg-primary-container transition-colors disabled:opacity-60 flex items-center gap-2">
                      {isInviting && <Loader2 size={14} className="animate-spin" />}
                      Add User
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
        <div className="flex flex-col gap-4 p-5 rounded-xl bg-surface-container-lowest border border-surface-variant shadow-sm">
          <h2 className="text-[14px] font-semibold text-on-surface">Change Password</h2>
          <form onSubmit={handlePasswordChange} className="flex flex-col gap-4">
            <div>
              <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Current Password</label>
              <input type="password" value={pwForm.current} onChange={(e) => setPwForm((f) => ({ ...f, current: e.target.value }))} required className={inputCls} />
            </div>
            <div>
              <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">New Password</label>
              <input type="password" value={pwForm.next} onChange={(e) => setPwForm((f) => ({ ...f, next: e.target.value }))} required minLength={8} className={inputCls} />
            </div>
            <div>
              <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Confirm New Password</label>
              <input type="password" value={pwForm.confirm} onChange={(e) => setPwForm((f) => ({ ...f, confirm: e.target.value }))} required className={inputCls} />
            </div>
            <button type="submit" disabled={isPwLoading} className="w-fit px-4 h-9 rounded-lg bg-primary text-on-primary text-[13px] font-medium hover:bg-primary-container transition-colors disabled:opacity-60 flex items-center gap-2">
              {isPwLoading && <Loader2 size={14} className="animate-spin" />}
              Update Password
            </button>
          </form>
        </div>
      )}
    </div>
  )
}

export default SettingsPage
