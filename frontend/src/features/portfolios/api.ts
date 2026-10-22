import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiGet, uploadWithProgress } from '../../lib/http'
import { downloadAndSave } from '../../lib/download'
import type { PortfolioListParams, PortfolioRow, Submission } from './types'

interface Items<T> {
  items: T[]
}

const POLL_MS = 5000

export function isInFlight(row: PortfolioRow): boolean {
  const status = row.latest_submission?.status
  return status === 'queued' || status === 'processing'
}

export function usePortfolios(params: PortfolioListParams | null) {
  return useQuery({
    queryKey: ['portfolios', params],
    queryFn: async () => (await apiGet<Items<PortfolioRow>>('/portfolios', { ...params })).items,
    enabled: params !== null,
    refetchInterval: (query) => (query.state.data?.some(isInFlight) ? POLL_MS : false),
  })
}

export function useSubmission(id: string | null) {
  return useQuery({
    queryKey: ['submission', id],
    queryFn: () => apiGet<Submission>(`/submissions/${id}`),
    enabled: id !== null,
  })
}

interface UploadVars {
  offeringId: string
  file: File
  onProgress?: (percent: number) => void
}

export function useUploadSubmission() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ offeringId, file, onProgress }: UploadVars) => {
      const form = new FormData()
      form.append('file', file)
      return uploadWithProgress<Submission>(`/offerings/${offeringId}/submissions`, form, onProgress)
    },
    onSuccess: async (submission) => {
      queryClient.setQueryData(['submission', submission.id], submission)
      await queryClient.invalidateQueries({ queryKey: ['portfolios'] })
    },
    meta: { inlineError: true },
  })
}

export function useDownloadPortfolio() {
  return useMutation({
    mutationFn: (row: PortfolioRow) =>
      downloadAndSave(
        `/offerings/${row.offering_id}/portfolio.pdf`,
        `${row.course_code}_${row.academic_year}_S${row.semester}_portfolio.pdf`,
      ),
  })
}

export function useBulkDownload() {
  return useMutation({
    mutationFn: ({ offeringIds, year }: { offeringIds: string[]; year: number }) =>
      downloadAndSave('/portfolios/download', `ABET_Course_Portfolios_${year}.zip`, {
        method: 'POST',
        body: { offering_ids: offeringIds },
      }),
  })
}

export function useDownloadTemplate() {
  return useMutation({
    mutationFn: (kind: 'instructor-package.zip' | 'course-instructor-mapping.csv') =>
      downloadAndSave(`/templates/${kind}`, kind),
  })
}
