import { useEffect, useState } from 'react'
import { policyApi, jobApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import type { BackupPolicy } from '@/types'
import EmptyState from '@/components/EmptyState'
import ConfirmDialog from '@/components/ConfirmDialog'
import SearchInput from '@/components/ui/SearchInput'
import { Page, PageHeader } from '@/components/Page'
import { Plus, Pencil, Trash2, X, SlidersHorizontal } from 'lucide-react'

const CRON_PRESETS = [
  { label: 'Manual (no schedule)', cron: '' },
  { label: 'Hourly', cron: '0 * * * *' },
  { label: 'Every 6 hours', cron: '0 */6 * * *' },
  { label: 'Daily 11 PM', cron: '0 23 * * *' },
  { label: 'Weekly Sunday', cron: '0 0 * * 0' },
]

const TIMEZONES = ['UTC', 'America/New_York', 'America/Chicago', 'America/Los_Angeles', 'Europe/London', 'Europe/Berlin', 'Asia/Tokyo']

const blankForm = {
  name: '', strategy: 'FULL', cron_expr: '0 23 * * *', timezone: 'UTC', enabled: true,
  mode: 'COMPRESSED', encrypted: true, retention_days: 30, verification_enabled: true,
  max_retries: 0, retry_delay_seconds: 300,
  sla_target_minutes: 0, rpo_target_minutes: 0, rto_target_minutes: 0,
}

const inputCls =
  'w-full h-9 px-3 rounded-lg border border-surface-variant bg-surface-container-low text-[13px] text-on-surface focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all'

const PoliciesPage = () => {
  const { currentOrg } = useAuthStore()
  const { addToast } = useUIStore()
  const [policies, setPolicies] = useState<BackupPolicy[]>([])
  const [attachedCounts, setAttachedCounts] = useState<Record<string, number>>({})
  const [search, setSearch] = useState('')
  const [isLoading, setIsLoading] = useState(true)
  const [showModal, setShowModal] = useState(false)
  const [editing, setEditing] = useState<BackupPolicy | null>(null)
  const [form, setForm] = useState(blankForm)
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [deleteId, setDeleteId] = useState<string | null>(null)

  const load = () => {
    if (!currentOrg) return
    setIsLoading(true)
    Promise.all([policyApi.list(currentOrg.id), jobApi.list(currentOrg.id)])
      .then(([p, jobs]) => {
        setPolicies(p)
        const counts: Record<string, number> = {}
        for (const j of jobs) {
          if (j.policy_id) counts[j.policy_id] = (counts[j.policy_id] ?? 0) + 1
        }
        setAttachedCounts(counts)
      })
      .catch(() => addToast('error', 'Failed to load policies'))
      .finally(() => setIsLoading(false))
  }

  useEffect(() => { load() }, [currentOrg])

  const openAdd = () => {
    setEditing(null)
    setForm(blankForm)
    setShowModal(true)
  }

  const openEdit = (p: BackupPolicy) => {
    setEditing(p)
    setForm({
      name: p.name, strategy: p.strategy, cron_expr: p.cron_expr, timezone: p.timezone,
      enabled: p.enabled, mode: p.mode, encrypted: p.encrypted, retention_days: p.retention_days,
      verification_enabled: p.verification_enabled, max_retries: p.max_retries,
      retry_delay_seconds: p.retry_delay_seconds, sla_target_minutes: p.sla_target_minutes,
      rpo_target_minutes: p.rpo_target_minutes, rto_target_minutes: p.rto_target_minutes,
    })
    setShowModal(true)
  }

  const update = (key: string, value: unknown) => setForm((f) => ({ ...f, [key]: value }))

  const handleSave = async () => {
    if (!currentOrg) return
    if (!form.name.trim()) {
      addToast('error', 'Policy name is required')
      return
    }
    setIsSubmitting(true)
    try {
      const payload: Record<string, unknown> = { ...form, name: form.name.trim() }
      if (editing) {
        await policyApi.update(currentOrg.id, editing.id, payload)
        addToast('success', 'Policy updated — attached jobs pick it up on their next run')
      } else {
        await policyApi.create(currentOrg.id, payload)
        addToast('success', 'Policy created — attach it to jobs from the job editor')
      }
      setShowModal(false)
      setEditing(null)
      load()
    } catch (e) {
      addToast('error', e instanceof Error ? e.message : 'Save failed')
    } finally {
      setIsSubmitting(false)
    }
  }

  const handleDelete = async () => {
    if (!currentOrg || !deleteId) return
    try {
      await policyApi.delete(currentOrg.id, deleteId)
      addToast('success', 'Policy deleted — attached jobs fell back to their own settings')
      setDeleteId(null)
      load()
    } catch {
      addToast('error', 'Failed to delete policy')
    }
  }

  const filtered = policies.filter((p) =>
    !search || p.name.toLowerCase().includes(search.toLowerCase()),
  )

  const scheduleLabel = (p: BackupPolicy) =>
    !p.cron_expr ? 'Manual' : (CRON_PRESETS.find((c) => c.cron === p.cron_expr)?.label ?? p.cron_expr)

  return (
    <Page>
      <PageHeader
        title="Backup Policies"
        description="Reusable how-to-back-up: schedule, processing, retention, verification, retry. Jobs attach optionally; unattached jobs use their own settings."
        actions={
          <button
            type="button"
            onClick={openAdd}
            className="flex items-center gap-2 px-4 h-9 rounded-lg bg-primary text-on-primary text-[13px] font-medium hover:bg-primary-container transition-colors shadow-sm"
          >
            <Plus size={16} />
            New Policy
          </button>
        }
      />

      <div className="flex flex-col lg:flex-row lg:items-center gap-3">
        <SearchInput
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Search policies…"
          className="lg:max-w-xs"
        />
      </div>

      {isLoading ? (
        <div className="flex flex-col gap-2">{[...Array(4)].map((_, i) => <div key={i} className="animate-pulse h-20 rounded-xl bg-surface-container-lowest border border-surface-variant" />)}</div>
      ) : filtered.length === 0 ? (
        <EmptyState
          icon={SlidersHorizontal}
          title="No backup policies"
          description="Policies capture schedule, compression, retention, verification and retry in one reusable place. Jobs keep working without them."
          action={
            <button type="button" onClick={openAdd} className="px-4 h-9 rounded-lg bg-primary text-on-primary text-[13px] font-medium hover:bg-primary-container transition-colors">
              Create First Policy
            </button>
          }
        />
      ) : (
        <div className="flex-1 min-h-0 overflow-y-auto grid grid-cols-1 md:grid-cols-2 gap-4 content-start pr-0.5">
          {filtered.map((p) => (
            <div key={p.id} className="flex flex-col gap-3 p-5 rounded-xl bg-surface-container-lowest border border-surface-variant shadow-sm">
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0">
                  <h3 className="text-[15px] font-semibold text-on-surface truncate">{p.name}</h3>
                  <p className="font-mono text-[11px] text-outline">
                    {scheduleLabel(p)} · {p.retention_days}d retention · {attachedCounts[p.id] ?? 0} jobs
                  </p>
                </div>
                <span className={`shrink-0 px-2 py-0.5 rounded-full text-[11px] font-medium ${p.enabled ? 'bg-primary-container text-on-primary-container' : 'bg-surface-variant text-outline'}`}>
                  {p.enabled ? 'Enabled' : 'Disabled'}
                </span>
              </div>
              <div className="flex flex-wrap gap-1.5 text-[11px]">
                <span className="px-2 py-0.5 rounded bg-surface-container-high text-on-surface-variant">{p.mode === 'COMPRESSED' ? 'Compressed' : 'Normal'}</span>
                <span className="px-2 py-0.5 rounded bg-surface-container-high text-on-surface-variant">{p.encrypted ? 'Encrypted' : 'Unencrypted'}</span>
                {p.verification_enabled && <span className="px-2 py-0.5 rounded bg-surface-container-high text-on-surface-variant">Verified</span>}
                {p.max_retries > 0 && <span className="px-2 py-0.5 rounded bg-surface-container-high text-on-surface-variant">↻ {p.max_retries} retries</span>}
              </div>
              <div className="flex items-center justify-end gap-1.5">
                <button
                  type="button"
                  onClick={() => openEdit(p)}
                  className="flex items-center gap-1 px-3 h-8 rounded-lg text-[12px] font-medium bg-surface-container-low text-on-surface-variant hover:bg-surface-container-high transition-colors"
                >
                  <Pencil size={13} /> Edit
                </button>
                <button
                  type="button"
                  onClick={() => setDeleteId(p.id)}
                  className="p-2 rounded-lg text-outline hover:text-error hover:bg-error-container/30 transition-colors"
                >
                  <Trash2 size={15} />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {showModal && (
        <div className="fixed inset-0 z-[200] flex items-center justify-center">
          <div className="absolute inset-0 bg-on-surface/30 backdrop-blur-[2px]" onClick={() => setShowModal(false)} />
          <div className="relative bg-surface-container-lowest rounded-xl shadow-2xl p-6 w-full max-w-lg mx-4 border border-surface-variant max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-[16px] font-semibold text-on-surface">{editing ? 'Edit Policy' : 'New Policy'}</h2>
              <button type="button" onClick={() => setShowModal(false)} className="p-1.5 rounded-lg text-on-surface-variant hover:bg-surface-container-high">
                <X size={18} />
              </button>
            </div>

            <div className="flex flex-col gap-4">
              <div>
                <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Name *</label>
                <input type="text" value={form.name} onChange={(e) => update('name', e.target.value)} placeholder="e.g. Nightly Standard" className={inputCls} />
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Schedule</label>
                  <select value={form.cron_expr} onChange={(e) => update('cron_expr', e.target.value)} className={inputCls}>
                    {CRON_PRESETS.map((c) => <option key={c.label} value={c.cron}>{c.label}</option>)}
                    {!CRON_PRESETS.some((c) => c.cron === form.cron_expr) && <option value={form.cron_expr}>Custom: {form.cron_expr}</option>}
                  </select>
                </div>
                <div>
                  <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Custom cron</label>
                  <input type="text" value={form.cron_expr} onChange={(e) => update('cron_expr', e.target.value)} placeholder="0 2 * * * (empty = manual)" className={`${inputCls} font-mono text-[13px]`} />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Timezone</label>
                  <select value={form.timezone} onChange={(e) => update('timezone', e.target.value)} className={inputCls}>
                    {TIMEZONES.map((tz) => <option key={tz} value={tz}>{tz}</option>)}
                  </select>
                </div>
                <div>
                  <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Retention (days)</label>
                  <input type="number" value={form.retention_days} onChange={(e) => update('retention_days', parseInt(e.target.value) || 0)} min={0} max={3650} className={inputCls} />
                </div>
              </div>

              {[
                { key: 'enabled', label: 'Enabled', desc: 'Disabled policies never schedule or retry' },
                { key: 'mode', label: 'Compression', desc: 'Zstandard payload compression', isMode: true },
                { key: 'encrypted', label: 'Encryption', desc: 'AES-256-GCM before upload' },
                { key: 'verification_enabled', label: 'Verification', desc: 'Require verified recovery points' },
              ].map((t) => (
                <label key={t.key} className="flex items-center justify-between gap-3 p-3 rounded-xl border border-surface-variant bg-surface-container-low cursor-pointer">
                  <span>
                    <span className="block text-[13px] font-medium text-on-surface">{t.label}</span>
                    <span className="block text-[11px] text-outline">{t.desc}</span>
                  </span>
                  <input
                    type="checkbox"
                    checked={t.isMode ? form.mode === 'COMPRESSED' : Boolean(form[t.key as keyof typeof form])}
                    onChange={(e) => update(t.isMode ? 'mode' : t.key, t.isMode ? (e.target.checked ? 'COMPRESSED' : 'NORMAL') : e.target.checked)}
                    className="w-4 h-4 accent-[#1a56db]"
                  />
                </label>
              ))}

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Max retries (0–10)</label>
                  <input type="number" value={form.max_retries} onChange={(e) => update('max_retries', Math.max(0, Math.min(10, parseInt(e.target.value) || 0)))} min={0} max={10} className={inputCls} />
                </div>
                <div>
                  <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">Retry delay (sec)</label>
                  <input type="number" value={form.retry_delay_seconds} onChange={(e) => update('retry_delay_seconds', Math.max(0, parseInt(e.target.value) || 0))} min={0} className={inputCls} />
                </div>
              </div>

              <div className="grid grid-cols-3 gap-3">
                {[
                  { key: 'sla_target_minutes', label: 'SLA (min)' },
                  { key: 'rpo_target_minutes', label: 'RPO (min)' },
                  { key: 'rto_target_minutes', label: 'RTO (min)' },
                ].map((f) => (
                  <div key={f.key}>
                    <label className="block text-[13px] font-medium text-on-surface-variant mb-1.5">{f.label}</label>
                    <input
                      type="number"
                      value={Number(form[f.key as keyof typeof form])}
                      onChange={(e) => update(f.key, Math.max(0, parseInt(e.target.value) || 0))}
                      min={0}
                      className={inputCls}
                    />
                  </div>
                ))}
              </div>
              <p className="text-[11px] text-outline">Strategy is FULL-only — incremental backups are not implemented and cannot be selected.</p>
            </div>

            <div className="flex items-center justify-end gap-2 mt-5">
              <button
                type="button"
                onClick={() => setShowModal(false)}
                className="px-4 h-9 rounded-lg bg-surface-container-lowest border border-surface-variant text-on-surface text-[13px] font-medium hover:bg-surface-container-low transition-colors"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={() => void handleSave()}
                disabled={isSubmitting}
                className="px-4 h-9 rounded-lg bg-primary text-on-primary text-[13px] font-medium hover:bg-primary-container transition-colors disabled:opacity-50"
              >
                {isSubmitting ? 'Saving…' : editing ? 'Save Changes' : 'Create Policy'}
              </button>
            </div>
          </div>
        </div>
      )}

      {deleteId !== null && (
        <ConfirmDialog
          title="Delete policy?"
          message="Attached jobs keep working on their own inline settings. This cannot be undone."
          confirmLabel="Delete"
          danger
          onConfirm={() => void handleDelete()}
          onCancel={() => setDeleteId(null)}
        />
      )}
    </Page>
  )
}

export default PoliciesPage
