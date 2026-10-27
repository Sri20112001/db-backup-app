import type { HTMLAttributes } from 'react'

export interface CardTitleProps extends HTMLAttributes<HTMLHeadingElement> {
  as?: 'h1' | 'h2' | 'h3' | 'h4' | 'h5' | 'h6'
}

export function CardTitle({ as: Comp = 'h3', className = '', children, ...props }: CardTitleProps) {
  return (
    <Comp className={`text-[15px] font-semibold text-on-surface ${className}`} {...props}>
      {children}
    </Comp>
  )
}

export default CardTitle
