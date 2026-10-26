import { useEffect, useState } from 'react'
import { regionApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import type { S3Region, StorageType } from '@/types'
import { CloudUpload, HardDrive, FolderOpen, Lock, Loader2 } from 'lucide-react'

interface StorageTargetFormProps {
  onSubmit: (data: Record<string, unknown>) => void | Promise<void>
  isSubmitting?: boolean
  submitLabel?: string
}

const typeIcon: Record<StorageType, typeof CloudUpload> = { S3: CloudUpload, LOCAL: HardDrive, SMB: FolderOpen }

const emptyForm = { name: '', type: 'S3' as StorageType, bucket: '', region: '', endpoint: '', use_path_style: false, access_key: '', secret_key: '', path: '' }

// Shared storage-target create form used by the Storage page drawer and the
// onboarding wizard. Owns its own field state and calls onSubmit with a
// ready-to-POST payload; the caller handles the request, toasts and closing.
const StorageTargetForm = ({ onSubmit, isSubmitting = false, submitLabel = 'Save Storage Target' }: StorageTargetFormProps) => {
  const { currentOrg } = useAuthStore()
  const [form, setForm] = useState(emptyForm)
  const [regions, setRegions] = useState<S3Region[]>([])

  useEffect(() => {
    if (!currentOrg) { setRegions([]); return }
    regionApi.list(currentOrg.id).then(setRegions).catch(() => setRegions([]))
  }, [currentOrg])

  const regionIsCustom = form.region !== '' && !regions.some((r) => r.code === form.region)
  const handleRegionSelect = (value: string) => {
    if (value === '__custom') {
      setForm((f) => ({ ...f, region: '' }))
      return
    }
    setForm((f) => {
      const picked = regions.find((r) => r.code === value)
      // Preset endpoints (e.g. a private clone registered with one) fill
      // the endpoint field unless the user already typed their own.
      const endpoint = picked?.endpoint && !f.endpoint.trim() ? picked.endpoint : f.endpoint
      return { ...f, region: value, endpoint }
    })
  }

  const inputCls = 'w-full h-9 px-3 rounded-lg border border-surface-variant bg-surface-container-low text-[14px] text-on-surface focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all'
  const labelCls = 'block text-[13px] font-medium text-on-surface-variant mb-1.5'

  return (
    <form
      onSubmit={(e) => { e.preventDefault(); void onSubmit(form as Record<string, unknown>) }}
      className="flex flex-col gap-4"
    >
      <div><label className={labelCls}>Name *</label><input type="text" value={form.name} onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))} required className={inputCls} /></div>
      <div>
        <label className={labelCls}>Type *</label>
        <div className="grid grid-cols-3 gap-2">
          {(['S3', 'LOCAL', 'SMB'] as StorageType[]).map((t) => {
            const TIcon = typeIcon[t]
            const disabled = t === 'SMB'
            return (
              <button key={t} type="button" disabled={disabled} title={disabled ? 'SMB support is planned' : t} onClick={() => setForm((f) => ({ ...f, type: t }))}
                className={`flex flex-col items-center gap-1.5 p-3 rounded-xl border-2 transition-all ${disabled ? 'opacity-40 cursor-not-allowed border-surface-variant' : form.type === t ? 'border-primary bg-primary-container/20' : 'border-surface-variant hover:border-outline'}`}>
                <TIcon size={20} className={form.type === t && !disabled ? 'text-primary' : 'text-on-surface-variant'} />
                <span className="text-[12px] font-medium text-on-surface">{t}</span>
              </button>
            )
          })}
        </div>
      </div>
      {form.type === 'S3' && (
        <>
          <div><label className={labelCls}>Bucket *</label><input type="text" value={form.bucket} onChange={(e) => setForm((f) => ({ ...f, bucket: e.target.value }))} required placeholder="my-backup-bucket" className={inputCls} /></div>
          <div>
            <label className={labelCls}>Region</label>
            <select
              value={regionIsCustom ? '__custom' : form.region}
              onChange={(e) => handleRegionSelect(e.target.value)}
              className={inputCls}
            >
              <option value="">Select a region…</option>
              {regions.map((r) => (
                <option key={r.code} value={r.code}>
                  {r.code} — {r.name}
                </option>
              ))}
              <option value="__custom">Custom…</option>
            </select>
          </div>
          {regionIsCustom && (
            <div><label className={labelCls}>Custom region code</label><input type="text" value={form.region} onChange={(e) => setForm((f) => ({ ...f, region: e.target.value }))} placeholder="auto, custom-region-1…" className={`${inputCls} font-mono text-[13px]`} /></div>
          )}
          <div><label className={labelCls}>Endpoint (S3-compatible)</label><input type="text" value={form.endpoint} onChange={(e) => setForm((f) => ({ ...f, endpoint: e.target.value }))} placeholder="https://minio:9000 or https://xyz.r2.cloudflarestorage.com" className={inputCls} /></div>
          <label className="flex items-start gap-2.5 p-3 rounded-lg bg-surface-container-low border border-surface-variant cursor-pointer">
            <input type="checkbox" checked={form.use_path_style} onChange={(e) => setForm((f) => ({ ...f, use_path_style: e.target.checked }))} className="mt-0.5 accent-primary" />
            <span>
              <span className="block text-[13px] font-medium text-on-surface">Path-style addressing</span>
              <span className="block text-[12px] text-on-surface-variant">Needed for MinIO (<span className="font-mono">host/bucket/key</span>). AWS, R2, Wasabi use virtual-hosted style — leave off.</span>
            </span>
          </label>
          <div><label className={labelCls}>Access Key</label><input type="text" value={form.access_key} onChange={(e) => setForm((f) => ({ ...f, access_key: e.target.value }))} className={inputCls} /></div>
          <div><label className={labelCls}>Secret Key</label><input type="password" value={form.secret_key} onChange={(e) => setForm((f) => ({ ...f, secret_key: e.target.value }))} className={inputCls} /></div>
          <p className="text-[12px] text-on-surface-variant">After saving, use <strong>Test</strong> on the card — it runs a live PUT/HEAD/GET/DELETE round-trip.</p>
        </>
      )}
      {(form.type === 'LOCAL' || form.type === 'SMB') && (
        <div><label className={labelCls}>Path</label><input type="text" value={form.path} onChange={(e) => setForm((f) => ({ ...f, path: e.target.value }))} placeholder={form.type === 'SMB' ? '\\\\server\\share' : 'D:\\Backups\\'} className={`${inputCls} font-mono text-[13px]`} /></div>
      )}
      <p className="text-[12px] text-on-surface-variant flex items-center gap-1.5">
        <Lock size={12} className="text-on-primary-container" />
        Credentials are encrypted with AES-256-GCM before storage
      </p>
      <button type="submit" disabled={isSubmitting} className="w-full h-9 rounded-lg bg-primary text-on-primary text-[13px] font-medium hover:bg-primary-container transition-colors disabled:opacity-60 flex items-center justify-center gap-2">
        {isSubmitting && <Loader2 size={14} className="animate-spin" />}
        {submitLabel}
      </button>
    </form>
  )
}

export default StorageTargetForm
