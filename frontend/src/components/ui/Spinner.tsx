import { Loader2 } from 'lucide-react'
import { cn } from '../../lib/cn'

export function Spinner({ size = 18, className }: { size?: number; className?: string }) {
  return <Loader2 size={size} className={cn('animate-spin text-indigo-600', className)} aria-hidden="true" />
}

export function PageSpinner({ label = 'Loading…' }: { label?: string }) {
  return (
    <div className="flex items-center justify-center gap-2 py-16 text-sm text-gray-500" role="status">
      <Spinner />
      <span>{label}</span>
    </div>
  )
}
