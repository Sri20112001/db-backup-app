import { useState } from 'react'
import { NavLink } from 'react-router-dom'
import { useAuthStore } from '../store/authStore'
import { useUIStore } from '../store/uiStore'
import { orgApi } from '@/services/api'
import type { Organization } from '@/types'
import CreateOrgForm from './CreateOrgForm'
import {
  Shield, Archive, History, RotateCcw, Server,
  Database, Bell, Settings, LogOut, ChevronDown,
  Check, Plus, Building2,
} from 'lucide-react'

const navItems = [
  { path: '/', icon: Shield, label: 'Command Center', exact: true },
  { path: '/jobs', icon: Archive, label: 'Backup Jobs' },
  { path: '/history', icon: History, label: 'History & Runs' },
  { path: '/restores', icon: RotateCcw, label: 'Restores' },
  { path: '/agents', icon: Server, label: 'Agents & Nodes' },
  { path: '/storage', icon: Database, label: 'Storage Targets' },
  { path: '/alerts', icon: Bell, label: 'Alerts' },
]

const FloatingDock = () => {
  const { logout, user, currentOrg, setCurrentOrg } = useAuthStore()
  const { addToast } = useUIStore()
  const [menuOpen, setMenuOpen] = useState(false)
  const [orgs, setOrgs] = useState<Organization[]>([])
  const [orgsLoading, setOrgsLoading] = useState(false)
  const [showCreate, setShowCreate] = useState(false)

  const initials = user?.name
    ? user.name.split(' ').map((n) => n[0]).join('').toUpperCase().slice(0, 2)
    : 'U'

  const loadOrgs = async () => {
    setOrgsLoading(true)
    try {
      setOrgs(await orgApi.list())
    } catch {
      addToast('error', 'Failed to load organizations')
    } finally {
      setOrgsLoading(false)
    }
  }

  const toggleMenu = () => {
    setMenuOpen((open) => {
      if (!open) void loadOrgs()
      return !open
    })
  }

  const handleSwitch = (org: Organization) => {
    setCurrentOrg(org)
    setMenuOpen(false)
    setShowCreate(false)
  }

  const handleCreated = () => {
    setShowCreate(false)
    setMenuOpen(false)
    void loadOrgs()
  }

  return (
    <>
      {/* Left floating dock */}
      <aside className="fixed left-4 top-1/2 -translate-y-1/2 w-14 bg-[#ffffff] border border-[#c3c6d7]/40 rounded-xl shadow-[0_4px_16px_rgba(20,27,43,0.06)] z-50 flex flex-col items-center py-3 gap-2">
        {/* Logo */}
        <div className="flex items-center justify-center p-1 mb-1">
          <div className="w-8 h-8 rounded-lg bg-[#2563eb] flex items-center justify-center">
            <Shield size={16} className="text-white" />
          </div>
        </div>

        {/* Nav items */}
        <nav className="flex flex-col items-center gap-1.5 w-full px-2">
          {navItems.map((item) => (
            <NavLink
              key={item.path}
              to={item.path}
              end={item.exact}
              title={item.label}
              className={({ isActive }) =>
                `relative group flex items-center justify-center w-10 h-10 rounded-lg transition-colors ${
                  isActive
                    ? 'bg-[#2563eb] text-white'
                    : 'text-[#434655] hover:bg-[#e9edff] hover:text-[#141b2b]'
                }`
              }
            >
              <item.icon size={20} />
              <span className="absolute left-14 px-2 py-1 bg-[#293040] text-[#edf0ff] text-[12px] font-medium rounded shadow-md opacity-0 pointer-events-none group-hover:opacity-100 transition-opacity z-50 whitespace-nowrap">
                {item.label}
              </span>
            </NavLink>
          ))}
        </nav>

        {/* Settings + logout at bottom */}
        <div className="mt-auto pt-2 border-t border-[#c3c6d7]/40 w-full px-2 flex flex-col gap-1.5">
          <NavLink
            to="/settings"
            title="Settings"
            className={({ isActive }) =>
              `relative group flex items-center justify-center w-10 h-10 rounded-lg transition-colors ${
                isActive ? 'bg-[#2563eb] text-white' : 'text-[#434655] hover:bg-[#e9edff] hover:text-[#141b2b]'
              }`
            }
          >
            <Settings size={20} />
            <span className="absolute left-14 px-2 py-1 bg-[#293040] text-[#edf0ff] text-[12px] font-medium rounded shadow-md opacity-0 pointer-events-none group-hover:opacity-100 transition-opacity z-50 whitespace-nowrap">
              Settings
            </span>
          </NavLink>
        </div>
      </aside>

      {/* Top-right context chip */}
      <div className="fixed top-4 right-6 z-40">
        {menuOpen && (
          <div className="fixed inset-0 z-40" onClick={() => setMenuOpen(false)} />
        )}
        <div className="relative z-50">
          <button
            type="button"
            onClick={toggleMenu}
            className="flex items-center gap-3 px-3 py-1.5 bg-[#ffffff] border border-[#c3c6d7]/40 rounded-full shadow-[0_2px_8px_rgba(20,27,43,0.05)] hover:border-[#737686] transition-colors cursor-pointer"
          >
            <div className="flex items-center gap-2">
              <span className="text-[13px] font-medium text-[#141b2b]">
                {currentOrg?.name ?? 'Select Org'}
              </span>
            </div>
            <div className="h-4 w-px bg-[#c3c6d7]/50" />
            <div className="w-7 h-7 rounded-full bg-[#2563eb] flex items-center justify-center text-white text-[11px] font-bold">
              {initials}
            </div>
            <ChevronDown size={16} className="text-[#434655]" />
          </button>

          {/* Dropdown */}
          {menuOpen && (
            <div className="absolute top-full right-0 mt-2 w-64 bg-[#ffffff] border border-[#e9edff] rounded-lg shadow-lg z-50 overflow-hidden">
              {/* Organizations */}
              <div className="px-3 pt-2 pb-1">
                <p className="text-[11px] font-semibold uppercase tracking-wide text-[#737686]">
                  Organizations
                </p>
              </div>
              <div className="max-h-48 overflow-y-auto">
                {orgsLoading ? (
                  <p className="px-3 py-2 text-[13px] text-[#737686]">Loading...</p>
                ) : orgs.length === 0 ? (
                  <p className="px-3 py-2 text-[13px] text-[#737686]">
                    No organizations yet.
                  </p>
                ) : (
                  orgs.map((org) => (
                    <button
                      key={org.id}
                      type="button"
                      onClick={() => handleSwitch(org)}
                      className={`w-full flex items-center gap-2 px-3 py-2 text-[13px] transition-colors hover:bg-[#f1f3ff] ${
                        currentOrg?.id === org.id
                          ? 'text-[#141b2b] font-medium'
                          : 'text-[#434655]'
                      }`}
                    >
                      <Building2 size={14} className="shrink-0 text-[#737686]" />
                      <span className="truncate flex-1 text-left">{org.name}</span>
                      {currentOrg?.id === org.id && (
                        <Check size={14} className="shrink-0 text-[#2563eb]" />
                      )}
                    </button>
                  ))
                )}
              </div>

              {/* Create organization */}
              <div className="px-3 py-2 border-t border-[#e9edff]">
                {showCreate ? (
                  <CreateOrgForm onCreated={handleCreated} autoFocus />
                ) : (
                  <button
                    type="button"
                    onClick={() => setShowCreate(true)}
                    className="w-full flex items-center gap-2 px-2 py-1.5 text-[13px] font-medium text-[#004ac6] hover:bg-[#e9edff]/60 rounded-lg transition-colors"
                  >
                    <Plus size={14} />
                    New organization
                  </button>
                )}
              </div>

              {/* User + sign out */}
              <div className="px-3 py-2 border-t border-[#e9edff]">
                <p className="text-[13px] font-medium text-[#141b2b] truncate">{user?.name}</p>
                <p className="text-[12px] text-[#737686] truncate">{user?.email}</p>
              </div>
              <button
                type="button"
                onClick={logout}
                className="w-full flex items-center gap-2 px-3 py-2 text-[13px] text-[#ba1a1a] hover:bg-[#ffdad6]/30 transition-colors"
              >
                <LogOut size={14} />
                Sign out
              </button>
            </div>
          )}
        </div>
      </div>
    </>
  )
}

export default FloatingDock
