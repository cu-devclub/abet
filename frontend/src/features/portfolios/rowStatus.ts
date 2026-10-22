import type { PortfolioRow, SubmissionStatus } from './types'

/** Table / badge label: queued+processing → Processing, invalid+failed → Failed. */
export function statusLabel(status: SubmissionStatus | null): string {
  switch (status) {
    case 'generated':
      return 'Generated'
    case 'queued':
    case 'processing':
      return 'Processing'
    case 'invalid':
    case 'failed':
      return 'Failed'
    default:
      return 'No Files'
  }
}

export function latestUpdate(row: PortfolioRow): string | null {
  const sub = row.latest_submission
  return sub ? (sub.completed_at ?? sub.created_at) : null
}
