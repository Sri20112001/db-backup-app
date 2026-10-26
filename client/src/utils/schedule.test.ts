import { describe, expect, it } from 'vitest'
import {
  DEFAULT_SCHEDULE,
  buildCron,
  cronIssue,
  describeSchedule,
  formatTime12,
  parseSchedule,
  type ScheduleFields,
} from './schedule'

const fields = (over: Partial<ScheduleFields> = {}): ScheduleFields => ({ ...DEFAULT_SCHEDULE, ...over })

describe('buildCron', () => {
  it('renders each frequency as the cron the scheduler executes', () => {
    expect(buildCron(fields({ freq: 'manual' }))).toBe('')
    expect(buildCron(fields({ freq: 'hourly' }))).toBe('0 * * * *')
    expect(buildCron(fields({ freq: 'daily', time: '23:00' }))).toBe('0 23 * * *')
    expect(buildCron(fields({ freq: 'weekly', time: '02:30', weekday: 2 }))).toBe('30 2 * * 2')
    expect(buildCron(fields({ freq: 'monthly', time: '04:15', dom: 15 }))).toBe('15 4 15 * *')
    expect(buildCron(fields({ freq: 'interval', intervalHours: 6 }))).toBe('0 */6 * * *')
  })

  it('clamps out-of-range picks instead of producing invalid cron', () => {
    // Day-of-month 0 or 40 → 1…31; weekday 9 → Saturday; interval 0/30 → 1…23.
    expect(buildCron(fields({ freq: 'monthly', time: '04:00', dom: 0 }))).toBe('0 4 1 * *')
    expect(buildCron(fields({ freq: 'monthly', time: '04:00', dom: 40 }))).toBe('0 4 31 * *')
    expect(buildCron(fields({ freq: 'weekly', time: '04:00', weekday: 9 }))).toBe('0 4 * * 6')
    expect(buildCron(fields({ freq: 'interval', intervalHours: 0 }))).toBe('0 */1 * * *')
    expect(buildCron(fields({ freq: 'interval', intervalHours: 30 }))).toBe('0 */23 * * *')
  })

  it('clamps malformed times', () => {
    expect(buildCron(fields({ freq: 'daily', time: '25:99' }))).toBe('59 23 * * *')
    expect(buildCron(fields({ freq: 'daily', time: 'garbage' }))).toBe('0 0 * * *')
  })
})

describe('parseSchedule', () => {
  it('reads plain crons back into builder fields', () => {
    expect(parseSchedule('0 23 * * *')).toMatchObject({ freq: 'daily', time: '23:00' })
    expect(parseSchedule('30 2 * * 2')).toMatchObject({ freq: 'weekly', time: '02:30', weekday: 2 })
    expect(parseSchedule('15 4 15 * *')).toMatchObject({ freq: 'monthly', time: '04:15', dom: 15 })
    expect(parseSchedule('0 */6 * * *')).toMatchObject({ freq: 'interval', intervalHours: 6 })
    expect(parseSchedule('0 * * * *')).toMatchObject({ freq: 'hourly' })
  })

  it('maps @-descriptors onto builder states', () => {
    expect(parseSchedule('@hourly')).toMatchObject({ freq: 'hourly' })
    expect(parseSchedule('@daily')).toMatchObject({ freq: 'daily', time: '00:00' })
    expect(parseSchedule('@midnight')).toMatchObject({ freq: 'daily', time: '00:00' })
    expect(parseSchedule('@weekly')).toMatchObject({ freq: 'weekly', time: '00:00', weekday: 0 })
    expect(parseSchedule('@monthly')).toMatchObject({ freq: 'monthly', time: '00:00', dom: 1 })
    expect(parseSchedule('@reboot')).toMatchObject({ freq: 'custom' })
  })

  it('treats an empty expression as manual', () => {
    expect(parseSchedule('')).toMatchObject({ freq: 'manual' })
    expect(parseSchedule('   ')).toMatchObject({ freq: 'manual' })
  })

  it('normalizes cron weekday 7 to Sunday (0)', () => {
    expect(parseSchedule('0 0 * * 7')).toMatchObject({ freq: 'weekly', weekday: 0 })
  })

  it('falls back to custom for unrecognized or invalid shapes', () => {
    expect(parseSchedule('*/5 * * * *')).toMatchObject({ freq: 'custom' })
    expect(parseSchedule('61 25 * * *')).toMatchObject({ freq: 'custom' })
    expect(parseSchedule('not a cron at all')).toMatchObject({ freq: 'custom' })
  })

  it('round-trips every builder state through build → parse', () => {
    const cases: ScheduleFields[] = [
      fields({ freq: 'hourly' }),
      fields({ freq: 'daily', time: '23:00' }),
      fields({ freq: 'weekly', time: '02:30', weekday: 5 }),
      fields({ freq: 'monthly', time: '04:15', dom: 15 }),
      fields({ freq: 'interval', intervalHours: 6 }),
    ]
    for (const f of cases) {
      expect(parseSchedule(buildCron(f))).toEqual(f)
    }
  })
})

describe('describeSchedule', () => {
  it('describes each frequency in plain language with the timezone', () => {
    expect(describeSchedule('0 23 * * *')).toBe('Every day at 11:00 PM (UTC)')
    expect(describeSchedule('30 2 * * 2', 'Asia/Kolkata')).toBe('Every Tuesday at 2:30 AM (Asia/Kolkata)')
    expect(describeSchedule('0 4 5 * *')).toBe('Every month on day 5 at 4:00 AM (UTC)')
    expect(describeSchedule('0 * * * *')).toBe('Every hour, on the hour (UTC)')
    expect(describeSchedule('0 */6 * * *')).toBe('Every 6 hours')
    expect(describeSchedule('0 */1 * * *')).toBe('Every 1 hour')
    expect(describeSchedule('')).toBe('Manual — no schedule; runs only when started by hand.')
  })

  it('shows custom cron verbatim', () => {
    expect(describeSchedule('*/5 * * * *')).toBe('Custom schedule: */5 * * * *')
  })

  it('surfaces field-count mistakes instead of describing them', () => {
    expect(describeSchedule('0 2 * *')).toMatch(/^Schedule needs attention — A cron expression has 5 fields/)
  })
})

describe('cronIssue', () => {
  it('accepts empty, @-descriptors and well-formed expressions', () => {
    expect(cronIssue('')).toBeNull()
    expect(cronIssue('@daily')).toBeNull()
    expect(cronIssue('0 2 * * *')).toBeNull()
  })

  it('rejects expressions that are not 5 fields', () => {
    expect(cronIssue('0 2 * *')).toMatch(/5 fields/)
    expect(cronIssue('0 2 * * * *')).toMatch(/5 fields/)
  })
})

describe('formatTime12', () => {
  it('converts 24-hour times to 12-hour with AM/PM', () => {
    expect(formatTime12('23:00')).toBe('11:00 PM')
    expect(formatTime12('00:30')).toBe('12:30 AM')
    expect(formatTime12('12:00')).toBe('12:00 PM')
    expect(formatTime12('09:05')).toBe('9:05 AM')
  })
})
