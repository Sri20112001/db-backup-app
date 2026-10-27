import { useEffect, lazy, Suspense } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { useAuthStore } from './store/authStore'
import { useUIStore } from './store/uiStore'
import AppLayout from './components/AppLayout'

const LoginPage = lazy(() => import('./features/auth/components/LoginPage'))
const RegisterPage = lazy(() => import('./features/auth/components/RegisterPage'))
const SetupPage = lazy(() => import('./features/setup/SetupPage'))
const OnboardingPage = lazy(() => import('./features/onboarding/OnboardingPage'))
const DashboardPage = lazy(() => import('./features/dashboard/components/DashboardPage'))
const JobsPage = lazy(() => import('./features/jobs/components/JobsPage'))
const JobDetailPage = lazy(() => import('./features/jobs/components/JobDetailPage'))
const HistoryPage = lazy(() => import('./features/history/components/HistoryPage'))
const RestoresPage = lazy(() => import('./features/restores/components/RestoresPage'))
const AgentsPage = lazy(() => import('./features/agents/components/AgentsPage'))
const AgentDetailPage = lazy(() => import('./features/agents/components/AgentDetailPage'))
const StoragePage = lazy(() => import('./features/storage/components/StoragePage'))
const ConnectionsPage = lazy(() => import('./features/connections/components/ConnectionsPage'))
const ConnectionDetailPage = lazy(() => import('./features/connections/components/ConnectionDetailPage'))
const PoliciesPage = lazy(() => import('./features/policies/components/PoliciesPage'))
const AlertsPage = lazy(() => import('./features/alerts/components/AlertsPage'))
const SettingsPage = lazy(() => import('./features/settings/components/SettingsPage'))

const PageLoadingFallback = () => (
  <div className="flex-1 min-h-[300px] flex flex-col items-center justify-center gap-3 animate-in fade-in duration-150">
    <div className="w-8 h-8 rounded-full border-2 border-primary border-t-transparent animate-spin" />
    <span className="text-[12.5px] text-on-surface-variant font-medium">Loading view…</span>
  </div>
)

const RequireAuth = ({ children }: { children: React.ReactNode }) => {
  const { user } = useAuthStore()
  if (!user) return <Navigate to="/login" replace />
  return <>{children}</>
}

// Requires auth AND an active org. Redirects to /setup if org is missing.
const RequireOrg = ({ children }: { children: React.ReactNode }) => {
  const { user, currentOrg } = useAuthStore()
  if (!user) return <Navigate to="/login" replace />
  if (!currentOrg) return <Navigate to="/setup" replace />
  return <>{children}</>
}

const App = () => {
  useEffect(() => {
    useUIStore.getState().initTheme()
  }, [])

  return (
    <BrowserRouter basename="/vaultguard">
      <Suspense fallback={<PageLoadingFallback />}>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/register" element={<RegisterPage />} />
          <Route
            path="/setup"
            element={
              <RequireAuth>
                <SetupPage />
              </RequireAuth>
            }
          />
          <Route
            path="/onboarding"
            element={
              <RequireOrg>
                <OnboardingPage />
              </RequireOrg>
            }
          />
          <Route
            path="/"
            element={
              <RequireOrg>
                <AppLayout />
              </RequireOrg>
            }
          >
            <Route index element={<DashboardPage />} />
            <Route path="jobs" element={<JobsPage />} />
            <Route path="jobs/:id" element={<JobDetailPage />} />
            <Route path="policies" element={<PoliciesPage />} />
            <Route path="history" element={<HistoryPage />} />
            <Route path="restores" element={<RestoresPage />} />
            <Route path="agents" element={<AgentsPage />} />
            <Route path="agents/:id" element={<AgentDetailPage />} />
            <Route path="storage" element={<StoragePage />} />
            <Route path="connections" element={<ConnectionsPage />} />
            <Route path="connections/:id" element={<ConnectionDetailPage />} />
            <Route path="alerts" element={<AlertsPage />} />
            <Route path="settings" element={<SettingsPage />} />
          </Route>
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </Suspense>
    </BrowserRouter>
  )
}

export default App
