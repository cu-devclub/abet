import { useRef, useState, type ChangeEvent } from 'react'
import { Download, RotateCcw, Upload } from 'lucide-react'
import { PageHeader } from '../../components/layout/PageHeader'
import { Alert } from '../../components/ui/Alert'
import { Badge } from '../../components/ui/Badge'
import { Button } from '../../components/ui/Button'
import { ConfirmDialog } from '../../components/ui/ConfirmDialog'
import { Spinner } from '../../components/ui/Spinner'
import { useToast } from '../../components/ui/useToast'
import { formatBytes, formatDateTime } from '../../lib/format'
import { useDownloadTemplate } from '../portfolios/api'
import { useCourseTemplate, useRestoreCourseTemplate, useUploadCourseTemplate } from './api'

const MAX_MB = 20

export function ConfigPage() {
  const toast = useToast()
  const query = useCourseTemplate()
  const upload = useUploadCourseTemplate()
  const restore = useRestoreCourseTemplate()
  const download = useDownloadTemplate()
  const input = useRef<HTMLInputElement>(null)
  const [error, setError] = useState<string | null>(null)
  const [confirming, setConfirming] = useState(false)
  const status = query.data
  const custom = status?.source === 'custom'

  function onPick(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    event.target.value = ''
    if (!file) return
    setError(null)
    if (!file.name.toLowerCase().endsWith('.zip')) return setError('The template must be a .zip file.')
    if (file.size > MAX_MB * 1024 * 1024) return setError(`The file is too large (max ${MAX_MB} MB).`)
    upload.mutate(file, {
      onSuccess: () => toast.success('Course portfolio template updated.'),
      onError: (e) => setError(e.message),
    })
  }

  return (
    <div className="p-6 space-y-5 w-full">
      <PageHeader title="System Config" subtitle="Settings that apply to the whole system. Super admin only." />
      <section className="bg-white border border-gray-200 rounded-xl shadow-sm p-6 space-y-4">
        <div className="flex flex-wrap items-center gap-3">
          <h2 className="text-lg font-semibold text-gray-900">Course portfolio template</h2>
          {status && <Badge variant={custom ? 'custom' : 'staff'}>{custom ? 'Custom' : 'Default'}</Badge>}
        </div>
        {query.isLoading && <Spinner />}
        {query.isError && <Alert>Could not load the template status.</Alert>}
        {custom && (
          <p className="text-sm text-gray-600">
            Uploaded {formatDateTime(status.uploaded_at)} by {status.uploaded_by} · {formatBytes(status.size_bytes ?? 0)}
          </p>
        )}
        <p className="text-sm text-gray-500">
          The folder and CSV templates instructors fill in before uploading a course portfolio. Users get it from Download Template on
          Course Portfolios and in the upload dialog. The zip must contain the 4 core CSV templates.
        </p>
        {error && <Alert>{error}</Alert>}
        <div className="flex flex-wrap gap-3">
          <Button
            variant="secondary"
            icon={Download}
            loading={download.isPending}
            onClick={() => download.mutate('instructor-package.zip', { onSuccess: () => toast.success('Downloaded course-template.zip.') })}
          >
            Download current
          </Button>
          <Button icon={Upload} loading={upload.isPending} onClick={() => input.current?.click()}>
            Upload new template
          </Button>
          {custom && (
            <Button variant="dangerSoft" icon={RotateCcw} onClick={() => setConfirming(true)}>
              Restore default
            </Button>
          )}
          <input ref={input} type="file" accept=".zip" className="hidden" aria-label="Template zip file" onChange={onPick} />
        </div>
      </section>
      <ConfirmDialog
        isOpen={confirming}
        title="Restore default template"
        confirmLabel="Restore default"
        loading={restore.isPending}
        onClose={() => setConfirming(false)}
        onConfirm={() =>
          restore.mutate(undefined, {
            onSuccess: () => {
              setConfirming(false)
              setError(null)
              toast.success('Restored the default course portfolio template.')
            },
          })
        }
      >
        The custom template will be removed and everyone will download the built-in default again.
      </ConfirmDialog>
    </div>
  )
}
