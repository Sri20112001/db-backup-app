import { create } from 'zustand'
import { persist } from 'zustand/middleware'

// Onboarding completion is always derived from live resources (storage
// targets, agents, jobs) — this store only remembers which orgs have
// dismissed the getting-started checklist so it stays hidden for them.
interface OnboardingState {
  dismissed: Record<string, boolean>
  dismiss: (orgId: string) => void
  undismiss: (orgId: string) => void
  isDismissed: (orgId: string | undefined) => boolean
}

export const useOnboardingStore = create<OnboardingState>()(
  persist(
    (set, get) => ({
      dismissed: {},
      dismiss: (orgId) => set((s) => ({ dismissed: { ...s.dismissed, [orgId]: true } })),
      undismiss: (orgId) =>
        set((s) => {
          const next = { ...s.dismissed }
          delete next[orgId]
          return { dismissed: next }
        }),
      isDismissed: (orgId) => (orgId ? !!get().dismissed[orgId] : false),
    }),
    { name: 'vaultguard-onboarding' }
  )
)
