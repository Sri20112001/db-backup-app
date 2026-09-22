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
    (set) => ({
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
        localStorage.removeItem('access_token')
        localStorage.removeItem('refresh_token')
        set({ user: null, accessToken: null, refreshToken: null, currentOrg: null })
      },
    }),
    { name: 'vaultguard-auth', partialize: (s) => ({ user: s.user, currentOrg: s.currentOrg }) }
  )
)
