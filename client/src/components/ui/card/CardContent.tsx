import type { HTMLAttributes } from 'react'

export function CardContent({ className = '', children, ...props }: HTMLAttributes<HTMLDivElement>) {
  return (
    <div className={`text-[14px] text-on-surface ${className}`} {...props}>
      {children}
    </div>
  )
}

export default CardContent
