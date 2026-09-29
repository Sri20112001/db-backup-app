import { BadgeCheck, FolderArchive, LayoutGrid, Radio } from 'lucide-react'
import { SkeletonCard } from '@/components/Skeleton'
import { formatBytes } from '@/utils/format'
import type { DashboardOverview } from '@/types'
import MetricTile from './MetricTile'

interface MetricsRowProps {
  overview: DashboardOverview | null
  isLoading: boolean
}

// Single-row KPI strip (always 4 tiles). An SLA-breach count folds into the
// execution tile so the row never wraps to a second line.
const MetricsRow = ({ overview, isLoading }: MetricsRowProps) => {
  const breaches = overview?.sla_breaches_24h ?? 0
  const agentsTotal = (overview?.agents_online ?? 0) + (overview?.agents_offline ?? 0)
  
  return (
    <div className="grid grid-cols-2 xl:grid-cols-4 gap-3 shrink-0">
      {isLoading ? (
        [...Array(4)].map((_, i) => <SkeletonCard key={i} />)
      ) : (
        <>
          <MetricTile 
            Icon={LayoutGrid} iconBg="bg-primary-container" iconColor="text-primary"
            label="Backup Jobs" 
            value={overview?.total_jobs ?? 0}
            sub={`${overview?.enabled_jobs ?? 0} active`} 
          />
          <MetricTile 
            Icon={Radio} iconBg="bg-primary-container" iconColor="text-on-primary-container"
            label="Agents Online"
            value={overview?.agents_online ?? 0}
            valueFormatter={(val) => `${Math.round(val)}/${agentsTotal}`}
            sub={overview && overview.agents_offline > 0 ? `${overview.agents_offline} offline` : 'Fleet healthy'} 
          />
          <MetricTile 
            Icon={FolderArchive} iconBg="bg-primary-container/50" iconColor="text-on-primary-container"
            label="Storage" 
            value={overview?.total_bytes ?? 0}
            valueFormatter={formatBytes}
            sub="Total uploaded" 
          />
          <MetricTile 
            Icon={BadgeCheck} iconBg={breaches > 0 ? 'bg-red-100' : 'bg-primary-container'} iconColor={breaches > 0 ? 'text-red-600' : 'text-primary'}
            label="24h Runs" 
            value={overview?.successful_runs ?? 0}
            valueFormatter={(val) => `${Math.round(val)}✓ ${overview?.failed_runs ?? 0}✗`}
            sub={breaches > 0 ? `${breaches} SLA breach${breaches > 1 ? 'es' : ''}` : 'Successful / Failed'} 
          />
        </>
      )}
    </div>
  )
}

export default MetricsRow
