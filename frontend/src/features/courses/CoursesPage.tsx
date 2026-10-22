import { useMemo, useState } from 'react'
import { Plus, Trash2 } from 'lucide-react'
import { PageHeader } from '../../components/layout/PageHeader'
import { Alert } from '../../components/ui/Alert'
import { Button } from '../../components/ui/Button'
import { Count, FilterToolbar } from '../../components/ui/FilterToolbar'
import { PageSpinner } from '../../components/ui/Spinner'
import { plural } from '../../lib/format'
import { sortRows, useSort } from '../../lib/useSort'
import { DeptCurriculumFilters } from '../shared/DeptCurriculumFilters'
import { orgPills } from '../shared/orgPills'
import { useOrgFilters } from '../shared/useOrgFilters'
import { useOrgOptions } from '../shared/useOrgOptions'
import { YearSelect } from '../shared/YearSelect'
import { useOfferings } from './api'
import { AssignUserModal } from './components/AssignUserModal'
import { ConfirmDeleteModal } from './components/ConfirmDeleteModal'
import { EditAccessModal } from './components/EditAccessModal'
import { ImportMappingModal } from './components/ImportMappingModal'
import { OfferingFormModal } from './components/OfferingFormModal'
import { OfferingTable, type OfferingSortField } from './components/OfferingTable'
import type { AccessEntry, Offering } from './types'

type Dialog =
  | { kind: 'form'; offering: Offering | null }
  | { kind: 'import' }
  | { kind: 'delete'; offerings: Offering[] }
  | { kind: 'assign'; offering: Offering; entries: AccessEntry[] }
  | { kind: 'edit-access'; offering: Offering; entry: AccessEntry }
  | null

function sortValue(o: Offering, field: OfferingSortField): string | number {
  switch (field) {
    case 'id':
      return o.course_code
    case 'name':
      return o.course_name
    case 'semester':
      return o.semester
  }
}

export function CoursesPage() {
  const org = useOrgOptions()
  const filters = useOrgFilters()
  const sort = useSort<OfferingSortField>('id')
  const [selected, setSelected] = useState<string[]>([])
  const [expanded, setExpanded] = useState<string[]>([])
  const [dialog, setDialog] = useState<Dialog>(null)

  const { year, departmentId, curriculumId } = filters
  const params = useMemo(
    () =>
      year === undefined
        ? null
        : { academic_year: year, department_id: departmentId || undefined, curriculum_id: curriculumId || undefined },
    [year, departmentId, curriculumId],
  )
  const offerings = useOfferings(params)

  const query = filters.search.trim().toLowerCase()
  const rows = useMemo(() => {
    const filtered = (offerings.data ?? []).filter((o) => {
      if (filters.semesters.length > 0 && !filters.semesters.includes(o.semester)) return false
      if (!query) return true
      return [o.course_code, o.course_name, o.department.name, o.curriculum.name].some((v) => v.toLowerCase().includes(query))
    })
    return sortRows(filtered, (o) => sortValue(o, sort.field), sort.direction)
  }, [offerings.data, filters.semesters, query, sort.field, sort.direction])

  const visibleSelected = selected.filter((id) => rows.some((o) => o.id === id))
  const toggle = (list: string[], id: string) => (list.includes(id) ? list.filter((x) => x !== id) : [...list, id])
  const selectedOfferings = rows.filter((o) => visibleSelected.includes(o.id))
  const curriculum = org.curricula.find((c) => c.id === curriculumId) ?? null
  const closeDialog = () => setDialog(null)

  return (
    <div className="p-6 space-y-5 w-full">
      <div className="space-y-4">
        <PageHeader
          title="Courses"
          actions={
            <>
              {selectedOfferings.length > 0 && (
                <Button variant="danger" icon={Trash2} onClick={() => setDialog({ kind: 'delete', offerings: selectedOfferings })}>
                  Delete Selected ({selectedOfferings.length})
                </Button>
              )}
              <Button icon={Plus} onClick={() => setDialog({ kind: 'form', offering: null })} disabled={year === undefined || org.isLoading}>
                Add New Course
              </Button>
            </>
          }
        />
        <div className="flex flex-wrap items-center gap-2.5">
          <DeptCurriculumFilters
            departments={org.departments}
            curricula={org.curricula}
            departmentId={departmentId}
            curriculumId={curriculumId}
            onDepartmentChange={filters.setDepartment}
            onCurriculumChange={filters.setCurriculum}
          />
          <YearSelect value={year} years={filters.years} onChange={filters.setYear} />
        </div>
      </div>

      <FilterToolbar
        search={filters.search}
        onSearch={filters.setSearch}
        placeholder="Search course ID, name, department, curriculum..."
        summary={
          <>
            Showing <Count>{rows.length}</Count> {plural(rows.length, 'course')} in <span className="font-semibold text-indigo-600">{year ?? '…'}</span>
          </>
        }
        filters={orgPills(filters, org.departments, org.curricula)}
        onClearAll={filters.clearAll}
      />

      {offerings.error && <Alert>{offerings.error.message}</Alert>}
      {offerings.isPending && params !== null ? (
        <PageSpinner />
      ) : (
        year !== undefined && (
          <OfferingTable
            rows={rows}
            year={year}
            sort={sort}
            selected={visibleSelected}
            expanded={expanded}
            semesters={filters.semesters}
            onSemestersChange={filters.setSemesters}
            onToggleSelect={(id) => setSelected((prev) => toggle(prev, id))}
            onToggleSelectAll={() => setSelected(visibleSelected.length === rows.length ? [] : rows.map((o) => o.id))}
            onToggleExpand={(id) => setExpanded((prev) => toggle(prev, id))}
            onEdit={(offering) => setDialog({ kind: 'form', offering })}
            onAssign={(offering, entries) => setDialog({ kind: 'assign', offering, entries })}
            onEditAccess={(offering, entry) => setDialog({ kind: 'edit-access', offering, entry })}
          />
        )
      )}

      {dialog?.kind === 'form' && year !== undefined && (
        <OfferingFormModal
          key={dialog.offering?.id ?? 'new'}
          offering={dialog.offering}
          departments={org.departments}
          curricula={org.curricula}
          years={filters.years}
          defaults={{ departmentId, curriculumId, year, semester: filters.semesters.length === 1 ? filters.semesters[0] : null }}
          onClose={closeDialog}
          onDelete={(offering) => setDialog({ kind: 'delete', offerings: [offering] })}
          onSwitchToImport={() => setDialog({ kind: 'import' })}
        />
      )}
      {dialog?.kind === 'import' && year !== undefined && (
        <ImportMappingModal curriculum={curriculum} year={year} onClose={closeDialog} onSwitchToSingle={() => setDialog({ kind: 'form', offering: null })} />
      )}
      {dialog?.kind === 'delete' && (
        <ConfirmDeleteModal
          offerings={dialog.offerings}
          onClose={closeDialog}
          onDeleted={(ids) => {
            setSelected((prev) => prev.filter((id) => !ids.includes(id)))
            setExpanded((prev) => prev.filter((id) => !ids.includes(id)))
          }}
        />
      )}
      {dialog?.kind === 'assign' && <AssignUserModal offering={dialog.offering} entries={dialog.entries} onClose={closeDialog} />}
      {dialog?.kind === 'edit-access' && <EditAccessModal offering={dialog.offering} entry={dialog.entry} onClose={closeDialog} />}
    </div>
  )
}
