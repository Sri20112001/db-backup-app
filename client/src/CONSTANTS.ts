// ── Central application configuration ────────────────────────────────────────
// All `import.meta.env` reads live in this file. Import these constants
// everywhere else instead of touching `import.meta.env` directly.
//
// NOTE: Vite replaces `import.meta.env.*` at build time, so every value here
// is baked into the bundle. Changing `.env` requires a rebuild (`npm run
// build` / docker rebuild) to take effect.

const env = import.meta.env

// Base URL of the VaultGuard REST API (no trailing slash).
// Relative by default so the dashboard works on localhost, LAN IPs, and
// behind the docker nginx proxy. In `npm run dev`, vite.config.ts proxies
// this path to the Go server.
export const API_BASE_URL: string = env.VITE_API_URL || '/vaultguard/api'

// Subpath the SPA is served from (vite `base`, e.g. `/vaultguard`).
export const APP_BASE_PATH: string = (env.BASE_URL ?? '/').replace(/\/$/, '')

// In-app auth routes as absolute paths, for hard redirects (which can't use
// react-router's basename-aware `navigate`).
export const LOGIN_URL = `${APP_BASE_PATH}/login`
export const REGISTER_URL = `${APP_BASE_PATH}/register`

// Loopback address of the agent folder-browser used by the folder picker and
// database pickers. Remote browsers can't reach it; type paths manually there.
export const AGENT_BROWSE_BASE_URL: string =
  env.VITE_AGENT_BROWSE_URL || 'http://127.0.0.1:7546'

// Derives an absolute ws(s):// URL (e.g. `<base>/ws`) from an HTTP(S) base
// URL. Relative bases resolve against the page origin so dev, LAN, and
// proxied deployments all work. Returns null when no base is configured.
export function toWebSocketUrl(base: string | undefined, origin = window.location.origin): string | null {
  if (!base) return null
  const absolute = base.startsWith('http')
    ? base
    : `${origin}${base.startsWith('/') ? '' : '/'}${base}`
  return absolute.replace(/^http/, 'ws').replace(/\/$/, '')
}
