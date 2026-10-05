import { useState, useRef, useEffect } from 'react'
import { NavLink, useNavigate, useLocation } from 'react-router-dom'
import { useAuthStore } from '../store/authStore'
import { useUIStore } from '../store/uiStore'
import { alertApi } from '../services/api'
import { useRealtimeStore } from '../stores/realtimeStore'
import {
  Shield, Archive, History, RotateCcw, Server,
  Database, Bell, Settings, LogOut, Building2, Sun, Moon, PlugZap
} from 'lucide-react'
import { Logo } from './ui/Logo'

const navItems = [
  { path: '/', icon: Shield, label: 'Command Center', exact: true },
  { path: '/jobs', icon: Archive, label: 'Backup Jobs' },
  { path: '/history', icon: History, label: 'History & Runs' },
  { path: '/restores', icon: RotateCcw, label: 'Restores' },
  { path: '/agents', icon: Server, label: 'Agents & Nodes' },
  { path: '/connections', icon: PlugZap, label: 'Connections' },
  { path: '/storage', icon: Database, label: 'Storage Targets' },
  { path: '/alerts', icon: Bell, label: 'Alerts' },
]

const FloatingDock = () => {
  const { logout, user, currentOrg } = useAuthStore()
  const navigate = useNavigate()
  const location = useLocation()
  const [menuOpen, setMenuOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)
  const [unreadCount, setUnreadCount] = useState(0)
  const alertTick = useRealtimeStore((s) => s.alertTick)

  // Unread badge: total only (limit 1 keeps it cheap). Refreshes on org or
  // route change, and instantly on socket-pushed alerts.
  useEffect(() => {
    if (!currentOrg) {
      setUnreadCount(0)
      return
    }
    let cancelled = false
    alertApi
      .list(currentOrg.id, { unread: true, limit: 1 })
      .then((res) => {
        if (!cancelled) setUnreadCount(res.total)
      })
      .catch(() => {})
    return () => {
      cancelled = true
    }
  }, [currentOrg, location.pathname, alertTick])

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
    <aside className="fixed left-4 top-1/2 -translate-y-1/2 w-14 bg-surface-container-lowest border border-outline-variant/40 rounded-xl shadow-[0_4px_16px_rgba(20,27,43,0.06)] z-50 flex flex-col items-center py-3 gap-2">
      {/* Brand Icon */}
      <div className="flex items-center justify-center p-1 mb-1">
        <div className="w-8 h-8 rounded-lg bg-surface-container-highest flex items-center justify-center border border-outline-variant/30 shadow-inner">
          <Logo className="w-6 h-6" />
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
                  ? 'bg-primary text-on-primary'
                  : 'text-on-surface-variant hover:bg-surface-container-high hover:text-on-surface'
              }`
            }
          >
            <item.icon size={20} />
            {item.path === '/alerts' && unreadCount > 0 && (
              <span className="absolute -top-1 -right-1 min-w-[18px] h-[18px] px-1 rounded-full bg-error text-on-primary text-[10px] font-bold flex items-center justify-center shadow-sm">
                {unreadCount > 99 ? '99+' : unreadCount}
              </span>
            )}
            <span className="absolute left-14 px-2 py-1 bg-[#293040] text-[#edf0ff] text-[12px] font-medium rounded shadow-md opacity-0 pointer-events-none group-hover:opacity-100 transition-opacity z-50 whitespace-nowrap">
              {item.label}
            </span>
          </NavLink>
        ))}
      </nav>

      {/* Footer: Settings & Profile Popover */}
      <div className="mt-auto pt-2 border-t border-outline-variant/40 w-full px-2 flex flex-col items-center gap-2">
        <button
          type="button"
          onClick={() => useUIStore.getState().toggleTheme()}
          className="relative group flex items-center justify-center w-10 h-10 rounded-lg text-on-surface-variant hover:bg-surface-container-high hover:text-on-surface transition-colors focus:outline-none"
        >
          {useUIStore((s) => s.theme) === 'dark' ? <Sun size={20} /> : <Moon size={20} />}
          <span className="absolute left-14 px-2 py-1 bg-[#293040] text-[#edf0ff] text-[12px] font-medium rounded shadow-md opacity-0 pointer-events-none group-hover:opacity-100 transition-opacity z-50 whitespace-nowrap">
            Toggle Theme
          </span>
        </button>

        <NavLink
          to="/settings"
          title="Settings"
          className={({ isActive }) =>
            `relative group flex items-center justify-center w-10 h-10 rounded-lg transition-colors ${
              isActive
                ? 'bg-primary text-on-primary'
                : 'text-on-surface-variant hover:bg-surface-container-high hover:text-on-surface'
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
            className={`w-8 h-8 rounded-full bg-primary text-on-primary text-[11px] font-bold flex items-center justify-center transition-all focus:outline-none ${
              menuOpen ? 'ring-2 ring-primary ring-offset-2' : 'hover:ring-2 hover:ring-primary/40'
            }`}
          >
            {initials}
          </button>

          {/* Flyout panel */}
          {menuOpen && (
            <div className="absolute left-12 bottom-0 w-52 bg-surface-container-lowest border border-outline-variant/40 rounded-xl shadow-[0_8px_24px_rgba(20,27,43,0.12)] z-50 overflow-hidden animate-in fade-in zoom-in-95 duration-100">
              {/* Invisible hover bridge to prevent edge flickering */}
              <div className="absolute -left-3 top-0 w-3 h-full" />

              {/* Org Header */}
              <div className="px-3 py-2 bg-[#f4f6fb] border-b border-outline-variant/30 flex items-center gap-2">
                <Building2 size={13} className="text-on-surface-variant shrink-0" />
                <span className="text-[12px] font-semibold text-on-surface truncate">
                  {currentOrg?.name ?? 'Select Org'}
                </span>
              </div>

              {/* User Details */}
              <div className="px-3 py-2.5 border-b border-outline-variant/30">
                <p className="text-[13px] font-medium text-on-surface truncate">
                  {user?.name ?? 'User'}
                </p>
                <p className="text-[11px] text-outline truncate">
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
                className="w-full flex items-center gap-2 px-3 py-2 text-[13px] text-error hover:bg-error-container/30 transition-colors"
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