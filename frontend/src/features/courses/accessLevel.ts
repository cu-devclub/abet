import type { AccessInput } from './types'

export type AccessLevel = 'both' | 'download' | 'upload'

export const ACCESS_LEVEL_LABELS: Record<AccessLevel, string> = {
  both: 'Upload & Download',
  download: 'Download',
  upload: 'Upload',
}

export function accessLevelOf(entry: AccessInput): AccessLevel {
  if (entry.can_upload && entry.can_download) return 'both'
  return entry.can_upload ? 'upload' : 'download'
}

export function accessInputOf(level: AccessLevel): AccessInput {
  return { can_upload: level !== 'download', can_download: level !== 'upload' }
}
