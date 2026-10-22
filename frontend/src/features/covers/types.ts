export type CoverStatus = 'has_cover' | 'no_cover'

export const COVER_STATUS_OPTIONS = [
  { value: 'has_cover', label: 'Cover Uploaded' },
  { value: 'no_cover', label: 'Missing Cover' },
] as const
