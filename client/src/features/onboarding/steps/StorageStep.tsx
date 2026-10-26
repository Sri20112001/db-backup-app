import { useState } from 'react'
import { storageApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import StorageTargetForm from '@/features/storage/components/StorageTargetForm'
import StepShell from '../components/StepShell'

interface StorageStepProps {
  onDone: () => void
  onBack: () => void
}

const StorageStep = ({ onDone, onBack }: StorageStepProps) => {
  const { currentOrg } = useAuthStore()
  const { addToast } = useUIStore()
  const [isSubmitting, setIsSubmitting] = useState(false)

  const handleSubmit = async (data: Record<string, unknown>) => {
    if (!currentOrg) return
    setIsSubmitting(true)
    try {
      await storageApi.create(currentOrg.id, data)
      addToast('success', 'Storage target added')
      onDone()
    } catch {
      addToast('error', 'Failed to add storage target')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <StepShell title="Add a storage target" subtitle="Backups need somewhere to go. Configure an S3-compatible bucket or a local path.">
      <StorageTargetForm onSubmit={handleSubmit} isSubmitting={isSubmitting} submitLabel="Save & continue" />
      <button type="button" onClick={onBack} className="self-start text-[13px] text-on-surface-variant hover:text-on-surface transition-colors">
        ← Back
      </button>
    </StepShell>
  )
}

export default StorageStep
