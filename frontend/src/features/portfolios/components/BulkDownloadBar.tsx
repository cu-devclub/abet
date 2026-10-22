import { Download } from 'lucide-react'
import { Button } from '../../../components/ui/Button'

interface BulkDownloadBarProps {
  count: number
  loading: boolean
  onDownload: () => void
}

/** "Download Selected (n)" action; only rendered while rows are selected. */
export function BulkDownloadBar({ count, loading, onDownload }: BulkDownloadBarProps) {
  return (
    <Button icon={Download} loading={loading} onClick={onDownload} title={`Download generated portfolio PDFs for ${count} selected course(s)`}>
      Download Selected
      <span className="ml-2 bg-indigo-500 text-white text-xs px-2 py-0.5 rounded-full font-bold">{count}</span>
    </Button>
  )
}
