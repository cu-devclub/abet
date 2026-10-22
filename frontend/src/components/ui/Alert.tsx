import type { ReactNode } from 'react'
import { AlertCircle, Info } from 'lucide-react'
import { cn } from '../../lib/cn'

interface AlertProps {
  variant?: 'error' | 'info' | 'warning'
  children: ReactNode
  className?: string
}

const STYLES = {
  error: 'bg-red-50 text-red-700 border-red-100',
  info: 'bg-indigo-50/70 text-indigo-900 border-indigo-100',
  warning: 'bg-amber-50 text-amber-900 border-amber-200',
}

export function Alert({ variant = 'error', children, className }: AlertProps) {
  const Icon = variant === 'error' ? AlertCircle : Info
  return (
    <div role={variant === 'error' ? 'alert' : undefined} className={cn('p-3 rounded-lg text-sm flex items-start border', STYLES[variant], className)}>
      <Icon size={16} className="mt-0.5 mr-2 shrink-0" />
      <div className="min-w-0">{children}</div>
    </div>
  )
}
