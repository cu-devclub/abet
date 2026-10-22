import { useNavigate } from 'react-router'
import { Undo2, UserCog } from 'lucide-react'
import { displayName, roleLabel } from '../../lib/format'
import { useActAs } from '../../features/auth/api'
import type { Me } from '../../features/auth/types'

/** Sticky amber strip shown on every page while a super admin is acting as another user (dev only). */
export function ActAsBanner({ me }: { me: Me }) {
  const navigate = useNavigate()
  const { stop } = useActAs(() => void navigate('/'))
  const real = me.act_as?.real_user
  if (!real) return null
  return (
    <div role="status" className="sticky top-0 z-40 shrink-0 bg-amber-400 text-amber-950 px-4 py-2 flex flex-wrap items-center gap-x-4 gap-y-1.5 text-sm">
      <UserCog size={16} className="shrink-0" />
      <p className="flex-1 min-w-0 wrap-break-word">
        <span className="font-bold">
          Acting as {displayName(me.user)} ({roleLabel(me.user.role)})
        </span>{' '}
        — dev testing as {real.email}
      </p>
      <button
        type="button"
        onClick={() => stop.mutate()}
        disabled={stop.isPending}
        className="inline-flex items-center gap-1.5 px-3 py-1 rounded-lg bg-amber-950 text-amber-50 text-xs font-semibold hover:bg-amber-900 disabled:opacity-60 transition-colors cursor-pointer"
      >
        <Undo2 size={14} /> Back to Super Admin
      </button>
    </div>
  )
}
