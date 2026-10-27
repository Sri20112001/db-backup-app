import { useState, useRef, useEffect } from 'react'
import { NavLink, useNavigate, useLocation } from 'react-router-dom'
import { useAuthStore } from '../store/authStore'
import { useUIStore } from '../store/uiStore'
import {
  Archive, History, RotateCcw, Server,
  Database, Settings, LogOut, Building2, Sun, Moon, PlugZap, SlidersHorizontal,
  Menu, X
} from 'lucide-react'
import { Logo } from './ui/Logo'
import NotificationBell from './NotificationBell'
import ConnectionStatusBadge from './ConnectionStatusBadge'

const navItems = [
  { path: '/jobs', icon: Archive, label: 'Backup Jobs' },
  { path: '/policies', icon: SlidersHorizontal, label: 'Backup Policies' },
  { path: '/history', icon: History, label: 'History & Runs' },
  { path: '/restores', icon: RotateCcw, label: 'Restores' },
  { path: '/agents', icon: Server, label: 'Agents & Nodes' },
  { path: '/connections', icon: PlugZap, label: 'Connections' },
  { path: '/storage', icon: Database, label: 'Storage Targets' },
]

const iconBtn =
  'flex items-center justify-center w-9 h-9 rounded-lg text-on-surface-variant hover:bg-surface-container-high hover:text-on-surface transition-colors focus:outline-none focus:ring-2 focus:ring-primary/20'

const AppHeader = () => {
  const { logout, user, currentOrg } = useAuthStore()
  const navigate = useNavigate()
  const location = useLocation()
  const [menuOpen, setMenuOpen] = useState(false)
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  const handleLogout = () => {
    logout()
    navigate('/login', { replace: true })
  }

  // Close menus when route changes
  useEffect(() => {
    setMobileMenuOpen(false)
  }, [location.pathname])

  // Close the avatar flyout when clicking anywhere outside it.
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
    <>
      <header className="shrink-0 h-16 bg-surface-container-lowest border-b border-outline-variant/40 z-40">
        <div className="h-full px-4 lg:px-6 flex items-center justify-between gap-3">
          {/* Brand lockup */}
          <div className="flex items-center gap-2">
            {/* Mobile Hamburger toggle */}
            <button
              type="button"
              onClick={() => setMobileMenuOpen((prev) => !prev)}
              aria-label={mobileMenuOpen ? 'Close navigation menu' : 'Open navigation menu'}
              aria-expanded={mobileMenuOpen}
              className={`xl:hidden ${iconBtn}`}
            >
              {mobileMenuOpen ? <X size={20} /> : <Menu size={20} />}
            </button>

            <NavLink to="/" className="flex items-center gap-2.5 shrink-0 group" title="VaultGuard — Command Center">
              <div className="w-9 h-9 rounded-lg bg-surface-container-highest flex items-center justify-center border border-outline-variant/30 shadow-inner">
                <Logo className="w-[22px] h-[22px]" />
              </div>
              <div className="flex flex-col leading-tight">
                <span className="text-[15px] font-semibold text-on-surface tracking-tight">
                  VaultGuard
                </span>
                <span className="hidden 2xl:block text-[10.5px] text-on-surface-variant">
                  Backup & Recovery Platform
                </span>
              </div>
            </NavLink>
          </div>

          {/* Desktop Primary nav: labels from xl up */}
          <nav className="hidden xl:flex flex-1 min-w-0 items-center gap-1 overflow-x-auto px-2">
            {navItems.map((item) => (
              <NavLink
                key={item.path}
                to={item.path}
                title={item.label}
                className={({ isActive }) =>
                  `shrink-0 flex items-center gap-1.5 h-9 px-2.5 rounded-lg text-[12.5px] font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-primary/20 ${
                    isActive
                      ? 'bg-primary text-on-primary shadow-sm'
                      : 'text-on-surface-variant hover:bg-surface-container-high hover:text-on-surface'
                  }`
                }
              >
                <item.icon size={16} />
                <span>{item.label}</span>
              </NavLink>
            ))}
          </nav>

          {/* Utilities */}
          <div className="shrink-0 flex items-center gap-1.5">
            {/* Live WebSocket sync status badge */}
            <ConnectionStatusBadge />

            <NotificationBell />

            <button
              type="button"
              onClick={() => useUIStore.getState().toggleTheme()}
              aria-label="Toggle theme"
              className={iconBtn}
            >
              {useUIStore((s) => s.theme) === 'dark' ? <Sun size={17} /> : <Moon size={17} />}
            </button>

            <NavLink
              to="/settings"
              aria-label="Settings"
              className={({ isActive }) =>
                `${iconBtn} ${isActive ? 'bg-primary text-on-primary' : ''}`
              }
            >
              <Settings size={17} />
            </NavLink>

            {/* User / Org flyout */}
            <div ref={menuRef} className="relative">
              <button
                type="button"
                onClick={() => setMenuOpen((prev) => !prev)}
                aria-label="User account and organization menu"
                className={`w-8 h-8 rounded-full bg-primary text-on-primary text-[11px] font-bold flex items-center justify-center transition-all focus:outline-none ${
                  menuOpen ? 'ring-2 ring-primary ring-offset-2 ring-offset-surface-container-lowest' : 'hover:ring-2 hover:ring-primary/40'
                }`}
              >
                {initials}
              </button>

              {menuOpen && (
                <div className="absolute right-0 top-[calc(100%+8px)] w-56 bg-surface-container-lowest border border-outline-variant/40 rounded-xl shadow-[0_8px_24px_rgba(20,27,43,0.12)] z-50 overflow-hidden animate-in fade-in zoom-in-95 duration-100">
                  {/* Org Header */}
                  <div className="px-3 py-2 bg-surface-container-low border-b border-outline-variant/30 flex items-center gap-2">
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
                    <p className="text-[11px] text-on-surface-variant truncate">
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
        </div>
      </header>

      {/* Mobile drawer overlay */}
      {mobileMenuOpen && (
        <div
          role="dialog"
          aria-modal="true"
          aria-label="Mobile Navigation"
          className="fixed inset-0 top-16 z-30 bg-black/40 backdrop-blur-xs xl:hidden animate-in fade-in duration-100"
          onClick={() => setMobileMenuOpen(false)}
        >
          <div
            className="w-72 max-w-[85vw] h-full bg-surface-container-lowest border-r border-outline-variant/40 p-4 shadow-xl flex flex-col justify-between overflow-y-auto animate-in slide-in-from-left duration-150"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex flex-col gap-1">
              <div className="px-2.5 py-2 mb-2 rounded-lg bg-surface-container-low flex items-center gap-2">
                <Building2 size={14} className="text-on-surface-variant shrink-0" />
                <span className="text-[12.5px] font-medium text-on-surface truncate">
                  {currentOrg?.name ?? 'Organization'}
                </span>
              </div>

              <div className="text-[11px] font-semibold uppercase tracking-wider text-on-surface-variant px-2.5 py-1">
                Navigation
              </div>

              {navItems.map((item) => (
                <NavLink
                  key={item.path}
                  to={item.path}
                  className={({ isActive }) =>
                    `flex items-center gap-2.5 px-3 py-2.5 rounded-lg text-[13px] font-medium transition-colors ${
                      isActive
                        ? 'bg-primary text-on-primary shadow-xs'
                        : 'text-on-surface-variant hover:bg-surface-container-high hover:text-on-surface'
                    }`
                  }
                >
                  <item.icon size={17} />
                  <span>{item.label}</span>
                </NavLink>
              ))}
            </div>

            <div className="pt-4 border-t border-outline-variant/30 flex flex-col gap-2">
              <div className="px-3 py-1">
                <p className="text-[13px] font-medium text-on-surface truncate">{user?.name ?? 'User'}</p>
                <p className="text-[11px] text-on-surface-variant truncate">{user?.email ?? ''}</p>
              </div>

              <button
                type="button"
                onClick={handleLogout}
                className="w-full flex items-center gap-2 px-3 py-2 text-[13px] font-medium text-error hover:bg-error-container/30 rounded-lg transition-colors"
              >
                <LogOut size={16} />
                Sign out
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  )
}

export default AppHeader
