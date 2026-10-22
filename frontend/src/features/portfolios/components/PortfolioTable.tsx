import { AlertTriangle, CheckCircle2, Download, UploadCloud } from 'lucide-react'
import { Checkbox } from '../../../components/ui/Checkbox'
import { ColumnFilterTh } from '../../../components/ui/ColumnFilterPopover'
import { EmptyState } from '../../../components/ui/EmptyState'
import { IconButton } from '../../../components/ui/IconButton'
import { StatusBadge } from '../../../components/ui/StatusBadge'
import { SortableTh, TableCard, Th } from '../../../components/ui/Table'
import { cn } from '../../../lib/cn'
import { formatDateTime, SEMESTER_OPTIONS, semesterText } from '../../../lib/format'
import type { SortState } from '../../../lib/useSort'
import { latestUpdate } from '../rowStatus'
import type { PortfolioRow } from '../types'

export type PortfolioSortField = 'id' | 'name' | 'semester' | 'status' | 'time'

const SEMESTER_FILTER_OPTIONS = SEMESTER_OPTIONS.map((o) => ({ value: o.value, label: o.label }))

interface PortfolioTableProps {
  rows: PortfolioRow[]
  year: number
  sort: SortState<PortfolioSortField>
  selected: string[]
  semesters: number[]
  onSemestersChange: (semesters: number[]) => void
  onToggle: (offeringId: string) => void
  onToggleAll: () => void
  onDownload: (row: PortfolioRow) => void
  onOpenIssues: (row: PortfolioRow) => void
  onUpload: (row: PortfolioRow) => void
}

export function PortfolioTable({
  rows,
  year,
  sort,
  selected,
  semesters,
  onSemestersChange,
  onToggle,
  onToggleAll,
  onDownload,
  onOpenIssues,
  onUpload,
}: PortfolioTableProps) {
  const allSelected = selected.length === rows.length && rows.length > 0

  return (
    <TableCard>
      <thead className="bg-gray-50">
        <tr>
          <Th className="w-10 px-4">
            <Checkbox checked={allSelected} onChange={onToggleAll} aria-label={allSelected ? 'Deselect all' : 'Select all'} />
          </Th>
          <SortableTh label="Course ID" field="id" sort={sort} />
          <SortableTh label="Course Name" field="name" sort={sort} />
          <ColumnFilterTh label="Semester" field="semester" sort={sort} options={SEMESTER_FILTER_OPTIONS} selected={semesters} onChange={onSemestersChange} />
          <SortableTh label="Status" field="status" sort={sort} />
          <SortableTh label="Latest Update" field="time" sort={sort} />
          <Th align="right">Actions</Th>
        </tr>
      </thead>
      <tbody className="bg-white divide-y divide-gray-200">
        {rows.map((row) => {
          const sub = row.latest_submission
          const isSelected = selected.includes(row.offering_id)
          const failed = sub?.status === 'invalid' || sub?.status === 'failed'
          const updated = latestUpdate(row)
          return (
            <tr
              key={row.offering_id}
              onClick={() => onToggle(row.offering_id)}
              className={cn('transition-colors cursor-pointer', isSelected ? 'bg-indigo-50/50 hover:bg-indigo-50/70' : 'hover:bg-gray-50')}
            >
              <td className="w-10 px-4 py-4 whitespace-nowrap" onClick={(e) => e.stopPropagation()}>
                <Checkbox checked={isSelected} onChange={() => onToggle(row.offering_id)} aria-label={`Select ${row.course_code}`} />
              </td>
              <td className="px-5 py-4 whitespace-nowrap text-sm font-semibold text-indigo-700">
                <span className="bg-indigo-50 px-2.5 py-1 rounded-md text-xs tracking-wider uppercase">{row.course_code}</span>
              </td>
              <td className="px-5 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{row.course_name}</td>
              <td className="px-5 py-4 whitespace-nowrap text-sm text-gray-600">
                <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-700">{semesterText(row.semester)}</span>
              </td>
              <td className="px-5 py-4 whitespace-nowrap text-sm" onClick={(e) => e.stopPropagation()}>
                <StatusBadge status={sub?.status ?? null} errorCount={sub?.error_count} createdAt={sub?.created_at} onFailedClick={() => onOpenIssues(row)} />
              </td>
              <td className="px-5 py-4 whitespace-nowrap text-sm text-gray-500">
                {updated ? (
                  <div className="flex items-center text-xs text-gray-500">
                    <CheckCircle2 size={12} className="mr-1 text-gray-400" />
                    {formatDateTime(updated)}
                  </div>
                ) : (
                  '-'
                )}
              </td>
              <td className="px-5 py-4 whitespace-nowrap text-right text-sm font-medium space-x-2" onClick={(e) => e.stopPropagation()}>
                {row.pdf_available && row.can_download && (
                  <IconButton icon={Download} label={`Download ${row.course_code} portfolio PDF`} tone="green" onClick={() => onDownload(row)} />
                )}
                {failed && <IconButton icon={AlertTriangle} label="View Processing Errors" tone="red" onClick={() => onOpenIssues(row)} />}
                {row.can_upload && (
                  <IconButton icon={UploadCloud} label={failed ? 'Re-upload Files' : 'Upload Files'} tone="indigo" onClick={() => onUpload(row)} />
                )}
              </td>
            </tr>
          )
        })}
        {rows.length === 0 && (
          <tr>
            <td colSpan={7}>
              <EmptyState>No courses found matching the selected filters for academic year {year}.</EmptyState>
            </td>
          </tr>
        )}
      </tbody>
    </TableCard>
  )
}
