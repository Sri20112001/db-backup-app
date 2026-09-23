import { Outlet } from 'react-router-dom'
import FloatingDock from './FloatingDock'
import ToastContainer from './ToastContainer'

const AppLayout = () => (
  <div className="min-h-screen bg-[#f9f9ff]">
    <FloatingDock />
    <main className="w-full min-h-screen pl-24 pr-8 py-6">
      <Outlet />
    </main>
    <ToastContainer />
  </div>
)

export default AppLayout
