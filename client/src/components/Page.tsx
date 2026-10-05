import type { ReactNode } from 'react'

// Page is the single root container every full-page view must use, so the
// app layout never shifts between routes: full-bleed column, consistent
// rhythm (gap-4), scroll only when opted in. No max-width caps here —
// centered narrow columns belong to modals, not pages.
export const Page = ({
  children,
  scroll = false,
  className = '',
}: {
  children: ReactNode
  scroll?: boolean
  className?: string
}) => (
  <div className={`h-full min-h-0 flex flex-col gap-4 ${scroll ? 'overflow-y-auto pr-0.5 pb-1' : ''} ${className}`}>
    {children}
  </div>
)

interface PageHeaderProps {
  eyebrow?: ReactNode
  title: ReactNode
  description?: ReactNode
  actions?: ReactNode
}

// PageHeader is the standard page title row: optional eyebrow (breadcrumbs),
// 20px semibold title with optional inline badge, 12px description, and an
// optional right-aligned actions slot.
export const PageHeader = ({ eyebrow, title, description, actions }: PageHeaderProps) => (
  <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
    <div className="min-w-0">
      {eyebrow}
      <h1 className="text-[20px] font-semibold text-on-surface tracking-tight flex items-center gap-2.5">
        {title}
      </h1>
      {description && (
        <p className="text-[12px] text-on-surface-variant mt-0.5">{description}</p>
      )}
    </div>
    {actions && <div className="flex items-center gap-2.5 shrink-0">{actions}</div>}
  </div>
)

export default Page
