import { MutationCache, QueryClient } from '@tanstack/react-query'
import { ApiError } from '../lib/http'

declare module '@tanstack/react-query' {
  interface Register {
    mutationMeta: {
      /** The calling modal shows the error itself, so skip the global error toast. */
      inlineError?: boolean
    }
  }
}

export function errorText(error: unknown): string {
  return error instanceof Error ? error.message : 'Something went wrong.'
}

export function createQueryClient(onMutationError: (message: string) => void): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        refetchOnWindowFocus: false,
        staleTime: 15_000,
        retry: (failureCount, error) => !(error instanceof ApiError && error.status < 500) && failureCount < 2,
      },
    },
    mutationCache: new MutationCache({
      onError: (error, _variables, _context, mutation) => {
        if (mutation.meta?.inlineError) return
        onMutationError(errorText(error))
      },
    }),
  })
}
