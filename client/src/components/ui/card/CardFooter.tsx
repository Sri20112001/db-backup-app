import type { HTMLAttributes } from 'react'

export function CardFooter({ className = '', children, ...props }: HTMLAttributes<HTMLDivElement>) {
  return (
    <div className={`flex items-center justify-between gap-3 pt-3 mt-3 border-t border-surface-variant ${className}`} {...props}>
      {children}
    </div>
  )
}

export default CardFooter
