import { ArrowDown, ArrowUp, ChevronsUpDown } from 'lucide-react'
import type { SortDir } from '@/hooks/useSort'

interface SortableThProps {
  label: string
  sortKey: string
  activeKey: string
  dir: SortDir
  onToggle: (key: string) => void
  className?: string
}

// Table header cell that toggles useSort ordering. Asc/desc arrow when
// active, muted affordance otherwise.
const SortableTh = ({
  label,
  sortKey,
  activeKey,
  dir,
  onToggle,
  className = 'py-2.5 px-4 whitespace-nowrap',
}: SortableThProps) => {
  const active = activeKey === sortKey
  return (
    <th className={className}>
      <button
        type="button"
        onClick={() => onToggle(sortKey)}
        title={`Sort by ${label}`}
        className={`inline-flex items-center gap-1 uppercase tracking-wider transition-colors ${
          active ? 'text-primary' : 'hover:text-on-surface'
        }`}
      >
        {label}
        {active ? (
          dir === 'asc' ? <ArrowUp size={12} /> : <ArrowDown size={12} />
        ) : (
          <ChevronsUpDown size={12} className="opacity-40" />
        )}
      </button>
    </th>
  )
}

export default SortableTh
