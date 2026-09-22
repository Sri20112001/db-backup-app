import { useEffect, useState } from 'react'
import { runApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import type { BackupRun, BackupArtifact } from '@/types'
import StatusBadge from '@/components/StatusBadge'
import { formatBytes, formatDuration, formatDate, truncate } from '@/utils/format'
import { X, Check, Loader2 } from 'lucide-react'

const STATUSES = ['PENDING', 'RUNNING', 'UPLOADING', 'VERIFYING', 'COMPLETED']

const RunDetailDrawer = ({ runId }: { runId: string }) => {
  const { currentOrg } = useAuthStore()
  const { closeRunDetail, addToast } = useUIStore()
  const [run, setRun] = useState<BackupRun | null>(null)
  const [artifacts, setArtifacts] = useState<BackupArtifact[]>([])
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    if (!currentOrg || !runId) return
    setIsLoading(true)
    Promise.all([
      runApi.get(currentOrg.id, runId),
      runApi.artifacts(currentOrg.id, runId),
    ])
      .then(([r, a]) => { setRun(r); setArtifacts(a) })
      .catch(() => addToast('error', 'Failed to load run details'))
      .finally(() => setIsLoading(false))
  }, [runId, currentOrg])

  const handleCancel = async () => {
    if (!currentOrg || !run) return
    try {
      await runApi.cancel(currentOrg.id, run.id)
      addToast('success', 'Backup cancelled')
      closeRunDetail()
    } catch {
      addToast('error', 'Failed to cancel backup')
    }
  }

  const currentStep = run ? STATUSES.indexOf(run.status) : -1

  return (
    <div className="fixed inset-0 z-[100] flex justify-end">
      <div className="absolute inset-0 bg-[#141b2b]/30 backdrop-blur-[2px]" onClick={closeRunDetail} />
      <div className="relative w-full max-w-[520px] bg-[#ffffff] h-full overflow-y-auto shadow-2xl border-l border-[#e9edff] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-5 border-b border-[#e9edff] sticky top-0 bg-[#ffffff] z-10">
          <div className="flex items-center gap-3">
            <h2 className="text-[16px] font-semibold text-[#141b2b]">Run Details</h2>
            {run && <StatusBadge status={run.status} />}
          </div>
          <button type="button" onClick={closeRunDetail} className="p-1.5 rounded-lg text-[#434655] hover:bg-[#e9edff] transition-colors">
            <X size={18} />
          </button>
        </div>

        {isLoading ? (
          <div className="flex-1 flex items-center justify-center">
            <Loader2 size={32} className="text-[#c3c6d7] animate-spin" />
          </div>
        ) : run ? (
          <div className="flex flex-col gap-5 p-5">
            {/* Status timeline */}
            <div className="flex items-center gap-1">
              {STATUSES.map((s, i) => {
                const done = i < currentStep
                const active = i === currentStep
                const failed = run.status === 'FAILED' && i === currentStep
                const cancelled = run.status === 'CANCELLED' && i === currentStep
                return (
                  <div key={s} className="flex items-center flex-1">
                    <div className="flex flex-col items-center flex-1">
                      <div className={`w-6 h-6 rounded-full flex items-center justify-center text-[11px] font-bold transition-colors ${
                        done ? 'bg-[#006591] text-white' :
                        active && (failed || cancelled) ? 'bg-[#ba1a1a] text-white' :
                        active ? 'bg-[#2563eb] text-white' :
                        'bg-[#e9edff] text-[#737686]'
                      }`}>
                        {done ? <Check size={12} /> : i + 1}
                      </div>
                      <span className="text-[10px] text-[#737686] mt-1 text-center leading-tight">{s}</span>
                    </div>
                    {i < STATUSES.length - 1 && (
                      <div className={`h-0.5 flex-1 mx-1 ${done ? 'bg-[#006591]' : 'bg-[#e9edff]'}`} />
                    )}
                  </div>
                )
              })}
            </div>

            {/* Metrics */}
            <div className="grid grid-cols-2 gap-3">
              {[
                { label: 'Original Size', value: formatBytes(run.bytes_read) },
                { label: 'Compressed', value: formatBytes(run.bytes_compressed) },
                { label: 'Uploaded', value: formatBytes(run.bytes_uploaded) },
                { label: 'Duration', value: formatDuration(run.duration_seconds) },
              ].map((m) => (
                <div key={m.label} className="p-3 rounded-lg bg-[#f1f3ff] border border-[#e9edff]">
                  <p className="text-[11px] text-[#737686] uppercase tracking-wider font-semibold">{m.label}</p>
                  <p className="text-[16px] font-bold text-[#141b2b] mt-0.5">{m.value}</p>
                </div>
              ))}
            </div>

            {/* Details */}
            <div className="flex flex-col gap-2">
              <h3 className="text-[12px] font-semibold uppercase tracking-wider text-[#434655]">Details</h3>
              <div className="flex flex-col gap-1.5 text-[13px]">
                {[
                  { label: 'Job', value: run.backup_job?.name },
                  { label: 'Started', value: formatDate(run.started_at) },
                  { label: 'Completed', value: formatDate(run.completed_at) },
                  { label: 'Source Type', value: run.source_type || run.backup_job?.source_type },
                  { label: 'Storage Path', value: run.storage_path, mono: true },
                  { label: 'Checksum', value: run.checksum, mono: true },
                ].map((d) => d.value ? (
                  <div key={d.label} className="flex items-start gap-3">
                    <span className="text-[#737686] w-28 shrink-0">{d.label}</span>
                    <span className={`text-[#141b2b] break-all ${d.mono ? 'font-mono text-[11px]' : ''}`}>
                      {d.value}
                    </span>
                  </div>
                ) : null)}
              </div>
            </div>

            {/* Error */}
            {run.error_message && (
              <div className="p-3 rounded-lg bg-[#ffdad6] border border-[#fca5a5]">
                <p className="text-[12px] font-semibold text-[#93000a] mb-1">Error</p>
                <p className="font-mono text-[12px] text-[#ba1a1a]">{run.error_message}</p>
              </div>
            )}

            {/* Artifacts */}
            {artifacts.length > 0 && (
              <div className="flex flex-col gap-2">
                <h3 className="text-[12px] font-semibold uppercase tracking-wider text-[#434655]">
                  Artifacts ({artifacts.length})
                </h3>
                <div className="flex flex-col gap-1.5">
                  {artifacts.map((a) => (
                    <div key={a.id} className="flex items-center justify-between p-2.5 rounded-lg bg-[#f1f3ff] border border-[#e9edff]">
                      <div className="min-w-0">
                        <p className="text-[13px] font-medium text-[#141b2b] truncate">{a.name}</p>
                        <p className="font-mono text-[11px] text-[#737686]">{truncate(a.checksum, 8)}</p>
                      </div>
                      <span className="text-[12px] font-mono text-[#434655] shrink-0 ml-3">{formatBytes(a.size)}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Cancel action */}
            {(run.status === 'RUNNING' || run.status === 'UPLOADING') && (
              <button
                type="button"
                onClick={handleCancel}
                className="w-full h-9 rounded-lg bg-[#ffffff] border border-[#fca5a5] text-[#ba1a1a] text-[13px] font-medium hover:bg-[#ffdad6] transition-colors"
              >
                Cancel Backup
              </button>
            )}
          </div>
        ) : null}
      </div>
    </div>
  )
}

export default RunDetailDrawer
