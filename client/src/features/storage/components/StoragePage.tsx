import { useEffect, useState } from 'react'
import { regionApi, storageApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import type { S3Region, StorageTarget, StorageType } from '@/types'
import EmptyState from '@/components/EmptyState'
import Pagination from '@/components/Pagination'
import { usePagination } from '@/hooks/usePagination'
import ConfirmDialog from '@/components/ConfirmDialog'
import { CloudUpload, HardDrive, FolderOpen, Lock, Trash2, Plus, X, Loader2, Database, PlugZap, CheckCircle2, XCircle } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import SearchInput from '@/components/ui/SearchInput'
import SortSelect from '@/components/ui/SortSelect'
import { Page, PageHeader } from '@/components/Page'

const TYPE_FILTERS: { label: string; value: StorageType | '' }[] = [
  { label: 'All', value: '' },
  { label: 'S3', value: 'S3' },
  { label: 'Local', value: 'LOCAL' },
  { label: 'SMB', value: 'SMB' },
]

const typeIcon: Record<StorageType, LucideIcon> = { S3: CloudUpload, LOCAL: HardDrive, SMB: FolderOpen }
const typeColor: Record<StorageType, string> = { S3: 'text-primary', LOCAL: 'text-on-surface-variant', SMB: 'text-primary' }
const typeBg: Record<StorageType, string> = { S3: 'bg-primary-container', LOCAL: 'bg-surface-variant', SMB: 'bg-primary-container' }

const StoragePage = () => {
  const { currentOrg } = useAuthStore()
  const { addToast } = useUIStore()
  const [targets, setTargets] = useState<StorageTarget[]>([])
  const [regions, setRegions] = useState<S3Region[]>([])
  const [search, setSearch] = useState('')
  const [typeFilter, setTypeFilter] = useState<StorageType | ''>('')
  const [sort, setSort] = useState('name-asc')
  const filtered = targets.filter((t) => {
    const matchType = !typeFilter || t.type === typeFilter
    const q = search.toLowerCase()
    const matchSearch =
      !search ||
      t.name.toLowerCase().includes(q) ||
      (t.bucket || '').toLowerCase().includes(q) ||
      (t.path || '').toLowerCase().includes(q)
    return matchType && matchSearch
  })
  const sorted = [...filtered].sort((a, b) =>
    sort === 'name-desc' ? b.name.localeCompare(a.name) : a.name.localeCompare(b.name),
  )
  const paged = usePagination(sorted, 6, `${search}|${typeFilter}|${sort}|${currentOrg?.id ?? ''}`)
  const [isLoading, setIsLoading] = useState(true)
  const [showAdd, setShowAdd] = useState(false)
  const [deleteId, setDeleteId] = useState<string | null>(null)
  const [form, setForm] = useState({ name: '', type: 'S3' as StorageType, bucket: '', region: '', endpoint: '', use_path_style: false, access_key: '', secret_key: '', path: '' })
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [testingId, setTestingId] = useState<string | null>(null)
  const [testResults, setTestResults] = useState<Record<string, { ok: boolean; detail: string }>>({})

  const resetForm = () => setForm({ name: '', type: 'S3', bucket: '', region: '', endpoint: '', use_path_style: false, access_key: '', secret_key: '', path: '' })

  const handleTest = async (id: string) => {
    if (!currentOrg) return
    setTestingId(id)
    try {
      const res = await storageApi.testConnection(currentOrg.id, id)
      setTestResults((prev) => ({ ...prev, [id]: { ok: true, detail: `Connected${res.latency_ms != null ? ` · ${res.latency_ms}ms` : ''}` } }))
      addToast('success', 'Storage connection verified')
    } catch (e) {
      setTestResults((prev) => ({ ...prev, [id]: { ok: false, detail: e instanceof Error ? e.message : 'Connection failed' } }))
      addToast('error', 'Storage connection test failed')
    } finally {
      setTestingId(null)
    }
  }

  const loadTargets = () => {
    if (!currentOrg) return
    setIsLoading(true)
    storageApi.list(currentOrg.id).then(setTargets).catch(() => addToast('error', 'Failed to load storage')).finally(() => setIsLoading(false))
  }

  useEffect(() => {
    loadTargets()
    if (currentOrg) {
      regionApi.list(currentOrg.id).then(setRegions).catch(() => setRegions([]))
    } else {
      setRegions([])
    }
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

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!currentOrg) return
    setIsSubmitting(true)
    try {
      await storageApi.create(currentOrg.id, form as Record<string, unknown>)
      addToast('success', `${form.name} added`)
      setShowAdd(false)
      resetForm()
      loadTargets()
    } catch { addToast('error', 'Failed to add storage target') }
    finally { setIsSubmitting(false) }
  }

  const handleDelete = async () => {
    if (!currentOrg || !deleteId) return
    try {
      await storageApi.delete(currentOrg.id, deleteId)
      addToast('success', 'Storage target removed')
      setDeleteId(null)
      loadTargets()
    } catch { addToast('error', 'Failed to delete') }
  }

  const inputCls = "w-full h-9 px-3 rounded-lg border border-surface-variant bg-surface-container-low text-[14px] text-on-surface focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all"
  const labelCls = "block text-[13px] font-medium text-on-surface-variant mb-1.5"

  return (
    <Page>
      <PageHeader
        title="Storage Targets"
        description="Configure where backup data is stored"
        actions={
          <button type="button" onClick={() => setShowAdd(true)} className="flex items-center gap-2 px-4 h-9 rounded-lg bg-primary text-on-primary text-[13px] font-medium hover:bg-primary-container transition-colors shadow-sm">
            <Plus size={16} />
            Add Storage
          </button>
        }
      />

      {/* Search + filters */}
      <div className="flex flex-col lg:flex-row lg:items-center gap-3">
        <SearchInput
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Search by name, bucket, or path…"
          className="lg:max-w-xs"
        />
        <div className="flex items-center gap-1.5 overflow-x-auto">
          <SortSelect
            value={sort}
            onChange={setSort}
            options={[
              { label: 'Name A–Z', value: 'name-asc' },
              { label: 'Name Z–A', value: 'name-desc' },
            ]}
          />
          {TYPE_FILTERS.map((f) => (
            <button
              key={f.value}
              type="button"
              onClick={() => setTypeFilter(f.value)}
              className={`px-3 h-8 rounded-lg text-[12px] font-medium whitespace-nowrap transition-colors ${
                typeFilter === f.value
                  ? 'bg-primary text-on-primary shadow-sm'
                  : 'bg-surface-container-low text-on-surface-variant hover:bg-surface-container-high'
              }`}
            >
              {f.label}
            </button>
          ))}
        </div>
      </div>

      {isLoading ? (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {[...Array(3)].map((_, i) => <div key={i} className="animate-pulse h-40 rounded-xl bg-surface-container-lowest border border-surface-variant" />)}
        </div>
      ) : filtered.length === 0 ? (
        <EmptyState icon={Database} title="No storage targets" description="Add your first storage target to start saving backups."
          action={<button type="button" onClick={() => setShowAdd(true)} className="px-4 h-9 rounded-lg bg-primary text-on-primary text-[13px] font-medium hover:bg-primary-container transition-colors">+ Add Storage</button>}
        />
      ) : (
        <div className="flex-1 min-h-0 overflow-y-auto grid grid-cols-1 md:grid-cols-3 gap-4 content-start pr-0.5">
          {paged.pageItems.map((t) => {
            const TypeIcon = typeIcon[t.type]
            const result = testResults[t.id]
            return (
              <div key={t.id} className="flex flex-col gap-3 p-5 rounded-xl bg-surface-container-lowest border border-surface-variant shadow-sm">
                <div className="flex items-start justify-between">
                  <div className={`w-10 h-10 rounded-lg ${typeBg[t.type]} flex items-center justify-center ${typeColor[t.type]}`}>
                    <TypeIcon size={20} />
                  </div>
                  <span className="px-2 py-0.5 rounded bg-surface-container-high text-on-surface-variant text-[11px] font-medium uppercase">{t.type}</span>
                </div>
                <div>
                  <h3 className="text-[15px] font-semibold text-on-surface">{t.name}</h3>
                  <p className="font-mono text-[11px] text-outline mt-0.5 truncate">{t.bucket || t.path || '—'}</p>
                  {t.region && <p className="text-[11px] text-outline">{t.region}</p>}
                  {t.type === 'S3' && t.endpoint && <p className="font-mono text-[11px] text-outline truncate">{t.endpoint}{t.use_path_style ? ' · path-style' : ''}</p>}
                </div>
                {result && (
                  <div className={`flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-[12px] font-medium ${result.ok ? 'bg-emerald-50 text-emerald-700' : 'bg-red-50 text-red-700'}`}>
                    {result.ok ? <CheckCircle2 size={14} /> : <XCircle size={14} />}
                    <span className="truncate">{result.detail}</span>
                  </div>
                )}
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-1.5 text-[12px] text-on-primary-container">
                    <Lock size={12} />
                    <span>{t.type === 'S3' ? (t.has_credentials ? 'Credentials encrypted' : 'No credentials stored') : 'No credentials needed'}</span>
                  </div>
                  <div className="flex items-center gap-1">
                    {t.type === 'S3' && (
                      <button
                        type="button"
                        onClick={() => handleTest(t.id)}
                        disabled={testingId === t.id}
                        title="PUT/HEAD/GET/DELETE round-trip (credentials never leave the server)"
                        className="flex items-center gap-1 px-2 py-1 rounded-lg text-[12px] font-medium text-on-primary-container hover:bg-surface-container-high transition-colors disabled:opacity-60"
                      >
                        {testingId === t.id ? <Loader2 size={13} className="animate-spin" /> : <PlugZap size={13} />}
                        Test
                      </button>
                    )}
                    <button type="button" onClick={() => setDeleteId(t.id)} className="p-1.5 rounded-lg text-outline hover:text-error hover:bg-error-container/30 transition-colors">
                      <Trash2 size={16} />
                    </button>
                  </div>
                </div>
              </div>
            )
          })}
        </div>
      )}

      <Pagination
        page={paged.page}
        totalPages={paged.totalPages}
        total={paged.total}
        perPage={paged.perPage}
        onPage={paged.setPage}
      />

      {/* Add drawer */}
      {showAdd && (
        <div className="fixed inset-0 z-[100] flex justify-end">
          <div className="absolute inset-0 bg-on-surface/30 backdrop-blur-[2px]" onClick={() => setShowAdd(false)} />
          <div className="relative w-full max-w-[480px] bg-surface-container-lowest h-full overflow-y-auto shadow-2xl border-l border-surface-variant">
            <div className="flex items-center justify-between p-5 border-b border-surface-variant sticky top-0 bg-surface-container-lowest z-10">
              <h2 className="text-[16px] font-semibold text-on-surface">Add Storage Target</h2>
              <button type="button" onClick={() => setShowAdd(false)} className="p-1.5 rounded-lg text-on-surface-variant hover:bg-surface-container-high">
                <X size={18} />
              </button>
            </div>
            <form onSubmit={handleCreate} className="flex flex-col gap-4 p-5">
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
                      <span className="block text-[12px] text-outline">Needed for MinIO (<span className="font-mono">host/bucket/key</span>). AWS, R2, Wasabi use virtual-hosted style — leave off.</span>
                    </span>
                  </label>
                  <div><label className={labelCls}>Access Key</label><input type="text" value={form.access_key} onChange={(e) => setForm((f) => ({ ...f, access_key: e.target.value }))} className={inputCls} /></div>
                  <div><label className={labelCls}>Secret Key</label><input type="password" value={form.secret_key} onChange={(e) => setForm((f) => ({ ...f, secret_key: e.target.value }))} className={inputCls} /></div>
                  <p className="text-[12px] text-outline">After saving, use <strong>Test</strong> on the card — it runs a live PUT/HEAD/GET/DELETE round-trip.</p>
                </>
              )}
              {(form.type === 'LOCAL' || form.type === 'SMB') && (
                <div><label className={labelCls}>Path</label><input type="text" value={form.path} onChange={(e) => setForm((f) => ({ ...f, path: e.target.value }))} placeholder={form.type === 'SMB' ? '\\\\server\\share' : 'D:\\Backups\\'} className={`${inputCls} font-mono text-[13px]`} /></div>
              )}
              <p className="text-[12px] text-outline flex items-center gap-1.5">
                <Lock size={12} className="text-on-primary-container" />
                Credentials are encrypted with AES-256-GCM before storage
              </p>
              <button type="submit" disabled={isSubmitting} className="w-full h-9 rounded-lg bg-primary text-on-primary text-[13px] font-medium hover:bg-primary-container transition-colors disabled:opacity-60 flex items-center justify-center gap-2">
                {isSubmitting && <Loader2 size={14} className="animate-spin" />}
                Save Storage Target
              </button>
            </form>
          </div>
        </div>
      )}

      {deleteId && (
        <ConfirmDialog title="Remove Storage Target?" message="This will not delete any backup data. The storage configuration will be removed." confirmLabel="Remove" danger onConfirm={handleDelete} onCancel={() => setDeleteId(null)} />
      )}
    </Page>
  )
}

export default StoragePage
