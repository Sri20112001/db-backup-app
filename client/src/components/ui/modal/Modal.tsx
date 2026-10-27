import { useEffect, useRef, type ReactNode } from 'react'
import { X } from 'lucide-react'

export interface ModalProps {
  isOpen?: boolean
  onClose: () => void
  title?: ReactNode
  description?: ReactNode
  children: ReactNode
  size?: 'sm' | 'md' | 'lg' | 'xl' | '2xl' | '3xl' | '4xl' | 'full'
  className?: string
  hideCloseButton?: boolean
  closeOnBackdrop?: boolean
  closeOnEscape?: boolean
  zIndex?: number
}

const SIZE_CLASSES = {
  sm: 'max-w-sm',
  md: 'max-w-md',
  lg: 'max-w-lg',
  xl: 'max-w-xl',
  '2xl': 'max-w-2xl',
  '3xl': 'max-w-3xl',
  '4xl': 'max-w-4xl',
  full: 'max-w-[96vw] max-h-[96vh]',
}

export function Modal({
  isOpen = true,
  onClose,
  title,
  description,
  children,
  size = 'md',
  className = '',
  hideCloseButton = false,
  closeOnBackdrop = true,
  closeOnEscape = true,
  zIndex = 200,
}: ModalProps) {
  const closeRef = useRef<HTMLButtonElement>(null)
  const onCloseRef = useRef(onClose)

  useEffect(() => {
    onCloseRef.current = onClose
  })

  useEffect(() => {
    if (!isOpen) return
    const prevActive = document.activeElement as HTMLElement | null
    closeRef.current?.focus()

    const onKey = (e: KeyboardEvent) => {
      if (closeOnEscape && e.key === 'Escape') {
        onCloseRef.current()
      }
    }

    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('keydown', onKey)
      prevActive?.focus?.()
    }
  }, [isOpen, closeOnEscape])

  if (!isOpen) return null

  const sizeClass = SIZE_CLASSES[size] ?? SIZE_CLASSES.md

  return (
    <div
      className="fixed inset-0 flex items-center justify-center p-4"
      style={{ zIndex }}
      role="dialog"
      aria-modal="true"
    >
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-on-surface/30 backdrop-blur-[2px] transition-opacity"
        onClick={() => {
          if (closeOnBackdrop) onClose()
        }}
        aria-hidden="true"
      />

      {/* Modal Dialog Box */}
      <div
        className={`relative w-full ${sizeClass} bg-surface-container-lowest rounded-xl shadow-2xl border border-surface-variant flex flex-col max-h-[92vh] overflow-hidden transition-all ${className}`}
      >
        {(title || !hideCloseButton) && (
          <div className="flex items-center justify-between px-6 py-4 border-b border-surface-variant sticky top-0 bg-surface-container-lowest z-10 shrink-0">
            <div>
              {typeof title === 'string' ? (
                <h2 className="text-[16px] font-semibold text-on-surface">{title}</h2>
              ) : (
                title
              )}
              {description && (
                <p className="text-[12px] text-on-surface-variant mt-0.5">{description}</p>
              )}
            </div>
            {!hideCloseButton && (
              <button
                ref={closeRef}
                type="button"
                onClick={onClose}
                aria-label="Close modal"
                className="p-1.5 rounded-lg text-on-surface-variant hover:text-on-surface hover:bg-surface-container-high transition-colors"
              >
                <X size={18} />
              </button>
            )}
          </div>
        )}

        <div className="p-6 overflow-y-auto min-h-0 flex-1">{children}</div>
      </div>
    </div>
  )
}

export default Modal
