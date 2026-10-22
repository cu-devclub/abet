import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { apiDelete, apiGet, apiPost, authApi } from '../../lib/http'
import type { ActAsUser, Me, Role } from './types'

export const ME_KEY = ['me'] as const

export function useMe() {
  return useQuery({
    queryKey: ME_KEY,
    queryFn: () => apiGet<Me>('/me', undefined, { skipUnauthorized: true }),
    retry: false,
    staleTime: 5 * 60 * 1000,
  })
}

export function useLogout() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async () => {
      if (queryClient.getQueryData<Me>(ME_KEY)?.act_as?.real_user) {
        // Drop the dev act-as cookie first so it does not outlive the session; a failure must not block sign-out.
        await apiDelete('/dev/act-as').catch(() => undefined)
      }
      return apiPost<void>('/auth/logout', undefined, authApi)
    },
    onSettled: () => {
      queryClient.clear()
    },
  })
}

export function useActAsUsers(role: Role, q: string, enabled: boolean) {
  return useQuery({
    queryKey: ['dev-act-as-users', role, q],
    queryFn: async () => (await apiGet<{ users: ActAsUser[] }>('/dev/act-as/users', { role, q })).users,
    enabled,
    staleTime: 0,
  })
}

/** Start/stop acting as another user. Every cached query belongs to the previous identity, so drop them all. */
export function useActAs(onDone: () => void) {
  const queryClient = useQueryClient()
  const done = () => {
    queryClient.clear()
    onDone()
  }
  const start = useMutation({
    mutationFn: (userId: string) => apiPost<void>('/dev/act-as', { user_id: userId }),
    onSuccess: done,
  })
  const stop = useMutation({
    mutationFn: () => apiDelete('/dev/act-as'),
    onSuccess: done,
  })
  return { start, stop }
}
