import { useEffect, useRef } from 'react'
import { X } from 'lucide-react'
import type { PreflightResult } from '@/types'

interface PreflightModalProps {
  result: PreflightResult
  onClose: () => void
  onRunAnyway: () => void
}

// Real dialog semantics: focus moves into the dialog on open, Escape and
// backdrop clicks dismiss it, and focus returns to the trigger on close.
const PreflightModal = ({ result, onClose, onRunAnyway }: PreflightModalProps) => {
  const closeRef = useRef<HTMLButtonElement>(null)
  const onCloseRef = useRef(onClose)

  useEffect(() => {
    onCloseRef.current = onClose
  })

  useEffect(() => {
    const previous = document.activeElement as HTMLElement | null
    closeRef.current?.focus()
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onCloseRef.current()
    }
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('keydown', onKey)
      previous?.focus?.()
    }
  }, [])

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose()
      }}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="preflight-title"
        className="bg-surface-container-lowest rounded-xl shadow-xl w-full max-w-md p-6 flex flex-col gap-4"
      >
        <div className="flex items-center justify-between">
          <h2 id="preflight-title" className="text-[15px] font-semibold text-on-surface">Pre-flight Checks</h2>
          <button
            ref={closeRef}
            type="button"
            onClick={onClose}
            aria-label="Close pre-flight checks"
            className="text-on-surface-variant hover:text-on-surface"
          >
            <X size={18} />
          </button>
        </div>
        <div className="flex flex-col gap-2">
          {result.checks.map((ch, i) => (
            <div key={`${i}-${ch.name}`} className="flex items-start gap-3 p-2.5 rounded-lg bg-surface-container-low">
              <span className={`mt-0.5 text-[13px] font-bold ${ch.status === 'OK' ? 'text-emerald-600' : ch.status === 'WARN' ? 'text-amber-500' : 'text-red-600'}`}>
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
          <div className="flex gap-3 justify-end">
            <button type="button" onClick={onClose} className="px-4 py-2 rounded-lg text-[13px] text-on-surface-variant hover:bg-surface-container-high">Cancel</button>
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
    </div>
  )
}

export default PreflightModal
