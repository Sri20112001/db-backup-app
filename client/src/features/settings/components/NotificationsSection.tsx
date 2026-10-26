import { useEffect, useState } from 'react'
import { notificationApi } from '@/services/api'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import type { NotificationSettings, AlertSeverity, UserNotificationPrefs } from '@/types'
import { Loader2, Send, Webhook, Moon, Gauge, Mail } from 'lucide-react'

const SEVERITIES: AlertSeverity[] = ['info', 'warning', 'critical']

const NotificationsSection = () => {
  const { currentOrg } = useAuthStore()
  const { addToast } = useUIStore()
  const [settings, setSettings] = useState<NotificationSettings | null>(null)
  const [prefs, setPrefs] = useState<UserNotificationPrefs | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [webhookSecret, setWebhookSecret] = useState('')
  const [isLoading, setIsLoading] = useState(true)
  const [isSaving, setIsSaving] = useState(false)
  const [isSavingPrefs, setIsSavingPrefs] = useState(false)
  const [isTesting, setIsTesting] = useState(false)

  useEffect(() => {
    if (!currentOrg) return
    load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentOrg])

  const load = () => {
    if (!currentOrg) return
    setIsLoading(true)
    setLoadError(null)
    Promise.all([
      notificationApi.getSettings(currentOrg.id),
      notificationApi.getMyPrefs(),
    ])
      .then(([s, p]) => { setSettings(s); setPrefs(p) })
      .catch((e: unknown) => {
        const msg = e instanceof Error ? e.message : 'unknown error'
        setLoadError(msg)
        addToast('error', `Failed to load notification settings: ${msg}`)
      })
      .finally(() => setIsLoading(false))
  }

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!currentOrg || !settings) return
    setIsSaving(true)
    try {
      const body: Record<string, unknown> = {
        webhook_url: settings.webhook_url,
        webhook_enabled: settings.webhook_enabled,
        quiet_hours_start: settings.quiet_hours_start,
        quiet_hours_end: settings.quiet_hours_end,
        min_email_severity: settings.min_email_severity,
        offline_threshold_minutes: settings.offline_threshold_minutes,
        missed_window_multiplier: settings.missed_window_multiplier,
      }
      // Only send the secret when the operator typed one; empty keeps existing.
      if (webhookSecret.trim() !== '') body.webhook_secret = webhookSecret.trim()
      const saved = await notificationApi.updateSettings(currentOrg.id, body)
      setSettings(saved)
      setWebhookSecret('')
      addToast('success', 'Notification settings saved')
    } catch (err: unknown) {
      addToast('error', err instanceof Error ? err.message : 'Failed to save settings')
    } finally {
      setIsSaving(false)
    }
  }

  const handleTestEmail = async () => {
    if (!currentOrg) return
    setIsTesting(true)
    try {
      const res = await notificationApi.testEmail(currentOrg.id)
      addToast('success', `Test email sent to ${res.to}`)
    } catch (err: unknown) {
      addToast('error', err instanceof Error ? err.message : 'Failed to send test email')
    } finally {
      setIsTesting(false)
    }
  }

  const handleSavePrefs = async (next: { email_enabled?: boolean; min_severity?: AlertSeverity }) => {
    setIsSavingPrefs(true)
    try {
      const saved = await notificationApi.updateMyPrefs(next)
      setPrefs(saved)
      addToast('success', 'Email preferences saved')
    } catch (err: unknown) {
      addToast('error', err instanceof Error ? err.message : 'Failed to save preferences')
    } finally {
      setIsSavingPrefs(false)
    }
  }

  const inputCls = 'w-full h-9 px-3 rounded-lg border border-surface-variant bg-surface-container-low text-[14px] text-on-surface focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all'
  const labelCls = 'block text-[13px] font-medium text-on-surface-variant mb-1.5'

  if (isLoading) {
    return (
      <div className="flex flex-col gap-4 p-5 rounded-xl bg-surface-container-lowest border border-surface-variant shadow-sm">
        {[...Array(3)].map((_, i) => <div key={i} className="animate-pulse h-14 rounded-lg bg-surface-container-high" />)}
      </div>
    )
  }

  // A failed load shows a real error (with retry) instead of hanging on the
  // skeleton forever — the previous behaviour hid the cause from the user.
  if (loadError || !settings || !prefs) {
    return (
      <div role="alert" className="flex flex-col items-center gap-3 p-8 rounded-xl bg-surface-container-lowest border border-surface-variant shadow-sm text-center">
        <p className="text-[14px] font-semibold text-on-surface">Couldn't load notification settings</p>
        <p className="text-[12.5px] text-on-surface-variant max-w-md">{loadError ?? 'The server returned an unexpected response.'}</p>
        <button
          type="button"
          onClick={load}
          className="px-4 h-9 rounded-lg bg-primary text-on-primary text-[13px] font-medium hover:bg-primary-container transition-colors"
        >
          Retry
        </button>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-6">
      {/* Org-level routing */}
      <form onSubmit={handleSave} className="flex flex-col gap-5 p-5 rounded-xl bg-surface-container-lowest border border-surface-variant shadow-sm">
        <div className="flex items-center gap-2">
          <Webhook size={16} className="text-primary" />
          <h2 className="text-[14px] font-semibold text-on-surface">Alert Delivery</h2>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <div>
            <label className={labelCls}>Webhook URL</label>
            <input
              type="url"
              value={settings.webhook_url}
              onChange={(e) => setSettings((s) => s && { ...s, webhook_url: e.target.value })}
              placeholder="https://hooks.example.com/vaultguard"
              className={inputCls}
            />
          </div>
          <div>
            <label className={labelCls}>Webhook Secret <span className="font-normal text-on-surface-variant">(HMAC-SHA256 signing)</span></label>
            <input
              type="password"
              value={webhookSecret}
              onChange={(e) => setWebhookSecret(e.target.value)}
              placeholder="••••••••  (leave blank to keep current)"
              autoComplete="new-password"
              className={inputCls}
            />
          </div>
        </div>

        <label className="flex items-center gap-2 text-[13px] text-on-surface cursor-pointer w-fit">
          <input
            type="checkbox"
            checked={settings.webhook_enabled}
            onChange={(e) => setSettings((s) => s && { ...s, webhook_enabled: e.target.checked })}
            className="w-4 h-4 rounded accent-primary"
          />
          Deliver new alerts to webhook
        </label>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
          <div>
            <label className={labelCls}>Email Severity Floor</label>
            <select
              value={settings.min_email_severity}
              onChange={(e) => setSettings((s) => s && { ...s, min_email_severity: e.target.value as AlertSeverity })}
              className={inputCls}
            >
              {SEVERITIES.map((v) => <option key={v} value={v}>{v}</option>)}
            </select>
          </div>
          <div>
            <label className={labelCls}>Quiet Hours Start (UTC hour, -1 off)</label>
            <input
              type="number"
              min={-1}
              max={23}
              value={settings.quiet_hours_start}
              onChange={(e) => setSettings((s) => s && { ...s, quiet_hours_start: parseInt(e.target.value, 10) })}
              className={inputCls}
            />
          </div>
          <div>
            <label className={labelCls}>Quiet Hours End (UTC hour, -1 off)</label>
            <input
              type="number"
              min={-1}
              max={23}
              value={settings.quiet_hours_end}
              onChange={(e) => setSettings((s) => s && { ...s, quiet_hours_end: parseInt(e.target.value, 10) })}
              className={inputCls}
            />
          </div>
        </div>
        <p className="text-[12px] text-on-surface-variant -mt-2 flex items-center gap-1">
          <Moon size={12} />
          Quiet hours suppress non-critical email only (UTC, wrap-midnight supported, e.g. 22 → 6).
        </p>

        <div className="flex items-end justify-between gap-4 flex-wrap">
          <div className="flex items-center gap-4 flex-wrap">
            <div className="flex items-center gap-1.5 text-[13px] text-on-surface-variant">
              <Gauge size={14} className="text-primary" />
              <label className="font-medium">Offline threshold (min)</label>
              <input
                type="number"
                min={1}
                max={60}
                value={settings.offline_threshold_minutes}
                onChange={(e) => setSettings((s) => s && { ...s, offline_threshold_minutes: parseInt(e.target.value, 10) })}
                className="w-16 h-8 px-2 rounded-lg border border-surface-variant bg-surface-container-low text-[13px] text-on-surface focus:outline-none"
              />
            </div>
            <div className="flex items-center gap-1.5 text-[13px] text-on-surface-variant">
              <label className="font-medium">Missed window ×</label>
              <input
                type="number"
                min={1}
                max={24}
                value={settings.missed_window_multiplier}
                onChange={(e) => setSettings((s) => s && { ...s, missed_window_multiplier: parseInt(e.target.value, 10) })}
                className="w-16 h-8 px-2 rounded-lg border border-surface-variant bg-surface-container-low text-[13px] text-on-surface focus:outline-none"
              />
            </div>
          </div>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={handleTestEmail}
              disabled={isTesting}
              className="flex items-center gap-1.5 px-3 h-9 rounded-lg bg-surface-container-low border border-surface-variant text-on-surface text-[13px] font-medium hover:bg-surface-container-high transition-colors disabled:opacity-60"
            >
              {isTesting ? <Loader2 size={14} className="animate-spin" /> : <Send size={14} />}
              Send test email
            </button>
            <button
              type="submit"
              disabled={isSaving}
              className="flex items-center gap-1.5 px-4 h-9 rounded-lg bg-primary text-on-primary text-[13px] font-medium hover:bg-primary-container transition-colors disabled:opacity-60"
            >
              {isSaving && <Loader2 size={14} className="animate-spin" />}
              Save
            </button>
          </div>
        </div>
      </form>

      {/* Per-user prefs */}
      <div className="flex flex-col gap-4 p-5 rounded-xl bg-surface-container-lowest border border-surface-variant shadow-sm">
        <div className="flex items-center gap-2">
          <Mail size={16} className="text-primary" />
          <h2 className="text-[14px] font-semibold text-on-surface">My Email Notifications</h2>
        </div>
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 items-end">
          <label className="flex items-center gap-2 text-[13px] text-on-surface cursor-pointer w-fit">
            <input
              type="checkbox"
              checked={prefs.email_enabled}
              disabled={isSavingPrefs}
              onChange={(e) => handleSavePrefs({ email_enabled: e.target.checked })}
              className="w-4 h-4 rounded accent-primary"
            />
            Email me alerts
          </label>
          <div>
            <label className={labelCls}>Minimum Severity</label>
            <select
              value={prefs.min_severity}
              disabled={isSavingPrefs}
              onChange={(e) => handleSavePrefs({ min_severity: e.target.value as AlertSeverity })}
              className={inputCls}
            >
              {SEVERITIES.map((v) => <option key={v} value={v}>{v}</option>)}
            </select>
          </div>
        </div>
        <p className="text-[12px] text-on-surface-variant">
          Applies to you only, in addition to the organization severity floor above.
        </p>
      </div>
    </div>
  )
}

export default NotificationsSection
