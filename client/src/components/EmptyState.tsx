import type { LucideIcon } from 'lucide-react'

interface Props {
  icon: LucideIcon
  title: string
  description: string
  action?: React.ReactNode
}

const EmptyState = ({ icon: Icon, title, description, action }: Props) => (
  <div className="flex flex-col flex-1 items-center justify-center py-16 text-center">
    <Icon size={48} className="text-[#c3c6d7] mb-4" />
    <h3 className="text-[16px] font-semibold text-on-surface mb-1">{title}</h3>
    <p className="text-[13px] text-outline max-w-sm mb-4">{description}</p>
    {action}
  </div>
)

export default EmptyState
