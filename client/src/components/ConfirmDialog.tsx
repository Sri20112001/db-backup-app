interface Props {
  title: string
  message: string
  confirmLabel?: string
  danger?: boolean
  onConfirm: () => void
  onCancel: () => void
}

const ConfirmDialog = ({ title, message, confirmLabel = 'Confirm', danger = false, onConfirm, onCancel }: Props) => (
  <div className="fixed inset-0 z-[200] flex items-center justify-center">
    <div className="absolute inset-0 bg-on-surface/30 backdrop-blur-[2px]" onClick={onCancel} />
    <div className="relative bg-surface-container-lowest rounded-xl shadow-2xl p-6 w-full max-w-md mx-4 border border-surface-variant">
      <h2 className="text-[16px] font-semibold text-on-surface mb-2">{title}</h2>
      <p className="text-[14px] text-on-surface-variant mb-6">{message}</p>
      <div className="flex items-center justify-end gap-3">
        <button
          type="button"
          onClick={onCancel}
          className="px-4 h-9 rounded-lg bg-surface-container-lowest border border-surface-variant text-on-surface text-[13px] font-medium hover:bg-surface-container-low transition-colors"
        >
          Cancel
        </button>
        <button
          type="button"
          onClick={onConfirm}
          className={`px-4 h-9 rounded-lg text-[13px] font-medium transition-colors ${
            danger
              ? 'bg-surface-container-lowest border border-error text-error hover:bg-error-container'
              : 'bg-primary text-on-primary hover:bg-primary-container'
          }`}
        >
          {confirmLabel}
        </button>
      </div>
    </div>
  </div>
)

export default ConfirmDialog
