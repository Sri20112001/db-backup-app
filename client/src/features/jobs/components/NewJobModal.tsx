import { X } from 'lucide-react'
import JobWizard from './JobWizard'

interface Props {
  onClose: () => void
  onSaved: () => void
}

const NewJobModal = ({ onClose, onSaved }: Props) => {
  return (
    <div className="fixed inset-0 z-[200] flex items-center justify-center">
      <div className="absolute inset-0 bg-on-surface/30 backdrop-blur-[2px]" onClick={onClose} />
      <div className="relative bg-surface-container-lowest rounded-xl shadow-2xl w-full max-w-3xl mx-4 border border-surface-variant max-h-[92vh] overflow-y-auto">
        <div className="flex items-center justify-between px-6 py-4 border-b border-surface-variant sticky top-0 bg-surface-container-lowest z-10 rounded-t-xl">
          <h2 className="text-[16px] font-semibold text-on-surface">Create Backup Job</h2>
          <button
            type="button"
            onClick={onClose}
            className="p-1.5 rounded-lg text-on-surface-variant hover:bg-surface-container-high transition-colors"
          >
            <X size={18} />
          </button>
        </div>
        <div className="p-6">
          <JobWizard onClose={onClose} onSaved={onSaved} />
        </div>
      </div>
    </div>
  )
}

export default NewJobModal
