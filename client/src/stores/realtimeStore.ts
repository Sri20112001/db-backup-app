import { create } from 'zustand'

export interface RunPatch {
  id: string
  backup_job_id?: string
  agent_id?: string
  organization_id?: string
  status?: string
  bytes_read?: number
  bytes_compressed?: number
  bytes_uploaded?: number
  checksum?: string
  error_message?: string
  storage_path?: string
}

export interface IncomingEvent {
  type: 'run' | 'presence' | 'alert' | 'jobs' | 'restores' | 'agents' | 'log'
  org_id?: string
  payload?: unknown
}

interface ToastMsg {
  id: number
  kind: 'success' | 'error' | 'warning' | 'info'
  title: string
  message: string
}

export interface PresenceMsg {
  id?: string
  name?: string
  status?: string
  seq: number
}

interface RealtimeState {
  status: 'connecting' | 'open' | 'closed'
  setStatus: (s: RealtimeState['status']) => void
  // Latest run event (with sequence so identical payloads still trigger).
  lastRunEvent: (RunPatch & { seq: number }) | null
  // Bump counters: pages refetch their lists when these change.
  entitySeq: { jobs: number; restores: number; agents: number }
  presenceSeq: number
  alertTick: number
  // Alert pushed from the socket, consumed once by the provider into a toast.
  pendingToast: ToastMsg | null
  consumeToast: () => void
  // Live log lines per run id (capped).
  logs: Record<string, string[]>
  // Latest presence payload (agent id/name/status + seq).
  presence: PresenceMsg | null
  _ingest: (ev: IncomingEvent) => void
  _seq: number
}

const asRecord = (p: unknown): Record<string, unknown> =>
  typeof p === 'object' && p !== null ? (p as Record<string, unknown>) : {}

const str = (v: unknown): string => (typeof v === 'string' ? v : '')
const num = (v: unknown): number | undefined => (typeof v === 'number' ? v : undefined)

// mergeRunPatch overlays a realtime run event onto a cached row, keeping
// every field the event omits. Returns the rows unchanged when the id is
// unknown so callers can refetch instead.
export function mergeRunPatch<T extends { id: string }>(rows: T[], patch: RunPatch): T[] {
  let known = false
  const next = rows.map((r) => {
    if (r.id !== patch.id) return r
    known = true
    const merged: Record<string, unknown> = { ...r }
    for (const [k, v] of Object.entries(patch)) {
      if (k !== 'id' && v !== undefined) merged[k] = v
    }
    return merged as T
  })
  return known ? next : rows
}

let toastId = 0

export const useRealtimeStore = create<RealtimeState>()((set, get) => ({
  status: 'closed',
  setStatus: (status) => set({ status }),
  lastRunEvent: null,
  entitySeq: { jobs: 0, restores: 0, agents: 0 },
  presenceSeq: 0,
  alertTick: 0,
  pendingToast: null,
  consumeToast: () => set({ pendingToast: null }),
  logs: {},
  presence: null,
  _seq: 0,
  _ingest: (ev) => {
    const s = get()
    const seq = s._seq + 1
    switch (ev.type) {
      case 'run': {
        const p = asRecord(ev.payload)
        const id = str(p.id)
        if (!id) return
        set({
          _seq: seq,
          lastRunEvent: {
            seq,
            id,
            backup_job_id: str(p.backup_job_id) || undefined,
            agent_id: str(p.agent_id) || undefined,
            organization_id: str(p.organization_id) || undefined,
            status: str(p.status) || undefined,
            bytes_read: num(p.bytes_read),
            bytes_compressed: num(p.bytes_compressed),
            bytes_uploaded: num(p.bytes_uploaded),
            checksum: str(p.checksum) || undefined,
            error_message: str(p.error_message) || undefined,
            storage_path: str(p.storage_path) || undefined,
          },
        })
        return
      }
      case 'presence': {
        const p = asRecord(ev.payload)
        set({
          _seq: seq,
          presenceSeq: s.presenceSeq + 1,
          presence: { id: str(p.id) || undefined, name: str(p.name) || undefined, status: str(p.status) || undefined, seq },
        })
        return
      }
      case 'alert': {
        const p = asRecord(ev.payload)
        const title = str(p.title) || 'Alert'
        const message = str(p.message) || ''
        const kind =
          String(p.type ?? '').includes('FAILED') || String(p.type ?? '').includes('OFFLINE')
            ? 'error'
            : 'warning';
        set({
          _seq: seq,
          alertTick: s.alertTick + 1,
          pendingToast: { id: ++toastId, kind: kind as ToastMsg['kind'], title, message },
        })
        return
      }
      case 'jobs':
      case 'restores':
      case 'agents': {
        const key = ev.type as 'jobs' | 'restores' | 'agents'
        set({ _seq: seq, entitySeq: { ...s.entitySeq, [key]: s.entitySeq[key] + 1 } })
        return
      }
      case 'log': {
        const p = asRecord(ev.payload)
        const runId = str(p.run_id)
        const line = str(p.line)
        if (!runId || !line) return
        const prev = s.logs[runId] ?? []
        set({ _seq: seq, logs: { ...s.logs, [runId]: [...prev.slice(-199), line] } })
        return
      }
      default:
        return
    }
  },
}))