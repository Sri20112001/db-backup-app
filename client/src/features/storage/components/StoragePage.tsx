import { useEffect, useState } from 'react'
import { storageApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import type { StorageTarget, StorageType } from '@/types'
import EmptyState from '@/components/EmptyState'
import ConfirmDialog from '@/components/ConfirmDialog'
import { CloudUpload, HardDrive, FolderOpen, Lock, Trash2, Plus, X, Loader2, Database } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

const typeIcon: Record<StorageType, LucideIcon> = { S3: CloudUpload, LOCAL: HardDrive, SMB: FolderOpen }
const typeColor: Record<StorageType, string> = { S3: 'text-[#004ac6]', LOCAL: 'text-[#434655]', SMB: 'text-[#3e3fcc]' }
const typeBg: Record<StorageType, string> = { S3: 'bg-[#dbe1ff]', LOCAL: 'bg-[#dce2f7]', SMB: 'bg-[#e1e0ff]' }

const StoragePage = () => {
  const { currentOrg } = useAuthStore()
  const { addToast } = useUIStore()
  const [targets, setTargets] = useState<StorageTarget[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [showAdd, setShowAdd] = useState(false)
  const [deleteId, setDeleteId] = useState<string | null>(null)
  const [form, setForm] = useState({ name: '', type: 'S3' as StorageType, bucket: '', region: '', endpoint: '', access_key: '', secret_key: '', path: '' })
  const [isSubmitting, setIsSubmitting] = useState(false)

  const loadTargets = () => {
    if (!currentOrg) return
    setIsLoading(true)
    storageApi.list(currentOrg.id).then(setTargets).catch(() => addToast('error', 'Failed to load storage')).finally(() => setIsLoading(false))
  }

  useEffect(() => { loadTargets() }, [currentOrg])

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!currentOrg) return
    setIsSubmitting(true)
    try {
      await storageApi.create(currentOrg.id, form as Record<string, unknown>)
      addToast('success', `${form.name} added`)
      setShowAdd(false)
      setForm({ name: '', type: 'S3', bucket: '', region: '', endpoint: '', access_key: '', secret_key: '', path: '' })
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

  const inputCls = "w-full h-9 px-3 rounded-lg border border-[#e9edff] bg-[#f9f9ff] text-[14px] text-[#141b2b] focus:outline-none focus:ring-2 focus:ring-[#2563eb]/20 focus:border-[#2563eb] transition-all"
  const labelCls = "block text-[13px] font-medium text-[#434655] mb-1.5"

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-[20px] font-semibold text-[#141b2b] tracking-tight">Storage Targets</h1>
          <p className="text-[12px] text-[#434655] mt-0.5">Configure where backup data is stored</p>
        </div>
        <button type="button" onClick={() => setShowAdd(true)} className="flex items-center gap-2 px-4 h-9 rounded-lg bg-[#2563eb] text-white text-[13px] font-medium hover:bg-[#1d4ed8] transition-colors shadow-sm">
          <Plus size={16} />
          Add Storage
        </button>
      </div>

      {isLoading ? (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {[...Array(3)].map((_, i) => <div key={i} className="animate-pulse h-40 rounded-xl bg-[#ffffff] border border-[#e9edff]" />)}
        </div>
      ) : targets.length === 0 ? (
        <EmptyState icon={Database} title="No storage targets" description="Add your first storage target to start saving backups."
          action={<button type="button" onClick={() => setShowAdd(true)} className="px-4 h-9 rounded-lg bg-[#2563eb] text-white text-[13px] font-medium hover:bg-[#1d4ed8] transition-colors">+ Add Storage</button>}
        />
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {targets.map((t) => {
            const TypeIcon = typeIcon[t.type]
            return (
              <div key={t.id} className="flex flex-col gap-3 p-5 rounded-xl bg-[#ffffff] border border-[#e9edff] shadow-sm">
                <div className="flex items-start justify-between">
                  <div className={`w-10 h-10 rounded-lg ${typeBg[t.type]} flex items-center justify-center ${typeColor[t.type]}`}>
                    <TypeIcon size={20} />
                  </div>
                  <span className="px-2 py-0.5 rounded bg-[#e9edff] text-[#434655] text-[11px] font-medium uppercase">{t.type}</span>
                </div>
                <div>
                  <h3 className="text-[15px] font-semibold text-[#141b2b]">{t.name}</h3>
                  <p className="font-mono text-[11px] text-[#737686] mt-0.5 truncate">{t.bucket || t.path || '—'}</p>
                  {t.region && <p className="text-[11px] text-[#737686]">{t.region}</p>}
                </div>
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-1.5 text-[12px] text-[#006591]">
                    <Lock size={12} />
                    <span>Credentials encrypted</span>
                  </div>
                  <button type="button" onClick={() => setDeleteId(t.id)} className="p-1.5 rounded-lg text-[#737686] hover:text-[#ba1a1a] hover:bg-[#ffdad6]/30 transition-colors">
                    <Trash2 size={16} />
                  </button>
                </div>
              </div>
            )
          })}
        </div>
      )}

      {/* Add drawer */}
      {showAdd && (
        <div className="fixed inset-0 z-[100] flex justify-end">
          <div className="absolute inset-0 bg-[#141b2b]/30 backdrop-blur-[2px]" onClick={() => setShowAdd(false)} />
          <div className="relative w-full max-w-[480px] bg-[#ffffff] h-full overflow-y-auto shadow-2xl border-l border-[#e9edff]">
            <div className="flex items-center justify-between p-5 border-b border-[#e9edff] sticky top-0 bg-[#ffffff] z-10">
              <h2 className="text-[16px] font-semibold text-[#141b2b]">Add Storage Target</h2>
              <button type="button" onClick={() => setShowAdd(false)} className="p-1.5 rounded-lg text-[#434655] hover:bg-[#e9edff]">
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
                    return (
                      <button key={t} type="button" onClick={() => setForm((f) => ({ ...f, type: t }))}
                        className={`flex flex-col items-center gap-1.5 p-3 rounded-xl border-2 transition-all ${form.type === t ? 'border-[#2563eb] bg-[#dbe1ff]/20' : 'border-[#e9edff] hover:border-[#c3c6d7]'}`}>
                        <TIcon size={20} className={form.type === t ? 'text-[#2563eb]' : 'text-[#434655]'} />
                        <span className="text-[12px] font-medium text-[#141b2b]">{t}</span>
                      </button>
                    )
                  })}
                </div>
              </div>
              {form.type === 'S3' && (
                <>
                  <div><label className={labelCls}>Bucket</label><input type="text" value={form.bucket} onChange={(e) => setForm((f) => ({ ...f, bucket: e.target.value }))} placeholder="my-backup-bucket" className={inputCls} /></div>
                  <div><label className={labelCls}>Region</label><input type="text" value={form.region} onChange={(e) => setForm((f) => ({ ...f, region: e.target.value }))} placeholder="us-east-1" className={inputCls} /></div>
                  <div><label className={labelCls}>Endpoint (optional)</label><input type="text" value={form.endpoint} onChange={(e) => setForm((f) => ({ ...f, endpoint: e.target.value }))} placeholder="https://s3.example.com" className={inputCls} /></div>
                  <div><label className={labelCls}>Access Key</label><input type="text" value={form.access_key} onChange={(e) => setForm((f) => ({ ...f, access_key: e.target.value }))} className={inputCls} /></div>
                  <div><label className={labelCls}>Secret Key</label><input type="password" value={form.secret_key} onChange={(e) => setForm((f) => ({ ...f, secret_key: e.target.value }))} className={inputCls} /></div>
                </>
              )}
              {(form.type === 'LOCAL' || form.type === 'SMB') && (
                <div><label className={labelCls}>Path</label><input type="text" value={form.path} onChange={(e) => setForm((f) => ({ ...f, path: e.target.value }))} placeholder={form.type === 'SMB' ? '\\\\server\\share' : 'D:\\Backups\\'} className={`${inputCls} font-mono text-[13px]`} /></div>
              )}
              <p className="text-[12px] text-[#737686] flex items-center gap-1.5">
                <Lock size={12} className="text-[#006591]" />
                Credentials are encrypted with AES-256-GCM before storage
              </p>
              <button type="submit" disabled={isSubmitting} className="w-full h-9 rounded-lg bg-[#2563eb] text-white text-[13px] font-medium hover:bg-[#1d4ed8] transition-colors disabled:opacity-60 flex items-center justify-center gap-2">
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
    </div>
  )
}

export default StoragePage
