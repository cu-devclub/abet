import type { ReactNode } from 'react'
import { cn } from '../../lib/cn'
import { roleLabel } from '../../lib/format'
import type { Role } from '../../features/auth/types'

const VARIANTS: Record<string, string> = {
  super_admin: 'bg-purple-100 text-purple-800 border-purple-200',
  admin: 'bg-blue-100 text-blue-800 border-blue-200',
  instructor: 'bg-teal-100 text-teal-800 border-teal-200',
  staff: 'bg-gray-100 text-gray-800 border-gray-200',
  processing: 'bg-yellow-100 text-yellow-800 border-yellow-200',
  generated: 'bg-green-100 text-green-800 border-green-200',
  failed: 'bg-red-50 text-red-700 border-red-200 hover:bg-red-100 hover:border-red-300 font-semibold',
}

interface BadgeProps {
  variant: string
  children: ReactNode
  onClick?: () => void
  title?: string
}

export function Badge({ variant, children, onClick, title }: BadgeProps) {
  const base = cn(
    'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium border',
    VARIANTS[variant] ?? 'bg-gray-100 border-gray-200',
  )
  if (onClick) {
    return (
      <button type="button" onClick={onClick} title={title} className={cn(base, 'transition-all cursor-pointer')}>
        {children}
      </button>
    )
  }
  return (
    <span title={title} className={base}>
      {children}
    </span>
  )
}

export function RoleBadge({ role }: { role: Role }) {
  return <Badge variant={role}>{roleLabel(role).toUpperCase()}</Badge>
}
