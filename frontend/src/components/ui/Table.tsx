import type { ReactNode } from 'react'
import { ArrowUpDown, ChevronDown, ChevronUp } from 'lucide-react'
import { cn } from '../../lib/cn'
import type { SortState } from '../../lib/useSort'

export const TH_CLASS = 'px-5 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider whitespace-nowrap'

export function TableCard({ children }: { children: ReactNode }) {
  return (
    <div className="bg-white rounded-xl shadow-2xs border border-gray-200 overflow-hidden">
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-gray-200">{children}</table>
      </div>
    </div>
  )
}

export function Th({ children, align = 'left', className }: { children?: ReactNode; align?: 'left' | 'right'; className?: string }) {
  return <th className={cn(TH_CLASS, align === 'right' && 'text-right', className)}>{children}</th>
}

export function SortIcon({ active, direction }: { active: boolean; direction: 'asc' | 'desc' }) {
  if (!active) return <ArrowUpDown size={12} className="text-gray-400" />
  return direction === 'asc' ? (
    <ChevronUp size={14} className="text-indigo-600" />
  ) : (
    <ChevronDown size={14} className="text-indigo-600" />
  )
}

interface SortableThProps<F extends string> {
  label: string
  field: F
  sort: SortState<F>
  className?: string
}

export function SortableTh<F extends string>({ label, field, sort, className }: SortableThProps<F>) {
  const active = sort.field === field
  return (
    <th
      aria-sort={active ? (sort.direction === 'asc' ? 'ascending' : 'descending') : 'none'}
      className={cn(TH_CLASS, 'p-0', className)}
    >
      <button
        type="button"
        onClick={() => sort.toggle(field)}
        className="flex w-full items-center gap-1 px-5 py-3 uppercase tracking-wider font-medium cursor-pointer hover:bg-gray-100 transition-colors"
      >
        {label}
        <SortIcon active={active} direction={sort.direction} />
      </button>
    </th>
  )
}
