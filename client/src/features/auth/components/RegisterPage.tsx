import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { authApi, orgApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import { Loader2 } from 'lucide-react'
import { Logo } from '@/components/ui/Logo'

const RegisterPage = () => {
  const navigate = useNavigate()
  const { setAuth, setCurrentOrg } = useAuthStore()
  const { addToast } = useUIStore()
  const [form, setForm] = useState({ name: '', email: '', password: '' })
  const [orgName, setOrgName] = useState('')
  const [step, setStep] = useState<'account' | 'org'>('account')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) =>
    setForm((f) => ({ ...f, [e.target.name]: e.target.value }))

  const handleAccountSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setIsLoading(true)
    try {
      await authApi.register(form.email, form.password, form.name)
      // Auto-login immediately after registration
      const data = await authApi.login(form.email, form.password)
      setAuth(data.user, data.access_token, data.refresh_token)
      setStep('org')
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Registration failed')
    } finally {
      setIsLoading(false)
    }
  }

  const handleOrgSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!orgName.trim()) return
    setError('')
    setIsLoading(true)
    try {
      const org = await orgApi.create(orgName.trim())
      setCurrentOrg(org)
      addToast('success', `Welcome to VaultGuard, ${form.name}!`)
      navigate('/')
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to create organization')
    } finally {
      setIsLoading(false)
    }
  }

  const inputCls = 'w-full h-9 px-3 rounded-lg border border-surface-variant bg-surface-container-low text-[14px] text-on-surface placeholder:text-outline focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all'

  return (
    <div className="min-h-screen bg-transparent flex items-center justify-center px-4">
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

        {/* Step indicator */}
        <div className="flex items-center gap-2 mb-6">
          {(['account', 'org'] as const).map((s, i) => (
            <div key={s} className="flex items-center gap-2 flex-1">
              <div className={`w-6 h-6 rounded-full flex items-center justify-center text-[11px] font-bold shrink-0 ${
                step === s ? 'bg-primary text-on-primary' :
                (step === 'org' && s === 'account') ? 'bg-primary text-on-primary' :
                'bg-surface-container-high text-outline'
              }`}>
                {step === 'org' && s === 'account' ? '✓' : i + 1}
              </div>
              <span className="text-[12px] text-outline">{s === 'account' ? 'Account' : 'Organization'}</span>
              {i === 0 && <div className={`flex-1 h-0.5 ${step === 'org' ? 'bg-primary' : 'bg-surface-container-high'}`} />}
            </div>
          ))}
        </div>

        {error && (
          <div className="mb-4 px-3 py-2.5 rounded-lg bg-error-container border border-error text-on-error-container text-[13px]">
            {error}
          </div>
        )}

        {step === 'account' ? (
          <>
            <h2 className="text-[18px] font-semibold text-on-surface mb-6">Create account</h2>
            <form onSubmit={handleAccountSubmit} className="flex flex-col gap-4">
              {(['name', 'email', 'password'] as const).map((field) => (
                <div key={field}>
                  <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5" htmlFor={field}>
                    {field === 'name' ? 'Full name' : field === 'email' ? 'Email address' : 'Password'}
                  </label>
                  <input
                    id={field}
                    name={field}
                    type={field === 'password' ? 'password' : field === 'email' ? 'email' : 'text'}
                    value={form[field]}
                    onChange={handleChange}
                    required
                    minLength={field === 'password' ? 8 : undefined}
                    className={inputCls}
                  />
                </div>
              ))}
              <button
                type="submit"
                disabled={isLoading}
                className="w-full h-10 rounded-lg bg-primary text-on-primary text-[13px] font-medium hover:bg-primary-container transition-colors disabled:opacity-60 flex items-center justify-center gap-2"
              >
                {isLoading && <Loader2 size={14} className="animate-spin" />}
                {isLoading ? 'Creating account...' : 'Continue'}
              </button>
            </form>
            <p className="mt-6 text-center text-[13px] text-outline">
              Already have an account?{' '}
              <a href="/login" className="text-primary hover:underline font-medium">Sign in</a>
            </p>
          </>
        ) : (
          <>
            <h2 className="text-[18px] font-semibold text-on-surface mb-1">Create your organization</h2>
            <p className="text-[13px] text-outline mb-6">
              Your organization is the workspace for your team's backup infrastructure.
            </p>
            <form onSubmit={handleOrgSubmit} className="flex flex-col gap-4">
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
                  className={inputCls}
                  autoFocus
                />
              </div>
              <button
                type="submit"
                disabled={isLoading || !orgName.trim()}
                className="w-full h-10 rounded-lg bg-primary text-on-primary text-[13px] font-medium hover:bg-primary-container transition-colors disabled:opacity-60 flex items-center justify-center gap-2"
              >
                {isLoading && <Loader2 size={14} className="animate-spin" />}
                {isLoading ? 'Creating...' : 'Create Organization & Continue'}
              </button>
            </form>
          </>
        )}
      </div>
    </div>
  )
}

export default RegisterPage
