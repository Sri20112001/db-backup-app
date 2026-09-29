// Shared SLA / RPO display helpers for the dashboard feature.

export function slaColor(s: string) {
  if (s === 'OK') return 'text-emerald-600'
  if (s === 'BREACH') return 'text-red-600'
  return 'text-slate-400'
}

export function slaLabel(s: string) {
  if (s === 'OK') return '✓'
  if (s === 'BREACH') return '✗ BREACH'
  return '—'
}

export function fmtMinutes(m: number) {
  if (m <= 0) return '—'
  if (m < 60) return `${m}m`
  const h = Math.floor(m / 60)
  const rem = m % 60
  return rem > 0 ? `${h}h ${rem}m` : `${h}h`
}
