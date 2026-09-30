import type {
  AuthTokens, BackupJob, BackupRun, BackupArtifact,
  Agent, Machine, StorageTarget, RestoreJob, Alert,
  DashboardOverview, JobHealth, OrganizationMember,
  Organization, PaginatedResponse, PreflightResult, S3Region, User
} from '../types'

import { useAuthStore } from '../store/authStore'

import { API_BASE_URL as BASE, LOGIN_URL, REGISTER_URL } from '../CONSTANTS'

function getToken() {
  return useAuthStore.getState().accessToken
}

function getRefreshToken() {
  return useAuthStore.getState().refreshToken
}

// The session is unrecoverable: wipe auth state and bounce to the login
// page. Skips the redirect on the auth pages themselves (a login/register
// 401 is a credential error for the form, not session expiry). Never
// settles the caller — the redirect unmounts the app anyway.
function forceLogout() {
  useAuthStore.getState().logout()
  const p = window.location.pathname
  if (!p.startsWith(LOGIN_URL) && !p.startsWith(REGISTER_URL)) {
    window.location.href = LOGIN_URL
  }
}

let refreshPromise: Promise<void> | null = null

async function refreshAccessToken(): Promise<void> {
  const rt = getRefreshToken()
  if (!rt) throw new Error('no refresh token')
  const res = await fetch(`${BASE}/auth/refresh`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refresh_token: rt }),
  })
  if (!res.ok) {
    forceLogout()
    throw new Error('session expired')
  }
  const data = await res.json()
  const { user } = useAuthStore.getState()
  useAuthStore.getState().setAuth(user!, data.access_token, data.refresh_token)
}

async function request<T>(path: string, options: RequestInit = {}, retry = true): Promise<T> {
  const token = getToken()
  const res = await fetch(`${BASE}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options.headers,
    },
  })
  if (res.status === 401) {
    // Login/register 401s are credential errors — surface them so the form
    // can show a message instead of logging out.
    const isAuthForm = path === '/auth/login' || path === '/auth/register'
    if (!isAuthForm && token) {
      if (retry && getRefreshToken()) {
        if (!refreshPromise) {
          refreshPromise = refreshAccessToken().finally(() => { refreshPromise = null })
        }
        await refreshPromise
        return request<T>(path, options, false)
      }
      // No refresh token left, or the retried request was rejected again —
      // the session is dead, so log out.
      forceLogout()
      return new Promise<never>(() => {})
    }
  }
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(err.error || 'Request failed')
  }
  if (res.status === 204) return undefined as T
  return res.json()
}

// Auth
export const authApi = {
  login: async (email: string, password: string) => {
    const data = await request<AuthTokens>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    })
    // Tokens are stored via setAuth into Zustand persist — no direct localStorage
    const { setAuth } = useAuthStore.getState()
    setAuth(data.user as User, data.access_token, data.refresh_token)
    return data
  },
  register: (email: string, password: string, name: string) =>
    request<{ id: string; email: string; name: string }>('/auth/register', {
      method: 'POST', body: JSON.stringify({ email, password, name }),
    }),
  refresh: (refresh_token: string) =>
    request<{ access_token: string; refresh_token: string }>('/auth/refresh', {
      method: 'POST', body: JSON.stringify({ refresh_token }),
    }),
  logout: (refresh_token: string) =>
    request('/auth/logout', { method: 'POST', body: JSON.stringify({ refresh_token }) }),
  changePassword: (currentPassword: string, newPassword: string) =>
    request('/auth/change-password', { method: 'POST', body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }) }),
}

// Organizations
export const orgApi = {
  list: () => request<Organization[]>('/organizations'),
  create: (name: string) =>
    request<Organization>('/organizations', { method: 'POST', body: JSON.stringify({ name }) }),
  get: (orgId: string) => request<Organization>(`/organizations/${orgId}`),
}

// Members
export const memberApi = {
  list: (orgId: string) => request<OrganizationMember[]>(`/organizations/${orgId}/members`),
  invite: (orgId: string, email: string, name: string, role: string, password?: string) =>
    request(`/organizations/${orgId}/members`, {
      method: 'POST', body: JSON.stringify({ email, name, role, password }),
    }),
  updateRole: (orgId: string, userId: string, role: string) =>
    request(`/organizations/${orgId}/members/${userId}/role`, {
      method: 'PUT', body: JSON.stringify({ role }),
    }),
  remove: (orgId: string, userId: string) =>
    request(`/organizations/${orgId}/members/${userId}`, { method: 'DELETE' }),
}

// Dashboard
export const dashboardApi = {
  overview: (orgId: string) => request<DashboardOverview>(`/organizations/${orgId}/dashboard`),
  health: (orgId: string) => request<JobHealth[]>(`/organizations/${orgId}/dashboard/health`),
}

// Agents
export const agentApi = {
  list: (orgId: string) => request<Agent[]>(`/organizations/${orgId}/agents`),
  get: (orgId: string, id: string) => request<Agent>(`/organizations/${orgId}/agents/${id}`),
  generateToken: (orgId: string) =>
    request<{ agent_id: string; registration_key: string }>(`/organizations/${orgId}/agents/token`, { method: 'POST' }),
  delete: (orgId: string, id: string) =>
    request(`/organizations/${orgId}/agents/${id}`, { method: 'DELETE' }),
}

// Machines
export const machineApi = {
  list: (orgId: string) => request<Machine[]>(`/organizations/${orgId}/machines`),
  get: (orgId: string, id: string) => request<Machine>(`/organizations/${orgId}/machines/${id}`),
}

// Storage
export const storageApi = {
  list: (orgId: string) => request<StorageTarget[]>(`/organizations/${orgId}/storage-targets`),
  create: (orgId: string, data: Record<string, unknown>) =>
    request<StorageTarget>(`/organizations/${orgId}/storage-targets`, {
      method: 'POST', body: JSON.stringify(data),
    }),
  delete: (orgId: string, id: string) =>
    request(`/organizations/${orgId}/storage-targets/${id}`, { method: 'DELETE' }),
  testConnection: (orgId: string, id: string) =>
    request<{ status: string; latency_ms?: number; bucket?: string; stage?: string; error?: string }>(
      `/organizations/${orgId}/storage-targets/${id}/test`, { method: 'POST' }),
}

// S3 region reference data (seeded server-side; admins can extend it).
export const regionApi = {
  list: (orgId: string) =>
    request<S3Region[]>(`/organizations/${orgId}/storage-regions`),
  create: (orgId: string, data: { code: string; name: string; provider?: string; endpoint?: string }) =>
    request<S3Region>(`/organizations/${orgId}/storage-regions`, {
      method: 'POST', body: JSON.stringify(data),
    }),
  remove: (orgId: string, code: string) =>
    request(`/organizations/${orgId}/storage-regions/${encodeURIComponent(code)}`, { method: 'DELETE' }),
}

// Backup Jobs
export const jobApi = {
  list: (orgId: string) => request<BackupJob[]>(`/organizations/${orgId}/backup-jobs`),
  get: (orgId: string, id: string) => request<BackupJob>(`/organizations/${orgId}/backup-jobs/${id}`),
  create: (orgId: string, data: Record<string, unknown>) =>
    request<BackupJob>(`/organizations/${orgId}/backup-jobs`, {
      method: 'POST', body: JSON.stringify(data),
    }),
  update: (orgId: string, id: string, data: Record<string, unknown>) =>
    request<BackupJob>(`/organizations/${orgId}/backup-jobs/${id}`, {
      method: 'PUT', body: JSON.stringify(data),
    }),
  delete: (orgId: string, id: string) =>
    request(`/organizations/${orgId}/backup-jobs/${id}`, { method: 'DELETE' }),
  runNow: (orgId: string, id: string) =>
    request<BackupRun>(`/organizations/${orgId}/backup-jobs/${id}/run`, { method: 'POST' }),
  enable: (orgId: string, id: string) =>
    request(`/organizations/${orgId}/backup-jobs/${id}/enable`, { method: 'POST' }),
  disable: (orgId: string, id: string) =>
    request(`/organizations/${orgId}/backup-jobs/${id}/disable`, { method: 'POST' }),
  preflight: (orgId: string, id: string) =>
    request<PreflightResult>(`/organizations/${orgId}/backup-jobs/${id}/preflight`),
}

// Backup Runs
export const runApi = {
  list: (orgId: string, params?: { job_id?: string; status?: string; search?: string; sort?: string; order?: 'asc' | 'desc'; page?: number; limit?: number }) => {
    const q = new URLSearchParams()
    if (params?.job_id) q.set('job_id', params.job_id)
    if (params?.status) q.set('status', params.status)
    if (params?.search) q.set('search', params.search)
    if (params?.sort) q.set('sort', params.sort)
    if (params?.order) q.set('order', params.order)
    if (params?.page) q.set('page', String(params.page))
    if (params?.limit) q.set('limit', String(params.limit))
    return request<PaginatedResponse<BackupRun>>(`/organizations/${orgId}/backup-runs?${q}`)
  },
  get: (orgId: string, id: string) => request<BackupRun>(`/organizations/${orgId}/backup-runs/${id}`),
  cancel: (orgId: string, id: string) =>
    request(`/organizations/${orgId}/backup-runs/${id}/cancel`, { method: 'POST' }),
  verify: (orgId: string, id: string) =>
    request<{ match: boolean; checksum?: string; expected?: string; actual?: string }>(`/organizations/${orgId}/backup-runs/${id}/verify`, { method: 'POST' }),
  artifacts: (orgId: string, id: string) =>
    request<BackupArtifact[]>(`/organizations/${orgId}/backup-runs/${id}/artifacts`),
}

// Restores
export const restoreApi = {
  list: (orgId: string) => request<RestoreJob[]>(`/organizations/${orgId}/restores`),
  get: (orgId: string, id: string) => request<RestoreJob>(`/organizations/${orgId}/restores/${id}`),
  create: (orgId: string, data: Record<string, unknown>) =>
    request<RestoreJob>(`/organizations/${orgId}/restores`, {
      method: 'POST', body: JSON.stringify(data),
    }),
}

// Alerts
export const alertApi = {
  list: (orgId: string, params?: { unread?: boolean; search?: string; order?: 'asc' | 'desc'; page?: number; limit?: number }) => {
    const q = new URLSearchParams()
    if (params?.unread) q.set('unread', 'true')
    if (params?.search) q.set('search', params.search)
    if (params?.order) q.set('order', params.order)
    if (params?.page) q.set('page', String(params.page))
    if (params?.limit) q.set('limit', String(params.limit))
    return request<PaginatedResponse<Alert>>(`/organizations/${orgId}/alerts?${q}`)
  },
  markRead: (orgId: string, id: string) =>
    request(`/organizations/${orgId}/alerts/${id}/read`, { method: 'PUT' }),
  markAllRead: (orgId: string) =>
    request<{ read: boolean; count: number }>(`/organizations/${orgId}/alerts/read-all`, { method: 'PUT' }),
}
