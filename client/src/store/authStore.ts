import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { User, Organization } from '../types'
import { API_BASE_URL as BASE } from '../CONSTANTS'

interface AuthState {
  user: User | null
  accessToken: string | null
  refreshToken: string | null
  currentOrg: Organization | null
  setAuth: (user: User, accessToken: string, refreshToken: string) => void
  setCurrentOrg: (org: Organization) => void
  logout: () => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      accessToken: null,
      refreshToken: null,
      currentOrg: null,
      setAuth: (user, accessToken, refreshToken) => {
        set({ user, accessToken, refreshToken })
      },
      setCurrentOrg: (org) => set({ currentOrg: org }),
      logout: () => {
        const rt = get().refreshToken
        if (rt) {
          // Fire-and-forget: revoke server-side refresh token
          fetch(`${BASE}/auth/logout`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ refresh_token: rt }),
          }).catch(() => {})
        }
        set({ user: null, accessToken: null, refreshToken: null, currentOrg: null })
      },
    }),
    {
      name: 'vaultguard-auth',
      partialize: (s) => ({
        user: s.user,
        currentOrg: s.currentOrg,
        accessToken: s.accessToken,
        refreshToken: s.refreshToken,
      }),
    }
  )
)
