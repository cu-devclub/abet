import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiDelete, apiGet, apiPatch, apiPost } from '../../lib/http'
import type { User } from '../auth/types'
import type { UserCreateInput, UserListParams, UserUpdateInput } from './types'

interface UserList {
  items: User[]
}

export function useUsers(params: UserListParams = {}) {
  return useQuery({
    queryKey: ['users', params],
    queryFn: async () => (await apiGet<UserList>('/users', { ...params })).items,
  })
}

function useInvalidateUsers() {
  const queryClient = useQueryClient()
  return () => queryClient.invalidateQueries({ queryKey: ['users'] })
}

export function useCreateUser() {
  const invalidate = useInvalidateUsers()
  return useMutation({
    mutationFn: (input: UserCreateInput) => apiPost<User>('/users', input),
    onSuccess: invalidate,
    meta: { inlineError: true },
  })
}

export function useUpdateUser() {
  const invalidate = useInvalidateUsers()
  return useMutation({
    mutationFn: ({ id, ...input }: UserUpdateInput & { id: string }) => apiPatch<User>(`/users/${id}`, input),
    onSuccess: invalidate,
    meta: { inlineError: true },
  })
}

export function useDeleteUser() {
  const invalidate = useInvalidateUsers()
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/users/${id}`),
    onSuccess: invalidate,
    meta: { inlineError: true },
  })
}
