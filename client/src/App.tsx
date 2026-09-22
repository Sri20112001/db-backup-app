import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { useAuthStore } from './store/authStore'
import AppLayout from './components/AppLayout'
import LoginPage from './features/auth/components/LoginPage'
import RegisterPage from './features/auth/components/RegisterPage'
import DashboardPage from './features/dashboard/components/DashboardPage'
import JobsPage from './features/jobs/components/JobsPage'
import NewJobPage from './features/jobs/components/NewJobPage'
import JobDetailPage from './features/jobs/components/JobDetailPage'
import HistoryPage from './features/history/components/HistoryPage'
import RestoresPage from './features/restores/components/RestoresPage'
import AgentsPage from './features/agents/components/AgentsPage'
import StoragePage from './features/storage/components/StoragePage'
import AlertsPage from './features/alerts/components/AlertsPage'
import SettingsPage from './features/settings/components/SettingsPage'

const RequireAuth = ({ children }: { children: React.ReactNode }) => {
  const { user } = useAuthStore()
  if (!user) return <Navigate to="/login" replace />
  return <>{children}</>
}

const App = () => (
  <BrowserRouter>
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/register" element={<RegisterPage />} />
      <Route
        path="/"
        element={
          <RequireAuth>
            <AppLayout />
          </RequireAuth>
        }
      >
        <Route index element={<DashboardPage />} />
        <Route path="jobs" element={<JobsPage />} />
        <Route path="jobs/new" element={<NewJobPage />} />
        <Route path="jobs/:id" element={<JobDetailPage />} />
        <Route path="history" element={<HistoryPage />} />
        <Route path="restores" element={<RestoresPage />} />
        <Route path="agents" element={<AgentsPage />} />
        <Route path="storage" element={<StoragePage />} />
        <Route path="alerts" element={<AlertsPage />} />
        <Route path="settings" element={<SettingsPage />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  </BrowserRouter>
)

export default App
