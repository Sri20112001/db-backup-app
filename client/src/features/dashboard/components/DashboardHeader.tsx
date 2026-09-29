import { Plus } from 'lucide-react'
import type { DashboardOverview } from '@/types'

interface DashboardHeaderProps {
  overview: DashboardOverview | null
  onNewJob: () => void
}

const DashboardHeader = ({ overview, onNewJob }: DashboardHeaderProps) => (
  <div className="flex items-center justify-between gap-3 shrink-0">
    <div className="flex items-baseline gap-3 min-w-0">
      <h1 className="text-[18px] font-semibold text-on-surface tracking-tight whitespace-nowrap">Command Center</h1>
      <p className="text-[11px] text-on-surface-variant truncate hidden xl:block">
        Live status of backup infrastructure, agent nodes, and replication pipelines.
      </p>
    </div>
    <div className="flex items-center gap-2 shrink-0">
      {overview && (
        <div className="flex items-center gap-2 px-3 py-1 rounded-full bg-surface-container-lowest shadow-sm border border-surface-variant">
          <span className="relative flex h-2 w-2">
            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-[#39b8fd] opacity-75" />
            <span className="relative inline-flex rounded-full h-2 w-2 bg-primary" />
          </span>
          <span className="text-[12px] font-medium text-on-surface whitespace-nowrap">
            Agents: <strong className="text-on-primary-container">{overview.agents_online}/{overview.agents_online + overview.agents_offline} Online</strong>
          </span>
        </div>
      )}
      <button
        type="button"
        onClick={onNewJob}
        className="flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg bg-primary text-on-primary text-[13px] font-medium hover:bg-primary-container transition-colors shadow-sm whitespace-nowrap"
      >
        <Plus size={15} />
        New Backup Job
      </button>
    </div>
  </div>
)

export default DashboardHeader
