import { useMemo, useState } from 'react'
import { FolderDown } from 'lucide-react'
import { PageHeader } from '../../components/layout/PageHeader'
import { Alert } from '../../components/ui/Alert'
import { Button } from '../../components/ui/Button'
import { Count, FilterToolbar } from '../../components/ui/FilterToolbar'
import { PageSpinner } from '../../components/ui/Spinner'
import { useToast } from '../../components/ui/useToast'
import { plural, semesterText } from '../../lib/format'
import { sortRows, useSort } from '../../lib/useSort'
import { DeptCurriculumFilters } from '../shared/DeptCurriculumFilters'
import { orgPills } from '../shared/orgPills'
import { useOrgFilters } from '../shared/useOrgFilters'
import { useOrgOptions } from '../shared/useOrgOptions'
import { YearSelect } from '../shared/YearSelect'
import { useBulkDownload, useDownloadPortfolio, useDownloadTemplate, usePortfolios } from './api'
import { BulkDownloadBar } from './components/BulkDownloadBar'
import { IssuesModal } from './components/IssuesModal'
import { PortfolioTable, type PortfolioSortField } from './components/PortfolioTable'
import { UploadPortfolioModal } from './components/UploadPortfolioModal'
import { latestUpdate, statusLabel } from './rowStatus'
import type { PortfolioRow, Submission } from './types'

function sortValue(row: PortfolioRow, field: PortfolioSortField): string | number {
  switch (field) {
    case 'id':
      return row.course_code
    case 'name':
      return row.course_name
    case 'semester':
      return row.semester
    case 'status':
      return statusLabel(row.latest_submission?.status ?? null)
    case 'time':
      return latestUpdate(row) ?? ''
  }
}

function matchesSearch(row: PortfolioRow, query: string): boolean {
  const status = statusLabel(row.latest_submission?.status ?? null)
  return [row.course_code, row.course_name, row.department.name, row.curriculum.name, status].some((v) => v.toLowerCase().includes(query))
}

type IssuesTarget = { row: PortfolioRow; submissionId: string }

export function PortfoliosPage() {
  const toast = useToast()
  const template = useDownloadTemplate()
  const filters = useOrgFilters()
  const sort = useSort<PortfolioSortField>('id')
  const [selected, setSelected] = useState<string[]>([])
  const [uploadRow, setUploadRow] = useState<PortfolioRow | null>(null)
  const [issues, setIssues] = useState<IssuesTarget | null>(null)
  const download = useDownloadPortfolio()
  const bulk = useBulkDownload()

  const { year, departmentId, curriculumId } = filters
  const params = useMemo(
    () =>
      year === undefined
        ? null
        : { academic_year: year, department_id: departmentId || undefined, curriculum_id: curriculumId || undefined },
    [year, departmentId, curriculumId],
  )
  const portfolios = usePortfolios(params)
  const org = useOrgOptions(portfolios.data)

  const query = filters.search.trim().toLowerCase()
  const rows = useMemo(() => {
    const filtered = (portfolios.data ?? []).filter(
      (row) => (filters.semesters.length === 0 || filters.semesters.includes(row.semester)) && (!query || matchesSearch(row, query)),
    )
    return sortRows(filtered, (row) => sortValue(row, sort.field), sort.direction)
  }, [portfolios.data, filters.semesters, query, sort.field, sort.direction])

  // Only rows currently in view stay selected (filters/year changes drop the rest).
  const visibleSelected = selected.filter((id) => rows.some((row) => row.offering_id === id))
  const toggle = (id: string) => setSelected((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]))
  const toggleAll = () => setSelected(visibleSelected.length === rows.length ? [] : rows.map((row) => row.offering_id))

  const downloadSelected = () => {
    if (year === undefined) return
    const downloadable = rows.filter((row) => visibleSelected.includes(row.offering_id) && row.pdf_available && row.can_download)
    if (downloadable.length === 0) {
      toast.error(`None of the ${visibleSelected.length} selected courses have a generated portfolio PDF to download.`)
      return
    }
    bulk.mutate(
      { offeringIds: downloadable.map((row) => row.offering_id), year },
      {
        onSuccess: (name) => {
          const skipped = visibleSelected.length - downloadable.length
          if (skipped > 0) toast.info(`Downloaded ${downloadable.length} PDFs (${skipped} selected ${plural(skipped, 'course has', 'courses have')} no generated PDF).`)
          else toast.success(`Downloaded ${downloadable.length} portfolio ${plural(downloadable.length, 'PDF')} (${name}).`)
        },
      },
    )
  }

  const handleUploaded = (submission: Submission) => {
    const row = uploadRow
    setUploadRow(null)
    if (submission.status === 'invalid' && row) setIssues({ row, submissionId: submission.id })
    else toast.success('Uploaded — generating PDF')
  }

  const openIssues = (row: PortfolioRow) => {
    if (row.latest_submission) setIssues({ row, submissionId: row.latest_submission.id })
  }

  return (
    <div className="p-6 space-y-5 w-full">
      <div className="space-y-4">
        <PageHeader
          title="Course Portfolios"
          actions={
            <>
              {visibleSelected.length > 0 && (
                <BulkDownloadBar count={visibleSelected.length} loading={bulk.isPending} onDownload={downloadSelected} />
              )}
              <Button
                variant="secondary"
                icon={FolderDown}
                loading={template.isPending}
                title="The folder structure and CSV templates to fill in before uploading a course portfolio"
                onClick={() =>
                  template.mutate('instructor-package.zip', {
                    onSuccess: () => toast.success('Downloaded the course portfolio template (course-template.zip).'),
                  })
                }
              >
                Download Template
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
        placeholder="Search course ID, name, status..."
        summary={
          <>
            Showing <Count>{rows.length}</Count> {plural(rows.length, 'course')} in <span className="font-semibold text-indigo-600">{year ?? '…'}</span>
            {visibleSelected.length > 0 && (
              <span className="ml-1.5 text-indigo-600 font-semibold bg-indigo-50 px-2 py-0.5 rounded-full border border-indigo-200">
                {visibleSelected.length} selected
              </span>
            )}
          </>
        }
        filters={orgPills(filters, org.departments, org.curricula)}
        onClearAll={filters.clearAll}
      />

      {portfolios.error && <Alert>{portfolios.error.message}</Alert>}
      {portfolios.isPending && params !== null ? (
        <PageSpinner />
      ) : (
        year !== undefined && (
          <PortfolioTable
            rows={rows}
            year={year}
            sort={sort}
            selected={visibleSelected}
            semesters={filters.semesters}
            onSemestersChange={filters.setSemesters}
            onToggle={toggle}
            onToggleAll={toggleAll}
            onDownload={(row) => download.mutate(row)}
            onOpenIssues={openIssues}
            onUpload={setUploadRow}
          />
        )
      )}

      {uploadRow && <UploadPortfolioModal row={uploadRow} onClose={() => setUploadRow(null)} onUploaded={handleUploaded} />}
      {issues && (
        <IssuesModal
          submissionId={issues.submissionId}
          courseCode={issues.row.course_code}
          courseName={`${issues.row.course_name} (${semesterText(issues.row.semester)})`}
          canUpload={issues.row.can_upload}
          onClose={() => setIssues(null)}
          onReupload={() => {
            setUploadRow(issues.row)
            setIssues(null)
          }}
        />
      )}
    </div>
  )
}
