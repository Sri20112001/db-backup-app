import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { User, Organization } from '../types'

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
        localStorage.setItem('access_token', accessToken)
        localStorage.setItem('refresh_token', refreshToken)
        set({ user, accessToken, refreshToken })
      },
      setCurrentOrg: (org) => set({ currentOrg: org }),
      logout: () => {
        const rt = get().refreshToken || localStorage.getItem('refresh_token')
        if (rt) {
          // Fire-and-forget: revoke server-side refresh token
          fetch(`${import.meta.env.VITE_API_URL}/auth/logout`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ refresh_token: rt }),
          }).catch(() => {})
        }
        localStorage.removeItem('access_token')
        localStorage.removeItem('refresh_token')
        set({ user: null, accessToken: null, refreshToken: null, currentOrg: null })
      },
    }),
    {
      name: 'vaultguard-auth',
      partialize: (s) => ({ user: s.user, currentOrg: s.currentOrg, refreshToken: s.refreshToken }),
      onRehydrateStorage: () => (state) => {
        // Sync persisted refresh token back to localStorage on page load
        if (state?.refreshToken) {
          localStorage.setItem('refresh_token', state.refreshToken)
        }
      },
    }
  )
)
