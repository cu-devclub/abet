import { useState } from 'react'
import { createPortal } from 'react-dom'
import { Filter } from 'lucide-react'
import { cn } from '../../lib/cn'
import type { SortState } from '../../lib/useSort'
import { SortIcon, TH_CLASS } from './Table'
import { useSmartPopover } from './useSmartPopover'

export interface ColumnFilterOption<V extends string | number> {
  value: V
  label: string
  /** Optional right-aligned chip (role colours). */
  chip?: { text: string; className: string }
}

interface ColumnFilterThProps<F extends string, V extends string | number> {
  label: string
  field: F
  sort: SortState<F>
  options: readonly ColumnFilterOption<V>[]
  selected: readonly V[]
  onChange: (next: V[]) => void
  popoverWidth?: string
}

/** Sortable table header with a checkbox-list filter popover (Semester, Role). */
export function ColumnFilterTh<F extends string, V extends string | number>({
  label,
  field,
  sort,
  options,
  selected,
  onChange,
  popoverWidth = 'w-48',
}: ColumnFilterThProps<F, V>) {
  const [isOpen, setIsOpen] = useState(false)
  const { buttonRef, popoverRef, popoverStyle } = useSmartPopover(isOpen, setIsOpen)
  const hasFilter = selected.length > 0
  const active = sort.field === field

  const toggle = (value: V) =>
    onChange(selected.includes(value) ? selected.filter((v) => v !== value) : [...selected, value])

  return (
    <th
      aria-sort={active ? (sort.direction === 'asc' ? 'ascending' : 'descending') : 'none'}
      className={cn(TH_CLASS, 'relative select-none')}
    >
      <div className="flex items-center gap-1.5">
        <button
          type="button"
          onClick={() => sort.toggle(field)}
          title={`Click to sort by ${label.toLowerCase()}`}
          className="flex items-center gap-1 uppercase tracking-wider font-medium cursor-pointer hover:text-gray-900 transition-colors"
        >
          <span>{label}</span>
          <SortIcon active={active} direction={sort.direction} />
        </button>

        <button
          ref={buttonRef}
          type="button"
          onClick={() => setIsOpen(!isOpen)}
          aria-label={`Filter by ${label}`}
          aria-expanded={isOpen}
          title={`Filter by ${label}`}
          className={cn(
            'p-1 rounded-md transition-all cursor-pointer flex items-center gap-0.5',
            hasFilter ? 'bg-indigo-100 text-indigo-700 hover:bg-indigo-200' : 'text-gray-400 hover:text-gray-600 hover:bg-gray-200/60',
          )}
        >
          <Filter size={13} className={hasFilter ? 'fill-indigo-600 text-indigo-600' : ''} />
          {hasFilter && (
            <span className="text-[10px] font-bold leading-none px-1 py-0.5 bg-indigo-600 text-white rounded-full">{selected.length}</span>
          )}
        </button>

        {isOpen &&
          createPortal(
            <div
              ref={popoverRef}
              style={popoverStyle}
              role="dialog"
              aria-label={`Filter ${label}`}
              className={cn(
                'bg-white rounded-xl shadow-2xl border border-gray-200/90 p-2.5 normal-case font-normal text-gray-700',
                popoverWidth,
              )}
            >
              <div className="flex items-center justify-between pb-2 mb-2 border-b border-gray-100">
                <span className="text-xs font-bold text-gray-800">Filter {label}</span>
                {hasFilter ? (
                  <button type="button" onClick={() => onChange([])} className="text-[11px] text-indigo-600 hover:text-indigo-800 font-semibold cursor-pointer">
                    Clear
                  </button>
                ) : (
                  <button
                    type="button"
                    onClick={() => onChange(options.map((o) => o.value))}
                    className="text-[11px] text-gray-500 hover:text-gray-700 cursor-pointer"
                  >
                    Select All
                  </button>
                )}
              </div>

              <div className="space-y-1">
                {options.map((option) => {
                  const checked = selected.includes(option.value)
                  return (
                    <label
                      key={option.value}
                      className={cn(
                        'flex items-center justify-between px-2 py-1.5 rounded-lg text-xs cursor-pointer transition-colors',
                        checked ? 'bg-indigo-50 text-indigo-900 font-medium' : 'hover:bg-gray-50 text-gray-700',
                      )}
                    >
                      <span className="flex items-center gap-2">
                        <input
                          type="checkbox"
                          checked={checked}
                          onChange={() => toggle(option.value)}
                          className="h-3.5 w-3.5 text-indigo-600 rounded border-gray-300 focus:ring-indigo-500 cursor-pointer"
                        />
                        <span>{option.label}</span>
                      </span>
                      {option.chip && (
                        <span className={cn('text-[9px] px-1.5 rounded border font-semibold', option.chip.className)}>{option.chip.text}</span>
                      )}
                    </label>
                  )
                })}
              </div>

              <div className="pt-2 mt-2 border-t border-gray-100 flex justify-end">
                <button
                  type="button"
                  onClick={() => setIsOpen(false)}
                  className="px-2.5 py-1 bg-indigo-600 text-white rounded-md text-[11px] font-semibold hover:bg-indigo-700 transition-colors cursor-pointer"
                >
                  Done
                </button>
              </div>
            </div>,
            document.body,
          )}
      </div>
    </th>
  )
}
