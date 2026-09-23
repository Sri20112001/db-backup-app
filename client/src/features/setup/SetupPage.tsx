import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { orgApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import { Shield, Loader2, Building2 } from 'lucide-react'

const SetupPage = () => {
  const navigate = useNavigate()
  const { setCurrentOrg, user, logout } = useAuthStore()
  const { addToast } = useUIStore()
  const [orgName, setOrgName] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')

  const handleLogout = () => {
    logout()
    navigate('/login', { replace: true })
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!orgName.trim()) return
    setError('')
    setIsLoading(true)
    try {
      const org = await orgApi.create(orgName.trim())
      setCurrentOrg(org)
      addToast('success', 'Organization created')
      navigate('/')
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to create organization')
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-[#f9f9ff] flex items-center justify-center px-4">
      <div className="w-full max-w-[420px] bg-[#ffffff] rounded-xl border border-[#e9edff] shadow-[0_4px_24px_rgba(20,27,43,0.08)] p-8">
        <div className="flex items-center gap-3 mb-8">
          <div className="w-10 h-10 rounded-xl bg-[#2563eb] flex items-center justify-center">
            <Shield size={20} className="text-white" />
          </div>
          <div>
            <h1 className="text-[20px] font-semibold text-[#141b2b] tracking-tight">VaultGuard</h1>
            <p className="text-[12px] text-[#737686]">Backup & Recovery Platform</p>
          </div>
        </div>

        <div className="flex items-center gap-3 mb-6">
          <div className="w-10 h-10 rounded-lg bg-[#dbe1ff] flex items-center justify-center text-[#004ac6]">
            <Building2 size={20} />
          </div>
          <div>
            <h2 className="text-[18px] font-semibold text-[#141b2b]">Set up your organization</h2>
            <p className="text-[13px] text-[#737686]">Hi {user?.name} — one more step before you start.</p>
          </div>
        </div>

        {error && (
          <div className="mb-4 px-3 py-2.5 rounded-lg bg-[#ffdad6] border border-[#fca5a5] text-[#93000a] text-[13px]">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <div>
            <label className="block text-[13px] font-medium text-[#434655] mb-1.5" htmlFor="orgName">
              Organization name
            </label>
            <input
              id="orgName"
              type="text"
              value={orgName}
              onChange={(e) => setOrgName(e.target.value)}
              required
              placeholder="Acme Corp"
              autoFocus
              className="w-full h-9 px-3 rounded-lg border border-[#e9edff] bg-[#f9f9ff] text-[14px] text-[#141b2b] placeholder:text-[#737686] focus:outline-none focus:ring-2 focus:ring-[#2563eb]/20 focus:border-[#2563eb] transition-all"
            />
            <p className="mt-1.5 text-[12px] text-[#737686]">
              This is your team's workspace. You can invite members after setup.
            </p>
          </div>
          <button
            type="submit"
            disabled={isLoading || !orgName.trim()}
            className="w-full h-10 rounded-lg bg-[#2563eb] text-white text-[13px] font-medium hover:bg-[#1d4ed8] transition-colors disabled:opacity-60 flex items-center justify-center gap-2"
          >
            {isLoading && <Loader2 size={14} className="animate-spin" />}
            {isLoading ? 'Creating...' : 'Create Organization'}
          </button>
        </form>

        <button
          type="button"
          onClick={handleLogout}
          className="mt-4 w-full text-center text-[13px] text-[#737686] hover:text-[#ba1a1a] transition-colors"
        >
          Sign out
        </button>
      </div>
    </div>
  )
}

export default SetupPage
