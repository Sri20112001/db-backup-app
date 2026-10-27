import type { HTMLAttributes } from 'react'

export function CardDescription({ className = '', children, ...props }: HTMLAttributes<HTMLParagraphElement>) {
  return (
    <p className={`text-[13px] text-on-surface-variant mt-0.5 ${className}`} {...props}>
      {children}
    </p>
  )
}

export default CardDescription
