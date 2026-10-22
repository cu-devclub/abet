import { useMemo, useState } from 'react'
import { FileText, Plus, Trash2 } from 'lucide-react'
import { PageHeader } from '../../components/layout/PageHeader'
import { Button } from '../../components/ui/Button'
import { ConfirmDialog } from '../../components/ui/ConfirmDialog'
import { Count, FilterToolbar, type ActiveFilter } from '../../components/ui/FilterToolbar'
import { SearchableSelect } from '../../components/ui/SearchableSelect'
import { PageSpinner } from '../../components/ui/Spinner'
import { useToast } from '../../components/ui/useToast'
import { plural } from '../../lib/format'
import { sortRows, useSort } from '../../lib/useSort'
import type { Curriculum } from '../curricula/types'
import { DeptCurriculumFilters } from '../shared/DeptCurriculumFilters'
import { orgPills } from '../shared/orgPills'
import { useOrgFilters } from '../shared/useOrgFilters'
import { useOrgOptions } from '../shared/useOrgOptions'
import { useDownloadCover, useRemoveCovers } from './api'
import { CoverPreviewModal } from './components/CoverPreviewModal'
import { CoverTable, type CoverSortField } from './components/CoverTable'
import { UploadCoverModal } from './components/UploadCoverModal'
import { COVER_STATUS_OPTIONS, type CoverStatus } from './types'

type Dialog =
  | { kind: 'upload'; curriculum: Curriculum | null }
  | { kind: 'preview'; curriculum: Curriculum }
  | { kind: 'remove'; curriculum: Curriculum }
  | { kind: 'bulk-remove' }
  | null

function sortValue(curr: Curriculum, field: CoverSortField): string {
  switch (field) {
    case 'code':
      return curr.code
    case 'name':
      return curr.name
    case 'department':
      return curr.department.name
    case 'cover':
      return curr.cover?.filename ?? ''
    case 'time':
      return curr.cover?.updated_at ?? ''
  }
}

export function CoversPage() {
  const toast = useToast()
  const org = useOrgOptions()
  const filters = useOrgFilters()
  const sort = useSort<CoverSortField>('code')
  const [status, setStatus] = useState<CoverStatus | ''>('')
  const [selected, setSelected] = useState<string[]>([])
  const [dialog, setDialog] = useState<Dialog>(null)
  const remove = useRemoveCovers()
  const download = useDownloadCover()
  const { curricula, departments } = org

  const query = filters.search.trim().toLowerCase()
  const rows = useMemo(() => {
    const filtered = curricula.filter((curr) => {
      if (filters.departmentId && curr.department.id !== filters.departmentId) return false
      if (filters.curriculumId && curr.id !== filters.curriculumId) return false
      if (status === 'has_cover' && !curr.cover) return false
      if (status === 'no_cover' && curr.cover) return false
      if (!query) return true
      return [curr.name, curr.code, curr.department.name, curr.cover?.filename].some((v) => v?.toLowerCase().includes(query))
    })
    return sortRows(filtered, (c) => sortValue(c, sort.field), sort.direction)
  }, [curricula, filters.departmentId, filters.curriculumId, status, query, sort.field, sort.direction])

  const selectedWithCovers = curricula.filter((c) => selected.includes(c.id) && c.cover)
  const pills: ActiveFilter[] = orgPills(filters, departments, curricula).filter((p) => p.key !== 'sem')
  if (status) {
    const label = COVER_STATUS_OPTIONS.find((o) => o.value === status)?.label
    pills.splice(2, 0, { key: 'status', label: `Status: ${label}`, onClear: () => setStatus('') })
  }
  const clearAll = () => {
    filters.clearAll()
    setStatus('')
  }

  const toggle = (id: string) => setSelected((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]))
  const toggleAll = () => setSelected(selected.length === rows.length ? [] : rows.map((c) => c.id))

  const doRemove = (ids: string[], message: string) => {
    setDialog(null)
    remove.mutate(ids, {
      onSuccess: () => {
        setSelected((prev) => prev.filter((id) => !ids.includes(id)))
        toast.info(message)
      },
    })
  }

  const closeDialog = () => setDialog(null)

  return (
    <div className="p-6 space-y-5 w-full">
      <div className="space-y-4">
        <PageHeader
          title="Cover Page"
          subtitle="Manage standard course cover PDFs by curriculum. All courses under the same curriculum share the same cover."
          actions={
            <>
              {selectedWithCovers.length > 0 && (
                <Button variant="danger" icon={Trash2} onClick={() => setDialog({ kind: 'bulk-remove' })}>
                  Remove Covers ({selectedWithCovers.length})
                </Button>
              )}
              <Button icon={Plus} onClick={() => setDialog({ kind: 'upload', curriculum: null })} disabled={curricula.length === 0}>
                Upload Cover PDF
              </Button>
            </>
          }
        />
        <div className="flex flex-wrap items-center gap-2.5">
          <DeptCurriculumFilters
            departments={departments}
            curricula={curricula}
            departmentId={filters.departmentId}
            curriculumId={filters.curriculumId}
            onDepartmentChange={filters.setDepartment}
            onCurriculumChange={filters.setCurriculum}
          />
          <SearchableSelect
            label="Cover Status"
            value={status}
            onChange={(v) => setStatus(v as CoverStatus | '')}
            options={COVER_STATUS_OPTIONS.map((o) => ({ value: o.value, label: o.label }))}
            allLabel="All Statuses"
            icon={FileText}
          />
        </div>
      </div>

      <FilterToolbar
        search={filters.search}
        onSearch={filters.setSearch}
        placeholder="Search curriculum code, name, department, cover..."
        summary={
          <>
            Showing <Count>{rows.length}</Count> {plural(rows.length, 'curriculum', 'curriculums')}
          </>
        }
        filters={pills}
        onClearAll={clearAll}
      />

      {org.isLoading ? (
        <PageSpinner />
      ) : (
        <CoverTable
          curricula={rows}
          sort={sort}
          selected={selected}
          onToggle={toggle}
          onToggleAll={toggleAll}
          onPreview={(curriculum) => setDialog({ kind: 'preview', curriculum })}
          onDownload={(curriculum) => download.mutate(curriculum)}
          onUpload={(curriculum) => setDialog({ kind: 'upload', curriculum })}
          onRemove={(curriculum) => setDialog({ kind: 'remove', curriculum })}
        />
      )}

      {dialog?.kind === 'upload' && <UploadCoverModal curricula={curricula} fixed={dialog.curriculum} onClose={closeDialog} />}
      {dialog?.kind === 'preview' && <CoverPreviewModal curriculum={dialog.curriculum} onClose={closeDialog} />}
      <ConfirmDialog
        isOpen={dialog?.kind === 'remove'}
        title="Remove Cover PDF"
        confirmLabel="Remove Cover"
        loading={remove.isPending}
        onClose={closeDialog}
        onConfirm={() => {
          if (dialog?.kind === 'remove') {
            doRemove([dialog.curriculum.id], `Cover removed from ${dialog.curriculum.name}.`)
          }
        }}
      >
        <p className="text-sm text-gray-600">
          Are you sure you want to remove the cover PDF from curriculum <strong>"{dialog?.kind === 'remove' ? dialog.curriculum.name : ''}"</strong>? Courses
          under this curriculum will no longer have a cover sheet attached until a new one is uploaded.
        </p>
      </ConfirmDialog>
      <ConfirmDialog
        isOpen={dialog?.kind === 'bulk-remove'}
        title="Remove Selected Covers"
        confirmLabel={`Remove Covers (${selectedWithCovers.length})`}
        loading={remove.isPending}
        onClose={closeDialog}
        onConfirm={() => doRemove(selectedWithCovers.map((c) => c.id), 'Covers removed from selected curriculums.')}
      >
        <p className="text-sm text-gray-600">
          Are you sure you want to remove cover PDFs from the <strong>{selectedWithCovers.length}</strong> selected curriculum(s)? This action cannot be undone.
        </p>
      </ConfirmDialog>
    </div>
  )
}
