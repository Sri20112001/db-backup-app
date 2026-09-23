import { useState, useRef, useEffect } from 'react'
import { NavLink, useNavigate } from 'react-router-dom'
import { useAuthStore } from '../store/authStore'
import {
  Shield, Archive, History, RotateCcw, Server,
  Database, Bell, Settings, LogOut, Building2,
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
  const { logout, user, currentOrg } = useAuthStore()
  const navigate = useNavigate()
  const [menuOpen, setMenuOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  const handleLogout = () => {
    logout()
    navigate('/login', { replace: true })
  }

  // Close when clicking anywhere outside the menu
  useEffect(() => {
    const handleOutsideClick = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setMenuOpen(false)
      }
    }

    if (menuOpen) {
      document.addEventListener('mousedown', handleOutsideClick)
    }
    return () => {
      document.removeEventListener('mousedown', handleOutsideClick)
    }
  }, [menuOpen])

  const initials = user?.name
    ? user.name.split(' ').map((n) => n[0]).join('').toUpperCase().slice(0, 2)
    : 'U'

  return (
    <aside className="fixed left-4 top-1/2 -translate-y-1/2 w-14 bg-[#ffffff] border border-[#c3c6d7]/40 rounded-xl shadow-[0_4px_16px_rgba(20,27,43,0.06)] z-50 flex flex-col items-center py-3 gap-2">
      {/* Brand Icon */}
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

      {/* Footer: Settings & Profile Popover */}
      <div className="mt-auto pt-2 border-t border-[#c3c6d7]/40 w-full px-2 flex flex-col items-center gap-2">
        <NavLink
          to="/settings"
          title="Settings"
          className={({ isActive }) =>
            `relative group flex items-center justify-center w-10 h-10 rounded-lg transition-colors ${
              isActive
                ? 'bg-[#2563eb] text-white'
                : 'text-[#434655] hover:bg-[#e9edff] hover:text-[#141b2b]'
            }`
          }
        >
          <Settings size={20} />
          <span className="absolute left-14 px-2 py-1 bg-[#293040] text-[#edf0ff] text-[12px] font-medium rounded shadow-md opacity-0 pointer-events-none group-hover:opacity-100 transition-opacity z-50 whitespace-nowrap">
            Settings
          </span>
        </NavLink>

        {/* User / Org context trigger */}
        <div ref={menuRef} className="relative flex items-center justify-center w-full">
          <button
            type="button"
            onClick={() => setMenuOpen((prev) => !prev)}
            aria-label="User account and organization menu"
            className={`w-8 h-8 rounded-full bg-[#2563eb] text-white text-[11px] font-bold flex items-center justify-center transition-all focus:outline-none ${
              menuOpen ? 'ring-2 ring-[#2563eb] ring-offset-2' : 'hover:ring-2 hover:ring-[#2563eb]/40'
            }`}
          >
            {initials}
          </button>

          {/* Flyout panel */}
          {menuOpen && (
            <div className="absolute left-12 bottom-0 w-52 bg-[#ffffff] border border-[#c3c6d7]/40 rounded-xl shadow-[0_8px_24px_rgba(20,27,43,0.12)] z-50 overflow-hidden animate-in fade-in zoom-in-95 duration-100">
              {/* Invisible hover bridge to prevent edge flickering */}
              <div className="absolute -left-3 top-0 w-3 h-full" />

              {/* Org Header */}
              <div className="px-3 py-2 bg-[#f4f6fb] border-b border-[#c3c6d7]/30 flex items-center gap-2">
                <Building2 size={13} className="text-[#434655] shrink-0" />
                <span className="text-[12px] font-semibold text-[#141b2b] truncate">
                  {currentOrg?.name ?? 'Select Org'}
                </span>
              </div>

              {/* User Details */}
              <div className="px-3 py-2.5 border-b border-[#c3c6d7]/30">
                <p className="text-[13px] font-medium text-[#141b2b] truncate">
                  {user?.name ?? 'User'}
                </p>
                <p className="text-[11px] text-[#737686] truncate">
                  {user?.email ?? ''}
                </p>
              </div>

              {/* Logout Action */}
              <button
                type="button"
                onClick={() => {
                  setMenuOpen(false)
                  handleLogout()
                }}
                className="w-full flex items-center gap-2 px-3 py-2 text-[13px] text-[#ba1a1a] hover:bg-[#ffdad6]/30 transition-colors"
              >
                <LogOut size={14} />
                Sign out
              </button>
            </div>
          )}
        </div>
      </div>
    </aside>
  )
}

export default FloatingDock