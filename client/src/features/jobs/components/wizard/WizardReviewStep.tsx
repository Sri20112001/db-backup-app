import type { Agent, BackupPolicy, DatabaseConnection, StorageTarget } from '@/types'
import type { JobWizardFormData } from './types'

interface WizardReviewStepProps {
  form: JobWizardFormData
  selectedConnection?: DatabaseConnection
  selectedAgent?: Agent
  selectedPolicy?: BackupPolicy
  selectedStorage?: StorageTarget
  policyGoverns: (field: 'schedule' | 'processing') => boolean
}

export default function WizardReviewStep({
  form,
  selectedConnection,
  selectedAgent,
  selectedPolicy,
  selectedStorage,
  policyGoverns,
}: WizardReviewStepProps) {
  const reviewItems = [
    { label: 'Job Name', value: form.name },
    { label: 'Source Type', value: form.source_type.replace('_', ' ') },
    ...(selectedConnection ? [{ label: 'Connection', value: selectedConnection.name }] : []),
    { label: 'Source', value: form.source_path || form.source_database || '—', mono: true },
    ...(form.source_type === 'MONGODB'
      ? [{ label: 'Export Format', value: form.export_format || 'ARCHIVE' }]
      : []),
    { label: 'Agent', value: selectedAgent?.name ?? '—' },
    ...(selectedPolicy
      ? [{ label: 'Policy', value: `${selectedPolicy.name} (schedule, processing & retry managed)` }]
      : []),
    {
      label: 'Schedule',
      value: policyGoverns('schedule') && selectedPolicy ? selectedPolicy.cron_expr : form.cron_expr,
      mono: true,
    },
    {
      label: 'Timezone',
      value: policyGoverns('schedule') && selectedPolicy ? selectedPolicy.timezone : form.timezone,
    },
    {
      label: 'Compression',
      value:
        policyGoverns('processing') && selectedPolicy
          ? selectedPolicy.mode === 'COMPRESSED'
            ? 'Zstandard (policy)'
            : 'Disabled (policy)'
          : form.mode === 'COMPRESSED'
          ? 'Zstandard'
          : 'Disabled',
    },
    {
      label: 'Encryption',
      value:
        policyGoverns('processing') && selectedPolicy
          ? selectedPolicy.encrypted
            ? 'AES-256-GCM (policy)'
            : 'Disabled (policy)'
          : form.encrypted
          ? 'AES-256-GCM'
          : 'Disabled',
    },
    { label: 'Storage', value: selectedStorage?.name ?? '—' },
    {
      label: 'Retention',
      value:
        policyGoverns('processing') && selectedPolicy
          ? `${selectedPolicy.retention_days} days (policy)`
          : `${form.retention_days} days`,
    },
  ]

  return (
    <div className="flex flex-col gap-4">
      <h2 className="text-[16px] font-semibold text-on-surface">Review & Create</h2>
      <div className="grid grid-cols-2 gap-x-8 gap-y-3 text-[13px]">
        {reviewItems.map((r) => (
          <div key={r.label} className="flex flex-col gap-0.5">
            <span className="text-[11px] font-semibold uppercase tracking-wider text-on-surface-variant">
              {r.label}
            </span>
            <span className={`text-on-surface ${r.mono ? 'font-mono text-[12px]' : 'font-medium'}`}>
              {r.value}
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}
