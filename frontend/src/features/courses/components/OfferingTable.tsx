import { Fragment } from 'react'
import { AlertTriangle, ChevronRight, Edit2, Users } from 'lucide-react'
import { Checkbox } from '../../../components/ui/Checkbox'
import { ColumnFilterTh } from '../../../components/ui/ColumnFilterPopover'
import { EmptyState } from '../../../components/ui/EmptyState'
import { IconButton } from '../../../components/ui/IconButton'
import { SortableTh, TableCard, Th } from '../../../components/ui/Table'
import { cn } from '../../../lib/cn'
import { plural, SEMESTER_OPTIONS, semesterText } from '../../../lib/format'
import type { SortState } from '../../../lib/useSort'
import type { AccessEntry, Offering } from '../types'
import { AccessSubTable } from './AccessSubTable'

export type OfferingSortField = 'id' | 'name' | 'semester'

const SEMESTER_FILTER_OPTIONS = SEMESTER_OPTIONS.map((o) => ({ value: o.value, label: o.label }))
const COLUMNS = 6

interface OfferingTableProps {
  rows: Offering[]
  year: number
  sort: SortState<OfferingSortField>
  selected: string[]
  expanded: string[]
  semesters: number[]
  onSemestersChange: (semesters: number[]) => void
  onToggleSelect: (id: string) => void
  onToggleSelectAll: () => void
  onToggleExpand: (id: string) => void
  onEdit: (offering: Offering) => void
  onAssign: (offering: Offering, entries: AccessEntry[]) => void
  onEditAccess: (offering: Offering, entry: AccessEntry) => void
}

export function OfferingTable({
  rows,
  year,
  sort,
  selected,
  expanded,
  semesters,
  onSemestersChange,
  onToggleSelect,
  onToggleSelectAll,
  onToggleExpand,
  onEdit,
  onAssign,
  onEditAccess,
}: OfferingTableProps) {
  const allSelected = selected.length === rows.length && rows.length > 0

  return (
    <TableCard>
      <thead className="bg-gray-50">
        <tr>
          <Th className="w-9 px-3">
            <span className="sr-only">Expand</span>
          </Th>
          <Th className="w-10 px-3">
            <Checkbox checked={allSelected} onChange={onToggleSelectAll} aria-label={allSelected ? 'Deselect all' : 'Select all'} />
          </Th>
          <SortableTh label="Course ID" field="id" sort={sort} />
          <SortableTh label="Course Name" field="name" sort={sort} />
          <ColumnFilterTh label="Semester" field="semester" sort={sort} options={SEMESTER_FILTER_OPTIONS} selected={semesters} onChange={onSemestersChange} />
          <Th align="right">Actions</Th>
        </tr>
      </thead>
      <tbody className="bg-white divide-y divide-gray-200">
        {rows.map((offering) => {
          const isSelected = selected.includes(offering.id)
          const isExpanded = expanded.includes(offering.id)
          const memberCount = offering.instructor_count + offering.staff_count
          const unassigned = memberCount === 0
          return (
            <Fragment key={offering.id}>
              <tr
                onClick={() => onToggleExpand(offering.id)}
                className={cn(
                  'transition-colors cursor-pointer',
                  unassigned
                    ? isSelected
                      ? 'bg-amber-100/70 hover:bg-amber-100/90'
                      : isExpanded
                        ? 'bg-amber-50/90 hover:bg-amber-100/80'
                        : 'bg-amber-50/50 hover:bg-amber-100/60'
                    : isSelected
                      ? 'bg-indigo-50/40 hover:bg-indigo-50/60'
                      : isExpanded
                        ? 'bg-slate-50/60'
                        : 'hover:bg-gray-50/80',
                )}
              >
                <td className="pl-3.5 pr-1 py-4 whitespace-nowrap text-center" onClick={(e) => e.stopPropagation()}>
                  <button
                    type="button"
                    onClick={() => onToggleExpand(offering.id)}
                    aria-expanded={isExpanded}
                    aria-label={`${isExpanded ? 'Collapse' : 'Expand'} assigned users of ${offering.course_code}`}
                    title={isExpanded ? 'Collapse assigned users' : 'Expand assigned users'}
                    className="p-1 rounded-md text-gray-400 hover:text-indigo-600 hover:bg-indigo-50 transition-colors cursor-pointer"
                  >
                    <ChevronRight size={16} className={cn('transition-transform duration-200', isExpanded ? 'rotate-90 text-indigo-600' : 'text-gray-400')} />
                  </button>
                </td>
                <td className="px-3 py-4 whitespace-nowrap" onClick={(e) => e.stopPropagation()}>
                  <Checkbox checked={isSelected} onChange={() => onToggleSelect(offering.id)} aria-label={`Select ${offering.course_code}`} />
                </td>
                <td className="px-5 py-4 whitespace-nowrap text-sm font-semibold text-indigo-700">
                  <span className="bg-indigo-50 px-2.5 py-1 rounded-md text-xs tracking-wider uppercase">{offering.course_code}</span>
                </td>
                <td className="px-5 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                  <div className="flex items-center gap-2">
                    <span>{offering.course_name}</span>
                    <button
                      type="button"
                      onClick={(e) => {
                        e.stopPropagation()
                        onToggleExpand(offering.id)
                      }}
                      className={cn(
                        'inline-flex items-center gap-1 text-[11px] font-medium px-2 py-0.5 rounded-full transition-colors cursor-pointer',
                        unassigned
                          ? 'bg-amber-100 text-amber-800 border border-amber-300/80 hover:bg-amber-200/80'
                          : isExpanded
                            ? 'bg-indigo-100 text-indigo-800'
                            : 'bg-gray-100 hover:bg-indigo-50 text-gray-600 hover:text-indigo-700',
                      )}
                      title={unassigned ? 'Warning: 0 users assigned. Click to assign.' : 'Click to view assigned users with access'}
                    >
                      {unassigned ? (
                        <>
                          <AlertTriangle size={11} className="text-amber-700 shrink-0" />
                          <span>0 assigned</span>
                        </>
                      ) : (
                        <>
                          <Users size={11} className={isExpanded ? 'text-indigo-700' : 'text-gray-500'} />
                          <span>
                            {memberCount} {plural(memberCount, 'user')}
                          </span>
                        </>
                      )}
                    </button>
                  </div>
                </td>
                <td className="px-5 py-4 whitespace-nowrap text-sm text-gray-600">
                  <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-700">{semesterText(offering.semester)}</span>
                </td>
                <td className="px-5 py-4 whitespace-nowrap text-right" onClick={(e) => e.stopPropagation()}>
                  <IconButton icon={Edit2} label={`Edit ${offering.course_code}`} size={13} onClick={() => onEdit(offering)} />
                </td>
              </tr>
              {isExpanded && (
                <tr className="bg-slate-50/70 border-b border-gray-200">
                  <td colSpan={COLUMNS} className="p-0">
                    <AccessSubTable offering={offering} onAssign={(entries) => onAssign(offering, entries)} onEdit={(entry) => onEditAccess(offering, entry)} />
                  </td>
                </tr>
              )}
            </Fragment>
          )
        })}
        {rows.length === 0 && (
          <tr>
            <td colSpan={COLUMNS}>
              <EmptyState>No courses found matching the selected filters for academic year {year}.</EmptyState>
            </td>
          </tr>
        )}
      </tbody>
    </TableCard>
  )
}
