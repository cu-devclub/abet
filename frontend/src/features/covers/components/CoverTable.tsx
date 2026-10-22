import { CheckCircle2, Download, Eye, FileText, Trash2, UploadCloud } from 'lucide-react'
import { Checkbox } from '../../../components/ui/Checkbox'
import { EmptyState } from '../../../components/ui/EmptyState'
import { IconButton } from '../../../components/ui/IconButton'
import { SortableTh, TableCard, Th } from '../../../components/ui/Table'
import { cn } from '../../../lib/cn'
import { formatDateTime } from '../../../lib/format'
import type { SortState } from '../../../lib/useSort'
import type { Curriculum } from '../../curricula/types'

export type CoverSortField = 'code' | 'name' | 'department' | 'cover' | 'time'

interface CoverTableProps {
  curricula: Curriculum[]
  sort: SortState<CoverSortField>
  selected: string[]
  onToggle: (id: string) => void
  onToggleAll: () => void
  onPreview: (curriculum: Curriculum) => void
  onDownload: (curriculum: Curriculum) => void
  onUpload: (curriculum: Curriculum) => void
  onRemove: (curriculum: Curriculum) => void
}

export function CoverTable({ curricula, sort, selected, onToggle, onToggleAll, onPreview, onDownload, onUpload, onRemove }: CoverTableProps) {
  const allSelected = selected.length === curricula.length && curricula.length > 0

  return (
    <TableCard>
      <thead className="bg-gray-50">
        <tr>
          <Th className="w-10 px-4">
            <Checkbox checked={allSelected} onChange={onToggleAll} aria-label={allSelected ? 'Deselect all' : 'Select all'} />
          </Th>
          <SortableTh label="Code" field="code" sort={sort} />
          <SortableTh label="Curriculum Name" field="name" sort={sort} />
          <SortableTh label="Department" field="department" sort={sort} />
          <SortableTh label="Cover PDF" field="cover" sort={sort} />
          <SortableTh label="Updated Time" field="time" sort={sort} />
          <Th align="right">Actions</Th>
        </tr>
      </thead>
      <tbody className="bg-white divide-y divide-gray-200">
        {curricula.map((curr) => {
          const isSelected = selected.includes(curr.id)
          const cover = curr.cover
          return (
            <tr
              key={curr.id}
              onClick={() => onToggle(curr.id)}
              className={cn('hover:bg-gray-50 transition-colors cursor-pointer', isSelected && 'bg-indigo-50/40 hover:bg-indigo-50/60')}
            >
              <td className="px-4 py-4 whitespace-nowrap" onClick={(e) => e.stopPropagation()}>
                <Checkbox checked={isSelected} onChange={() => onToggle(curr.id)} aria-label={`Select ${curr.name}`} />
              </td>
              <td className="px-5 py-4 whitespace-nowrap text-sm font-semibold text-indigo-700">
                <span className="bg-indigo-50 px-2.5 py-1 rounded-md text-xs tracking-wider uppercase font-mono">{curr.code}</span>
              </td>
              <td className="px-5 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{curr.name}</td>
              <td className="px-5 py-4 whitespace-nowrap text-sm text-gray-500">
                <div className="max-w-[140px] truncate" title={curr.department.name}>
                  {curr.department.name}
                </div>
              </td>
              <td className="px-5 py-4 whitespace-nowrap text-sm">
                {cover ? (
                  <div className="flex items-center gap-2">
                    <div className="p-1 bg-rose-50 text-rose-600 rounded shrink-0">
                      <FileText size={15} />
                    </div>
                    <span className="font-medium text-gray-900 text-xs cursor-help" title={cover.filename}>
                      {cover.filename.length > 12 ? `${cover.filename.slice(0, 12)}...` : cover.filename}
                    </span>
                  </div>
                ) : (
                  <span className="text-gray-400 text-xs italic">No Cover</span>
                )}
              </td>
              <td className="px-5 py-4 whitespace-nowrap text-sm text-gray-500">
                {cover ? (
                  <div className="flex items-center text-xs text-gray-500">
                    <CheckCircle2 size={12} className="mr-1 text-gray-400" />
                    {formatDateTime(cover.updated_at)}
                  </div>
                ) : (
                  '-'
                )}
              </td>
              <td className="px-5 py-4 whitespace-nowrap text-right text-sm font-medium space-x-1.5" onClick={(e) => e.stopPropagation()}>
                {cover && (
                  <>
                    <IconButton icon={Eye} label="Preview Cover PDF" tone="gray" onClick={() => onPreview(curr)} />
                    <IconButton icon={Download} label="Download PDF" tone="green" onClick={() => onDownload(curr)} />
                  </>
                )}
                <IconButton icon={UploadCloud} label={cover ? 'Replace Cover PDF' : 'Upload Cover PDF'} tone="indigo" onClick={() => onUpload(curr)} />
                {cover && <IconButton icon={Trash2} label="Remove Cover" tone="red" onClick={() => onRemove(curr)} />}
              </td>
            </tr>
          )
        })}
        {curricula.length === 0 && (
          <tr>
            <td colSpan={7}>
              <EmptyState>No curriculums found matching your search and filter criteria.</EmptyState>
            </td>
          </tr>
        )}
      </tbody>
    </TableCard>
  )
}
