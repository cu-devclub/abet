import { useState } from 'react'
import { FileText, Trash2 } from 'lucide-react'
import { Alert } from '../../../components/ui/Alert'
import { Button } from '../../../components/ui/Button'
import { FileDropzone } from '../../../components/ui/FileDropzone'
import { Field, NativeSelect } from '../../../components/ui/Field'
import { Modal } from '../../../components/ui/Modal'
import { ProgressBar } from '../../../components/ui/ProgressBar'
import { useToast } from '../../../components/ui/useToast'
import { errorText } from '../../../app/queryClient'
import { formatBytes } from '../../../lib/format'
import type { Curriculum } from '../../curricula/types'
import { useUploadCover } from '../api'
import { useAppConfig } from '../../config/api'
import { validateCover } from '../validateCover'

interface UploadCoverModalProps {
  curricula: Curriculum[]
  /** Set when replacing/uploading from a row action; otherwise the target is selectable. */
  fixed: Curriculum | null
  onClose: () => void
}

export function UploadCoverModal({ curricula, fixed, onClose }: UploadCoverModalProps) {
  const toast = useToast()
  const upload = useUploadCover()
  const limits = useAppConfig()
  const [targetId, setTargetId] = useState(fixed?.id ?? curricula[0]?.id ?? '')
  const [file, setFile] = useState<File | null>(null)
  const [error, setError] = useState('')
  const [progress, setProgress] = useState(0)

  const target = fixed ?? curricula.find((c) => c.id === targetId) ?? null

  const pick = (files: File[]) => {
    const picked = files[0]
    if (!picked) return
    const problem = validateCover(picked, limits.coverBytes)
    setError(problem ?? '')
    if (!problem) setFile(picked)
  }

  const save = async () => {
    if (!target) {
      setError('Please select a curriculum.')
      return
    }
    if (!file) {
      setError('Please upload a PDF cover sheet file.')
      return
    }
    setError('')
    setProgress(0)
    try {
      await upload.mutateAsync({ curriculumId: target.id, file, onProgress: setProgress })
      toast.success(`Cover PDF saved for ${target.name}.`)
      onClose()
    } catch (e) {
      setError(errorText(e))
    }
  }

  return (
    <Modal
      isOpen
      onClose={onClose}
      title={fixed ? `Upload Cover PDF - ${fixed.name}` : 'Upload Curriculum Cover PDF'}
      maxWidth="max-w-xl"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button onClick={() => void save()} disabled={!file} loading={upload.isPending}>
            Save Cover
          </Button>
        </>
      }
    >
      <div className="space-y-4">
        {error && <Alert>{error}</Alert>}

        <div>
          {fixed ? (
            <>
              <span className="block text-sm font-medium text-gray-700 mb-1">Target Curriculum</span>
              <div className="w-full border border-gray-200 bg-gray-50 rounded-lg px-3.5 py-2.5 text-sm text-gray-900 font-medium flex items-center justify-between shadow-2xs">
                <div className="flex items-center gap-2 min-w-0">
                  <span className="font-mono bg-indigo-100 text-indigo-700 text-xs px-2.5 py-0.5 rounded font-bold uppercase shrink-0">{fixed.code}</span>
                  <span className="truncate">{fixed.name}</span>
                </div>
                <span className="text-xs text-gray-500 bg-gray-200/70 px-2 py-0.5 rounded shrink-0 ml-2">{fixed.department.name}</span>
              </div>
            </>
          ) : (
            <Field label="Target Curriculum">
              <NativeSelect value={targetId} onChange={(e) => setTargetId(e.target.value)}>
                {curricula.map((curr) => (
                  <option key={curr.id} value={curr.id}>
                    [{curr.code}] {curr.name} — {curr.department.name}
                  </option>
                ))}
              </NativeSelect>
            </Field>
          )}
        </div>

        {target && (
          <Alert variant="info">
            This cover will be automatically shared with all courses under <strong>{target.name}</strong>.
          </Alert>
        )}

        <div>
          <span className="block text-sm font-medium text-gray-700 mb-1">Course Cover PDF (cover.pdf)</span>
          {file ? (
            <div className="flex items-center justify-between p-3 bg-rose-50/50 border border-rose-200 rounded-xl">
              <div className="flex items-center gap-3 min-w-0">
                <div className="p-2 bg-rose-100 text-rose-700 rounded-lg shrink-0">
                  <FileText size={20} />
                </div>
                <div className="min-w-0">
                  <p className="text-sm font-semibold text-gray-900 truncate">{file.name}</p>
                  <p className="text-xs text-gray-500">{formatBytes(file.size)} · Ready as curriculum cover</p>
                </div>
              </div>
              <button
                type="button"
                onClick={() => setFile(null)}
                disabled={upload.isPending}
                aria-label="Remove selected file"
                className="p-1 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded-lg transition-colors cursor-pointer shrink-0 ml-2"
              >
                <Trash2 size={16} />
              </button>
            </div>
          ) : (
            <FileDropzone
              accept=".pdf,application/pdf"
              onFiles={pick}
              inputLabel="Cover PDF"
              title={
                <>
                  <span className="text-indigo-600 font-semibold">Click to upload</span> or drag and drop{' '}
                  <span className="font-mono text-gray-600 font-medium">cover.pdf</span>
                </>
              }
              hint={`Standard A4 course cover sheet (PDF only, max ${limits.config.max_cover_mb} MB)`}
            />
          )}
        </div>

        {upload.isPending && <ProgressBar percent={progress} label="Uploading cover" />}
      </div>
    </Modal>
  )
}
