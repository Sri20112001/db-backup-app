import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { authApi } from '@/services/api'
import { useUIStore } from '@/store/uiStore'
import { Shield, Loader2 } from 'lucide-react'

const RegisterPage = () => {
  const navigate = useNavigate()
  const { addToast } = useUIStore()
  const [form, setForm] = useState({ name: '', email: '', password: '' })
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) =>
    setForm((f) => ({ ...f, [e.target.name]: e.target.value }))

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setIsLoading(true)
    try {
      await authApi.register(form.email, form.password, form.name)
      addToast('success', 'Account created. Please sign in.')
      navigate('/login')
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Registration failed')
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

        <h2 className="text-[18px] font-semibold text-[#141b2b] mb-6">Create account</h2>

        {error && (
          <div className="mb-4 px-3 py-2.5 rounded-lg bg-[#ffdad6] border border-[#fca5a5] text-[#93000a] text-[13px]">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          {(['name', 'email', 'password'] as const).map((field) => (
            <div key={field}>
              <label className="block text-[13px] font-medium text-[#434655] mb-1.5" htmlFor={field}>
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
                className="w-full h-9 px-3 rounded-lg border border-[#e9edff] bg-[#f9f9ff] text-[14px] text-[#141b2b] placeholder:text-[#737686] focus:outline-none focus:ring-2 focus:ring-[#2563eb]/20 focus:border-[#2563eb] transition-all"
              />
            </div>
          ))}

          <button
            type="submit"
            disabled={isLoading}
            className="w-full h-10 rounded-lg bg-[#2563eb] text-white text-[13px] font-medium hover:bg-[#1d4ed8] transition-colors disabled:opacity-60 flex items-center justify-center gap-2"
          >
            {isLoading && <Loader2 size={14} className="animate-spin" />}
            {isLoading ? 'Creating account...' : 'Create account'}
          </button>
        </form>

        <p className="mt-6 text-center text-[13px] text-[#737686]">
          Already have an account?{' '}
          <a href="/login" className="text-[#004ac6] hover:underline font-medium">Sign in</a>
        </p>
      </div>
    </div>
  )
}

export default RegisterPage
