import type { HTMLAttributes } from 'react'

export interface CardHeaderProps extends HTMLAttributes<HTMLDivElement> {
  action?: React.ReactNode
}

export function CardHeader({ action, className = '', children, ...props }: CardHeaderProps) {
  return (
    <div className={`flex items-start justify-between gap-4 mb-3 ${className}`} {...props}>
      <div className="flex-1 min-w-0">{children}</div>
      {action && <div className="shrink-0">{action}</div>}
    </div>
  )
}

export default CardHeader
