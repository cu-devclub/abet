import { formatBytes } from '../../lib/format'

/** Client-side cover checks (the server re-validates: `%PDF-` magic bytes, MAX_COVER_MB). */
export function validateCover(file: Pick<File, 'name' | 'type' | 'size'>, maxBytes: number): string | null {
  if (!file.name.toLowerCase().endsWith('.pdf') && file.type !== 'application/pdf') return 'Please select a valid PDF file (.pdf)'
  if (file.size > maxBytes) {
    return `This file is ${formatBytes(file.size)}. Cover PDFs can be at most ${formatBytes(maxBytes)}.`
  }
  return null
}
