import type { BackupPolicy, StorageTarget } from '@/types'
import { HardDrive } from 'lucide-react'
import NeoToggle from '@/components/ui/NeoToggle'
import { STORAGE_TYPE_ICON } from './types'
import type { JobWizardFormData } from './types'

interface WizardProcessingStepProps {
  form: JobWizardFormData
  update: (key: string, value: unknown) => void
  policies: BackupPolicy[]
  selectedPolicy?: BackupPolicy
  policyGovernsProcessing: boolean
  storageTargets: StorageTarget[]
  onNavigateStorage: () => void
}

export default function WizardProcessingStep({
  form,
  update,
  policies,
  selectedPolicy,
  policyGovernsProcessing,
  storageTargets,
  onNavigateStorage,
}: WizardProcessingStepProps) {
  const toggleOptions = [
    {
      key: 'mode',
      label: 'Compression',
      desc: 'Zstandard — reduces storage by up to 60%',
      value: form.mode === 'COMPRESSED',
      toggle: (v: boolean) => update('mode', v ? 'COMPRESSED' : 'NORMAL'),
    },
    {
      key: 'encrypted',
      label: 'Encryption',
      desc: 'AES-256-GCM — data encrypted before leaving your machine',
      value: form.encrypted,
      toggle: (v: boolean) => update('encrypted', v),
    },
  ]

  return (
    <div className="flex flex-col gap-4">
      <h2 className="text-[16px] font-semibold text-on-surface">Processing & Storage</h2>

      <div>
        <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Backup Policy (optional)</label>
        <select
          value={form.policy_id}
          onChange={(e) => update('policy_id', e.target.value)}
          className="w-full h-9 px-3 rounded-lg border border-surface-variant bg-surface-container-low text-[14px] text-on-surface focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all"
        >
          <option value="">Job-specific settings (no policy)</option>
          {policies.filter((p) => p.enabled).map((p) => (
            <option key={p.id} value={p.id}>
              {p.name} · {p.cron_expr || 'manual'} · {p.retention_days}d{p.max_retries > 0 ? ` · ↻${p.max_retries}` : ''}
            </option>
          ))}
        </select>
        {selectedPolicy && (
          <p className="text-[11px] text-on-surface-variant mt-1.5">
            {selectedPolicy.name} governs{selectedPolicy.cron_expr ? ' schedule,' : ' (manual — job schedule kept),'} compression, encryption, retention ({selectedPolicy.retention_days}d), verification{selectedPolicy.max_retries > 0 ? ` and ${selectedPolicy.max_retries} retries` : ', no retries'}. Detach by choosing job-specific settings.
          </p>
        )}
      </div>

      <div className="flex flex-col gap-3">
        {toggleOptions.map((t) => (
          <div
            key={t.key}
            className={`flex items-center justify-between p-4 rounded-xl border border-surface-variant bg-surface-container-low ${
              policyGovernsProcessing ? 'opacity-50' : ''
            }`}
          >
            <div>
              <p className="text-[14px] font-medium text-on-surface">{t.label}</p>
              <p className="text-[12px] text-on-surface-variant">
                {policyGovernsProcessing && selectedPolicy
                  ? `Managed by policy ${selectedPolicy.name}`
                  : t.desc}
              </p>
            </div>
            <NeoToggle
              id={`toggle-${t.key}`}
              checked={
                policyGovernsProcessing && selectedPolicy
                  ? (t.key === 'mode' ? selectedPolicy.mode === 'COMPRESSED' : selectedPolicy.encrypted)
                  : t.value
              }
              onChange={(checked) => {
                if (!policyGovernsProcessing) t.toggle(checked)
              }}
            />
          </div>
        ))}
      </div>

      <div>
        <label className="block text-[13px] font-medium text-on-surface-variant mb-2">Storage Target *</label>
        {storageTargets.length === 0 ? (
          <div className="text-center py-6 text-on-surface-variant text-[13px]">
            No storage targets.{' '}
            <button type="button" onClick={onNavigateStorage} className="text-primary hover:underline">
              Add one first.
            </button>
          </div>
        ) : (
          <div className="flex flex-col gap-2 max-h-[280px] overflow-y-auto pr-1">
            {storageTargets.map((s) => {
              const StorIcon = STORAGE_TYPE_ICON[s.type] ?? HardDrive
              return (
                <button
                  key={s.id}
                  type="button"
                  onClick={() => update('storage_target_id', s.id)}
                  className={`flex items-center gap-3 p-3.5 rounded-xl border-2 text-left transition-all ${
                    form.storage_target_id === s.id ? 'border-primary bg-primary-container/20' : 'border-surface-variant hover:border-outline'
                  }`}
                >
                  <StorIcon size={20} className="text-primary" />
                  <div>
                    <p className="text-[13px] font-semibold text-on-surface">{s.name}</p>
                    <p className="font-mono text-[11px] text-on-surface-variant">
                      {s.type} · {s.bucket || s.path || '—'}
                    </p>
                  </div>
                </button>
              )
            })}
          </div>
        )}
      </div>

      <div>
        <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">
          Retention (days){policyGovernsProcessing && selectedPolicy ? ` — managed by ${selectedPolicy.name}` : ''}
        </label>
        <input
          type="number"
          value={policyGovernsProcessing && selectedPolicy ? selectedPolicy.retention_days : form.retention_days}
          disabled={policyGovernsProcessing}
          onChange={(e) => update('retention_days', parseInt(e.target.value, 10))}
          min={1}
          max={365}
          className="w-32 h-9 px-3 rounded-lg border border-surface-variant bg-surface-container-low text-[14px] text-on-surface focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all disabled:opacity-50"
        />
      </div>
    </div>
  )
}
