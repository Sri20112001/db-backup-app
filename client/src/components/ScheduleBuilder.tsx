import { useRef, useState } from 'react'
import {
  WEEKDAYS,
  buildCron,
  cronIssue,
  describeSchedule,
  parseSchedule,
  type ScheduleFreq,
  type ScheduleFields,
} from '@/utils/schedule'

interface Props {
  /** Current cron expression — the source of truth, kept in sync both ways. */
  cron: string
  /** Used only to label the preview ("Every day at 11:00 PM (UTC)"). */
  timezone: string
  onChange: (cron: string) => void
  disabled?: boolean
  idPrefix?: string
}

const FREQ_OPTIONS: { value: ScheduleFreq; label: string }[] = [
  { value: 'manual', label: 'Manual (no schedule)' },
  { value: 'hourly', label: 'Hourly' },
  { value: 'daily', label: 'Daily at a time…' },
  { value: 'weekly', label: 'Every week on a day…' },
  { value: 'monthly', label: 'Every month on a day…' },
  { value: 'interval', label: 'Every N hours…' },
  { value: 'custom', label: 'Custom cron…' },
]

const inputCls =
  'h-9 px-3 rounded-lg border border-surface-variant bg-surface-container-low text-[13px] text-on-surface focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all disabled:opacity-50'

// Schedule picker for people who don't speak cron: choose a frequency and
// time with plain controls; the cron field on the right stays in sync and is
// still there for anyone who prefers to type an expression.
const ScheduleBuilder = ({ cron, timezone, onChange, disabled = false, idPrefix = 'vg-sched' }: Props) => {
  // The cron prop is the truth; these mirror it so the controls can round-trip.
  const [fields, setFields] = useState<ScheduleFields>(() => parseSchedule(cron))
  // Once the user touches the raw field we stop rewriting it from the picker.
  const [custom, setCustom] = useState(() => parseSchedule(cron).freq === 'custom')
  const cronRef = useRef<HTMLInputElement>(null)

  const issue = cronIssue(cron)
  const freq: ScheduleFreq = custom ? 'custom' : fields.freq

  const applyFields = (next: ScheduleFields) => {
    setFields(next)
    onChange(buildCron(next))
  }

  const handleFreq = (value: ScheduleFreq) => {
    if (value === 'custom') {
      setCustom(true)
      requestAnimationFrame(() => cronRef.current?.focus())
      return
    }
    setCustom(false)
    applyFields({ ...fields, freq: value })
  }

  return (
    <div>
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label htmlFor={`${idPrefix}-freq`} className="block text-[13px] font-medium text-on-surface-variant mb-1.5">
            Schedule
          </label>
          <select
            id={`${idPrefix}-freq`}
            value={freq}
            disabled={disabled}
            onChange={(e) => handleFreq(e.target.value as ScheduleFreq)}
            className={`w-full ${inputCls}`}
          >
            {FREQ_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label htmlFor={`${idPrefix}-cron`} className="block text-[13px] font-medium text-on-surface-variant mb-1.5">
            Custom cron
          </label>
          <input
            id={`${idPrefix}-cron`}
            ref={cronRef}
            type="text"
            value={cron}
            disabled={disabled}
            onChange={(e) => {
              setCustom(true)
              onChange(e.target.value)
            }}
            placeholder="0 2 * * *"
            spellCheck={false}
            autoComplete="off"
            className={`w-full ${inputCls} font-mono text-[13px]`}
          />
        </div>
      </div>

      {/* Contextual controls for the chosen frequency — no cron knowledge needed. */}
      {freq !== 'manual' && freq !== 'hourly' && freq !== 'custom' && (
        <div className="mt-2.5 flex flex-wrap items-end gap-3 rounded-lg border border-surface-variant bg-surface-container-low px-3 py-2.5">
          {freq === 'interval' ? (
            <div className="flex items-center gap-2 text-[13px] text-on-surface">
              <span>Every</span>
              <input
                type="number"
                min={1}
                max={23}
                disabled={disabled}
                value={fields.intervalHours}
                onChange={(e) => applyFields({ ...fields, intervalHours: Math.max(1, Math.min(23, parseInt(e.target.value, 10) || 1)) })}
                className={`${inputCls} w-16 text-center`}
                aria-label="Interval in hours"
              />
              <span>hours</span>
            </div>
          ) : (
            <>
              <div className="flex items-center gap-2 text-[13px] text-on-surface">
                <span>At</span>
                <input
                  type="time"
                  disabled={disabled}
                  value={fields.time}
                  onChange={(e) => applyFields({ ...fields, time: e.target.value || '00:00' })}
                  className={`${inputCls} w-28`}
                  aria-label="Time of day"
                />
              </div>
              {freq === 'weekly' && (
                <div className="flex items-center gap-2 text-[13px] text-on-surface">
                  <span>on</span>
                  <select
                    disabled={disabled}
                    value={fields.weekday}
                    onChange={(e) => applyFields({ ...fields, weekday: Number(e.target.value) })}
                    className={inputCls}
                    aria-label="Day of week"
                  >
                    {WEEKDAYS.map((d, i) => (
                      <option key={d} value={i}>
                        {d}
                      </option>
                    ))}
                  </select>
                </div>
              )}
              {freq === 'monthly' && (
                <div className="flex items-center gap-2 text-[13px] text-on-surface">
                  <span>on day</span>
                  <input
                    type="number"
                    min={1}
                    max={31}
                    disabled={disabled}
                    value={fields.dom}
                    onChange={(e) => applyFields({ ...fields, dom: Math.max(1, Math.min(31, parseInt(e.target.value, 10) || 1)) })}
                    className={`${inputCls} w-16 text-center`}
                    aria-label="Day of month"
                  />
                </div>
              )}
            </>
          )}
        </div>
      )}

      <p className="mt-2 text-[11.5px] leading-relaxed text-on-surface-variant" data-testid="schedule-preview">
        <span className="font-semibold text-on-surface">Runs:</span>{' '}
        {describeSchedule(cron, timezone)}
        {!custom && freq !== 'manual' && (
          <span className="text-on-surface-variant/80"> — pick any option on the left and this box writes the cron for you.</span>
        )}
      </p>
      {issue && <p className="mt-1 text-[11px] text-error">{issue}</p>}
    </div>
  )
}

export default ScheduleBuilder
