interface AlertBatchBarProps {
  selectedCount: number
  isBatchProcessing: boolean
  onAck: () => void
  onResolve: () => void
  onMarkRead: () => void
  onCancel: () => void
}

export const AlertBatchBar = ({
  selectedCount,
  isBatchProcessing,
  onAck,
  onResolve,
  onMarkRead,
  onCancel,
}: AlertBatchBarProps) => {
  if (selectedCount === 0) return null

  return (
    <div className="sticky top-0 z-20 flex items-center justify-between gap-3 p-3 rounded-xl bg-primary text-on-primary shadow-lg animate-in slide-in-from-top-2 duration-150">
      <div className="flex items-center gap-2 text-[13px] font-medium">
        <span>
          {selectedCount} alert{selectedCount > 1 ? 's' : ''} selected
        </span>
      </div>

      <div className="flex items-center gap-2">
        <button
          type="button"
          disabled={isBatchProcessing}
          onClick={onAck}
          className="px-3 py-1.5 rounded-lg bg-surface-container-lowest/20 hover:bg-surface-container-lowest/30 text-[12.5px] font-medium transition-colors disabled:opacity-50"
        >
          Acknowledge
        </button>
        <button
          type="button"
          disabled={isBatchProcessing}
          onClick={onResolve}
          className="px-3 py-1.5 rounded-lg bg-surface-container-lowest/20 hover:bg-surface-container-lowest/30 text-[12.5px] font-medium transition-colors disabled:opacity-50"
        >
          Resolve
        </button>
        <button
          type="button"
          disabled={isBatchProcessing}
          onClick={onMarkRead}
          className="px-3 py-1.5 rounded-lg bg-surface-container-lowest/20 hover:bg-surface-container-lowest/30 text-[12.5px] font-medium transition-colors disabled:opacity-50"
        >
          Mark Read
        </button>
        <button
          type="button"
          onClick={onCancel}
          className="px-2.5 py-1.5 text-[12px] underline opacity-90 hover:opacity-100 transition-opacity"
        >
          Cancel
        </button>
      </div>
    </div>
  )
}

export default AlertBatchBar
