import { useEffect, useId, useRef, useState, type ElementType } from 'react'
import { CheckCircle2, ChevronDown, Search, X } from 'lucide-react'
import { cn } from '../../lib/cn'

export interface SelectOption {
  value: string
  label: string
  subtext?: string
}

interface SearchableSelectProps {
  label: string
  value: string
  onChange: (value: string) => void
  options: SelectOption[]
  allLabel?: string
  icon?: ElementType
  disabled?: boolean
  className?: string
  /** A value is always selected: no "All" entry and no clear button. */
  required?: boolean
  /** Show the search box (default true; turn off for short lists). */
  searchable?: boolean
}

export function SearchableSelect({
  label,
  value,
  onChange,
  options,
  allLabel = 'All',
  icon: Icon,
  disabled = false,
  className,
  required = false,
  searchable = true,
}: SearchableSelectProps) {
  const [isOpen, setIsOpen] = useState(false)
  const [search, setSearch] = useState('')
  const containerRef = useRef<HTMLDivElement>(null)
  const searchInputRef = useRef<HTMLInputElement>(null)
  const listId = useId()

  useEffect(() => {
    if (!isOpen) return
    const onMouseDown = (event: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) setIsOpen(false)
    }
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setIsOpen(false)
    }
    document.addEventListener('mousedown', onMouseDown)
    document.addEventListener('keydown', onKeyDown)
    if (searchable) searchInputRef.current?.focus()
    return () => {
      document.removeEventListener('mousedown', onMouseDown)
      document.removeEventListener('keydown', onKeyDown)
    }
  }, [isOpen, searchable])

  const needle = search.toLowerCase()
  const filtered = options.filter(
    (opt) => opt.label.toLowerCase().includes(needle) || opt.subtext?.toLowerCase().includes(needle),
  )
  const selected = options.find((o) => o.value === value)

  const choose = (next: string) => {
    onChange(next)
    setIsOpen(false)
  }

  return (
    <div className={cn('relative', className)} ref={containerRef}>
      <div
        className={cn(
          'flex items-center gap-2 bg-white border rounded-lg shadow-2xs px-3 py-1.5 text-sm transition-all',
          disabled ? 'opacity-60 bg-gray-50 border-gray-200' : 'hover:border-indigo-300 border-gray-300',
          isOpen && 'ring-2 ring-indigo-500/20 border-indigo-500',
        )}
      >
        <button
          type="button"
          disabled={disabled}
          aria-haspopup="listbox"
          aria-expanded={isOpen}
          aria-controls={isOpen ? listId : undefined}
          aria-label={`${label}: ${selected ? selected.label : allLabel}`}
          onClick={() => {
            setIsOpen(!isOpen)
            setSearch('')
          }}
          className="flex items-center gap-2 cursor-pointer disabled:cursor-not-allowed focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 rounded"
        >
          {Icon && <Icon size={14} className="text-gray-400 shrink-0" />}
          <span className="text-xs font-semibold text-gray-500 uppercase tracking-wide shrink-0">{label}:</span>
          <span
            className={cn('font-semibold truncate max-w-[190px] text-left', value ? 'text-indigo-700' : 'text-gray-700')}
            title={selected ? selected.label : allLabel}
          >
            {selected ? selected.label : allLabel}
          </span>
          {(required || !value) && (
            <ChevronDown size={14} className={cn('text-gray-400 transition-transform', isOpen && 'rotate-180 text-indigo-600')} />
          )}
        </button>
        {value && !disabled && !required && (
          <button
            type="button"
            onClick={() => choose('')}
            aria-label={`Clear ${label}`}
            title="Clear selection"
            className="p-0.5 hover:bg-gray-200 rounded-full text-gray-400 hover:text-gray-600 ml-1 transition-colors cursor-pointer"
          >
            <X size={13} />
          </button>
        )}
      </div>

      {isOpen && (
        <div
          className={cn(
            'absolute left-0 mt-1.5 bg-white rounded-xl shadow-xl border border-gray-200 py-2 z-50',
            searchable ? 'w-72' : 'min-w-full w-max',
          )}
        >
          {searchable && (
            <div className="px-2.5 pb-2 border-b border-gray-100">
              <div className="relative">
                <Search size={14} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-gray-400" />
                <input
                  ref={searchInputRef}
                  type="text"
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  aria-label={`Search ${label.toLowerCase()}`}
                  placeholder={`Search ${label.toLowerCase()}...`}
                  className="w-full pl-8 pr-7 py-1.5 text-xs bg-gray-50 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:bg-white transition-colors"
                />
                {search && (
                  <button
                    type="button"
                    onClick={() => setSearch('')}
                    aria-label="Clear search"
                    className="absolute right-2 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 cursor-pointer"
                  >
                    <X size={12} />
                  </button>
                )}
              </div>
            </div>
          )}

          <div id={listId} role="listbox" aria-label={label} className="max-h-56 overflow-y-auto py-1 text-xs">
            {!required && (!search || allLabel.toLowerCase().includes(needle)) && (
              <button
                type="button"
                role="option"
                aria-selected={!value}
                onClick={() => choose('')}
                className={cn(
                  'w-full text-left px-3 py-2 flex items-center justify-between hover:bg-indigo-50/70 transition-colors cursor-pointer',
                  !value ? 'bg-indigo-50 text-indigo-700 font-semibold' : 'text-gray-700',
                )}
              >
                <span>{allLabel}</span>
                {!value && <CheckCircle2 size={14} className="text-indigo-600 shrink-0" />}
              </button>
            )}

            {filtered.map((opt) => {
              const isSelected = opt.value === value
              return (
                <button
                  key={opt.value}
                  type="button"
                  role="option"
                  aria-selected={isSelected}
                  onClick={() => choose(opt.value)}
                  className={cn(
                    'w-full text-left px-3 py-2 flex items-center justify-between hover:bg-indigo-50/70 transition-colors cursor-pointer',
                    isSelected ? 'bg-indigo-50 text-indigo-700 font-semibold' : 'text-gray-700',
                  )}
                >
                  <div className="flex items-center gap-2 min-w-0 pr-2">
                    {opt.subtext && (
                      <span className="bg-gray-100 text-gray-600 font-mono text-[10px] px-1.5 py-0.5 rounded font-semibold uppercase shrink-0">
                        {opt.subtext}
                      </span>
                    )}
                    <span className="truncate">{opt.label}</span>
                  </div>
                  {isSelected && <CheckCircle2 size={14} className="text-indigo-600 shrink-0" />}
                </button>
              )
            })}

            {filtered.length === 0 && <div className="px-3 py-4 text-center text-gray-400 italic">No matching results</div>}
          </div>
        </div>
      )}
    </div>
  )
}
