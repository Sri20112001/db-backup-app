import { X } from 'lucide-react'
import JobWizard from './JobWizard'

interface Props {
  onClose: () => void
  onSaved: () => void
}

const NewJobModal = ({ onClose, onSaved }: Props) => {
  return (
    <div className="fixed inset-0 z-[200] flex items-center justify-center">
      <div className="absolute inset-0 bg-[#141b2b]/30 backdrop-blur-[2px]" onClick={onClose} />
      <div className="relative bg-[#ffffff] rounded-xl shadow-2xl w-full max-w-3xl mx-4 border border-[#e9edff] max-h-[92vh] overflow-y-auto">
        <div className="flex items-center justify-between px-6 py-4 border-b border-[#e9edff] sticky top-0 bg-[#ffffff] z-10 rounded-t-xl">
          <h2 className="text-[16px] font-semibold text-[#141b2b]">Create Backup Job</h2>
          <button
            type="button"
            onClick={onClose}
            className="p-1.5 rounded-lg text-[#434655] hover:bg-[#e9edff] transition-colors"
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
