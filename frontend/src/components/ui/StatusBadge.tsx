import { useEffect, useState } from 'react'
import { AlertCircle } from 'lucide-react'
import type { SubmissionStatus } from '../../features/portfolios/types'
import { statusLabel } from '../../features/portfolios/rowStatus'
import { elapsedText } from '../../lib/format'
import { Badge } from './Badge'

interface StatusBadgeProps {
  status: SubmissionStatus | null
  errorCount?: number
  onFailedClick?: () => void
  /** Submission created_at; queued/processing badges show the elapsed time since then. */
  createdAt?: string
}

/** Re-renders every 30 s while `active` so the elapsed time advances even if a poll returns identical data. */
function useTick(active: boolean) {
  const [, setTick] = useState(0)
  useEffect(() => {
    if (!active) return
    const id = setInterval(() => setTick((n) => n + 1), 30_000)
    return () => clearInterval(id)
  }, [active])
}

export function StatusBadge({ status, errorCount = 0, onFailedClick, createdAt }: StatusBadgeProps) {
  useTick(status === 'queued' || status === 'processing')
  if (status === null) return <span className="text-gray-400 text-xs italic">No Files</span>
  if (status === 'invalid' || status === 'failed') {
    return (
      <Badge variant="failed" onClick={onFailedClick} title="Click to view processing errors">
        <AlertCircle size={12} className="mr-1 inline text-red-600 shrink-0" />
        <span>Failed</span>
        {errorCount > 0 && (
          <span className="ml-1 text-[10px] text-red-700 font-semibold bg-red-100/90 px-1.5 rounded-full">{errorCount}</span>
        )}
      </Badge>
    )
  }
  if (status === 'generated') return <Badge variant="generated">{statusLabel(status)}</Badge>
  const elapsed = createdAt ? elapsedText(createdAt) : ''
  return <Badge variant="processing">{elapsed ? `${statusLabel(status)} · ${elapsed}` : statusLabel(status)}</Badge>
}
