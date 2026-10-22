import { useQuery } from '@tanstack/react-query'
import { apiGet } from '../../lib/http'

const MB = 1024 * 1024

export interface AppConfig {
  max_upload_mb: number
  max_cover_mb: number
  max_import_mb: number
  build_timeout_minutes: number
}

/** Used only while GET /config is loading or failed; the server values (.env) are authoritative. */
export const DEFAULT_APP_CONFIG: AppConfig = {
  max_upload_mb: 300,
  max_cover_mb: 20,
  max_import_mb: 5,
  build_timeout_minutes: 60,
}

export interface Limits {
  uploadBytes: number
  coverBytes: number
  importBytes: number
  config: AppConfig
}

export function limitsFrom(config: AppConfig): Limits {
  return { uploadBytes: config.max_upload_mb * MB, coverBytes: config.max_cover_mb * MB, importBytes: config.max_import_mb * MB, config }
}

export function useAppConfig(): Limits {
  const query = useQuery({
    queryKey: ['config'],
    queryFn: () => apiGet<AppConfig>('/config'),
    staleTime: Infinity,
    retry: 1,
  })
  return limitsFrom({ ...DEFAULT_APP_CONFIG, ...query.data })
}
