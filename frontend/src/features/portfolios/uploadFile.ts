import { formatBytes } from '../../lib/format'

/** Client-side checks for a portfolio zip before upload (server re-validates). */
export function validatePortfolioZip(file: Pick<File, 'name' | 'size'>, maxBytes: number): string | null {
  if (!file.name.toLowerCase().endsWith('.zip')) return 'Please drop a folder or a single .zip file.'
  if (file.size > maxBytes) {
    return `This upload is ${formatBytes(file.size)}, which is over the ${formatBytes(maxBytes).replace('.0', '')} limit. Remove large files and try again.`
  }
  return null
}
