import type { ReactNode } from 'react'

interface StepShellProps {
  title: string
  subtitle?: string
  children: ReactNode
}

// Shared heading + width wrapper so every wizard step lines up with the rail.
const StepShell = ({ title, subtitle, children }: StepShellProps) => (
  <div className="w-full max-w-xl mx-auto flex flex-col gap-5">
    <div>
      <h2 className="text-[20px] font-semibold text-on-surface">{title}</h2>
      {subtitle && <p className="text-[13px] text-on-surface-variant mt-1">{subtitle}</p>}
    </div>
    {children}
  </div>
)

export default StepShell
