import { useState, type ReactNode } from 'react'
import { FileSpreadsheet, Trash2 } from 'lucide-react'
import { Alert } from '../../../components/ui/Alert'
import { Button } from '../../../components/ui/Button'
import { FileDropzone } from '../../../components/ui/FileDropzone'
import { Modal } from '../../../components/ui/Modal'
import { Tabs } from '../../../components/ui/Tabs'
import { useToast } from '../../../components/ui/useToast'
import { errorText } from '../../../app/queryClient'
import { cn } from '../../../lib/cn'
import { formatBytes, roleLabel, semesterText } from '../../../lib/format'
import { ApiError } from '../../../lib/http'
import { useAppConfig } from '../../config/api'
import { useDownloadTemplate } from '../../portfolios/api'
import type { Curriculum } from '../../curricula/types'
import { ADD_COURSE_TABS } from '../addCourseTabs'
import { useApplyImport, usePreviewImport } from '../api'
import { ACCESS_ACTION_LABELS, COURSE_ACTION_LABELS, isImportPlan, summaryItems, USER_ACTION_LABELS } from '../importPlan'
import type { ImportPlan } from '../types'


const TONES = {
  green: 'bg-emerald-50 text-emerald-800 border-emerald-200',
  blue: 'bg-blue-50 text-blue-800 border-blue-200',
  gray: 'bg-gray-50 text-gray-700 border-gray-200',
  red: 'bg-red-50 text-red-800 border-red-200',
  amber: 'bg-amber-50 text-amber-800 border-amber-200',
}

const ACTION_TONES: Record<string, string> = {
  create: TONES.green,
  add: TONES.green,
  update: TONES.blue,
  raise_role: TONES.blue,
  keep: TONES.gray,
  delete: TONES.red,
  remove: TONES.red,
  keep_has_uploads: TONES.amber,
}

function ActionChip({ action, label }: { action: string; label: string }) {
  return <span className={cn('inline-block px-2 py-0.5 rounded-full border text-[10px] font-semibold whitespace-nowrap', ACTION_TONES[action])}>{label}</span>
}

function PlanSection({ title, count, children }: { title: string; count: number; children: ReactNode }) {
  if (count === 0) return null
  return (
    <section className="space-y-1.5">
      <h5 className="text-[11px] font-bold uppercase tracking-wider text-gray-500">
        {title} ({count})
      </h5>
      <div className="border border-gray-200 rounded-lg overflow-hidden max-h-48 overflow-y-auto">
        <table className="min-w-full text-xs divide-y divide-gray-100">{children}</table>
      </div>
    </section>
  )
}

const CELL = 'px-3 py-1.5 whitespace-nowrap'
const HEAD = 'px-3 py-1.5 text-left text-[10px] uppercase tracking-wider text-gray-500 bg-gray-50 sticky top-0'

function PlanView({ plan }: { plan: ImportPlan }) {
  return (
    <div className="space-y-4">
      <div className="flex flex-wrap gap-2">
        {summaryItems(plan.summary).map((item) => (
          <span key={item.key} className={cn('px-2.5 py-1 rounded-lg border text-xs font-medium', TONES[item.tone])}>
            <strong className="mr-1">{item.value}</strong>
            {item.label}
          </span>
        ))}
      </div>

      {plan.errors.length > 0 && (
        <Alert>
          <p className="font-semibold mb-1">
            {plan.errors.length} error{plan.errors.length === 1 ? '' : 's'} found — fix the CSV and choose it again. Nothing will be applied until the file is valid.
          </p>
          <ul className="list-disc list-inside text-xs space-y-0.5 max-h-32 overflow-y-auto">
            {plan.errors.map((err, index) => (
              <li key={`${err.row}-${index}`}>
                <strong>Row {err.row}:</strong> {err.message}
              </li>
            ))}
          </ul>
        </Alert>
      )}

      <PlanSection title="Courses" count={plan.courses.length}>
        <thead>
          <tr>
            <th className={HEAD}>Action</th>
            <th className={HEAD}>Course ID</th>
            <th className={HEAD}>Name</th>
            <th className={HEAD}>Semester</th>
            <th className={HEAD}>Note</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-100">
          {plan.courses.map((c, i) => (
            <tr key={`${c.course_code}-${c.semester}-${i}`}>
              <td className={CELL}>
                <ActionChip action={c.action} label={COURSE_ACTION_LABELS[c.action]} />
              </td>
              <td className={cn(CELL, 'font-mono')}>{c.course_code}</td>
              <td className="px-3 py-1.5">{c.course_name}</td>
              <td className={CELL}>{semesterText(c.semester)}</td>
              <td className="px-3 py-1.5 text-gray-500">{c.note}</td>
            </tr>
          ))}
        </tbody>
      </PlanSection>

      <PlanSection title="Access changes" count={plan.access.length}>
        <thead>
          <tr>
            <th className={HEAD}>Action</th>
            <th className={HEAD}>Course</th>
            <th className={HEAD}>User</th>
            <th className={HEAD}>Role</th>
            <th className={HEAD}>Permission</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-100">
          {plan.access.map((a, i) => (
            <tr key={`${a.course_code}-${a.semester}-${a.email}-${i}`}>
              <td className={CELL}>
                <ActionChip action={a.action} label={ACCESS_ACTION_LABELS[a.action]} />
              </td>
              <td className={CELL}>
                <span className="font-mono">{a.course_code}</span> · {semesterText(a.semester)}
              </td>
              <td className={cn(CELL, 'font-mono')}>{a.email}</td>
              <td className={cn(CELL, 'capitalize')}>{roleLabel(a.role)}</td>
              <td className={CELL}>
                {a.can_upload && a.can_download ? 'Upload & Download' : a.can_upload ? 'Upload' : 'Download'}
              </td>
            </tr>
          ))}
        </tbody>
      </PlanSection>

      <PlanSection title="Users" count={plan.users.length}>
        <thead>
          <tr>
            <th className={HEAD}>Action</th>
            <th className={HEAD}>Email</th>
            <th className={HEAD}>Role</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-100">
          {plan.users.map((u, i) => (
            <tr key={`${u.email}-${i}`}>
              <td className={CELL}>
                <ActionChip action={u.action} label={USER_ACTION_LABELS[u.action]} />
              </td>
              <td className={cn(CELL, 'font-mono')}>{u.email}</td>
              <td className={cn(CELL, 'capitalize')}>{roleLabel(u.role)}</td>
            </tr>
          ))}
        </tbody>
      </PlanSection>
    </div>
  )
}

interface ImportMappingModalProps {
  /** Curriculum selected in the page filters; required to import. */
  curriculum: Curriculum | null
  year: number
  onClose: () => void
  onSwitchToSingle: () => void
}

export function ImportMappingModal({ curriculum, year, onClose, onSwitchToSingle }: ImportMappingModalProps) {
  const toast = useToast()
  const preview = usePreviewImport()
  const limits = useAppConfig()
  const apply = useApplyImport()
  const template = useDownloadTemplate()
  const [file, setFile] = useState<File | null>(null)
  const [plan, setPlan] = useState<ImportPlan | null>(null)
  const [error, setError] = useState('')

  const request = (f: File) => (curriculum ? { file: f, curriculumId: curriculum.id, academicYear: year } : null)

  /** A 422 carries the ImportPlan (row errors); show it instead of a bare message. */
  const handleFailure = (e: unknown) => {
    if (e instanceof ApiError && isImportPlan(e.body)) setPlan(e.body)
    else setError(errorText(e))
  }

  const choose = (files: File[]) => {
    const picked = files[0]
    const req = picked && request(picked)
    if (!picked || !req) return
    setPlan(null)
    setError('')
    if (!picked.name.toLowerCase().endsWith('.csv')) {
      setError('Please choose a .csv file.')
      return
    }
    if (picked.size > limits.importBytes) {
      setError(`This file is ${formatBytes(picked.size)}. The CSV can be at most ${limits.config.max_import_mb} MB.`)
      return
    }
    setFile(picked)
    preview.mutate(req, { onSuccess: setPlan, onError: handleFailure })
  }

  const clear = () => {
    setFile(null)
    setPlan(null)
    setError('')
  }

  const applyPlan = () => {
    const req = file && request(file)
    if (!req) return
    setError('')
    apply.mutate(req, {
      onSuccess: () => {
        toast.success('Import applied.')
        onClose()
      },
      onError: handleFailure,
    })
  }

  const canApply = Boolean(plan && plan.errors.length === 0 && !preview.isPending)

  return (
    <Modal
      isOpen
      onClose={onClose}
      title="Add New Course"
      maxWidth="max-w-4xl"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button onClick={applyPlan} disabled={!canApply} loading={apply.isPending}>
            Apply
          </Button>
        </>
      }
    >
      <div className="space-y-4">
        <Tabs
          label="Creation method"
          tabs={ADD_COURSE_TABS}
          active="csv"
          onChange={(tab) => {
            if (tab === 'single') onSwitchToSingle()
          }}
        />

        {curriculum ? (
          <div className="bg-indigo-50/70 border border-indigo-100 rounded-xl p-3.5 text-xs flex flex-wrap items-center justify-between gap-2">
            <div>
              <span className="text-gray-500">Importing into curriculum: </span>
              <span className="font-bold text-indigo-900 text-sm">
                {curriculum.code} — {curriculum.name}
              </span>
            </div>
            <div>
              <span className="text-gray-500">Academic year: </span>
              <span className="font-bold text-indigo-900 text-sm">{year}</span>
            </div>
          </div>
        ) : (
          <Alert variant="warning">Select a curriculum first. Use the Curriculum filter on the Courses page, then open this dialog again.</Alert>
        )}

        <Alert variant="info">
          One-shot batch for the selected curriculum and year: courses missing from the file are removed unless they already have uploads, and the course access
          of every course in the file mirrors the CSV.
        </Alert>

        {error && <Alert>{error}</Alert>}

        {file ? (
          <div className="flex items-center justify-between p-3 bg-emerald-50/40 border border-emerald-200 rounded-xl">
            <div className="flex items-center gap-3 min-w-0">
              <div className="p-2 bg-emerald-100 text-emerald-700 rounded-lg shrink-0">
                <FileSpreadsheet size={20} />
              </div>
              <div className="min-w-0">
                <p className="text-sm font-semibold text-gray-900 truncate">{file.name}</p>
                <p className="text-xs text-gray-500">
                  {formatBytes(file.size)} · {preview.isPending ? 'Checking file…' : 'Preview below'}
                </p>
              </div>
            </div>
            <button
              type="button"
              onClick={clear}
              disabled={apply.isPending}
              aria-label="Remove selected file"
              className="p-1 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded-lg transition-colors cursor-pointer shrink-0 ml-2"
            >
              <Trash2 size={16} />
            </button>
          </div>
        ) : (
          <FileDropzone
            accept=".csv,text/csv"
            disabled={!curriculum}
            inputLabel="Mapping CSV"
            onFiles={choose}
            title={
              <>
                <span className="text-indigo-600 font-semibold">Click to upload</span> or drag the course–instructor mapping CSV here
              </>
            }
            hint={`Columns: course id, course name, semester, user email, user role, upload permission, download permission (max ${limits.config.max_import_mb} MB)`}
          />
        )}

        <button
          type="button"
          onClick={() => template.mutate('course-instructor-mapping.csv')}
          disabled={template.isPending}
          className="text-xs font-medium text-indigo-600 hover:text-indigo-800 underline cursor-pointer disabled:opacity-50"
        >
          Download sample CSV template
        </button>

        {preview.isPending && <p className="text-sm text-gray-500">Checking the file…</p>}
        {plan && <PlanView plan={plan} />}
      </div>
    </Modal>
  )
}
