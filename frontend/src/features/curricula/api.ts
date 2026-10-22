import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiDelete, apiGet, apiPatch, apiPost } from '../../lib/http'
import type { Curriculum, CurriculumInput } from './types'

interface CurriculumList {
  items: Curriculum[]
}

export function useCurricula(departmentId?: string) {
  return useQuery({
    queryKey: ['curricula', departmentId ?? null],
    queryFn: async () => (await apiGet<CurriculumList>('/curricula', { department_id: departmentId })).items,
  })
}

function useInvalidateCurricula() {
  const queryClient = useQueryClient()
  return () => Promise.all([
    queryClient.invalidateQueries({ queryKey: ['curricula'] }),
    queryClient.invalidateQueries({ queryKey: ['departments'] }),
  ])
}

export function useCreateCurriculum() {
  const invalidate = useInvalidateCurricula()
  return useMutation({
    mutationFn: (input: CurriculumInput) => apiPost<Curriculum>('/curricula', input),
    onSuccess: invalidate,
    meta: { inlineError: true },
  })
}

export function useUpdateCurriculum() {
  const invalidate = useInvalidateCurricula()
  return useMutation({
    mutationFn: ({ id, ...input }: CurriculumInput & { id: string }) => apiPatch<Curriculum>(`/curricula/${id}`, input),
    onSuccess: invalidate,
    meta: { inlineError: true },
  })
}

export function useDeleteCurriculum() {
  const invalidate = useInvalidateCurricula()
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/curricula/${id}`),
    onSuccess: invalidate,
    meta: { inlineError: true },
  })
}
