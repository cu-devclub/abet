import { Download, FileText } from 'lucide-react'
import { Button } from '../../../components/ui/Button'
import { Modal } from '../../../components/ui/Modal'
import { formatBytes } from '../../../lib/format'
import type { Curriculum } from '../../curricula/types'
import { coverUrl, useDownloadCover } from '../api'

interface CoverPreviewModalProps {
  curriculum: Curriculum
  onClose: () => void
}

export function CoverPreviewModal({ curriculum, onClose }: CoverPreviewModalProps) {
  const download = useDownloadCover()
  const cover = curriculum.cover

  return (
    <Modal
      isOpen
      onClose={onClose}
      title={`Cover Sheet Preview - ${curriculum.name}`}
      maxWidth="max-w-3xl"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Close Preview
          </Button>
          <Button icon={Download} loading={download.isPending} onClick={() => download.mutate(curriculum)}>
            Download PDF
          </Button>
        </>
      }
    >
      {cover && (
        <div className="space-y-4">
          <div className="flex items-center justify-between bg-gray-900 text-white px-4 py-2 rounded-lg text-xs font-mono">
            <div className="flex items-center gap-2 min-w-0">
              <FileText size={14} className="text-rose-400 shrink-0" />
              <span className="font-semibold truncate max-w-[360px]">{cover.filename}</span>
            </div>
            <span className="text-gray-300 shrink-0">{formatBytes(cover.size_bytes)}</span>
          </div>
          <iframe
            title={`Cover PDF of ${curriculum.name}`}
            src={coverUrl(curriculum.id, cover.updated_at)}
            className="w-full h-[60vh] rounded-xl border border-gray-300 bg-gray-200"
          />
        </div>
      )}
    </Modal>
  )
}
