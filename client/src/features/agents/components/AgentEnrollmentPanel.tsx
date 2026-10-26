import { useEffect, useState } from 'react'
import { agentApi } from '@/services/api'
import { API_BASE_URL as BASE } from '@/CONSTANTS'
import { useAuthStore } from '@/store/authStore'
import { useUIStore } from '@/store/uiStore'
import type { EnrollmentTokenInfo, EnrollmentTokenResult } from '@/types'
import { formatRelative, maskEnrollmentToken } from '@/utils/format'
import { AlertTriangle, Check, Copy, Loader2 } from 'lucide-react'

interface AgentEnrollmentPanelProps {
  defaultName?: string
  onGenerated?: (result: EnrollmentTokenResult) => void
}

const copyToClipboard = (text: string) => {
  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(text)
    return
  }
  // Fallback for insecure contexts (like exposed IPs without HTTPS)
  const textArea = document.createElement('textarea')
  textArea.value = text
  textArea.style.position = 'fixed'
  textArea.style.left = '-9999px'
  document.body.appendChild(textArea)
  textArea.focus()
  textArea.select()
  try {
    document.execCommand('copy')
  } catch (err) {
    console.error('Fallback copy failed', err)
  }
  document.body.removeChild(textArea)
}

// Rail badge: numbered while pending/active, check when done.
const StepBadge = ({ n, state }: { n: number; state: 'done' | 'active' | 'pending' }) => (
  <span
    aria-hidden="true"
    className={`w-6 h-6 rounded-full flex items-center justify-center text-[11px] font-semibold shrink-0 border transition-colors ${
      state === 'done'
        ? 'bg-primary/15 text-primary border-primary/40'
        : state === 'active'
          ? 'bg-primary text-on-primary border-primary'
          : 'bg-surface-container-low text-on-surface-variant border-surface-variant'
    }`}
  >
    {state === 'done' ? <Check size={13} /> : n}
  </span>
)

const stepTitle = (active: boolean) =>
  `text-[13px] font-semibold leading-6 ${active ? 'text-on-surface' : 'text-on-surface-variant'}`

// Three-step enrollment flow shared by the Agents page modal and the
// onboarding wizard: name and generate, copy the one-time token, then run
// the setup on the target machine. Steps 2-3 stay visible but quiet until
// a token exists, so the whole sequence is always legible.
const AgentEnrollmentPanel = ({ defaultName = '', onGenerated }: AgentEnrollmentPanelProps) => {
  const { currentOrg } = useAuthStore()
  const { addToast } = useUIStore()
  const [agentName, setAgentName] = useState(defaultName)
  const [enroll, setEnroll] = useState<EnrollmentTokenResult | null>(null)
  const [isGenerating, setIsGenerating] = useState(false)
  const [copied, setCopied] = useState(false)
  const [copiedUrl, setCopiedUrl] = useState(false)
  const [pendingTokens, setPendingTokens] = useState<EnrollmentTokenInfo[]>([])
  const [revokingTokenId, setRevokingTokenId] = useState<string | null>(null)

  // Absolute API URL for the setup wizard's "Server URL" prompt.
  const serverUrl = BASE.startsWith('http') ? BASE : `${window.location.origin}${BASE}`

  const loadPendingTokens = async () => {
    if (!currentOrg) return
    try {
      const toks = await agentApi.listTokens(currentOrg.id)
      const now = Date.now()
      setPendingTokens(toks.filter((t) => !t.used_at && !t.revoked_at && new Date(t.expires_at).getTime() > now))
    } catch {
      // Token management is auxiliary; the panel works without the list.
      setPendingTokens([])
    }
  }

  useEffect(() => {
    void loadPendingTokens()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentOrg])

  const handleGenerateToken = async () => {
    if (!currentOrg) return
    setIsGenerating(true)
    try {
      const token = await agentApi.enrollmentToken(currentOrg.id, { name: agentName.trim() || undefined })
      setEnroll(token)
      onGenerated?.(token)
      void loadPendingTokens()
    } catch { addToast('error', 'Failed to generate enrollment token') }
    finally { setIsGenerating(false) }
  }

  const handleCopy = () => {
    if (!enroll) return
    copyToClipboard(enroll.enrollment_token)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const handleCopyUrl = () => {
    copyToClipboard(serverUrl)
    setCopiedUrl(true)
    setTimeout(() => setCopiedUrl(false), 2000)
  }

  const handleRevokeToken = async (id: string) => {
    if (!currentOrg) return
    setRevokingTokenId(id)
    try {
      await agentApi.revokeToken(currentOrg.id, id)
      addToast('success', 'Enrollment token revoked')
      setPendingTokens((prev) => prev.filter((t) => t.id !== id))
    } catch {
      addToast('error', 'Failed to revoke token')
    } finally {
      setRevokingTokenId(null)
    }
  }

  const hasToken = enroll !== null
  const named = agentName.trim()

  return (
    <div>
      <ol className="flex flex-col">
        {/* 1 — name and generate */}
        <li className="flex gap-3">
          <div className="flex flex-col items-center gap-1">
            <StepBadge n={1} state={hasToken ? 'done' : 'active'} />
            <span className="w-px flex-1 min-h-4 bg-surface-variant" aria-hidden="true" />
          </div>
          <div className="flex-1 min-w-0 pb-4">
            <p className={stepTitle(true)}>Name this agent</p>
            {hasToken ? (
              <div className="mt-0.5 flex items-center justify-between gap-2">
                <p className="text-[12.5px] text-on-surface-variant truncate">
                  Token generated{named ? ` for ${named}` : ''} · expires {formatRelative(enroll.expires_at)}
                </p>
                <button
                  type="button"
                  onClick={() => setEnroll(null)}
                  className="text-[12px] text-primary hover:underline shrink-0"
                >
                  Generate another
                </button>
              </div>
            ) : (
              <div className="mt-2 flex flex-col gap-2.5">
                <div className="flex items-center justify-between">
                  <label htmlFor="vg-agent-name" className="text-[12.5px] text-on-surface-variant">
                    Agent name
                  </label>
                  <span className="text-[11.5px] text-on-surface-variant">Optional — rename it later</span>
                </div>
                <input
                  id="vg-agent-name"
                  type="text"
                  value={agentName}
                  onChange={(e) => setAgentName(e.target.value)}
                  placeholder="e.g. db-primary-01"
                  autoComplete="off"
                  className="w-full h-9 px-3 rounded-lg border border-surface-variant bg-surface-container-low text-[13px] text-on-surface placeholder:text-on-surface-variant focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all"
                />
                <button
                  type="button"
                  onClick={() => void handleGenerateToken()}
                  disabled={isGenerating}
                  className="w-full h-9 rounded-lg bg-primary text-on-primary text-[13px] font-medium hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
                >
                  {isGenerating && <Loader2 size={14} className="animate-spin" />}
                  Generate token
                </button>
              </div>
            )}
          </div>
        </li>

        {/* 2 — copy the token (the one hero element) */}
        <li className="flex gap-3">
          <div className="flex flex-col items-center gap-1">
            <StepBadge n={2} state={hasToken ? 'active' : 'pending'} />
            <span className="w-px flex-1 min-h-4 bg-surface-variant" aria-hidden="true" />
          </div>
          <div className="flex-1 min-w-0 pb-4">
            <p className={stepTitle(hasToken)}>Copy the one-time token</p>
            {hasToken && (
              <div className="mt-2 flex flex-col gap-2">
                <div className="flex items-stretch gap-1.5 p-1.5 pl-3 rounded-lg bg-surface-container-highest border border-primary/40">
                  <code
                    className="flex-1 min-w-0 self-center font-mono text-[13px] text-on-surface break-all"
                    title="Full token hidden — use Copy"
                  >
                    {maskEnrollmentToken(enroll.enrollment_token)}
                  </code>
                  <button
                    type="button"
                    onClick={handleCopy}
                    aria-label={copied ? 'Token copied' : 'Copy enrollment token'}
                    className={`shrink-0 h-8 px-3 rounded-md flex items-center gap-1.5 text-[12px] font-medium transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 ${
                      copied
                        ? 'bg-success-container text-on-success-container'
                        : 'bg-primary text-on-primary hover:bg-primary/90'
                    }`}
                  >
                    {copied ? <Check size={14} /> : <Copy size={14} />}
                    {copied ? 'Copied' : 'Copy'}
                  </button>
                </div>
                <div className="flex flex-wrap items-center gap-1.5">
                  <span className="px-2 py-0.5 rounded-full border border-outline-variant bg-surface-container-low text-[11px] font-medium text-on-surface-variant">
                    Single-use
                  </span>
                  <span
                    className="px-2 py-0.5 rounded-full border border-outline-variant bg-surface-container-low font-mono text-[11px] text-on-surface-variant"
                    title={new Date(enroll.expires_at).toLocaleString()}
                  >
                    Expires {formatRelative(enroll.expires_at)}
                  </span>
                </div>
                <p className="flex items-start gap-1.5 text-[12px] text-on-surface-variant leading-relaxed">
                  <AlertTriangle size={13} className="text-warning shrink-0 mt-0.5" />
                  <span>
                    Shown once — copy it now; only the masked form stays on screen.
                    The agent receives its permanent credential when it enrolls.
                  </span>
                </p>
              </div>
            )}
          </div>
        </li>

        {/* 3 — run setup on the machine */}
        <li className="flex gap-3">
          <StepBadge n={3} state={hasToken ? 'active' : 'pending'} />
          <div className="flex-1 min-w-0">
            <p className={stepTitle(hasToken)}>Run setup on the machine</p>
            {hasToken && (
              <div className="mt-2 flex flex-col gap-2">
                <p className="text-[12.5px] text-on-surface-variant leading-relaxed">
                  Run{' '}
                  <code className="font-mono text-[12px] px-1.5 py-0.5 rounded bg-surface-container-high text-on-surface">
                    VaultGuard-Agent-Setup.exe
                  </code>{' '}
                  as administrator on the machine you want to protect, then paste the token when prompted.
                </p>
                <div className="flex items-center gap-2 px-2.5 py-2 rounded-lg bg-surface-container-low border border-surface-variant">
                  <span className="text-[11.5px] text-on-surface-variant shrink-0">Server URL</span>
                  <code
                    className="flex-1 min-w-0 truncate font-mono text-[11.5px] text-on-surface-variant"
                    title={serverUrl}
                  >
                    {serverUrl}
                  </code>
                  <button
                    type="button"
                    onClick={handleCopyUrl}
                    aria-label={copiedUrl ? 'Server URL copied' : 'Copy server URL'}
                    className={`shrink-0 p-1.5 rounded transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 ${
                      copiedUrl ? 'text-success' : 'text-on-surface-variant hover:bg-surface-container-high'
                    }`}
                  >
                    {copiedUrl ? <Check size={14} /> : <Copy size={14} />}
                  </button>
                </div>
                <p className="text-[11.5px] text-on-surface-variant leading-relaxed">
                  Console alternative:{' '}
                  <span className="font-mono">VaultGuard-Agent-Setup-Console.exe</span>{' '}
                  — paste the token at the prompt.
                </p>
              </div>
            )}
          </div>
        </li>
      </ol>

      {pendingTokens.length > 0 && (
        <div className="mt-4 pt-4 border-t border-surface-variant">
          <div className="flex items-center gap-2 mb-2">
            <p className="text-[12px] font-semibold text-on-surface-variant">Outstanding tokens</p>
            <span className="min-w-[18px] h-[18px] px-1 rounded-full bg-surface-container-high text-[10.5px] font-semibold text-on-surface-variant flex items-center justify-center">
              {pendingTokens.length}
            </span>
          </div>
          <div className="flex flex-col gap-1.5 max-h-32 overflow-y-auto">
            {pendingTokens.map((t) => (
              <div key={t.id} className="flex items-center justify-between gap-2 px-2.5 py-1.5 rounded-lg bg-surface-container-low border border-surface-variant">
                <span
                  className="font-mono text-[11px] text-on-surface-variant truncate"
                  title={`Agent ${t.agent_id} · expires ${new Date(t.expires_at).toLocaleString()}`}
                >
                  {t.agent_id.slice(0, 8)}… · expires {formatRelative(t.expires_at)}
                </span>
                <button
                  type="button"
                  disabled={revokingTokenId === t.id}
                  onClick={() => void handleRevokeToken(t.id)}
                  className="text-[11px] font-medium text-error hover:underline disabled:opacity-50 shrink-0 rounded px-1 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
                >
                  {revokingTokenId === t.id ? 'Revoking…' : 'Revoke'}
                </button>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

export default AgentEnrollmentPanel
