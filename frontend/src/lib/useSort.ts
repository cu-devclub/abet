import { useState } from 'react'

export type SortDirection = 'asc' | 'desc'

export interface SortState<F extends string> {
  field: F
  direction: SortDirection
  toggle: (field: F) => void
}

export function useSort<F extends string>(initialField: F, initialDirection: SortDirection = 'asc'): SortState<F> {
  const [state, setState] = useState<{ field: F; direction: SortDirection }>({
    field: initialField,
    direction: initialDirection,
  })

  const toggle = (field: F) =>
    setState((prev) =>
      prev.field === field
        ? { field, direction: prev.direction === 'asc' ? 'desc' : 'asc' }
        : { field, direction: 'asc' },
    )

  return { ...state, toggle }
}

export function sortRows<T>(rows: readonly T[], getValue: (row: T) => string | number, direction: SortDirection): T[] {
  const sign = direction === 'asc' ? 1 : -1
  return [...rows].sort((a, b) => {
    const av = getValue(a)
    const bv = getValue(b)
    if (typeof av === 'number' && typeof bv === 'number') return (av - bv) * sign
    return String(av).localeCompare(String(bv), undefined, { numeric: true, sensitivity: 'base' }) * sign
  })
}
