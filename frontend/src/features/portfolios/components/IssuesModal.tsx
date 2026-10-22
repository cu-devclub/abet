import { useState } from 'react'
import { AlertCircle, AlertTriangle, ChevronRight, UploadCloud } from 'lucide-react'
import { Button } from '../../../components/ui/Button'
import { Alert } from '../../../components/ui/Alert'
import { Modal } from '../../../components/ui/Modal'
import { PageSpinner } from '../../../components/ui/Spinner'
import { cn } from '../../../lib/cn'
import { formatDateTime, plural } from '../../../lib/format'
import { useSubmission } from '../api'
import { groupIssues } from '../issues'
import type { Issue, IssueSeverity, IssueStage } from '../types'

const STAGE_STYLES: Record<IssueStage, string> = {
  missing: 'bg-amber-100 text-amber-800 border-amber-200',
  structure: 'bg-purple-100 text-purple-800 border-purple-200',
  naming: 'bg-blue-100 text-blue-800 border-blue-200',
  csv: 'bg-rose-100 text-rose-800 border-rose-200',
  build: 'bg-indigo-100 text-indigo-800 border-indigo-200',
}

const STAGE_LABELS: Record<IssueStage, string> = {
  missing: 'Missing',
  structure: 'Structure',
  naming: 'Naming',
  csv: 'CSV',
  build: 'Build',
}

const SEVERITY_TITLES: Record<IssueSeverity, string> = { error: 'Errors', warning: 'Warnings' }

interface IssuesModalProps {
  submissionId: string
  courseCode: string
  courseName: string
  canUpload: boolean
  onClose: () => void
  onReupload: () => void
}

function IssueRow({ issue, expanded, onToggle }: { issue: Issue; expanded: boolean; onToggle: () => void }) {
  return (
    <div className="border border-gray-200 rounded-xl overflow-hidden bg-white shadow-sm">
      <button
        type="button"
        onClick={onToggle}
        aria-expanded={expanded}
        className="w-full flex items-center justify-between p-3.5 cursor-pointer hover:bg-gray-50/80 transition-colors gap-3 text-left"
      >
        <span className="flex items-center gap-2.5 min-w-0 flex-1">
          <ChevronRight size={16} className={cn('text-gray-400 transition-transform duration-150 shrink-0', expanded && 'rotate-90 text-indigo-600')} />
          <span className={cn('px-2 py-0.5 rounded text-[11px] font-semibold border shrink-0', STAGE_STYLES[issue.stage] ?? 'bg-gray-100 text-gray-700')}>
            {STAGE_LABELS[issue.stage] ?? issue.stage}
          </span>
          <span className="font-mono text-xs text-gray-500 shrink-0 font-medium">{issue.code}</span>
          <span className="text-sm font-semibold text-gray-900 truncate">{issue.title}</span>
        </span>
        <span className="text-xs text-indigo-600 font-medium shrink-0">{expanded ? 'Hide' : 'Details'}</span>
      </button>

      {expanded && (
        <div className="p-4 pt-2 border-t border-gray-100 bg-gray-50/60 space-y-3">
          <div>
            <span className="text-xs font-semibold text-gray-700 uppercase tracking-wide">Issue Description</span>
            <p className="text-sm text-gray-800 mt-1 leading-relaxed">{issue.description}</p>
          </div>
          {issue.location && (
            <div className="flex items-center gap-2 text-xs text-gray-600">
              <span className="font-semibold text-gray-700">Location:</span>
              <span className="bg-gray-200/80 text-gray-800 font-mono px-2 py-0.5 rounded text-[11px] break-all">{issue.location}</span>
            </div>
          )}
          {issue.suggested_fix && (
            <div className="bg-emerald-50 border border-emerald-200/80 rounded-lg p-3 text-xs text-emerald-900">
              <span className="font-bold flex items-center gap-1.5 mb-1 text-emerald-800">💡 Suggested Fix:</span>
              <p className="leading-relaxed text-emerald-800">{issue.suggested_fix}</p>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

export function IssuesModal({ submissionId, courseCode, courseName, canUpload, onClose, onReupload }: IssuesModalProps) {
  const { data: submission, isPending, error } = useSubmission(submissionId)
  const [expanded, setExpanded] = useState<string[]>([])

  const issues = submission?.issues ?? []
  const groups = groupIssues(issues)
  const allKeys = groups.flatMap((g) => g.items.map((i) => i.key))
  const allExpanded = allKeys.length > 0 && expanded.length === allKeys.length
  const toggle = (key: string) => setExpanded((prev) => (prev.includes(key) ? prev.filter((k) => k !== key) : [...prev, key]))
  const hasErrors = (submission?.error_count ?? 0) > 0

  return (
    <Modal
      isOpen
      onClose={onClose}
      title="Processing & Validation Errors"
      maxWidth="max-w-2xl"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Close
          </Button>
          {canUpload && (
            <Button icon={UploadCloud} onClick={onReupload}>
              Re-upload Files
            </Button>
          )}
        </>
      }
    >
      {isPending && <PageSpinner />}
      {error && <Alert>{error.message}</Alert>}
      {submission && (
        <div className="space-y-4">
          <div className={cn('border rounded-xl p-4', hasErrors ? 'bg-red-50 border-red-200' : 'bg-amber-50 border-amber-200')}>
            <div className="flex items-start gap-3">
              <div className={cn('p-2 rounded-lg shrink-0', hasErrors ? 'bg-red-100 text-red-700' : 'bg-amber-100 text-amber-700')}>
                {hasErrors ? <AlertCircle size={20} /> : <AlertTriangle size={20} />}
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <h4 className={cn('text-sm font-bold', hasErrors ? 'text-red-900' : 'text-amber-900')}>
                    {courseCode}: {courseName}
                  </h4>
                  <span className="inline-flex items-center gap-1.5 text-xs font-bold">
                    <span className="px-2 py-0.5 rounded-full bg-red-200 text-red-800">
                      {submission.error_count} {plural(submission.error_count, 'Error')}
                    </span>
                    {submission.warning_count > 0 && (
                      <span className="px-2 py-0.5 rounded-full bg-amber-200 text-amber-800">
                        {submission.warning_count} {plural(submission.warning_count, 'Warning')}
                      </span>
                    )}
                  </span>
                </div>
                <p className={cn('text-xs mt-1', hasErrors ? 'text-red-700' : 'text-amber-800')}>
                  File: <span className="font-semibold">{submission.original_filename}</span> • Processed:{' '}
                  {formatDateTime(submission.completed_at ?? submission.created_at)}
                </p>
                <p className={cn('text-xs mt-1', hasErrors ? 'text-red-600' : 'text-amber-700')}>
                  {hasErrors
                    ? 'The portfolio could not be generated. Click each row below to expand details and suggestions.'
                    : 'Review the warnings below. Click each row to expand details.'}
                </p>
              </div>
            </div>
          </div>

          {allKeys.length > 0 ? (
            <>
              <div className="flex items-center justify-between px-1">
                <span className="text-xs font-semibold text-gray-500 uppercase tracking-wider">Issue Breakdown ({allKeys.length})</span>
                <button
                  type="button"
                  onClick={() => setExpanded(allExpanded ? [] : allKeys)}
                  className="text-xs font-medium text-indigo-600 hover:text-indigo-800 transition-colors cursor-pointer"
                >
                  {allExpanded ? 'Collapse All' : 'Expand All'}
                </button>
              </div>
              {groups.map((group) => (
                <section key={group.severity} aria-label={SEVERITY_TITLES[group.severity]} className="space-y-2.5">
                  <h5 className="text-[11px] font-bold uppercase tracking-wider text-gray-500 px-1">
                    {SEVERITY_TITLES[group.severity]} ({group.items.length})
                  </h5>
                  {group.items.map(({ issue, key }) => (
                    <IssueRow key={key} issue={issue} expanded={expanded.includes(key)} onToggle={() => toggle(key)} />
                  ))}
                </section>
              ))}
            </>
          ) : (
            <div className="p-4 text-center text-sm text-gray-500 bg-gray-50 rounded-xl border border-gray-200 italic">
              An unexpected error occurred during processing. Please review your files and re-upload.
            </div>
          )}
        </div>
      )}
    </Modal>
  )
}
