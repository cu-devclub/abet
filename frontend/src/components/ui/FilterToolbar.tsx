import type { ReactNode } from 'react'
import { RotateCcw, X } from 'lucide-react'
import { cn } from '../../lib/cn'
import { SearchInput } from './SearchInput'

export interface ActiveFilter {
  key: string
  label: string
  onClear: () => void
  /** Search pills use the neutral gray style. */
  neutral?: boolean
}

export function FilterPill({ filter }: { filter: ActiveFilter }) {
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1 border px-2 py-0.5 rounded-full font-medium',
        filter.neutral ? 'bg-gray-100 text-gray-700 border-gray-200' : 'bg-indigo-50 text-indigo-700 border-indigo-200',
      )}
    >
      {filter.label}
      <button type="button" onClick={filter.onClear} aria-label={`Clear filter ${filter.label}`} className="hover:text-indigo-900 cursor-pointer ml-0.5">
        <X size={12} />
      </button>
    </span>
  )
}

interface FilterToolbarProps {
  search: string
  onSearch: (value: string) => void
  placeholder: string
  /** Right-hand "Showing N …" summary. */
  summary: ReactNode
  filters: ActiveFilter[]
  onClearAll: () => void
}

/** Search box + result summary + active-filter pills (shared by every list page). */
export function FilterToolbar({ search, onSearch, placeholder, summary, filters, onClearAll }: FilterToolbarProps) {
  return (
    <div className="bg-white p-3 rounded-xl border border-gray-200 shadow-2xs space-y-2">
      <div className="flex items-center justify-between gap-3">
        <SearchInput value={search} onChange={onSearch} placeholder={placeholder} />
        <div className="text-xs text-gray-500 px-2 whitespace-nowrap">{summary}</div>
      </div>
      {filters.length > 0 && (
        <div className="flex flex-wrap items-center gap-2 pt-2 border-t border-gray-100 text-xs">
          <span className="text-gray-500 font-medium">Active filters:</span>
          {filters.map((filter) => (
            <FilterPill key={filter.key} filter={filter} />
          ))}
          <button
            type="button"
            onClick={onClearAll}
            className="text-indigo-600 hover:text-indigo-800 font-semibold underline ml-1 cursor-pointer flex items-center gap-1"
          >
            <RotateCcw size={11} /> Clear all
          </button>
        </div>
      )}
    </div>
  )
}

export function Count({ children }: { children: ReactNode }) {
  return <span className="font-semibold text-gray-900">{children}</span>
}
