import type { LucideIcon } from 'lucide-react'
import CountUp from '@/components/ui/CountUp'

interface MetricTileProps {
  Icon: LucideIcon
  iconBg: string
  iconColor: string
  label: string
  value: React.ReactNode | number
  valueFormatter?: (val: number) => string
  sub: string
  hasAlert?: boolean
}

const MetricTile = ({
  Icon,
  iconBg,
  iconColor,
  label,
  value,
  valueFormatter,
  sub,
  hasAlert = false,
}: MetricTileProps) => {
  return (
    <div
      className={`group relative overflow-hidden rounded-2xl border p-4 sm:p-5 transition-all duration-300 ease-out hover:-translate-y-1 hover:shadow-lg ${
        hasAlert
          ? 'border-error/30 bg-error-container/30 shadow-[0_8px_30px_rgba(220,38,38,0.08)]'
          : 'border-outline-variant bg-surface-container-lowest shadow-sm hover:border-primary/40 hover:shadow-primary/10'
      } backdrop-blur-xl`}
    >
      {/* Dynamic ambient hover glow */}
      <div
        className={`pointer-events-none absolute -inset-px rounded-2xl opacity-0 transition-opacity duration-500 group-hover:opacity-100 ${
          hasAlert
            ? 'bg-[radial-gradient(ellipse_at_top,var(--tw-gradient-stops))] from-error/15 via-transparent to-transparent'
            : 'bg-[radial-gradient(ellipse_at_top,var(--tw-gradient-stops))] from-primary/10 via-transparent to-transparent'
        }`}
      />

      <div className="relative z-10 flex flex-col justify-between h-full gap-3">
        {/* Header: Label & Icon */}
        <div className="flex items-center justify-between">
          <span className="text-xs font-medium uppercase tracking-wider text-on-surface-variant group-hover:text-on-surface transition-colors">
            {label}
          </span>
          <div
            className={`flex h-9 w-9 items-center justify-center rounded-xl ring-1 ring-inset ring-outline-variant transition-transform duration-300 group-hover:scale-110 ${iconBg}`}
          >
            <Icon className={`h-4.5 w-4.5 ${iconColor}`} />
          </div>
        </div>

        {/* Content: CountUp / Animated value */}
        <div className="flex flex-col gap-0.5">
          <div className="text-2xl font-bold tracking-tight text-on-surface font-mono flex items-baseline gap-1">
            {typeof value === 'number' ? (
              <CountUp value={value} formatter={valueFormatter} duration={1200} />
            ) : (
              value
            )}
          </div>
          <span
            className={`text-xs font-medium transition-colors ${
              hasAlert ? 'text-error' : 'text-outline group-hover:text-on-surface-variant'
            }`}
          >
            {sub}
          </span>
        </div>
      </div>

      {/* Subtle top glare line */}
      <div className="pointer-events-none absolute inset-x-0 top-0 h-[1px] bg-gradient-to-r from-transparent via-outline-variant to-transparent opacity-50" />
    </div>
  )
}

export default MetricTile
