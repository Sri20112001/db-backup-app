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
    <div className="absolute inset-0 bg-[#141b2b]/30 backdrop-blur-[2px]" onClick={onCancel} />
    <div className="relative bg-[#ffffff] rounded-xl shadow-2xl p-6 w-full max-w-md mx-4 border border-[#e9edff]">
      <h2 className="text-[16px] font-semibold text-[#141b2b] mb-2">{title}</h2>
      <p className="text-[14px] text-[#434655] mb-6">{message}</p>
      <div className="flex items-center justify-end gap-3">
        <button
          type="button"
          onClick={onCancel}
          className="px-4 h-9 rounded-lg bg-[#ffffff] border border-[#e9edff] text-[#141b2b] text-[13px] font-medium hover:bg-[#f1f3ff] transition-colors"
        >
          Cancel
        </button>
        <button
          type="button"
          onClick={onConfirm}
          className={`px-4 h-9 rounded-lg text-[13px] font-medium transition-colors ${
            danger
              ? 'bg-[#ffffff] border border-[#fca5a5] text-[#ba1a1a] hover:bg-[#ffdad6]'
              : 'bg-[#2563eb] text-white hover:bg-[#1d4ed8]'
          }`}
        >
          {confirmLabel}
        </button>
      </div>
    </div>
  </div>
)

export default ConfirmDialog
