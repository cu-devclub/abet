import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiDelete, apiGet, apiPatch, apiPost, apiPut, uploadWithProgress } from '../../lib/http'
import type {
  AccessEntry,
  AccessInput,
  ImportPlan,
  ImportRequest,
  Offering,
  OfferingInput,
  OfferingListParams,
  YearsResponse,
} from './types'

interface Items<T> {
  items: T[]
}

export function useYears() {
  return useQuery({
    queryKey: ['years'],
    queryFn: () => apiGet<YearsResponse>('/offerings/years'),
  })
}

export function useOfferings(params: OfferingListParams | null) {
  return useQuery({
    queryKey: ['offerings', params],
    queryFn: async () => (await apiGet<Items<Offering>>('/offerings', { ...params })).items,
    enabled: params !== null,
  })
}

export function useOfferingAccess(offeringId: string, enabled: boolean) {
  return useQuery({
    queryKey: ['offering-access', offeringId],
    queryFn: async () => (await apiGet<Items<AccessEntry>>(`/offerings/${offeringId}/access`)).items,
    enabled,
  })
}

/** Offerings, their access lists, the portfolio list and the year list all depend on offering data. */
function useInvalidateOfferings() {
  const queryClient = useQueryClient()
  return () =>
    Promise.all([
      queryClient.invalidateQueries({ queryKey: ['offerings'] }),
      queryClient.invalidateQueries({ queryKey: ['offering-access'] }),
      queryClient.invalidateQueries({ queryKey: ['portfolios'] }),
      queryClient.invalidateQueries({ queryKey: ['years'] }),
    ])
}

export function useCreateOffering() {
  const invalidate = useInvalidateOfferings()
  return useMutation({
    mutationFn: (input: OfferingInput) => apiPost<Offering>('/offerings', input),
    onSuccess: invalidate,
    meta: { inlineError: true },
  })
}

export function useUpdateOffering() {
  const invalidate = useInvalidateOfferings()
  return useMutation({
    mutationFn: ({ id, ...input }: Partial<OfferingInput> & { id: string }) => apiPatch<Offering>(`/offerings/${id}`, input),
    onSuccess: invalidate,
    meta: { inlineError: true },
  })
}

export function useDeleteOffering() {
  const invalidate = useInvalidateOfferings()
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/offerings/${id}`),
    onSuccess: invalidate,
  })
}

export function useBulkDeleteOfferings() {
  const invalidate = useInvalidateOfferings()
  return useMutation({
    mutationFn: (ids: string[]) => apiPost<{ deleted: number }>('/offerings/bulk-delete', { ids }),
    onSuccess: invalidate,
  })
}

export function useSetAccess() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ offeringId, userId, ...input }: AccessInput & { offeringId: string; userId: string }) =>
      apiPut<AccessEntry>(`/offerings/${offeringId}/access/${userId}`, input),
    onSuccess: (_entry, vars) =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: ['offering-access', vars.offeringId] }),
        queryClient.invalidateQueries({ queryKey: ['offerings'] }),
        queryClient.invalidateQueries({ queryKey: ['portfolios'] }),
      ]),
    meta: { inlineError: true },
  })
}

export function useRemoveAccess() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ offeringId, userId }: { offeringId: string; userId: string }) =>
      apiDelete(`/offerings/${offeringId}/access/${userId}`),
    onSuccess: (_void, vars) =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: ['offering-access', vars.offeringId] }),
        queryClient.invalidateQueries({ queryKey: ['offerings'] }),
        queryClient.invalidateQueries({ queryKey: ['portfolios'] }),
      ]),
  })
}

function importForm({ file, curriculumId, academicYear }: ImportRequest): FormData {
  const form = new FormData()
  form.append('file', file)
  form.append('curriculum_id', curriculumId)
  form.append('academic_year', String(academicYear))
  return form
}

export function usePreviewImport() {
  return useMutation({
    mutationFn: (req: ImportRequest) => uploadWithProgress<ImportPlan>('/offerings/import/preview', importForm(req)),
    meta: { inlineError: true },
  })
}

export function useApplyImport() {
  const invalidate = useInvalidateOfferings()
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (req: ImportRequest) => uploadWithProgress<ImportPlan>('/offerings/import/apply', importForm(req)),
    onSuccess: async () => {
      await invalidate()
      await queryClient.invalidateQueries({ queryKey: ['users'] })
    },
    meta: { inlineError: true },
  })
}
