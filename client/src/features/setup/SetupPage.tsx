import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { orgApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import { Loader2, Building2 } from 'lucide-react'
import { Logo } from '@/components/ui/Logo'

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
    <div className="min-h-screen bg-surface-container-low flex items-center justify-center px-4">
      <div className="w-full max-w-[420px] bg-surface-container-lowest rounded-xl border border-surface-variant shadow-[0_4px_24px_rgba(20,27,43,0.08)] p-8">
        <div className="flex items-center gap-3 mb-8">
          <div className="w-10 h-10 rounded-xl bg-surface-container-highest flex items-center justify-center border border-outline-variant/30 shadow-inner">
            <Logo className="w-6 h-6" />
          </div>
          <div>
            <h1 className="text-[20px] font-semibold text-on-surface tracking-tight">VaultGuard</h1>
            <p className="text-[12px] text-outline">Backup & Recovery Platform</p>
          </div>
        </div>

        <div className="flex items-center gap-3 mb-6">
          <div className="w-10 h-10 rounded-lg bg-primary-container flex items-center justify-center text-primary">
            <Building2 size={20} />
          </div>
          <div>
            <h2 className="text-[18px] font-semibold text-on-surface">Set up your organization</h2>
            <p className="text-[13px] text-outline">Hi {user?.name} — one more step before you start.</p>
          </div>
        </div>

        {error && (
          <div className="mb-4 px-3 py-2.5 rounded-lg bg-error-container border border-error text-on-error-container text-[13px]">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <div>
            <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5" htmlFor="orgName">
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
              className="w-full h-9 px-3 rounded-lg border border-surface-variant bg-surface-container-low text-[14px] text-on-surface placeholder:text-outline focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all"
            />
            <p className="mt-1.5 text-[12px] text-outline">
              This is your team's workspace. You can invite members after setup.
            </p>
          </div>
          <button
            type="submit"
            disabled={isLoading || !orgName.trim()}
            className="w-full h-10 rounded-lg bg-primary text-on-primary text-[13px] font-medium hover:bg-primary-container transition-colors disabled:opacity-60 flex items-center justify-center gap-2"
          >
            {isLoading && <Loader2 size={14} className="animate-spin" />}
            {isLoading ? 'Creating...' : 'Create Organization'}
          </button>
        </form>

        <button
          type="button"
          onClick={handleLogout}
          className="mt-4 w-full text-center text-[13px] text-outline hover:text-error transition-colors"
        >
          Sign out
        </button>
      </div>
    </div>
  )
}

export default SetupPage
