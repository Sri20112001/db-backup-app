import { Outlet } from 'react-router-dom'
import ErrorBoundary from './ErrorBoundary'
import AppHeader from './AppHeader'
import ToastContainer from './ToastContainer'
import RealtimeProvider from './RealtimeProvider'

const AppLayout = () => (
  <div className="h-screen overflow-hidden bg-surface-container-low flex flex-col">
    <AppHeader />
    <RealtimeProvider />
    <main className="flex-1 min-h-0 w-full px-4 lg:px-8 py-6 flex flex-col">
      <div className="flex-1 min-h-0 flex flex-col">
        <ErrorBoundary>
          <Outlet />
        </ErrorBoundary>
      </div>
    </main>
    <ToastContainer />
  </div>
)

export default AppLayout
