import { useState } from 'react'
import { Loader2, Plus } from 'lucide-react'
import { orgApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'

interface Props {
  onCreated?: () => void
  autoFocus?: boolean
}

const inputCls =
  'w-full h-9 px-3 rounded-lg border border-[#e9edff] bg-[#f9f9ff] text-[14px] text-[#141b2b] placeholder:text-[#737686] focus:outline-none focus:ring-2 focus:ring-[#2563eb]/20 focus:border-[#2563eb] transition-all'

const CreateOrgForm = ({ onCreated, autoFocus = false }: Props) => {
  const { setCurrentOrg } = useAuthStore()
  const { addToast } = useUIStore()
  const [name, setName] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    const trimmed = name.trim()
    if (!trimmed) return
    setError('')
    setIsLoading(true)
    try {
      const org = await orgApi.create(trimmed)
      setCurrentOrg(org)
      addToast('success', `Organization "${org.name}" created`)
      setName('')
      onCreated?.()
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Failed to create organization'
      setError(msg)
      addToast('error', msg)
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-2">
      <input
        type="text"
        value={name}
        onChange={(e) => setName(e.target.value)}
        placeholder="Organization name"
        autoFocus={autoFocus}
        maxLength={100}
        className={inputCls}
      />
      {error && <p className="text-[12px] text-[#ba1a1a]">{error}</p>}
      <button
        type="submit"
        disabled={isLoading || !name.trim()}
        className="w-full h-9 rounded-lg bg-[#2563eb] text-white text-[13px] font-medium hover:bg-[#1d4ed8] transition-colors disabled:opacity-60 disabled:cursor-not-allowed flex items-center justify-center gap-2"
      >
        {isLoading ? <Loader2 size={14} className="animate-spin" /> : <Plus size={14} />}
        {isLoading ? 'Creating...' : 'Create organization'}
      </button>
    </form>
  )
}

export default CreateOrgForm
