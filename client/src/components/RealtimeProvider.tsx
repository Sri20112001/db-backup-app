import { useEffect, useRef } from 'react'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import { useRealtimeStore } from '@/stores/realtimeStore'
import type { IncomingEvent } from '@/stores/realtimeStore'

// Derive ws(s)://host/<base>/ws from the REST base URL so proxies and
// basenames (e.g. /vaultguard/api) carry over automatically.
function socketUrl(): string | null {
  const base = import.meta.env.VITE_API_URL as string | undefined
  if (!base) return null
  const ws = base.replace(/^http/, 'ws').replace(/\/$/, '')
  return `${ws}/ws`
}

// Owns the single dashboard socket: connects when signed in, subscribes to
// the active organization, reconnects with backoff, and funnels alert pushes
// into toasts. Pages consume useRealtimeStore for row patching/refetching.
const RealtimeProvider = () => {
  const { user, currentOrg } = useAuthStore()
  const addToast = useUIStore((s) => s.addToast)
  const pendingToast = useRealtimeStore((s) => s.pendingToast)
  const consumeToast = useRealtimeStore((s) => s.consumeToast)
  const orgIdRef = useRef<string | null>(null)
  const wsRef = useRef<WebSocket | null>(null)

  useEffect(() => {
    if (!user || !currentOrg) return
    const url = socketUrl()
    if (!url) return

    let stopped = false
    let socket: WebSocket | null = null
    let retryMs = 2000
    let retryTimer: ReturnType<typeof setTimeout> | null = null
    const store = useRealtimeStore.getState()
    store.setStatus('connecting')

    const connect = () => {
      if (stopped) return
      const token = localStorage.getItem('access_token')
      if (!token) {
        retryTimer = setTimeout(connect, retryMs)
        return
      }
      let ws: WebSocket
      try {
        ws = new WebSocket(`${url}?token=${encodeURIComponent(token)}`)
      } catch {
        retryTimer = setTimeout(connect, retryMs)
        return
      }
      socket = ws
      wsRef.current = ws

      ws.onopen = () => {
        retryMs = 2000
        useRealtimeStore.getState().setStatus('open')
        const orgId = orgIdRef.current
        if (orgId) ws.send(JSON.stringify({ type: 'subscribe', org_id: orgId }))
      }
      ws.onmessage = (msg) => {
        try {
          useRealtimeStore.getState()._ingest(JSON.parse(String(msg.data)) as IncomingEvent)
        } catch {
          // ignore malformed frames
        }
      }
      const schedule = () => {
        useRealtimeStore.getState().setStatus('closed')
        wsRef.current = null
        if (!stopped) retryTimer = setTimeout(() => {
          retryMs = Math.min(retryMs * 2, 30000)
          connect()
        }, retryMs)
      }
      ws.onclose = schedule
      ws.onerror = () => ws.close()
    }

    connect()
    return () => {
      stopped = true
      if (retryTimer) clearTimeout(retryTimer)
      wsRef.current = null
      try {
        socket?.close()
      } catch {
        // already gone
      }
      useRealtimeStore.getState().setStatus('closed')
    }
  }, [user, currentOrg])

  // (Re)subscribe whenever the active org changes.
  useEffect(() => {
    orgIdRef.current = currentOrg?.id ?? null
    const ws = wsRef.current
    if (ws && ws.readyState === WebSocket.OPEN && currentOrg) {
      ws.send(JSON.stringify({ type: 'subscribe', org_id: currentOrg.id }))
    }
  }, [currentOrg])

  // Alert pushes become toasts exactly once.
  useEffect(() => {
    if (!pendingToast) return
    addToast(pendingToast.kind, `${pendingToast.title}${pendingToast.message ? ` — ${pendingToast.message}` : ''}`)
    consumeToast()
  }, [pendingToast, addToast, consumeToast])

  return null
}

export default RealtimeProvider
