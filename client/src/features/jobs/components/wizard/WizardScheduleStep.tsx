import type { BackupPolicy } from '@/types'
import ScheduleBuilder from '@/components/ScheduleBuilder'
import type { JobWizardFormData } from './types'

interface WizardScheduleStepProps {
  form: JobWizardFormData
  update: (key: string, value: unknown) => void
  selectedPolicy?: BackupPolicy
  policyGovernsSchedule: boolean
}

const TIMEZONES = [
  'UTC',
  'America/New_York',
  'America/Chicago',
  'America/Los_Angeles',
  'Europe/London',
  'Europe/Berlin',
  'Asia/Tokyo',
]

export default function WizardScheduleStep({
  form,
  update,
  selectedPolicy,
  policyGovernsSchedule,
}: WizardScheduleStepProps) {
  return (
    <div className="flex flex-col gap-4">
      <h2 className="text-[16px] font-semibold text-on-surface">Schedule</h2>
      {policyGovernsSchedule && selectedPolicy && (
        <p className="text-[12px] text-on-surface-variant bg-primary-container/20 border border-primary/30 rounded-lg px-3 py-2">
          Schedule comes from policy <span className="font-semibold">{selectedPolicy.name}</span> ({selectedPolicy.cron_expr}). Change or detach the policy in Processing to edit it here.
        </p>
      )}
      <div className="flex flex-col gap-4">
        <ScheduleBuilder
          cron={form.cron_expr}
          timezone={form.timezone}
          onChange={(c) => update('cron_expr', c)}
          disabled={policyGovernsSchedule}
          idPrefix="vg-job"
        />
      </div>
      <div>
        <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Timezone</label>
        <select
          value={form.timezone}
          disabled={policyGovernsSchedule}
          onChange={(e) => update('timezone', e.target.value)}
          className="w-full h-9 px-3 rounded-lg border border-surface-variant bg-surface-container-low text-[14px] text-on-surface focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all disabled:opacity-50"
        >
          {TIMEZONES.map((tz) => (
            <option key={tz} value={tz}>
              {tz}
            </option>
          ))}
        </select>
      </div>
    </div>
  )
}
