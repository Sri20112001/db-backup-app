import type {
  AuthTokens, BackupJob, BackupRun, BackupArtifact,
  Agent, Machine, StorageTarget, RestoreJob, Alert,
  DashboardOverview, JobHealth, OrganizationMember,
  Organization, PaginatedResponse
} from '../types'

const BASE = import.meta.env.VITE_API_URL || 'http://localhost:7541/vaultguard/api'

function getToken() {
  return localStorage.getItem('access_token')
}

function getRefreshToken() {
  return localStorage.getItem('refresh_token')
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
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
    window.location.href = '/login'
    throw new Error('session expired')
  }
  const data = await res.json()
  localStorage.setItem('access_token', data.access_token)
  localStorage.setItem('refresh_token', data.refresh_token)
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
  if (res.status === 401 && retry) {
    if (!refreshPromise) {
      refreshPromise = refreshAccessToken().finally(() => { refreshPromise = null })
    }
    await refreshPromise
    return request<T>(path, options, false)
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
  login: (email: string, password: string) =>
    request<AuthTokens>('/auth/login', { method: 'POST', body: JSON.stringify({ email, password }) }),
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
  invite: (orgId: string, email: string, name: string, role: string) =>
    request(`/organizations/${orgId}/members`, {
      method: 'POST', body: JSON.stringify({ email, name, role }),
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
}

// Backup Runs
export const runApi = {
  list: (orgId: string, params?: { job_id?: string; status?: string; page?: number; limit?: number }) => {
    const q = new URLSearchParams()
    if (params?.job_id) q.set('job_id', params.job_id)
    if (params?.status) q.set('status', params.status)
    if (params?.page) q.set('page', String(params.page))
    if (params?.limit) q.set('limit', String(params.limit))
    return request<PaginatedResponse<BackupRun>>(`/organizations/${orgId}/backup-runs?${q}`)
  },
  get: (orgId: string, id: string) => request<BackupRun>(`/organizations/${orgId}/backup-runs/${id}`),
  cancel: (orgId: string, id: string) =>
    request(`/organizations/${orgId}/backup-runs/${id}/cancel`, { method: 'POST' }),
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
  list: (orgId: string, params?: { unread?: boolean; page?: number; limit?: number }) => {
    const q = new URLSearchParams()
    if (params?.unread) q.set('unread', 'true')
    if (params?.page) q.set('page', String(params.page))
    if (params?.limit) q.set('limit', String(params.limit))
    return request<PaginatedResponse<Alert>>(`/organizations/${orgId}/alerts?${q}`)
  },
  markRead: (orgId: string, id: string) =>
    request(`/organizations/${orgId}/alerts/${id}/read`, { method: 'PUT' }),
}
