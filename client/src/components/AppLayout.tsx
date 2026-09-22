import { Outlet } from 'react-router-dom'
import { Building2, Server, Database, Archive } from 'lucide-react'
import FloatingDock from './FloatingDock'
import ToastContainer from './ToastContainer'
import CreateOrgForm from './CreateOrgForm'
import { useAuthStore } from '@/store/authStore'

const steps = [
  { icon: Building2, text: 'Create your organization' },
  { icon: Server, text: 'Connect an agent to a machine' },
  { icon: Database, text: 'Add a storage target' },
  { icon: Archive, text: 'Create a backup job and run it' },
]

const OnboardingNoOrg = () => (
  <div className="min-h-[70vh] flex items-center justify-center">
    <div className="w-full max-w-[480px] bg-[#ffffff] rounded-xl border border-[#e9edff] shadow-[0_4px_24px_rgba(20,27,43,0.08)] p-8">
      <div className="w-12 h-12 rounded-xl bg-[#2563eb]/10 flex items-center justify-center mb-4">
        <Building2 size={24} className="text-[#2563eb]" />
      </div>
      <h1 className="text-[20px] font-semibold text-[#141b2b] tracking-tight mb-1">
        Welcome to VaultGuard
      </h1>
      <p className="text-[13px] text-[#737686] mb-6">
        Everything in VaultGuard lives inside an organization. Create your first
        one to get started.
      </p>
      <CreateOrgForm autoFocus />
      <div className="mt-6 pt-5 border-t border-[#e9edff]">
        <p className="text-[11px] font-semibold uppercase tracking-wide text-[#737686] mb-3">
          What&apos;s next
        </p>
        <div className="flex flex-col gap-2.5">
          {steps.slice(1).map((step) => (
            <div key={step.text} className="flex items-center gap-2.5">
              <step.icon size={16} className="text-[#737686] shrink-0" />
              <span className="text-[13px] text-[#434655]">{step.text}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  </div>
)

const AppLayout = () => {
  const { currentOrg } = useAuthStore()

  return (
    <div className="min-h-screen bg-[#f9f9ff]">
      <FloatingDock />
      <main className="w-full min-h-screen pl-24 pr-8 py-6">
        {currentOrg ? <Outlet /> : <OnboardingNoOrg />}
      </main>
      <ToastContainer />
    </div>
  )
}

export default AppLayout
