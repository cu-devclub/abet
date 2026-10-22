import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router'
import { CheckCircle2, Search, UserCog, X } from 'lucide-react'
import { cn } from '../../lib/cn'
import { displayName } from '../../lib/format'
import { useActAs, useActAsUsers } from '../../features/auth/api'
import type { Me, Role } from '../../features/auth/types'
import { Spinner } from '../ui/Spinner'
import { Tabs } from '../ui/Tabs'

const ROLE_TABS: readonly { id: Role; label: string }[] = [
  { id: 'admin', label: 'Admin' },
  { id: 'instructor', label: 'Instructor' },
  { id: 'staff', label: 'Staff' },
  { id: 'super_admin', label: 'Super admin' },
]

/** Dev-only "Switch role" button + panel; rendered only when `me.act_as.available` (D28). */
export function ActAsSwitcher({ me, compact = false }: { me: Me; compact?: boolean }) {
  const [open, setOpen] = useState(false)
  const [role, setRole] = useState<Role>('admin')
  const [search, setSearch] = useState('')
  const containerRef = useRef<HTMLDivElement>(null)
  const buttonRef = useRef<HTMLButtonElement>(null)
  const searchRef = useRef<HTMLInputElement>(null)
  const navigate = useNavigate()
  const { start } = useActAs(() => {
    setOpen(false)
    void navigate('/')
  })
  const available = Boolean(me.act_as?.available)
  const users = useActAsUsers(role, search.trim(), open && available)

  useEffect(() => {
    if (!open) return
    const onMouseDown = (event: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) setOpen(false)
    }
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== 'Escape') return
      setOpen(false)
      buttonRef.current?.focus()
    }
    document.addEventListener('mousedown', onMouseDown)
    document.addEventListener('keydown', onKeyDown)
    searchRef.current?.focus()
    return () => {
      document.removeEventListener('mousedown', onMouseDown)
      document.removeEventListener('keydown', onKeyDown)
    }
  }, [open])

  if (!available) return null

  return (
    <div className="relative" ref={containerRef}>
      <button
        ref={buttonRef}
        type="button"
        aria-haspopup="dialog"
        aria-expanded={open}
        aria-label="Switch role"
        title="Switch role (dev only)"
        onClick={() => setOpen(!open)}
        className={cn(
          'flex items-center rounded-lg text-sm font-medium border border-amber-300 bg-amber-50 text-amber-900 hover:bg-amber-100 transition-colors cursor-pointer focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500',
          compact ? 'p-2 justify-center' : 'w-full px-3 py-2 justify-center gap-2',
        )}
      >
        <UserCog size={compact ? 18 : 16} className="shrink-0" />
        {!compact && 'Switch role'}
        {!compact && <span className="bg-amber-400 text-amber-950 text-[10px] font-bold px-1 rounded">DEV</span>}
      </button>

      {open && (
        <div
          role="dialog"
          aria-label="Switch role panel"
          className="absolute bottom-full left-0 mb-2 w-80 max-w-[calc(100vw-1.5rem)] bg-white rounded-xl shadow-xl border border-gray-200 p-3 z-50 cursor-default"
        >
          <div className="flex items-center justify-between mb-2">
            <p className="text-sm font-bold text-gray-900">
              Act as user <span className="ml-1 bg-amber-400 text-amber-950 text-[10px] font-bold px-1 rounded">DEV</span>
            </p>
            <button
              type="button"
              onClick={() => setOpen(false)}
              aria-label="Close"
              className="p-1 text-gray-400 hover:text-gray-700 hover:bg-gray-100 rounded-lg cursor-pointer"
            >
              <X size={14} />
            </button>
          </div>
          <Tabs
            tabs={ROLE_TABS}
            active={role}
            onChange={(next) => {
              setRole(next)
              setSearch('')
            }}
            label="Role"
          />
          <div className="relative mb-2">
            <Search size={14} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-gray-400" />
            <input
              ref={searchRef}
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              aria-label="Search users"
              placeholder="Search name or email..."
              className="w-full pl-8 pr-3 py-1.5 text-xs bg-gray-50 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:bg-white transition-colors"
            />
          </div>
          <div role="listbox" aria-label="Users" className="max-h-64 overflow-y-auto text-xs">
            {users.isLoading && (
              <div className="flex justify-center py-4">
                <Spinner />
              </div>
            )}
            {users.isError && <p className="px-3 py-4 text-center text-red-600">{users.error.message}</p>}
            {users.data?.map((u) => {
              const current = u.id === me.user.id
              return (
                <button
                  key={u.id}
                  type="button"
                  role="option"
                  aria-selected={current}
                  disabled={start.isPending}
                  onClick={() => start.mutate(u.id)}
                  className={cn(
                    'w-full text-left px-3 py-2 flex items-center justify-between rounded-lg hover:bg-indigo-50/70 transition-colors cursor-pointer disabled:opacity-60',
                    current ? 'bg-indigo-50 text-indigo-700 font-semibold' : 'text-gray-700',
                  )}
                >
                  <span className="min-w-0 pr-2">
                    <span className="block truncate font-medium">{displayName(u)}</span>
                    {u.name.trim() && <span className="block truncate text-gray-500">{u.email}</span>}
                  </span>
                  {current && <CheckCircle2 size={14} className="text-indigo-600 shrink-0" />}
                </button>
              )
            })}
            {users.data?.length === 0 && <p className="px-3 py-4 text-center text-gray-400 italic">No matching users</p>}
          </div>
        </div>
      )}
    </div>
  )
}
