import { useState, type ElementType } from 'react'
import { NavLink, useNavigate } from 'react-router'
import { BookMarked, BookOpen, Building2, ChevronLeft, ChevronRight, GraduationCap, Layers, LogOut, Users } from 'lucide-react'
import { cn } from '../../lib/cn'
import { displayName, roleLabel } from '../../lib/format'
import { useLogout } from '../../features/auth/api'
import type { Capability, Me, Role } from '../../features/auth/types'
import { ActAsSwitcher } from './ActAsSwitcher'

const STORAGE_KEY = 'abet.sidebar.collapsed'

interface NavItem {
  to: string
  label: string
  icon: ElementType
  /** Omitted = visible to every authenticated user. */
  capability?: Capability
}

const NAV_ITEMS: NavItem[] = [
  { to: '/portfolios', label: 'Course Portfolios', icon: BookOpen },
  { to: '/courses', label: 'Courses', icon: BookMarked, capability: 'manage_courses' },
  { to: '/covers', label: 'Cover Page', icon: Layers, capability: 'manage_covers' },
  { to: '/users', label: 'Users', icon: Users, capability: 'manage_users' },
  { to: '/departments', label: 'Departments', icon: Building2, capability: 'manage_org' },
  { to: '/curricula', label: 'Curriculums', icon: GraduationCap, capability: 'manage_org' },
]

const AVATAR_STYLES: Record<Role, string> = {
  super_admin: 'bg-purple-100 text-purple-700 border border-purple-200',
  admin: 'bg-blue-100 text-blue-700 border border-blue-200',
  instructor: 'bg-teal-100 text-teal-700 border border-teal-200',
  staff: 'bg-slate-100 text-slate-700 border border-slate-200',
}

const ROLE_CHIP_STYLES: Record<Role, string> = {
  super_admin: 'bg-purple-100 text-purple-800',
  admin: 'bg-blue-100 text-blue-800',
  instructor: 'bg-teal-100 text-teal-800',
  staff: 'bg-gray-100 text-gray-800',
}

function readCollapsed(): boolean {
  try {
    return localStorage.getItem(STORAGE_KEY) === '1'
  } catch {
    return false
  }
}

function writeCollapsed(collapsed: boolean): void {
  try {
    localStorage.setItem(STORAGE_KEY, collapsed ? '1' : '0')
  } catch {
    // Storage may be unavailable (private mode); the preference simply isn't persisted.
  }
}

function Avatar({ me, size }: { me: Me; size: 'sm' | 'md' }) {
  const { user } = me
  const dimension = size === 'md' ? 'h-10 w-10' : 'h-9 w-9 mr-3'
  if (user.picture_url) {
    return <img src={user.picture_url} alt="" referrerPolicy="no-referrer" className={cn(dimension, 'rounded-full shrink-0 object-cover')} />
  }
  return (
    <div className={cn(dimension, 'rounded-full flex items-center justify-center font-bold text-sm shrink-0', AVATAR_STYLES[user.role])}>
      {displayName(user).charAt(0).toUpperCase()}
    </div>
  )
}

export function Sidebar({ me }: { me: Me }) {
  const [collapsed, setCollapsed] = useState(readCollapsed)
  const navigate = useNavigate()
  const logout = useLogout()

  const setAndStore = (next: boolean) => {
    setCollapsed(next)
    writeCollapsed(next)
  }

  const signOut = () => logout.mutate(undefined, { onSettled: () => void navigate('/login') })
  const { user, capabilities } = me
  const items = NAV_ITEMS.filter((item) => !item.capability || capabilities[item.capability])

  return (
    <aside
      className={cn(
        collapsed ? 'w-20' : 'w-64',
        'bg-white border-r border-gray-200 flex flex-col transition-all duration-300 ease-in-out shrink-0 select-none',
      )}
    >
      {collapsed ? (
        <div className="h-16 flex items-center justify-center border-b border-gray-200 px-2">
          <button
            type="button"
            onClick={() => setAndStore(false)}
            className="p-2 text-indigo-600 hover:bg-indigo-50 rounded-lg transition-colors cursor-pointer flex items-center justify-center group relative"
            title="Expand sidebar"
            aria-label="Expand sidebar"
          >
            <BookOpen className="h-6 w-6 text-indigo-600 transition-transform group-hover:scale-95" />
            <span className="absolute -bottom-1 -right-1 bg-white border border-gray-200 rounded-full p-0.5 shadow-xs text-gray-500 group-hover:text-indigo-600 group-hover:border-indigo-300">
              <ChevronRight size={12} />
            </span>
          </button>
        </div>
      ) : (
        <div className="h-16 flex items-center justify-between px-5 border-b border-gray-200">
          <div className="flex items-center min-w-0">
            <BookOpen className="h-6 w-6 text-indigo-600 mr-2.5 shrink-0" />
            <span className="text-lg font-bold text-gray-900 tracking-tight truncate">ABET</span>
          </div>
          <button
            type="button"
            onClick={() => setAndStore(true)}
            className="p-1.5 text-gray-400 hover:text-gray-700 hover:bg-gray-100 rounded-lg transition-colors cursor-pointer"
            title="Collapse sidebar"
            aria-label="Collapse sidebar"
          >
            <ChevronLeft size={18} />
          </button>
        </div>
      )}

      <nav aria-label="Main" className={cn('flex-1 space-y-2 overflow-y-auto', collapsed ? 'px-2 py-4' : 'px-4 py-6')}>
        {items.map(({ to, label, icon: Icon }) => (
          <NavLink
            key={to}
            to={to}
            title={collapsed ? label : undefined}
            aria-label={collapsed ? label : undefined}
            className={({ isActive }) =>
              cn(
                'w-full flex items-center rounded-lg text-sm font-medium transition-colors',
                collapsed ? 'justify-center p-2.5' : 'px-3 py-2.5',
                isActive ? 'bg-indigo-50 text-indigo-700 font-semibold' : 'text-gray-600 hover:bg-gray-100',
              )
            }
          >
            <Icon size={collapsed ? 20 : 18} className={collapsed ? 'shrink-0' : 'mr-3 shrink-0'} />
            {!collapsed && <span className="truncate">{label}</span>}
          </NavLink>
        ))}
      </nav>

      <div className={cn('border-t border-gray-200', collapsed ? 'p-3 flex flex-col items-center gap-3' : 'p-4 space-y-3')}>
        {collapsed ? (
          <>
            <div title={`${displayName(user)} (${roleLabel(user.role)})`}>
              <Avatar me={me} size="md" />
            </div>
            <ActAsSwitcher me={me} compact />
            <button
              type="button"
              onClick={signOut}
              className="p-2 text-red-600 border border-red-200 bg-red-50 hover:bg-red-100 hover:border-red-300 rounded-lg transition-colors cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-red-500"
              title="Sign Out"
              aria-label="Sign Out"
            >
              <LogOut size={18} />
            </button>
          </>
        ) : (
          <>
            <div className="flex items-center px-1">
              <Avatar me={me} size="sm" />
              <div className="overflow-hidden flex-1 min-w-0">
                <p className="text-sm font-bold text-gray-900 truncate" title={user.email}>
                  {displayName(user)}
                </p>
                <div className="mt-0.5">
                  <span className={cn('inline-block px-1.5 rounded text-[10px] font-semibold uppercase tracking-wider', ROLE_CHIP_STYLES[user.role])}>
                    {roleLabel(user.role)}
                  </span>
                </div>
              </div>
            </div>
            <ActAsSwitcher me={me} />
            <button
              type="button"
              onClick={signOut}
              className="w-full flex items-center justify-center gap-2 px-3 py-2 border border-red-200 rounded-lg text-sm font-medium text-red-600 bg-red-50 hover:bg-red-100 hover:border-red-300 hover:text-red-700 transition-colors cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-red-500"
            >
              <LogOut size={16} className="shrink-0" />
              <span className="truncate">Sign Out</span>
            </button>
          </>
        )}
      </div>
    </aside>
  )
}
