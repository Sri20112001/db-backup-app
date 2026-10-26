// Shared cron helpers for the schedule builder: turn simple, human choices
// (daily at 23:00, every Tuesday, every 6 hours…) into cron expressions and
// read them back, so users never have to know cron syntax to schedule work.

export type ScheduleFreq =
  | 'manual'
  | 'hourly'
  | 'daily'
  | 'weekly'
  | 'monthly'
  | 'interval'
  | 'custom'

export interface ScheduleFields {
  freq: ScheduleFreq
  time: string // "HH:MM", 24-hour
  weekday: number // 0 = Sunday … 6 = Saturday
  dom: number // day of month, 1–31
  intervalHours: number // 1–23
}

export const WEEKDAYS = [
  'Sunday',
  'Monday',
  'Tuesday',
  'Wednesday',
  'Thursday',
  'Friday',
  'Saturday',
] as const

// Sensible starting point for a new schedule (daily at 11 PM).
export const DEFAULT_SCHEDULE: ScheduleFields = {
  freq: 'daily',
  time: '23:00',
  weekday: 0,
  dom: 1,
  intervalHours: 6,
}

const clamp = (n: number, lo: number, hi: number) => Math.min(hi, Math.max(lo, n))

const timeParts = (time: string): { h: number; m: number } => {
  const [rawH, rawM] = time.split(':')
  const h = clamp(parseInt(rawH, 10) || 0, 0, 23)
  const m = clamp(parseInt(rawM, 10) || 0, 0, 59)
  return { h, m }
}

// 24-hour "23:00" → "11:00 PM" for previews and card labels.
export const formatTime12 = (time: string): string => {
  const { h, m } = timeParts(time)
  const suffix = h >= 12 ? 'PM' : 'AM'
  const hour12 = h % 12 === 0 ? 12 : h % 12
  return `${hour12}:${String(m).padStart(2, '0')} ${suffix}`
}

// Renders the builder state as the cron expression the scheduler executes.
export function buildCron(f: ScheduleFields): string {
  const { h, m } = timeParts(f.time)
  switch (f.freq) {
    case 'manual':
      return ''
    case 'hourly':
      return '0 * * * *'
    case 'daily':
      return `${m} ${h} * * *`
    case 'weekly':
      return `${m} ${h} * * ${clamp(f.weekday, 0, 6)}`
    case 'monthly':
      return `${m} ${h} ${clamp(f.dom, 1, 31)} * *`
    case 'interval':
      // Hourly step via "*/N"; 24+ hours would be a daily schedule instead.
      return `0 */${clamp(f.intervalHours, 1, 23)} * * *`
    default:
      return ''
  }
}

// Reads a cron expression back into builder fields. Unrecognized shapes come
// back as freq 'custom' so the raw field keeps the user's exact text.
export function parseSchedule(cron: string): ScheduleFields {
  const v = (cron ?? '').trim()
  const base = { ...DEFAULT_SCHEDULE }

  if (!v) return { ...base, freq: 'manual', time: '00:00' }

  // @-descriptors map onto the same builder states.
  switch (v) {
    case '@hourly':
      return { ...base, freq: 'hourly' }
    case '@daily':
    case '@midnight':
      return { ...base, freq: 'daily', time: '00:00' }
    case '@weekly':
      return { ...base, freq: 'weekly', time: '00:00', weekday: 0 }
    case '@monthly':
      return { ...base, freq: 'monthly', time: '00:00', dom: 1 }
    default:
      break
  }
  if (v.startsWith('@')) return { ...base, freq: 'custom' }

  const asTime = (m: string, h: string): string | null => {
    const mi = Number(m)
    const hi = Number(h)
    if (!Number.isInteger(mi) || mi < 0 || mi > 59) return null
    if (!Number.isInteger(hi) || hi < 0 || hi > 23) return null
    return `${String(hi).padStart(2, '0')}:${String(mi).padStart(2, '0')}`
  }

  let match: RegExpMatchArray | null

  if (v === '0 * * * *') return { ...base, freq: 'hourly' }

  // daily: "M H * * *"
  if ((match = v.match(/^(\d{1,2}) (\d{1,2}) \* \* \*$/))) {
    const time = asTime(match[1], match[2])
    if (time) return { ...base, freq: 'daily', time }
  }

  // weekly: "M H * * D"
  if ((match = v.match(/^(\d{1,2}) (\d{1,2}) \* \* (\d)$/))) {
    const time = asTime(match[1], match[2])
    const wd = Number(match[3])
    if (time && wd >= 0 && wd <= 7) {
      return { ...base, freq: 'weekly', time, weekday: wd === 7 ? 0 : wd }
    }
  }

  // monthly: "M H D * *"
  if ((match = v.match(/^(\d{1,2}) (\d{1,2}) (\d{1,2}) \* \*$/))) {
    const time = asTime(match[1], match[2])
    const dom = Number(match[3])
    if (time && dom >= 1 && dom <= 31) return { ...base, freq: 'monthly', time, dom }
  }

  // every N hours: "0 */N * * *"
  if ((match = v.match(/^0 \*\/(\d{1,2}) \* \* \*$/))) {
    const n = Number(match[1])
    if (n >= 1 && n <= 23) return { ...base, freq: 'interval', intervalHours: n }
  }

  return { ...base, freq: 'custom' }
}

// Plain-language description of a cron expression for previews and list rows.
export function describeSchedule(cron: string, timezone = 'UTC'): string {
  const issue = cronIssue(cron)
  if (issue) return `Schedule needs attention — ${issue}`

  const f = parseSchedule(cron)
  switch (f.freq) {
    case 'manual':
      return 'Manual — no schedule; runs only when started by hand.'
    case 'hourly':
      return `Every hour, on the hour (${timezone})`
    case 'daily':
      return `Every day at ${formatTime12(f.time)} (${timezone})`
    case 'weekly':
      return `Every ${WEEKDAYS[f.weekday]} at ${formatTime12(f.time)} (${timezone})`
    case 'monthly':
      return `Every month on day ${f.dom} at ${formatTime12(f.time)} (${timezone})`
    case 'interval':
      return `Every ${f.intervalHours} hour${f.intervalHours === 1 ? '' : 's'}`
    default:
      return `Custom schedule: ${cron.trim()}`
  }
}

// Lightweight shape check so a typo is caught before it reaches the
// scheduler. Empty (= manual) and @-descriptors (@hourly, @daily…) pass.
export function cronIssue(raw: string): string | null {
  const v = (raw ?? '').trim()
  if (!v || v.startsWith('@')) return null
  const fields = v.split(/\s+/)
  if (fields.length !== 5) {
    return 'A cron expression has 5 fields: minute hour day-of-month month day-of-week — for example 0 2 * * *'
  }
  return null
}
