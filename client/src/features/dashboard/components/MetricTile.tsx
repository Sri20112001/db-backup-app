interface MetricTileProps {
  Icon: React.ElementType
  iconBg: string
  iconColor: string
  label: string
  value: string
  sub: string
}

const MetricTile = ({ Icon, iconBg, iconColor, label, value, sub }: MetricTileProps) => (
  <div className="flex items-center gap-3 px-3.5 py-2.5 rounded-lg bg-surface-container-lowest shadow-sm border border-surface-variant min-w-0">
    <div className={`w-8 h-8 rounded-lg ${iconBg} flex items-center justify-center ${iconColor} shrink-0`}>
      <Icon size={16} />
    </div>
    <div className="min-w-0">
      <div className="text-[17px] font-bold text-on-surface leading-none truncate">{value}</div>
      <div className="text-[10px] font-semibold uppercase tracking-wider text-on-surface-variant mt-1 truncate">{label}</div>
      <div className="text-[10px] text-outline truncate">{sub}</div>
    </div>
  </div>
)

export default MetricTile
