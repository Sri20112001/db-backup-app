import { useUIStore } from '../store/uiStore'
import { CheckCircle, AlertCircle, AlertTriangle, Info, X } from 'lucide-react'

const borderColor = {
  success: 'border-[#006591]',
  error: 'border-[#ba1a1a]',
  warning: 'border-[#d97706]',
  info: 'border-primary',
}

const IconMap = {
  success: CheckCircle,
  error: AlertCircle,
  warning: AlertTriangle,
  info: Info,
}

const iconColor = {
  success: 'text-on-primary-container',
  error: 'text-error',
  warning: 'text-[#d97706]',
  info: 'text-primary',
}

const ToastContainer = () => {
  const { toasts, removeToast } = useUIStore()

  if (toasts.length === 0) return null

  return (
    <div className="fixed bottom-6 right-6 z-9999 flex flex-col gap-2">
      {toasts.map((t) => {
        const Icon = IconMap[t.type]
        return (
          <div
            key={t.id}
            className={`flex items-center gap-3 px-4 py-3 bg-surface-container-lowest border border-surface-variant border-l-4 ${borderColor[t.type]} rounded-lg shadow-lg min-w-[280px] max-w-[380px]`}
          >
            <Icon size={16} className={`shrink-0 ${iconColor[t.type]}`} />
            <span className="text-[13px] text-on-surface flex-1">{t.message}</span>
            <button
              type="button"
              onClick={() => removeToast(t.id)}
              className="text-outline hover:text-on-surface transition-colors"
            >
              <X size={14} />
            </button>
          </div>
        )
      })}
    </div>
  )
}

export default ToastContainer
