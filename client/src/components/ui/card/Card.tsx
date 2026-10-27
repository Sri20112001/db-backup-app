import { forwardRef, type HTMLAttributes } from 'react'

export interface CardProps extends HTMLAttributes<HTMLDivElement> {
  variant?: 'default' | 'flat' | 'interactive' | 'sunken'
  padding?: 'none' | 'sm' | 'md' | 'lg'
}

const VARIANT_CLASSES = {
  default: 'bg-surface-container-lowest border border-surface-variant shadow-sm',
  flat: 'bg-surface-container-lowest border border-surface-variant',
  interactive: 'bg-surface-container-lowest border border-surface-variant shadow-sm hover:border-primary/50 transition-all cursor-pointer',
  sunken: 'bg-surface-container-low border border-surface-variant',
}

const PADDING_CLASSES = {
  none: '',
  sm: 'p-3',
  md: 'p-4 sm:p-5',
  lg: 'p-6',
}

export const Card = forwardRef<HTMLDivElement, CardProps>(
  ({ variant = 'default', padding = 'md', className = '', children, ...props }, ref) => {
    return (
      <div
        ref={ref}
        className={`rounded-xl ${VARIANT_CLASSES[variant]} ${PADDING_CLASSES[padding]} ${className}`}
        {...props}
      >
        {children}
      </div>
    )
  }
)

Card.displayName = 'Card'

export default Card
