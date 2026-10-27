import type { ReactNode } from 'react'
import Modal from './Modal'

export interface DialogProps {
  isOpen?: boolean
  title: ReactNode
  message: ReactNode
  confirmLabel?: string
  cancelLabel?: string
  danger?: boolean
  isSubmitting?: boolean
  onConfirm: () => void
  onCancel: () => void
}

export function Dialog({
  isOpen = true,
  title,
  message,
  confirmLabel = 'Confirm',
  cancelLabel = 'Cancel',
  danger = false,
  isSubmitting = false,
  onConfirm,
  onCancel,
}: DialogProps) {
  return (
    <Modal isOpen={isOpen} onClose={onCancel} size="md" hideCloseButton>
      <div className="flex flex-col gap-2">
        {typeof title === 'string' ? (
          <h2 className="text-[16px] font-semibold text-on-surface">{title}</h2>
        ) : (
          title
        )}
        <div className="text-[14px] text-on-surface-variant mb-4">{message}</div>
        <div className="flex items-center justify-end gap-3 pt-2">
          <button
            type="button"
            onClick={onCancel}
            disabled={isSubmitting}
            className="px-4 h-9 rounded-lg bg-surface-container-lowest border border-surface-variant text-on-surface text-[13px] font-medium hover:bg-surface-container-low transition-colors disabled:opacity-50"
          >
            {cancelLabel}
          </button>
          <button
            type="button"
            onClick={onConfirm}
            disabled={isSubmitting}
            className={`px-4 h-9 rounded-lg text-[13px] font-medium transition-colors disabled:opacity-50 ${
              danger
                ? 'bg-surface-container-lowest border border-error text-error hover:bg-error-container'
                : 'bg-primary text-on-primary hover:bg-primary-container'
            }`}
          >
            {isSubmitting ? 'Processing...' : confirmLabel}
          </button>
        </div>
      </div>
    </Modal>
  )
}

export default Dialog
