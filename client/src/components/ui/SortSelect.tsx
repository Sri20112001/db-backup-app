import { ArrowUpDown } from 'lucide-react'

export interface SortOption {
  label: string
  value: string
}

interface SortSelectProps {
  value: string
  options: SortOption[]
  onChange: (value: string) => void
}

// Compact sort dropdown for card lists (jobs, agents, restores, storage).
const SortSelect = ({ value, options, onChange }: SortSelectProps) => (
  <label className="inline-flex items-center gap-1.5 pl-2.5 pr-1.5 h-8 rounded-lg bg-surface-container-low text-on-surface-variant shrink-0">
    <ArrowUpDown size={13} className="shrink-0" />
    <span className="text-[12px] font-medium whitespace-nowrap">Sort</span>
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className="h-full bg-transparent text-[12px] font-medium text-on-surface outline-none cursor-pointer max-w-36"
    >
      {options.map((o) => (
        <option key={o.value} value={o.value}>
          {o.label}
        </option>
      ))}
    </select>
  </label>
)

export default SortSelect
