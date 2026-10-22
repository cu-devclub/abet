import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiDelete, apiGet, apiPatch, apiPost } from '../../lib/http'
import type { Department, DepartmentInput } from './types'

interface DepartmentList {
  items: Department[]
}

export function useDepartments() {
  return useQuery({
    queryKey: ['departments'],
    queryFn: async () => (await apiGet<DepartmentList>('/departments')).items,
  })
}

function useInvalidateDepartments() {
  const queryClient = useQueryClient()
  return () => Promise.all([
    queryClient.invalidateQueries({ queryKey: ['departments'] }),
    queryClient.invalidateQueries({ queryKey: ['curricula'] }),
  ])
}

export function useCreateDepartment() {
  const invalidate = useInvalidateDepartments()
  return useMutation({
    mutationFn: (input: DepartmentInput) => apiPost<Department>('/departments', input),
    onSuccess: invalidate,
    meta: { inlineError: true },
  })
}

export function useUpdateDepartment() {
  const invalidate = useInvalidateDepartments()
  return useMutation({
    mutationFn: ({ id, ...input }: DepartmentInput & { id: string }) => apiPatch<Department>(`/departments/${id}`, input),
    onSuccess: invalidate,
    meta: { inlineError: true },
  })
}

export function useDeleteDepartment() {
  const invalidate = useInvalidateDepartments()
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/departments/${id}`),
    onSuccess: invalidate,
    meta: { inlineError: true },
  })
}
