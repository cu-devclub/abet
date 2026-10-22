import { useState } from 'react'
import { FileArchive, Trash2 } from 'lucide-react'
import { Alert } from '../../../components/ui/Alert'
import { Button } from '../../../components/ui/Button'
import { FileDropzone } from '../../../components/ui/FileDropzone'
import { Modal } from '../../../components/ui/Modal'
import { ProgressBar } from '../../../components/ui/ProgressBar'
import { errorText } from '../../../app/queryClient'
import { formatBytes, semesterText } from '../../../lib/format'
import { droppedEntries, hasDroppedDirectory, zipFromEntries, zipFromFileList } from '../../../lib/zipFolder'
import { useDownloadTemplate, useUploadSubmission } from '../api'
import type { PortfolioRow, Submission } from '../types'
import { useAppConfig } from '../../config/api'
import { validatePortfolioZip } from '../uploadFile'

interface UploadPortfolioModalProps {
  row: PortfolioRow
  onClose: () => void
  onUploaded: (submission: Submission) => void
}

export function UploadPortfolioModal({ row, onClose, onUploaded }: UploadPortfolioModalProps) {
  const upload = useUploadSubmission()
  const limits = useAppConfig()
  const template = useDownloadTemplate()
  const [file, setFile] = useState<File | null>(null)
  const [zipping, setZipping] = useState<number | null>(null)
  const [uploadPercent, setUploadPercent] = useState(0)
  const [error, setError] = useState('')

  const accept = (candidate: File) => {
    const problem = validatePortfolioZip(candidate, limits.uploadBytes)
    setError(problem ?? '')
    setFile(problem ? null : candidate)
  }

  const zip = async (build: (onProgress: (percent: number) => void) => Promise<File>) => {
    setError('')
    setFile(null)
    setZipping(0)
    try {
      accept(await build(setZipping))
    } catch (e) {
      setError(errorText(e))
    } finally {
      setZipping(null)
    }
  }

  const handleDrop = (dt: DataTransfer) => {
    // Entries and files must be read synchronously: the DataTransfer is cleared after the event.
    const entries = droppedEntries(dt)
    const files = Array.from(dt.files)
    if (hasDroppedDirectory(entries)) void zip((onProgress) => zipFromEntries(entries, onProgress))
    else handleFiles(files)
  }

  function handleFiles(files: File[]) {
    if (files.length === 1) accept(files[0])
    else if (files.length > 1) setError('Please drop a single folder or one .zip file.')
  }

  const submit = async () => {
    if (!file) return
    setError('')
    setUploadPercent(0)
    try {
      const submission = await upload.mutateAsync({ offeringId: row.offering_id, file, onProgress: setUploadPercent })
      onUploaded(submission)
    } catch (e) {
      setError(errorText(e))
    }
  }

  const busy = zipping !== null || upload.isPending

  return (
    <Modal
      isOpen
      onClose={onClose}
      title="Upload Course Portfolio Folder"
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={upload.isPending}>
            Cancel
          </Button>
          <Button onClick={() => void submit()} disabled={!file || zipping !== null} loading={upload.isPending}>
            Submit & Process
          </Button>
        </>
      }
    >
      <div className="space-y-4">
        <div className="bg-indigo-50/70 border border-indigo-100 rounded-xl p-3.5 space-y-1.5 text-xs">
          <div className="flex items-start justify-between gap-3">
            <div className="min-w-0 flex-1">
              <span className="text-gray-500">Course: </span>
              <span className="font-bold text-indigo-900 text-sm">
                {row.course_code}: {row.course_name}
              </span>
            </div>
            <div className="shrink-0 text-right whitespace-nowrap">
              <span className="text-gray-500">Year: </span>
              <span className="font-bold text-indigo-900 text-sm">
                {row.academic_year} · {semesterText(row.semester)}
              </span>
            </div>
          </div>
          <div>
            <span className="text-gray-500">Department: </span>
            <span className="font-semibold text-gray-800">{row.department.name}</span>
          </div>
          <div>
            <span className="text-gray-500">Curriculum: </span>
            <span className="font-semibold text-gray-800">{row.curriculum.name}</span>
          </div>
        </div>

        {error && <Alert>{error}</Alert>}

        <div>
          <div className="flex items-center justify-between mb-2">
            <span className="block text-sm font-medium text-gray-700">Upload Folder</span>
            <button
              type="button"
              onClick={() => template.mutate('instructor-package.zip')}
              disabled={template.isPending}
              className="text-xs font-medium text-indigo-600 hover:text-indigo-800 underline cursor-pointer disabled:opacity-50"
            >
              Download folder template
            </button>
          </div>

          {file ? (
            <div className="flex items-center justify-between p-3 bg-indigo-50/50 border border-indigo-200 rounded-xl">
              <div className="flex items-center gap-3 min-w-0">
                <div className="p-2 bg-indigo-100 text-indigo-700 rounded-lg shrink-0">
                  <FileArchive size={20} />
                </div>
                <div className="min-w-0">
                  <p className="text-sm font-semibold text-gray-900 truncate">{file.name}</p>
                  <p className="text-xs text-gray-500">{formatBytes(file.size)} · Ready to upload</p>
                </div>
              </div>
              <button
                type="button"
                onClick={() => setFile(null)}
                disabled={busy}
                aria-label="Remove selected folder"
                className="p-1 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded-lg transition-colors cursor-pointer shrink-0 ml-2"
              >
                <Trash2 size={16} />
              </button>
            </div>
          ) : (
            <FileDropzone
              accept=".zip,application/zip,application/x-zip-compressed"
              disabled={busy}
              inputLabel="Portfolio .zip"
              onDrop={handleDrop}
              onFiles={handleFiles}
              onFolder={(files) => void zip((onProgress) => zipFromFileList(files, onProgress))}
              title={
                <>
                  <span className="text-indigo-600 font-medium">Drag and drop</span> your folder here
                </>
              }
              hint={`Or choose a folder / a .zip file (max ${limits.config.max_upload_mb} MB). Files will be processed into a Course Portfolio PDF.`}
            />
          )}
        </div>

        {zipping !== null && <ProgressBar percent={zipping} label="Compressing folder" />}
        {upload.isPending && <ProgressBar percent={uploadPercent} label="Uploading" />}
      </div>
    </Modal>
  )
}
