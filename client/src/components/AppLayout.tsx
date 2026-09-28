import { Outlet } from 'react-router-dom'
import FloatingDock from './FloatingDock'
import ToastContainer from './ToastContainer'
import RealtimeProvider from './RealtimeProvider'

const AppLayout = () => (
  <div className="h-screen overflow-hidden bg-[#f9f9ff]">
    <FloatingDock />
    <RealtimeProvider />
    <main className="h-full w-full pl-24 pr-8 py-6 flex flex-col min-h-0">
      <div className="flex-1 min-h-0 flex flex-col">
        <Outlet />
      </div>
    </main>
    <ToastContainer />
  </div>
)

export default AppLayout
