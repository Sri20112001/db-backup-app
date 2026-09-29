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
      className={`group relative overflow-hidden rounded-2xl border p-4 sm:p-5 transition-all duration-300 ease-out hover:-translate-y-1 hover:shadow-xl ${
        hasAlert
          ? 'border-red-500/30 bg-linear-to-b from-red-950/20 via-zinc-900/60 to-zinc-950/80 shadow-[0_8px_30px_rgb(239,68,68,0.08)]'
          : 'border-white/10 bg-linear-to-b from-zinc-800/40 via-zinc-900/60 to-zinc-950/80 shadow-[0_8px_30px_rgb(0,0,0,0.25)] hover:border-cyan-500/40 hover:shadow-cyan-500/10'
      } backdrop-blur-xl`}
    >
      {/* Dynamic ambient hover glow */}
      <div
        className={`pointer-events-none absolute -inset-px rounded-2xl opacity-0 transition-opacity duration-500 group-hover:opacity-100 ${
          hasAlert
            ? 'bg-[radial-gradient(ellipse_at_top,var(--tw-gradient-stops))] from-red-500/15 via-transparent to-transparent'
            : 'bg-[radial-gradient(ellipse_at_top,var(--tw-gradient-stops))] from-cyan-500/15 via-transparent to-transparent'
        }`}
      />

      <div className="relative z-10 flex flex-col justify-between h-full gap-3">
        {/* Header: Label & Icon */}
        <div className="flex items-center justify-between">
          <span className="text-xs font-medium uppercase tracking-wider text-zinc-400 group-hover:text-zinc-200 transition-colors">
            {label}
          </span>
          <div
            className={`flex h-9 w-9 items-center justify-center rounded-xl ring-1 ring-inset ring-white/10 transition-transform duration-300 group-hover:scale-110 ${iconBg}`}
          >
            <Icon className={`h-4.5 w-4.5 ${iconColor}`} />
          </div>
        </div>

        {/* Content: CountUp / Animated value */}
        <div className="flex flex-col gap-0.5">
          <div className="text-2xl font-bold tracking-tight text-white font-mono flex items-baseline gap-1">
            {typeof value === 'number' ? (
              <CountUp value={value} formatter={valueFormatter} duration={1200} />
            ) : (
              value
            )}
          </div>
          <span
            className={`text-xs font-medium transition-colors ${
              hasAlert ? 'text-red-400' : 'text-zinc-400 group-hover:text-zinc-300'
            }`}
          >
            {sub}
          </span>
        </div>
      </div>

      {/* Subtle top glare line */}
      <div className="pointer-events-none absolute inset-x-0 top-0 h-[1px] bg-gradient-to-r from-transparent via-white/20 to-transparent" />
    </div>
  )
}

export default MetricTile
