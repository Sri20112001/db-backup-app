import Modal from '@/components/ui/modal/Modal'
import type { PreflightResult } from '@/types'

interface PreflightModalProps {
  result: PreflightResult
  onClose: () => void
  onRunAnyway: () => void
}

const PreflightModal = ({ result, onClose, onRunAnyway }: PreflightModalProps) => {
  return (
    <Modal isOpen onClose={onClose} title="Pre-flight Checks" size="md">
      <div className="flex flex-col gap-4">
        <div className="flex flex-col gap-2">
          {result.checks.map((ch, i) => (
            <div
              key={`${i}-${ch.name}`}
              className="flex items-start gap-3 p-2.5 rounded-lg bg-surface-container-low"
            >
              <span
                className={`mt-0.5 text-[13px] font-bold ${
                  ch.status === 'OK'
                    ? 'text-emerald-600'
                    : ch.status === 'WARN'
                    ? 'text-amber-500'
                    : 'text-red-600'
                }`}
              >
                {ch.status === 'OK' ? '✓' : ch.status === 'WARN' ? '⚠' : '✗'}
              </span>
              <div>
                <p className="text-[13px] font-medium text-on-surface">{ch.name}</p>
                {ch.detail && <p className="text-[12px] text-on-surface-variant mt-0.5">{ch.detail}</p>}
              </div>
            </div>
          ))}
        </div>
        {result.overall === 'FAIL' ? (
          <p className="text-[13px] text-red-600 font-medium">Fix the issues above before running this backup.</p>
        ) : (
          <div className="flex gap-3 justify-end pt-2">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 rounded-lg text-[13px] text-on-surface-variant hover:bg-surface-container-high"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={onRunAnyway}
              className="px-4 py-2 rounded-lg bg-amber-500 text-on-primary text-[13px] font-medium hover:bg-amber-600"
            >
              Run anyway
            </button>
          </div>
        )}
      </div>
    </Modal>
  )
}

export default PreflightModal
