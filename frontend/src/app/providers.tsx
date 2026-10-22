import { useEffect, useState, type ReactNode } from 'react'
import { QueryClientProvider } from '@tanstack/react-query'
import { ToastProvider } from '../components/ui/Toast'
import { useToast } from '../components/ui/useToast'
import { ME_KEY } from '../features/auth/api'
import { setUnauthorizedHandler } from '../lib/http'
import { createQueryClient } from './queryClient'
import { router } from './router'

function QueryProvider({ children }: { children: ReactNode }) {
  const toast = useToast()
  const [queryClient] = useState(() => createQueryClient((message) => toast.error(message)))

  useEffect(() => {
    setUnauthorizedHandler(() => {
      void queryClient.invalidateQueries({ queryKey: ME_KEY })
      if (!window.location.pathname.startsWith('/login')) {
        void router.navigate('/login', { state: { from: window.location.pathname + window.location.search } })
      }
    })
    return () => setUnauthorizedHandler(null)
  }, [queryClient])

  return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
}

export function Providers({ children }: { children: ReactNode }) {
  return (
    <ToastProvider>
      <QueryProvider>{children}</QueryProvider>
    </ToastProvider>
  )
}
